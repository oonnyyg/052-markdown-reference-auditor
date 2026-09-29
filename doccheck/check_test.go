package doccheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func joinLines(lines ...string) string { return strings.Join(lines, "\n") }

func checkTree(t *testing.T, files map[string]string, entry string) *CheckResult {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	scan, err := ScanDocs(root, entry)
	if err != nil {
		t.Fatalf("ScanDocs: %v", err)
	}
	res, err := Check(scan)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return res
}

func statusMap(t *testing.T, res *CheckResult) map[string]LinkStatus {
	t.Helper()
	m := map[string]LinkStatus{}
	for _, f := range res.Links {
		m[f.Target] = f.Status
	}
	return m
}

func TestCheckHappyPathAndReachability(t *testing.T) {
	files := map[string]string{
		"index.md":  "# Index\n\n[a](sub/a.md) [cycle](b.md)\n",
		"sub/a.md":  "# A\n\n[back](../b.md#beta) [self](../index.md#index)\n",
		"b.md":      "# Beta\n\n[a](sub/a.md)\n",
		"orphan.md": "# Orphan\n",
	}
	res := checkTree(t, files, "index.md")
	for target, st := range statusMap(t, res) {
		if st != StatusOK {
			t.Errorf("target %q: %s", target, st)
		}
	}
	if len(res.Unreachable) != 1 || res.Unreachable[0] != "orphan.md" {
		t.Fatalf("unreachable = %v, want [orphan.md]", res.Unreachable)
	}
}

func TestCheckCycleSelfLink(t *testing.T) {
	files := map[string]string{
		"index.md": "# I\n\n[me](#i) [b](b.md)\n",
		"b.md":     "# B\n\n[back](index.md) [me2](#b)\n",
	}
	res := checkTree(t, files, "index.md")
	for target, st := range statusMap(t, res) {
		if st != StatusOK {
			t.Errorf("target %q: %s", target, st)
		}
	}
	if len(res.Unreachable) != 0 {
		t.Fatalf("unreachable = %v", res.Unreachable)
	}
}

func TestCheckMissingAndAnchorStatuses(t *testing.T) {
	files := map[string]string{
		"index.md": joinLines(
			"# Index",
			"",
			"[gone](nope.md)",
			"[bad anchor](b.md#wrong)",
			"[case sensitive](b.md#BETA)",
			"[self bad](#nope)",
			"[external](https://example.com)",
			"[mail](mailto:x@y.z)",
		),
		"b.md": "# Beta\n",
	}
	res := checkTree(t, files, "index.md")
	m := statusMap(t, res)
	if m["nope.md"] != StatusFileMissing {
		t.Errorf("nope.md = %s", m["nope.md"])
	}
	if m["b.md#wrong"] != StatusAnchorMiss {
		t.Errorf("b.md#wrong = %s", m["b.md#wrong"])
	}
	if m["b.md#BETA"] != StatusAnchorMiss {
		t.Errorf("b.md#BETA = %s", m["b.md#BETA"])
	}
	if m["#nope"] != StatusAnchorMiss {
		t.Errorf("#nope = %s", m["#nope"])
	}
	if _, present := m["https://example.com"]; present {
		t.Errorf("https link should be ignored")
	}
}

func TestCheckIllegalAndOutsideTargets(t *testing.T) {
	files := map[string]string{
		"index.md": joinLines(
			"# Index",
			"[abs](/etc/passwd)",
			"[back](../escape.md)",
			"[query](b.md?x=1)",
			"[slash](a\\b.md)",
			"[dot inside](./b.md)",
			"[dotdot inside](sub/../b.md)",
		),
		"sub/x.md": "# X\n[up](../index.md)\n",
		"b.md":     "# B\n",
	}
	res := checkTree(t, files, "index.md")
	m := statusMap(t, res)
	if m["/etc/passwd"] != StatusIllegal {
		t.Errorf("abs = %s", m["/etc/passwd"])
	}
	if m["../escape.md"] != StatusOutsideRoot {
		t.Errorf("outside = %s", m["../escape.md"])
	}
	if m["b.md?x=1"] != StatusIllegal {
		t.Errorf("query = %s", m["b.md?x=1"])
	}
	if m[`a\b.md`] != StatusIllegal {
		t.Errorf("backslash = %s", m[`a\b.md`])
	}
	if m["./b.md"] != StatusOK {
		t.Errorf("./b.md = %s", m["./b.md"])
	}
	if m["sub/../b.md"] != StatusOK {
		t.Errorf("sub/../b.md = %s", m["sub/../b.md"])
	}
}

func TestCheckSymlinkTargetIllegal(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("# Outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "index.md"), "# Index\n\n[link](linked.md)\n")
	if err := os.Symlink(outside, filepath.Join(root, "linked.md")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	scan, err := ScanDocs(root, "index.md")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Check(scan)
	if err != nil {
		t.Fatal(err)
	}
	m := statusMap(t, res)
	if m["linked.md"] != StatusIllegal {
		t.Fatalf("symlink target = %s, want illegal_target", m["linked.md"])
	}
}

func TestCheckCleanTreeHasNoProblems(t *testing.T) {
	files := map[string]string{
		"index.md": "# Index\n\n[b](b.md)\n",
		"b.md":     "# B\n",
	}
	res := checkTree(t, files, "index.md")
	if res.HasProblems() {
		t.Fatalf("expected no problems, got %+v", res)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
