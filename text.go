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

// validRows keeps only well-formed message rows ([timestamp, text, author]) and cleans them.
// Board files live in each user's home folder, so a malformed row must not crash anyone else.
func validRows(rows [][]string) [][]string {
	out := [][]string{}
	for _, row := range rows {
		if len(row) != 3 {
			continue
		}
		out = append(out, []string{clean(row[0]), clean(row[1]), clean(row[2])})
	}
	return out
}

// hasRow reports whether rows already contains an identical message row.
func hasRow(rows [][]string, row []string) bool {
	for _, r := range rows {
		if len(r) != len(row) {
			continue
		}
		same := true
		for i := range r {
			if r[i] != row[i] {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}
	return false
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
