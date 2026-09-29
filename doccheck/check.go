package doccheck

import (
	"os"
	"path/filepath"
	"strings"
)

// LinkStatus classifies a single local link.
type LinkStatus string

const (
	StatusOK          LinkStatus = "ok"
	StatusOutsideRoot LinkStatus = "outside_root"
	StatusIllegal     LinkStatus = "illegal_target"
	StatusFileMissing LinkStatus = "file_missing"
	StatusAnchorMiss  LinkStatus = "anchor_missing"
)

// LinkFinding is one reported link occurrence.
type LinkFinding struct {
	Source string     `json:"source"`
	Line   int        `json:"line"`
	Column int        `json:"column"`
	Target string     `json:"target"`
	Path   string     `json:"path,omitempty"`
	Anchor string     `json:"anchor,omitempty"`
	Status LinkStatus `json:"status"`
}

// CheckResult aggregates findings and reachability.
type CheckResult struct {
	Root        string         `json:"root"`
	Entry       string         `json:"entry"`
	Links       []LinkFinding  `json:"links"`
	Unreachable []string       `json:"unreachable_pages"`
	Counts      map[string]int `json:"counts"`
}

type pageInfo struct {
	doc *Document
}

// Check parses every scanned file, validates all local links and computes
// the set of pages reachable from the entry.
func Check(scan *ScanResult) (*CheckResult, error) {
	rels := scan.SortedRels()
	docs := make(map[string]*Document, len(rels))
	for _, rel := range rels {
		f, err := os.Open(scan.Files[rel])
		if err != nil {
			return nil, inputError("cannot read %q: %v", rel, err)
		}
		docs[rel] = Parse(f)
		f.Close()
	}

	res := &CheckResult{
		Root:   scan.RootAbs,
		Entry:  scan.EntryRel,
		Links:  []LinkFinding{},
		Counts: map[string]int{},
	}

	// adjacency for valid markdown-to-markdown links
	adj := map[string]map[string]bool{}

	for _, rel := range rels {
		doc := docs[rel]
		dir := filepath.ToSlash(filepath.Dir(scan.Files[rel]))
		for _, lk := range doc.Links {
			if isIgnoredScheme(lk.Target) {
				continue
			}
			finding := LinkFinding{
				Source: rel,
				Line:   lk.Line,
				Column: lk.Column,
				Target: lk.Target,
			}
			pathPart, anchorPart, hasAnchor := splitTarget(lk.Target)
			finding.Path = pathPart
			if hasAnchor {
				finding.Anchor = anchorPart
			}

			status, targetRel := classifyTarget(scan, rel, dir, pathPart, anchorPart, hasAnchor, docs)
			finding.Status = status
			res.Links = append(res.Links, finding)
			res.Counts[string(status)]++

			if status == StatusOK && targetRel != "" {
				if adj[rel] == nil {
					adj[rel] = map[string]bool{}
				}
				adj[rel][targetRel] = true
			}
		}
	}

	res.Unreachable = reachableDiff(rels, adj, scan.EntryRel)
	res.Counts["unreachable_pages"] = len(res.Unreachable)
	return res, nil
}

// HasProblems reports whether the run should fail with exit code 1.
func (c *CheckResult) HasProblems() bool {
	if len(c.Unreachable) > 0 {
		return true
	}
	for _, f := range c.Links {
		if f.Status != StatusOK {
			return true
		}
	}
	return false
}

func isIgnoredScheme(target string) bool {
	lower := strings.ToLower(target)
	switch {
	case strings.HasPrefix(lower, "http://"),
		strings.HasPrefix(lower, "https://"),
		strings.HasPrefix(lower, "mailto:"):
		return true
	}
	return false
}

// splitTarget separates the optional #anchor. An empty anchor ("a.md#") is
// treated as absent.
func splitTarget(target string) (pathPart, anchor string, hasAnchor bool) {
	if i := strings.IndexByte(target, '#'); i >= 0 {
		pathPart = target[:i]
		anchor = target[i+1:]
		if anchor != "" {
			hasAnchor = true
		}
		return
	}
	return target, "", false
}

func classifyTarget(scan *ScanResult, sourceRel, dir, pathPart, anchor string, hasAnchor bool, docs map[string]*Document) (LinkStatus, string) {
	if strings.ContainsAny(pathPart, `\?`) || filepath.IsAbs(filepath.FromSlash(pathPart)) || strings.HasPrefix(pathPart, "/") {
		return StatusIllegal, ""
	}

	var resolvedAbs string
	if pathPart == "" {
		// Empty path means the containing document itself.
		resolvedAbs = scan.Files[sourceRel]
	} else {
		resolvedAbs = filepath.Join(filepath.FromSlash(dir), filepath.FromSlash(pathPart))
	}

	// Normalize dot segments (filepath.Join already cleans) and verify the
	// result stays inside the root.
	if !withinRoot(scan.RootAbs, resolvedAbs) {
		return StatusOutsideRoot, ""
	}

	// Every segment of the target path must be free of symlinks.
	if traversesSymlink(scan.RootAbs, resolvedAbs) {
		return StatusIllegal, ""
	}

	info, err := os.Lstat(resolvedAbs)
	if err != nil || !info.Mode().IsRegular() {
		return StatusFileMissing, ""
	}

	rel := filepath.ToSlash(mustRel(scan.RootAbs, resolvedAbs))

	if hasAnchor {
		if !strings.HasSuffix(rel, ".md") {
			// Anchors are only meaningful for markdown files.
			return StatusIllegal, ""
		}
		targetDoc, ok := docs[rel]
		if !ok {
			return StatusFileMissing, ""
		}
		if !containsAnchor(targetDoc.Anchors, anchor) {
			return StatusAnchorMiss, ""
		}
	} else if strings.HasSuffix(rel, ".md") {
		if _, ok := docs[rel]; !ok {
			return StatusFileMissing, ""
		}
	}

	targetRel := ""
	if strings.HasSuffix(rel, ".md") {
		targetRel = rel
	}
	return StatusOK, targetRel
}

func traversesSymlink(rootAbs, targetAbs string) bool {
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return true
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	cur := rootAbs
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func containsAnchor(anchors []string, want string) bool {
	for _, a := range anchors {
		if a == want {
			return true
		}
	}
	return false
}

func reachableDiff(rels []string, adj map[string]map[string]bool, entry string) []string {
	seen := map[string]bool{entry: true}
	queue := []string{entry}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for next := range adj[cur] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	var unreachable []string
	for _, rel := range rels {
		if !seen[rel] {
			unreachable = append(unreachable, rel)
		}
	}
	return unreachable
}
