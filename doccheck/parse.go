package doccheck

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

// Link is one inline link occurrence, ordered by file, line and column.
type Link struct {
	Line   int
	Column int
	Target string
}

// Document is the parsed view of one markdown file.
type Document struct {
	Anchors []string // generated anchors in document order
	Links   []Link
}

// Parse reads markdown limited to the documented subset: standalone ATX
// headings, fenced code blocks and simple inline links.
func Parse(r io.Reader) *Document {
	doc := &Document{}
	used := map[string]bool{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	fence := byte(0) // 0 outside a fence, '`' or '~' inside
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if fence != 0 {
			if isFenceMarker(trimmed, fence) {
				fence = 0
			}
			continue
		}
		if marker, ok := fenceOpening(trimmed); ok {
			fence = marker
			continue
		}
		if _, text, ok := parseATXHeading(line); ok {
			doc.Anchors = append(doc.Anchors, uniqueAnchor(text, used))
		}
		doc.Links = append(doc.Links, extractLinks(line, lineNo)...)
	}
	return doc
}

func isFenceMarker(trimmed string, ch byte) bool {
	if len(trimmed) != 3 {
		return false
	}
	return trimmed[0] == ch && trimmed[1] == ch && trimmed[2] == ch
}

func fenceOpening(trimmed string) (byte, bool) {
	if isFenceMarker(trimmed, '`') {
		return '`', true
	}
	if isFenceMarker(trimmed, '~') {
		return '~', true
	}
	return 0, false
}

func parseATXHeading(line string) (int, string, bool) {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level+1:]), true
}

// uniqueAnchor builds the GitHub-style slug described in the spec and
// disambiguates duplicates with -2, -3, ... suffixes.
func uniqueAnchor(text string, used map[string]bool) string {
	base := slug(text)
	if base == "" {
		base = "section"
	}
	if !used[base] {
		used[base] = true
		return base
	}
	for n := 2; ; n++ {
		candidate := base + "-" + itoa(n)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func slug(text string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevHyphen = false
		case unicode.IsSpace(r):
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// extractLinks finds [text](dest) occurrences. Content inside paired single
// backticks is skipped; images (![...]) and reference-style links are ignored.
func extractLinks(line string, lineNo int) []Link {
	var links []Link
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '`' {
			if closeAt := strings.IndexByte(line[i+1:], '`'); closeAt >= 0 {
				i += closeAt + 1
				continue
			}
			// Unpaired backtick: ordinary character, keep scanning from it.
			continue
		}
		if c != '[' {
			continue
		}
		if i > 0 && line[i-1] == '!' {
			continue
		}
		closeText := strings.IndexByte(line[i+1:], ']')
		if closeText < 0 {
			continue
		}
		j := i + 1 + closeText
		if j+1 >= len(line) || line[j+1] != '(' {
			continue
		}
		end := strings.IndexByte(line[j+2:], ')')
		if end < 0 {
			continue
		}
		destStart := j + 2
		destEnd := j + 2 + end
		dest := line[destStart:destEnd]
		if !validSimpleDest(dest) {
			i = destEnd
			continue
		}
		links = append(links, Link{Line: lineNo, Column: i + 1, Target: dest})
		i = destEnd
	}
	return links
}

// validSimpleDest enforces "no whitespace or parentheses" for the target.
func validSimpleDest(dest string) bool {
	if dest == "" {
		return false
	}
	for i := 0; i < len(dest); i++ {
		c := dest[i]
		if c == '(' || c == ')' || c == ' ' || c == '\t' {
			return false
		}
	}
	return true
}
