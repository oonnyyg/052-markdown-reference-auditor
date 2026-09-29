package doccheck

import (
	"strings"
	"testing"
)

func TestParseAnchors(t *testing.T) {
	src := strings.NewReader(strings.Join([]string{
		"# Hello World",
		"## Hello World",
		"### Section with 123 & stuff!",
		"#### ",
		"##### Multiple   Spaces",
		"###### 大标题 Alpha",
		"#not a heading",
		"####### too many",
		"#",
		"##trailing",
	}, "\n"))
	doc := Parse(src)
	want := []string{
		"hello-world",
		"hello-world-2",
		"section-with-123-stuff",
		"section",
		"multiple-spaces",
		"大标题-alpha",
	}
	if len(doc.Anchors) != len(want) {
		t.Fatalf("anchors = %v, want %v", doc.Anchors, want)
	}
	for i := range want {
		if doc.Anchors[i] != want[i] {
			t.Errorf("anchor[%d] = %q, want %q", i, doc.Anchors[i], want[i])
		}
	}
}

func TestParseFencedBlocks(t *testing.T) {
	src := strings.NewReader(strings.Join([]string{
		"# Real",
		"```",
		"# Fake Heading",
		"[fake](missing.md)",
		"```",
		"after [ok](other.md) link",
		"~~~",
		"## Also Fake",
		"``` not a fence since open tilde",
		"~~~",
	}, "\n"))
	doc := Parse(src)
	if len(doc.Anchors) != 1 || doc.Anchors[0] != "real" {
		t.Fatalf("anchors = %v, want [real]", doc.Anchors)
	}
	if len(doc.Links) != 1 || doc.Links[0].Target != "other.md" {
		t.Fatalf("links = %+v, want only other.md", doc.Links)
	}
}

func TestParseInlineCodeAndImages(t *testing.T) {
	src := strings.NewReader(strings.Join([]string{
		"use `[a](x.md)` inline",
		"plain [real](real.md) here",
		"image ![alt](img.md) ignored",
		"unpaired ` then [b](b.md)",
		"nested spans `a` [c](c.md) `b`",
	}, "\n"))
	doc := Parse(src)
	var got []string
	for _, l := range doc.Links {
		got = append(got, l.Target)
	}
	want := []string{"real.md", "b.md", "c.md"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("links = %v, want %v", got, want)
	}
}

func TestParseLinkOrderAndColumns(t *testing.T) {
	src := strings.NewReader("[a](a.md) then [b](b.md)")
	doc := Parse(src)
	if len(doc.Links) != 2 {
		t.Fatalf("got %d links", len(doc.Links))
	}
	if doc.Links[0].Column != 1 || doc.Links[1].Column != 16 {
		t.Fatalf("columns = %d,%d", doc.Links[0].Column, doc.Links[1].Column)
	}
}

func TestParseIgnoresSchemeLinks(t *testing.T) {
	// Scheme filtering happens in Check; parser still records them.
	src := strings.NewReader("[web](https://example.com) [mail](mailto:a@b.c)")
	doc := Parse(src)
	if len(doc.Links) != 2 {
		t.Fatalf("got %d links", len(doc.Links))
	}
}
