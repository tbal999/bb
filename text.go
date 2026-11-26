package main

import (
	"regexp"
	"strings"
	"unicode"
)

// Terminal escape sequences: CSI (colours, cursor moves, clear screen), OSC (window titles, hyperlinks)
// and two-character escapes. Everything on the board comes from other users' files or fetched pages,
// so none of it is allowed to drive the reader's terminal.
var escapeSeq = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)?|\x1b[@-_]`)

// clean strips escape sequences and control characters from text that will be shown to users.
func clean(s string) string {
	s = escapeSeq.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// firstURL returns the earliest URL in input that starts with one of the given schemes,
// with trailing punctuation removed. It returns "" if there is none.
func firstURL(input string, schemes ...string) string {
	start := -1
	for _, s := range schemes {
		if i := strings.Index(input, s); i >= 0 && (start < 0 || i < start) {
			start = i
		}
	}
	if start < 0 {
		return ""
	}
	link := input[start:]
	if end := strings.IndexFunc(link, unicode.IsSpace); end >= 0 {
		link = link[:end]
	}
	link = strings.TrimRight(link, ".,;:!?)]}'\"")
	for _, s := range schemes {
		if link == s {
			return "" // scheme with nothing after it
		}
	}
	return link
}

// grabgeminiurl returns the first gemini:// URL in input, or "".
func grabgeminiurl(input string) string {
	return firstURL(input, "gemini://")
}
