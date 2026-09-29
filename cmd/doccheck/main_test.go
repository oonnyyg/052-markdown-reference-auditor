package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeDoc(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCLICleanTree(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, filepath.Join(root, "index.md"), "# I\n\n[b](b.md)\n")
	writeDoc(t, filepath.Join(root, "b.md"), "# B\n")
	report := filepath.Join(root, "report.json")

	code := run([]string{"-root", root, "-entry", "index.md", "-report", report})
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if _, err := os.Stat(report); !os.IsNotExist(err) {
		t.Fatalf("no report should be written on success, stat err = %v", err)
	}
}

func TestCLIBrokenLinksExit1(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, filepath.Join(root, "index.md"), "# I\n\n[x](missing.md)\n")
	report := filepath.Join(root, "report.json")

	code := run([]string{"-root", root, "-entry", "index.md", "-report", report})
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestCLIInvalidInputExit2(t *testing.T) {
	root := t.TempDir()
	code := run([]string{"-root", root, "-entry", "nope.md", "-report", filepath.Join(root, "r.json")})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if code2 := run([]string{"-root", "", "-entry", "x.md"}); code2 != 2 {
		t.Fatalf("empty root exit = %d, want 2", code2)
	}
}
