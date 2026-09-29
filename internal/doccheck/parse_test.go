package doccheck

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseHeadingsAndLinks(t *testing.T) {
	src := strings.Join([]string{
		"# Title One",
		"## Section",
		"see [a](other.md) and [b](x.md#h)",
		"![img](pic.png) should be skipped",
		"code: `[q](code.go)` and [real](real.md)",
		"unpaired ` tick [still](z.md)",
	}, "\n")
	doc := Parse(strings.NewReader(src))
	wantHeadings := []string{"Title One", "Section"}
	if !reflect.DeepEqual(doc.Headings, wantHeadings) {
		t.Fatalf("headings = %v, want %v", doc.Headings, wantHeadings)
	}
	var targets []string
	for _, l := range doc.Links {
		targets = append(targets, l.Target)
	}
	wantLinks := []string{"other.md", "x.md#h", "real.md", "z.md"}
	if !reflect.DeepEqual(targets, wantLinks) {
		t.Fatalf("links = %v, want %v", targets, wantLinks)
	}
	if doc.Links[0].Line != 3 || doc.Links[0].Order != 1 {
		t.Fatalf("line/order = %d/%d", doc.Links[0].Line, doc.Links[0].Order)
	}
}

func TestParseFencedBlocks(t *testing.T) {
	for _, marker := range []string{"```", "~~~"} {
		src := strings.Join([]string{
			"# Before",
			marker,
			"# Not a heading",
			"[x](hidden.md)",
			marker,
			"# After",
			"[y](shown.md)",
		}, "\n")
		doc := Parse(strings.NewReader(src))
		if !reflect.DeepEqual(doc.Headings, []string{"Before", "After"}) {
			t.Fatalf("%s: headings = %v", marker, doc.Headings)
		}
		if len(doc.Links) != 1 || doc.Links[0].Target != "shown.md" {
			t.Fatalf("%s: links = %+v", marker, doc.Links)
		}
	}
}

func TestParseFenceMismatch(t *testing.T) {
	src := "# H\n```\n# Ignored\n~~~\n# Visible\n"
	doc := Parse(strings.NewReader(src))
	// ~~~ does not close a ``` fence; everything until EOF is ignored.
	if !reflect.DeepEqual(doc.Headings, []string{"H"}) {
		t.Fatalf("headings = %v", doc.Headings)
	}
}

func TestParseBadHeadings(t *testing.T) {
	src := "#No space\n##  \n####### Too deep\n  # leading spaces ignored? line does not start at col 0?\n## Real"
	doc := Parse(strings.NewReader(src))
	if !reflect.DeepEqual(doc.Headings, []string{"", "Real"}) {
		t.Fatalf("headings = %#v", doc.Headings)
	}
}
