package doccheck

import (
	"bufio"
	"io"
	"strings"
)

// RawLink is one inline link occurrence.
type RawLink struct {
	Line   int
	Order  int
	Target string
}

// ParsedDoc holds heading texts (document order) and local-ish raw links.
type ParsedDoc struct {
	Headings []string
	Links    []RawLink
}

// Parse reads the simplified Markdown subset:
// standalone ATX headings, simple inline links, fenced code blocks
// (``` or ~~~) and single-backtick inline code spans.
func Parse(r io.Reader) ParsedDoc {
	var doc ParsedDoc
	fence := byte(0)
	lineNo := 0

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)

		if fence != 0 {
			if isFenceMarker(trimmed) == fence {
				fence = 0
			}
			continue
		}
		if m := isFenceMarker(trimmed); m != 0 {
			fence = m
			continue
		}

		if h, ok := atxHeading(line); ok {
			doc.Headings = append(doc.Headings, h)
			// Headings on their own line cannot also contain links.
			continue
		}

		for _, t := range extractLinks(maskCodeSpans(line)) {
			doc.Links = append(doc.Links, RawLink{
				Line:   lineNo,
				Order:  len(doc.Links) + 1,
				Target: t,
			})
		}
}
	return doc
}

// isFenceMarker returns the marker byte when s is exactly ``` or ~~~.
func isFenceMarker(s string) byte {
	if s == "```" {
		return '`'
	}
	if s == "~~~" {
		return '~'
	}
	return 0
}

// atxHeading returns heading text when the line starts with 1-6 '#'
// immediately followed by a space.
func atxHeading(line string) (string, bool) {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n < 1 || n > 6 || n >= len(line) || line[n] != ' ' {
		return "", false
	}
	return strings.TrimSpace(line[n+1:]), true
}

// extractLinks finds [text](target) occurrences on a line whose backtick
// spans have already been masked.
func extractLinks(line string) []string {
	var targets []string
	for i := 0; i < len(line); i++ {
		if line[i] == '[' && (i == 0 || line[i-1] != '!') {
			if target, next, ok := matchLink(line, i); ok {
				targets = append(targets, target)
				i = next - 1
				continue
			}
		}
	}
	return targets
}

// maskCodeSpans replaces paired single-backtick spans with spaces so that
// headings and links inside them are invisible. An unpaired backtick is
// kept as an ordinary character.
func maskCodeSpans(line string) string {
	b := []byte(line)
	for i := 0; i < len(b); i++ {
		if b[i] != '`' {
			continue
		}
		close := strings.IndexByte(line[i+1:], '`')
		if close < 0 {
			return string(b)
		}
		for j := i; j <= i+1+close; j++ {
			b[j] = ' '
		}
		i += close + 1
	}
	return string(b)
}

func matchLink(line string, open int) (target string, next int, ok bool) {
	closeB := indexUnescaped(line, open+1, ']')
	if closeB < 0 || closeB == open+1 {
		return "", 0, false
}
	text := line[open+1 : closeB]
	if strings.ContainsAny(text, "[]") {
		return "", 0, false
	}
	if closeB+1 >= len(line) || line[closeB+1] != '(' {
		return "", 0, false
	}
	closeP := indexUnescaped(line, closeB+2, ')')
	if closeP < 0 || closeP == closeB+2 {
		return "", 0, false
	}
	t := line[closeB+2 : closeP]
	if strings.ContainsAny(t, " \t()") {
		return "", 0, false
	}
	return t, closeP + 1, true
}

func indexUnescaped(s string, from int, ch byte) int {
	for i := from; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
}
	return -1
}
