package parser_data

import (
	"reflect"
	"testing"
)

func TestParseDocumentationMarkdownWithCategoryPaths(t *testing.T) {
	source := []byte("### Decoration\n\nCategory: `decoration`\n\n" +
		"| Name | Description | Type | Default | Limits |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| rounding | Corner radius | int | `0` | [0 - 100] |\n\n" +
		"#### Blur\n\nCategory: `decoration.blur`\n\n" +
		"| Name | Description | Type | Default | Limits |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| enabled | Enable blur | bool | `true` | |\n\n" +
		"###### Acrylic\n\nUsed by: `acrylic`  \n" +
		"Category: `decoration.blur.acrylic`\n\n" +
		"| Name | Description | Type | Default | Limits |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| tint | Blur tint | color | `0x123456FF` | |\n")

	sections := parseDocumentationMarkdown(source, 3)
	if len(sections) != 3 {
		t.Fatalf("expected root and two category sections, got %d", len(sections))
	}
	root := sections[0]
	if root.Name() != "Decoration" || len(root.Variables) != 1 {
		t.Fatalf("unexpected root section: %#v", root)
	}
	if len(root.Subsections) != 2 {
		t.Fatalf("expected two category sections, got %d", len(root.Subsections))
	}
	if !reflect.DeepEqual(root.Subsections[0].Path, []string{"Decoration", "blur"}) ||
		!reflect.DeepEqual(root.Subsections[1].Path, []string{"Decoration", "blur", "acrylic"}) {
		t.Fatalf("unexpected category paths: %#v, %#v", root.Subsections[0].Path, root.Subsections[1].Path)
	}
	if got := root.Subsections[1].Variables[0].Default; got != "0x123456FF" {
		t.Fatalf("unexpected parsed default: %q", got)
	}
}
