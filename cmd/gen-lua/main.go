// Generates a typed schema for the table accepted by hl.config({...}).
//
// The canonical Lua API (the `hl` global, dispatchers, object types, etc.) is
// already produced by Hyprland upstream's meta/generateLuaStubs.py; we ship
// that file verbatim as library/hl.meta.lua. Upstream's signature for
// hl.config is just `fun(config: table): nil` though, so users get no
// completion inside the config table. This generator fills that gap by
// emitting per-section/per-option ---@class blocks (sourced from the wiki)
// and an `---@overload` for hl.config.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	pd "github.com/hyprland-community/hyprls/parser/data"
)

// Aligned with Hyprland upstream's HL.* aliases (see meta/generateLuaStubs.py).
func luaType(t string) string {
	switch t {
	case "int":
		return "integer|boolean"
	case "font_weight":
		return "integer|string"
	case "bool":
		return "boolean"
	case "float", "floatvalue":
		return "number|boolean"
	case "str", "string":
		return "string"
	case "color":
		return "string"
	case "vec2", "Vec2D":
		return "HL.Vec2Like"
	case "MOD":
		return "string"
	case "gradient":
		return "string|HL.Gradient"
	default:
		return "any"
	}
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func className(path []string) string {
	return "HL.Config." + strings.Join(path, ".")
}

type configCategory struct {
	variables []pd.VariableDefinition
	children  map[string]*configCategory
}

func flattenSubsections(sec pd.SectionDefinition, out *[]pd.SectionDefinition) {
	for _, subsection := range sec.Subsections {
		*out = append(*out, subsection)
		flattenSubsections(subsection, out)
	}
}

func categoryTree(root pd.SectionDefinition) *configCategory {
	tree := &configCategory{variables: root.Variables, children: map[string]*configCategory{}}
	for _, section := range root.Subsections {
		node := tree
		path := section.Path[1:]
		if !section.HasCategoryPath {
			path = []string{strings.ToLower(section.Path[len(section.Path)-1])}
		}
		for _, name := range path {
			name = strings.ReplaceAll(name, "-", "_")
			name = strings.ReplaceAll(name, " ", "_")
			if node.children[name] == nil {
				node.children[name] = &configCategory{children: map[string]*configCategory{}}
			}
			node = node.children[name]
		}
		node.variables = append(node.variables, section.Variables...)
	}
	return tree
}

func emitCategoryClasses(b *strings.Builder, root string, node *configCategory, path []string) map[string]string {
	children := map[string]string{}
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		childPath := append(append([]string(nil), path...), name)
		child := node.children[name]
		children[name] = className(append([]string{root}, childPath...))
		emitCategoryClasses(b, root, child, childPath)
	}
	if len(path) > 0 {
		emitClass(b, className(append([]string{root}, path...)), node.variables, children)
	}
	return children
}

// emitClass writes a ---@class block for a section. Variable names containing
// a dot (e.g. "col.active_border") are grouped under a nested anonymous class.
func emitClass(b *strings.Builder, name string, vars []pd.VariableDefinition, extraFields map[string]string) {
	groups := map[string][]pd.VariableDefinition{}
	flat := []pd.VariableDefinition{}
	for _, v := range vars {
		if i := strings.IndexByte(v.Name, '.'); i >= 0 {
			groups[v.Name[:i]] = append(groups[v.Name[:i]], pd.VariableDefinition{
				Name:        v.Name[i+1:],
				Description: v.Description,
				Type:        v.Type,
				Default:     v.Default,
			})
		} else {
			flat = append(flat, v)
		}
	}

	groupNames := make([]string, 0, len(groups))
	for k := range groups {
		groupNames = append(groupNames, k)
	}
	sort.Strings(groupNames)

	for _, g := range groupNames {
		emitClass(b, name+"."+g, groups[g], nil)
	}

	fmt.Fprintf(b, "---@class (exact) %s\n", name)
	for _, v := range flat {
		desc := sanitize(v.Description)
		if v.Default != "" && v.Default != "[[Empty]]" {
			desc = fmt.Sprintf("%s (default: `%s`)", desc, sanitize(v.Default))
		}
		fmt.Fprintf(b, "---@field %s? %s %s\n", v.Name, luaType(v.Type), desc)
	}
	for _, g := range groupNames {
		fmt.Fprintf(b, "---@field %s? %s.%s\n", g, name, g)
	}
	extraKeys := make([]string, 0, len(extraFields))
	for k := range extraFields {
		extraKeys = append(extraKeys, k)
	}
	sort.Strings(extraKeys)
	for _, k := range extraKeys {
		fmt.Fprintf(b, "---@field %s? %s\n", k, extraFields[k])
	}
	b.WriteString("\n")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-lua <output-file>")
		os.Exit(2)
	}

	roots := []pd.SectionDefinition{}
	for _, s := range pd.Sections {
		if len(s.Path) == 1 {
			roots = append(roots, s)
		}
	}

	var b strings.Builder
	b.WriteString(`---@meta
-- HL.Config — typed schema for the table accepted by hl.config({...}).
--
-- Auto-generated from the bundled Hyprland wiki by cmd/gen-lua.
-- The canonical hl.* API is provided by library/hl.meta.lua, generated
-- upstream by Hyprland's own meta/generateLuaStubs.py. This file augments
-- that stub with rich, per-option documentation for hl.config().
--
-- See https://wiki.hypr.land/Configuring/Configuring-Hyprland/

`)

	// Per-section classes, preserving nested category paths from the wiki.
	rootChildren := map[string]map[string]string{}
	for _, root := range roots {
		usesWikiCategories := false
		for _, subsection := range root.Subsections {
			if subsection.HasCategoryPath {
				usesWikiCategories = true
				break
			}
		}
		rootChildren[root.Name()] = map[string]string{}
		if usesWikiCategories {
			rootChildren[root.Name()] = emitCategoryClasses(&b, root.Name(), categoryTree(root), nil)
			continue
		}

		subsections := []pd.SectionDefinition{}
		flattenSubsections(root, &subsections)
		for _, subsection := range subsections {
			leaf := strings.ToLower(subsection.Path[len(subsection.Path)-1])
			class := className([]string{root.Name(), leaf})
			emitClass(&b, class, subsection.Variables, nil)
			rootChildren[root.Name()][leaf] = class
		}
	}

	// Root section classes.
	for _, root := range roots {
		emitClass(&b, className([]string{root.Name()}), root.Variables, rootChildren[root.Name()])
	}

	// Top-level configuration class accepted by hl.config().
	b.WriteString("---@class HL.Config\n")
	for _, root := range roots {
		fmt.Fprintf(&b, "---@field %s? %s\n", strings.ToLower(root.Name()), className([]string{root.Name()}))
	}
	b.WriteString("\n")

	// Override the upstream signature of hl.config so users get completion
	// inside the table literal. The duplicate-set-field diagnostic is
	// expected (we are augmenting the canonical stub).
	b.WriteString(`---@diagnostic disable: duplicate-set-field
---Apply (or merge) Hyprland configuration values.
---@overload fun(config: HL.Config): nil
function hl.config(config) end
`)

	if err := os.WriteFile(os.Args[1], []byte(b.String()), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
