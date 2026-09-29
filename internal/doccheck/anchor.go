package doccheck

import (
	"strings"
	"unicode"
)

// slugify builds the base anchor for heading text:
// lowercase, keep Unicode letters/digits, whitespace to '-', drop the rest,
// collapse and trim hyphens; empty text becomes "section".
func slugify(text string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevHyphen = false
		case unicode.IsSpace(r):
			if b.Len() > 0 && !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	base := strings.Trim(b.String(), "-")
	if base == "" {
		base = "section"
	}
	return base
}

// uniqueAnchors assigns document-order unique anchors to heading texts.
func uniqueAnchors(texts []string) []string {
	used := make(map[string]bool, len(texts))
	anchors := make([]string, len(texts))
	for i, text := range texts {
		base := slugify(text)
		name := base
		if used[name] {
			for n := 2; ; n++ {
				candidate := base + "-" + itoa(n)
				if !used[candidate] {
					name = candidate
					break
				}
			}
		}
		used[name] = true
		anchors[i] = name
	}
	return anchors
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
