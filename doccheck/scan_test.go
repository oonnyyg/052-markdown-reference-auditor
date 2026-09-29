package doccheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanDocsInputErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "index.md"), "# I\n")

	cases := []struct {
		name  string
		root  string
		entry string
	}{
		{"empty root", "", "index.md"},
		{"missing root", filepath.Join(root, "nope"), "index.md"},
		{"root is file", filepath.Join(root, "index.md"), "index.md"},
		{"missing entry", root, "nope.md"},
		{"entry outside", root, filepath.Join(t.TempDir(), "x.md")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ScanDocs(tc.root, tc.entry); err == nil {
				t.Fatalf("expected InputError")
			} else {
				var in *InputError
				if !asInput(err, &in) {
					t.Fatalf("error %v is not *InputError", err)
				}
			}
		})
	}
}

func asInput(err error, target **InputError) bool {
	in, ok := err.(*InputError)
	if !ok {
		return false
	}
	*target = in
	return true
}

func TestWriteReportRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "index.md"), "# I\n")
	scan, err := ScanDocs(root, "index.md")
	if err != nil {
		t.Fatal(err)
	}
	res := &CheckResult{Links: []LinkFinding{}, Unreachable: nil, Counts: map[string]int{}}
	err = WriteReport(filepath.Join(root, "index.md"), scan, res)
	if err == nil || !strings.Contains(err.Error(), "overwrite") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}

	reportPath := filepath.Join(root, "out", "report.json")
	if err := WriteReport(reportPath, scan, res); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("report missing: %v", err)
	}
}
