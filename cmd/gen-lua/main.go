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

// flattenSubsections returns all (sub)sections found anywhere under the root.
// The Hyprland wiki parser sometimes chains H4 siblings as nested paths; for
// our purposes we treat every section with len(Path) >= 2 as a direct child of
// its root section, keyed by the last path element.
func flattenSubsections(sec pd.SectionDefinition, out *[]pd.SectionDefinition) {
	for _, s := range sec.Subsections {
		*out = append(*out, s)
		flattenSubsections(s, out)
	}
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

	// Per-section classes (root + flattened subsections).
	rootChildren := map[string]map[string]string{}
	for _, root := range roots {
		rootChildren[root.Name()] = map[string]string{}
		subs := []pd.SectionDefinition{}
		flattenSubsections(root, &subs)
		for _, sub := range subs {
			leaf := strings.ToLower(sub.Path[len(sub.Path)-1])
			cls := className([]string{root.Name(), leaf})
			emitClass(&b, cls, sub.Variables, nil)
			rootChildren[root.Name()][leaf] = cls
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
