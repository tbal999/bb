package main

import (
	"bb/addons"
	"testing"
)

func TestClean(t *testing.T) {
	cases := map[string]string{
		"plain text":                     "plain text",
		"\x1b[2Jcleared":                 "cleared",
		"\x1b[31mred\x1b[0m":             "red",
		"\x1b]0;window title\x07after":   "after",
		"\x1b]8;;http://x\x1b\\link":     "link",
		"bell\x07 and c1 csi \u009b31m":  "bell and c1 csi 31m",
		"tab\there\nnewline":             "tab here newline",
		"unicode stays: héllo ☕ @person": "unicode stays: héllo ☕ @person",
	}
	for in, want := range cases {
		if got := clean(in); got != want {
			t.Errorf("clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstURL(t *testing.T) {
	all := []string{"http://", "https://", "gemini://"}
	cases := map[string]string{
		"gemini://example.org/page":         "gemini://example.org/page",
		"check foo.com and gemini://x.org":  "gemini://x.org", // used to give "gemifoo.com"
		"see https://a.com/x, it's good":    "https://a.com/x",
		"(link: http://b.org/y).":           "http://b.org/y",
		"gemini://g.org then https://h.org": "gemini://g.org",
		"no links here":                     "",
		"just the scheme https://":          "",
	}
	for in, want := range cases {
		if got := firstURL(in, all...); got != want {
			t.Errorf("firstURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseOnlyReversesWithRevCommand(t *testing.T) {
	cases := map[string]string{
		"rev hello":              "olleh",
		"review my patch please": "review my patch please",
		"revert that":            "revert that",
		"rev":                    "rev",
	}
	for in, want := range cases {
		if got := addons.Parse(in); got != want {
			t.Errorf("Parse(%q) = %q, want %q", in, got, want)
		}
	}
}
