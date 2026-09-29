package doccheck

import (
	"reflect"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":      "hello-world",
		"  Multiple   Gaps ": "multiple-gaps",
		"Foo - Bar":       "foo-bar",
		"Foo---Bar":       "foobar",
		"Café & Résumé":    "café-résumé",
		"100% done!":      "100-done",
		"!!!":             "section",
		"":                "section",
		"中文 标题":         "中文-标题",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueAnchors(t *testing.T) {
	got := uniqueAnchors([]string{"Setup", "setup", "Setup", "###", "###"})
	want := []string{"setup", "setup-2", "setup-3", "section", "section-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
