package doccheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InputError describes an invalid invocation (bad root, missing entry, etc.).
type InputError struct{ Msg string }

func (e *InputError) Error() string { return e.Msg }

func inputError(format string, args ...any) error {
	return &InputError{Msg: "doccheck: " + sprintf(format, args...)}
}

// ScanResult holds the absolute root and every scanned markdown file,
// keyed by slash-separated path relative to the root.
type ScanResult struct {
	RootAbs  string
	EntryRel string
	Files    map[string]string // rel path -> absolute path
}

// ScanDocs validates the document root and entry, then enumerates regular
// .md files below the root. Symlinks are never followed.
func ScanDocs(root, entry string) (*ScanResult, error) {
	if strings.TrimSpace(root) == "" {
		return nil, inputError("document root must not be empty")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, inputError("invalid document root %q: %v", root, err)
	}
	info, err := os.Lstat(rootAbs)
	if err != nil {
		return nil, inputError("document root %q not found", root)
	}
	if isSymlink(info) {
		return nil, inputError("document root %q must not be a symlink", root)
	}
	if !info.IsDir() {
		return nil, inputError("document root %q must be a directory", root)
	}

	entryAbs, err := resolveEntry(rootAbs, entry)
	if err != nil {
		return nil, err
	}

	files := map[string]string{}
	walkErr := filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() || isDirSymlink(path) {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Type().IsRegular() && strings.HasSuffix(path, ".md") {
			rel, rerr := filepath.Rel(rootAbs, path)
			if rerr != nil {
				return rerr
			}
			files[filepath.ToSlash(rel)] = path
		}
		return nil
	})
	if walkErr != nil {
		return nil, inputError("cannot scan document root %q: %v", root, walkErr)
	}

	res := &ScanResult{RootAbs: rootAbs, Files: files}
	res.EntryRel = filepath.ToSlash(mustRel(rootAbs, entryAbs))
	if _, ok := files[res.EntryRel]; !ok {
		return nil, inputError("entry %q is not a scanned markdown file inside the root", entry)
	}
	return res, nil
}

func resolveEntry(rootAbs, entry string) (string, error) {
	if strings.TrimSpace(entry) == "" {
		return "", inputError("entry file must not be empty")
	}
	var entryAbs string
	if filepath.IsAbs(entry) {
		entryAbs = filepath.Clean(entry)
	} else {
		entryAbs = filepath.Join(rootAbs, filepath.FromSlash(entry))
	}
	info, err := os.Lstat(entryAbs)
	if err != nil {
		return "", inputError("entry file %q not found", entry)
	}
	if isSymlink(info) {
		return "", inputError("entry file %q must not be a symlink", entry)
	}
	if !info.Mode().IsRegular() {
		return "", inputError("entry file %q must be a regular file", entry)
	}
	if !withinRoot(rootAbs, entryAbs) {
		return "", inputError("entry file %q must be inside the document root", entry)
	}
	if !strings.HasSuffix(entryAbs, ".md") {
		return "", inputError("entry file %q must be a markdown (.md) file", entry)
	}
	return entryAbs, nil
}

func withinRoot(rootAbs, p string) bool {
	rel, err := filepath.Rel(rootAbs, p)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel != ".." && !strings.HasPrefix(rel, "../") && !filepath.IsAbs(rel)
}

func isSymlink(info os.FileInfo) bool { return info.Mode()&fs.ModeSymlink != 0 }

func isDirSymlink(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func mustRel(base, p string) string {
	rel, err := filepath.Rel(base, p)
	if err != nil {
		panic(err)
	}
	return rel
}

// SortedRels returns the scanned file relative paths sorted.
func (s *ScanResult) SortedRels() []string {
	rels := make([]string, 0, len(s.Files))
	for rel := range s.Files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	return rels
}
