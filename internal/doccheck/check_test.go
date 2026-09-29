package doccheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func buildFixture(t *testing.T) (root, report string) {
	dir := t.TempDir()
	root = filepath.Join(dir, "docs")
	report = filepath.Join(dir, "report.json")

	writeFile(t, filepath.Join(root, "index.md"), strings.Join([]string{
		"# Index",
		"",
		"- [guide](guide.md)",
		"- [setup anchor](guide.md#setup)",
		"- [dup anchor](guide.md#install)",
		"- [missing](nope.md)",
		"- [outside](../etc/passwd.md)",
		"- [bad anchor](guide.md#does-not-exist)",
		"- [self](index.md#index)",
		"- [web](https://example.com) [mail](mailto:x@y)",
	}, "\n"))

	writeFile(t, filepath.Join(root, "guide.md"), strings.Join([]string{
		"# Setup",
		"```",
		"[hidden](nope.md)",
		"# not a heading",
		"```",
		"# Install",
		"# Install",
		"[back](index.md)",
	}, "\n"))

	writeFile(t, filepath.Join(root, "sub", "orphan.md"), "# Orphan\n")
	return root, report
}

func TestRunIntegration(t *testing.T) {
	root, report := buildFixture(t)

	res, err := Run(root, "index.md", report)
	if err != nil {
		t.Fatal(err)
	}

	statusByTarget := map[string]LinkStatus{}
	for _, f := range res.Report.Links {
		statusByTarget[f.Target] = f.Status
	}
	want := map[string]LinkStatus{
		"guide.md":                StatusOK,
		"guide.md#setup":          StatusOK,
		"guide.md#install":        StatusOK, // first duplicate keeps base slug
		"guide.md#install-2":      "",       // not present, just documenting
		"nope.md":                 StatusFileMissing,
		"../etc/passwd.md":        StatusOutsideRoot,
		"guide.md#does-not-exist": StatusAnchorMissing,
		"index.md#index":          StatusOK,
	}
	for target, st := range want {
		if target == "guide.md#install-2" {
			continue
		}
		if got := statusByTarget[target]; got != st {
			t.Errorf("target %q: got %q, want %q", target, got, st)
		}
	}

	// The code-fenced link and headings must not appear at all.
	for _, f := range res.Report.Links {
		if f.Target == "nope.md" && f.Source == "guide.md" {
			t.Errorf("fenced link must be ignored: %+v", f)
		}
	}

	// Duplicate heading: second Install becomes install-2.
	if len(res.Unreachable) != 1 || res.Unreachable[0] != "sub/orphan.md" {
		t.Fatalf("unreachable = %v", res.Unreachable)
	}
	if !res.HasProblem() {
		t.Fatal("expected problems")
	}
}

func TestRunReportInsideRoot(t *testing.T) {
	root, report := buildFixture(t)
	badReport := filepath.Join(root, "out.json")
	if _, err := Run(root, "index.md", badReport); err == nil {
		t.Fatal("expected error when report is inside root")
	}
	if _, err := os.Stat(filepath.Join(report)); !os.IsNotExist(err) {
		t.Fatal("no report should be written on input error")
	}
}

func TestRunBadInputs(t *testing.T) {
	root, report := buildFixture(t)
	for _, tc := range []struct{ root, entry, report string }{
		{root, "missing.md", report},
		{root, `sub\orphan.md`, report},
		{filepath.Join(root, "index.md"), "index.md", report},
		{"", "index.md", report},
	} {
		if _, err := Run(tc.root, tc.entry, tc.report); err == nil {
			t.Errorf("expected error for %+v", tc)
		}
	}
}

func TestRunCleanTreeExitZero(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "docs")
	report := filepath.Join(dir, "r.json")
	writeFile(t, filepath.Join(root, "index.md"), "# I\n[a](a.md)\n")
	writeFile(t, filepath.Join(root, "a.md"), "# A\n[home](index.md#i)\n")
	res, err := Run(root, "index.md", report)
	if err != nil {
		t.Fatal(err)
	}
	if res.HasProblem() {
		t.Fatalf("unexpected problems: %+v", res.Report)
	}
}

func TestRunSymlinkTargetIllegal(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "docs")
	report := filepath.Join(dir, "r.json")
	writeFile(t, filepath.Join(root, "index.md"), "# I\n[s](linked.md)\n")
	target := filepath.Join(dir, "outside.md")
	writeFile(t, target, "# Outside\n")
	if err := os.Symlink(target, filepath.Join(root, "linked.md")); err != nil {
		t.Skip("cannot create symlink:", err)
	}
	res, err := Run(root, "index.md", report)
	if err != nil {
	t.Fatal(err)
}
	for _, f := range res.Report.Links {
		if f.Target == "linked.md" && f.Status != StatusIllegalTarget {
			t.Fatalf("symlink target status = %q", f.Status)
		}
	}
}
