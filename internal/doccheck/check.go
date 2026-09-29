package doccheck

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Run performs the full audit. Input problems return an error (exit code 2).
func Run(rootArg, entryArg, reportArg string) (*Result, error) {
	if rootArg == "" || entryArg == "" || reportArg == "" {
		return nil, errors.New("-root, -entry and -report are required")
	}
	if strings.ContainsAny(entryArg, `\`) || filepath.IsAbs(entryArg) {
		return nil, errors.New("entry must be a forward-slash relative path inside root")
	}

	rootAbs, err := filepath.Abs(rootArg)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(rootAbs)
	if err != nil {
		return nil, errors.New("cannot access root: " + err.Error())
	}
	if !info.IsDir() {
		return nil, errors.New("root is not a directory")
	}

	reportAbs, err := filepath.Abs(reportArg)
	if err != nil {
		return nil, err
	}
	if within(rootAbs, reportAbs) {
		return nil, errors.New("report path must not be inside the documentation root")
	}

	entryRel := cleanRel(entryArg)
	if entryRel == "" || strings.HasSuffix(entryRel, "/") {
		return nil, errors.New("invalid entry path")
	}
	entryAbs := filepath.Join(rootAbs, filepath.FromSlash(entryRel))
	if !within(rootAbs, entryAbs) {
		return nil, errors.New("entry is outside root")
	}
	einfo, err := os.Lstat(entryAbs)
	if err != nil {
		return nil, errors.New("cannot access entry: " + err.Error())
	}
	if einfo.Mode()&fs.ModeSymlink != 0 || !einfo.Mode().IsRegular() {
		return nil, errors.New("entry must be a regular file")
	}
	if !strings.HasSuffix(strings.ToLower(entryRel), ".md") {
		return nil, errors.New("entry must be a Markdown (.md) file")
	}

	pages, err := scanTree(rootAbs)
	if err != nil {
		return nil, err
	}
	if _, ok := pages[entryRel]; !ok {
		return nil, errors.New("entry was not found among scanned files")
	}

	res := &Result{Report: PageReport{
		Root:  filepath.Clean(rootArg),
		Entry: entryRel,
		Links: []LinkFinding{},
	}}

	// anchors per page, built during parsing
	anchors := map[string]map[string]bool{}
	pageList := make([]string, 0, len(pages))
	for rel := range pages {
		pageList = append(pageList, rel)
	}
	sort.Strings(pageList)
	for _, rel := range pageList {
		doc := pages[rel]
		anchors[rel] = map[string]bool{}
		for _, a := range uniqueAnchors(doc.Headings) {
			anchors[rel][a] = true
		}
	}

	graph := map[string]map[string]bool{}
	for _, rel := range pageList {
		doc := pages[rel]
		orderInFile := 0
		for _, l := range doc.Links {
			if isIgnoredScheme(l.Target) {
				continue
			}
			f := LinkFinding{Source: rel, Line: l.Line, Order: l.Order, Target: l.Target}
			status := classify(rootAbs, rel, l.Target, anchors)
			f.Status = status.status
			f.Path = status.pathRel
			f.Anchor = status.anchor
			f.Detail = status.detail
			orderInFile++
			f.Order = orderInFile
			res.Report.Links = append(res.Report.Links, f)
			if status.status == StatusOK && status.pageTarget != "" && status.pageTarget != rel {
				if graph[rel] == nil {
					graph[rel] = map[string]bool{}
				}
				graph[rel][status.pageTarget] = true
			}
		}
	}

	reachable := map[string]bool{entryRel: true}
	queue := []string{entryRel}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for next := range graph[cur] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}
	for _, rel := range pageList {
		if !reachable[rel] {
			res.Unreachable = append(res.Unreachable, rel)
		}
	}
	res.Report.Unreachable = res.Unreachable

	for _, f := range res.Report.Links {
		res.Report.Counts.TotalLinks++
		switch f.Status {
		case StatusOK:
			res.Report.Counts.OK++
		case StatusOutsideRoot:
			res.Report.Counts.OutsideRoot++
		case StatusFileMissing:
			res.Report.Counts.FileMissing++
		case StatusIllegalTarget:
			res.Report.Counts.IllegalTarget++
		case StatusAnchorMissing:
			res.Report.Counts.AnchorMissing++
		}
	}
	res.Report.Counts.Unreachable = len(res.Unreachable)
	return res, nil
}

func isIgnoredScheme(target string) bool {
	for _, prefix := range []string{"http://", "https://", "mailto:"} {
		if strings.HasPrefix(strings.ToLower(target), prefix) {
			return true
		}
	}
	return false
}

type classification struct {
	status     LinkStatus
	pathRel    string
	anchor     string
	pageTarget string
	detail     string
}

func classify(rootAbs, sourceRel, target string, anchors map[string]map[string]bool) classification {
	c := classification{status: StatusIllegalTarget}

	if strings.Contains(target, "\\") {
		c.detail = "backslash is not allowed"
		return c
	}
	if strings.Contains(target, "?") {
		c.detail = "query string is not allowed"
		return c
	}
	pathPart := target
	anchor := ""
	if i := strings.IndexByte(target, '#'); i >= 0 {
		pathPart = target[:i]
		anchor = target[i+1:]
	}
	c.anchor = anchor
	if strings.HasPrefix(pathPart, "/") {
		c.detail = "absolute path is not allowed"
		return c
	}
	if strings.Contains(pathPart, ":") {
		c.detail = "drive letters and schemes are not allowed"
		return c
	}

	base := filepath.Dir(filepath.FromSlash(sourceRel))
	var rel string
	if pathPart == "" {
		rel = sourceRel
	} else {
		rel = cleanRel(filepath.ToSlash(filepath.Join(base, pathPart)))
	}
	c.pathRel = rel
	if rel == "" || strings.HasPrefix(rel, "../") || rel == ".." {
		c.status = StatusOutsideRoot
		c.detail = "target escapes the documentation root"
		return c
	}

	// Any symlink along the path makes the target illegal.
	segs := strings.Split(rel, "/")
	cur := rootAbs
	for _, s := range segs {
		cur = filepath.Join(cur, s)
		fi, err := os.Lstat(cur)
		if err != nil {
			c.status = StatusFileMissing
			c.detail = "no such file"
			return c
		}
		if fi.Mode()&fs.ModeSymlink != 0 {
			c.status = StatusIllegalTarget
			c.detail = "path component is a symbolic link: " + s
			return c
		}
	}
	fi, err := os.Lstat(filepath.Join(rootAbs, filepath.FromSlash(rel)))
	if err != nil || !fi.Mode().IsRegular() {
		c.status = StatusFileMissing
		c.detail = "target is not an existing regular file"
		return c
	}

	if anchor != "" || strings.HasPrefix(target, "#") {
		if !strings.HasSuffix(strings.ToLower(rel), ".md") {
			c.status = StatusAnchorMissing
			c.detail = "anchors are only valid in Markdown files"
			return c
		}
		if !anchors[rel][anchor] {
			c.status = StatusAnchorMissing
			c.detail = "anchor does not exist"
			return c
		}
	}

	c.status = StatusOK
	if strings.HasSuffix(strings.ToLower(rel), ".md") {
		c.pageTarget = rel
	}
	return c
}

func scanTree(rootAbs string) (map[string]ParsedDoc, error) {
	pages := map[string]ParsedDoc{}
	err := filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootAbs {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		rel := filepath.ToSlash(path[len(rootAbs)+1:])
		pages[rel] = Parse(f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func cleanRel(p string) string {
	p = filepath.ToSlash(p)
	var stack []string
	for _, s := range strings.Split(p, "/") {
		switch s {
		case "", ".":
		case "..":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			} else {
				return ".."
			}
		default:
			stack = append(stack, s)
		}
}
	return strings.Join(stack, "/")
}

func within(dir, target string) bool {
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(rel)
}

// WriteReport writes the JSON report, creating parent directories as needed.
func WriteReport(path string, res *Result) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(res.Report)
}
