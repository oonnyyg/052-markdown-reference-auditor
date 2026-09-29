package doccheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// WriteReport writes the JSON result. It refuses to overwrite any document
// that was part of the scan.
func WriteReport(path string, scan *ScanResult, res *CheckResult) error {
	if strings.TrimSpace(path) == "" {
		return inputError("report path must not be empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return inputError("invalid report path %q: %v", path, err)
	}
	for _, docAbs := range scan.Files {
		if samePath(abs, docAbs) {
			return inputError("report path %q would overwrite a checked document", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return inputError("cannot create report directory for %q: %v", path, err)
	}

	f, err := os.Create(abs)
	if err != nil {
		return inputError("cannot create report %q: %v", path, err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	werr := enc.Encode(res)
	cerr := f.Close()
	if werr != nil {
		return inputError("cannot write report %q: %v", path, werr)
	}
	if cerr != nil {
		return inputError("cannot close report %q: %v", path, cerr)
	}
	return nil
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
