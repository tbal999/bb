package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupHome points bb at a fresh fake /home containing an admin, the current user (alice)
// and any extra users, and resets all global state.
func setupHome(t *testing.T, users ...string) string {
	t.Helper()
	root := t.TempDir()
	homeroot = root + "/"
	admin = "admin"
	username = "alice"
	setPaths()
	for _, u := range append([]string{admin, username}, users...) {
		for _, d := range []string{".bb", ".bbsn", ".bbmod"} {
			if err := os.MkdirAll(filepath.Join(root, u, d), 0777); err != nil {
				t.Fatal(err)
			}
		}
	}
	bb, an, pin, mm, aa, ll, per = BB{}, Anon{}, Pin{}, Mod{}, Snap{}, Last{}, Personal{}
	ismod, back = false, 0
	return root
}

// writeRaw writes a board file straight into a user's .bb folder.
func writeRaw(t *testing.T, root, user, title, raw string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, user, ".bb", title+".json"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeBoard(t *testing.T, root, user string, b Board) {
	t.Helper()
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	writeRaw(t, root, user, b.Title, string(out))
}

func texts(rows [][]string) map[string]int {
	m := map[string]int{}
	for _, r := range rows {
		m[r[1]]++
	}
	return m
}

func TestLoadKeepsSameTextFromDifferentUsers(t *testing.T) {
	root := setupHome(t, "bob")
	writeBoard(t, root, "alice", Board{Title: "general", Owner: "alice", Date: "2021-02-12", Contents: [][]string{
		{"2021-02-12 10:00:00", "lol", "alice"},
		{"2021-02-12 10:05:00", "lol", "alice"},
	}})
	writeBoard(t, root, "bob", Board{Title: "general", Owner: "alice", Date: "2021-02-12", Contents: [][]string{
		{"2021-02-12 10:01:00", "lol", "bob"},
		{"2021-02-12 10:02:00", "alice", "bob"}, // text that matches a username used to be dropped
	}})
	bb.Load()
	if len(bb.B) != 1 {
		t.Fatalf("want 1 board, got %d", len(bb.B))
	}
	got := texts(bb.B[0].Contents)
	if got["lol"] != 3 || got["alice"] != 1 {
		t.Fatalf("messages were dropped while merging: %v", bb.B[0].Contents)
	}

	// Loading again into the same BB (as `mod del` does) must not duplicate anything.
	bb.Load()
	if n := len(bb.B[0].Contents); n != 4 {
		t.Fatalf("reload duplicated rows: want 4, got %d", n)
	}
}

func TestMalformedBoardFileDoesNotCrash(t *testing.T) {
	root := setupHome(t, "mallory")
	writeBoard(t, root, "alice", Board{Title: "general", Owner: "alice", Date: "2021-02-12", Contents: [][]string{
		{"2021-02-12 10:00:00", "hi", "alice"},
	}})
	writeRaw(t, root, "mallory", "general", `{"Title":"general","Owner":"alice","Date":"2021-02-12",
		"Contents":[[], ["one field"], ["2021-02-12 10:01:00", "\u001b[2J\u001b]0;pwned\u0007gotcha", "mallory"]]}`)

	bb.Load()
	if !bb.loadboard(0, "") { // sorting used to panic on short rows
		t.Fatal("board did not load")
	}
	got := texts(bb.B[0].Contents)
	if len(bb.B[0].Contents) != 2 || got["hi"] != 1 || got["gotcha"] != 1 {
		t.Fatalf("want the 2 valid rows with escapes stripped, got %q", bb.B[0].Contents)
	}
}

func TestAnonPostsFromOtherUsersSurvive(t *testing.T) {
	root := setupHome(t, "bob")
	writeBoard(t, root, "alice", Board{Title: "general", Owner: "alice", Date: "2021-02-12"})
	an.Save() // admin's `bb init` creates the empty anon file
	bb.Load() // alice's session starts, anon file is empty

	bob := Anon{} // bob posts anonymously from his own session
	bob.Add("general", "2021-02-12", []string{"2021-02-12 10:00:00", "from bob", "???"})
	bob.Save()

	rehash() // alice refreshes - this used to write her stale copy over bob's post
	bb.addtoboard("from alice", "general", "2021-02-12", true)

	saved := Anon{}
	if err := saved.Load(); err != nil {
		t.Fatal(err)
	}
	if len(saved.Board) != 1 {
		t.Fatalf("want 1 anon board, got %+v", saved)
	}
	got := texts(saved.Board[0].Contents)
	if got["from bob"] != 1 || got["from alice"] != 1 {
		t.Fatalf("anon posts lost: %q", saved.Board[0].Contents)
	}

	rehash()
	if got := texts(bb.B[0].Contents); got["from bob"] != 1 || got["from alice"] != 1 {
		t.Fatalf("anon posts not shown on board: %q", bb.B[0].Contents)
	}
}

func TestAnonBoardMatchesDateToo(t *testing.T) {
	root := setupHome(t)
	writeBoard(t, root, "alice", Board{Title: "general", Owner: "alice", Date: "2021-02-12"})
	other := Anon{}
	other.Add("general", "2020-01-01", []string{"2020-01-01 10:00:00", "old board", "???"}) // same title, different board
	other.Save()
	bb.Load()
	if len(bb.B[0].Contents) != 0 {
		t.Fatalf("anon post leaked onto a different board: %q", bb.B[0].Contents)
	}
}

func TestAddToUnknownBoardPostsNothing(t *testing.T) {
	root := setupHome(t)
	bb.addtoboard("hello", "", "", false) // no boards at all: used to panic

	writeBoard(t, root, "alice", Board{Title: "general", Owner: "alice", Date: "2021-02-12"})
	bb.Load()
	bb.addtoboard("hello", "deleted-board", "2021-01-01", false) // used to post to board 0
	rehash()
	if len(bb.B[0].Contents) != 0 {
		t.Fatalf("message landed on the wrong board: %q", bb.B[0].Contents)
	}
}

func TestTamperedSharedFilesDoNotCrash(t *testing.T) {
	setupHome(t)
	if err := os.WriteFile(masterfilepath+pinname, []byte(`{"Title":["a","b"],"Date":["d"]}`), 0666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(masterfilepath+anonname, []byte(`{"Title":["a","b"],"Date":["d"],"Board":[]}`), 0666); err != nil {
		t.Fatal(err)
	}
	if err := pin.Load(); err != nil || len(pin.Title) != 1 || len(pin.Date) != 1 {
		t.Fatalf("pin lists not trimmed to match: %+v (%v)", pin, err)
	}
	if err := an.Load(); err != nil || len(an.Title) != 0 {
		t.Fatalf("anon lists not trimmed to match: %+v (%v)", an, err)
	}
}

// Drives a whole interactive session from piped input, ending with EOF instead of 'q'.
func TestInteractiveSession(t *testing.T) {
	root := setupHome(t)
	writeBoard(t, root, "alice", Board{Title: "general", Owner: "alice", Date: "2021-02-12"})
	bb.Load()

	input := filepath.Join(t.TempDir(), "input")
	script := "new second\n" + // create a board
		"0\n" + // open "general"
		"file is broken\n" + // used to be swallowed as a filter
		"anonymous coward here\n" + // used to be posted anonymously as "mous coward here"
		"review this\n" + // used to be reversed
		"anon secret\n" + // a real anon post
		"\n" // back to the index, then EOF (ctrl-d), which used to redraw forever
	if err := os.WriteFile(input, []byte(script), 0644); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	stdout := os.Stdout
	if devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		os.Stdout = devnull
		defer devnull.Close()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ViewBB("", in, 1)
	}()
	select {
	case <-done:
		os.Stdout = stdout
	case <-time.After(10 * time.Second):
		os.Stdout = stdout
		t.Fatal("ViewBB did not return at end of input")
	}

	if _, err := os.Stat(filepath.Join(root, "alice", ".bb", "second.json")); err != nil {
		t.Errorf("'new second' did not create a board: %v", err)
	}
	saved := Board{}
	saved.Load(filepath.Join(root, "alice", ".bb", "general.json"))
	got := texts(saved.Contents)
	for _, want := range []string{"file is broken", "anonymous coward here", "review this"} {
		if got[want] != 1 {
			t.Errorf("message %q not posted as typed; board has %q", want, saved.Contents)
		}
	}
	if len(saved.Contents) != 3 {
		t.Errorf("want 3 messages in alice's file, got %q", saved.Contents)
	}
	anon := Anon{}
	_ = anon.Load()
	if len(anon.Board) != 1 || len(anon.Board[0].Contents) != 1 || anon.Board[0].Contents[0][1] != "secret" {
		t.Errorf("anon post missing: %+v", anon)
	}
}

func TestPageRange(t *testing.T) {
	defer func() { back = 0 }()
	cases := []struct {
		n, back     int
		search      bool
		first, last int
	}{
		{n: 0, first: 0, last: -1},
		{n: 4, first: 0, last: 3},
		{n: 25, first: 5, last: 24}, // exactly `length` items, not length+1
		{n: 25, back: 5, first: 0, last: 19},
		{n: 25, back: 99, first: 0, last: 19}, // stale scroll from a longer list
		{n: 25, back: 5, search: true, first: 0, last: 24},
	}
	for _, c := range cases {
		back = c.back
		if first, last := pageRange(c.n, c.search); first != c.first || last != c.last {
			t.Errorf("pageRange(%d, back=%d, search=%v) = %d..%d, want %d..%d", c.n, c.back, c.search, first, last, c.first, c.last)
		}
	}
}

func TestDeleteBoardWithDotInTitle(t *testing.T) {
	root := setupHome(t)
	writeBoard(t, root, "alice", Board{Title: "v1.2", Owner: "alice", Date: "2021-02-12"})
	bb.Load()
	bb.delboard(0)
	if _, err := os.Stat(filepath.Join(root, "alice", ".bb", "v1.2.json")); !os.IsNotExist(err) {
		t.Fatalf("board file still there: %v", err)
	}
}

func TestHomeUser(t *testing.T) {
	homeroot = "/home/"
	cases := map[string]string{
		"/home/alice":      "alice",
		"/home/alice/":     "alice",
		"/root":            "",
		"/home/":           "",
		"/var/home/alice":  "",
		"/home/club/alice": "",
	}
	for in, want := range cases {
		if got := homeUser(in); got != want {
			t.Errorf("homeUser(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHelpShowsModAndAdminCommandsOnlyToThem(t *testing.T) {
	setupHome(t)
	defer func() { ismod = false }()
	check := func(who string, wantMod, wantAdmin bool) {
		t.Helper()
		h := helpText()
		if strings.Contains(h, "MODERATOR commands") != wantMod || strings.Contains(h, "ADMIN commands") != wantAdmin {
			t.Errorf("%s: moderator section %v (want %v), admin section %v (want %v)", who,
				strings.Contains(h, "MODERATOR commands"), wantMod, strings.Contains(h, "ADMIN commands"), wantAdmin)
		}
		if !strings.HasSuffix(h, "PRESS ENTER TO CONTINUE...") {
			t.Errorf("%s: help should end with the enter prompt", who)
		}
	}
	username, ismod = "alice", false
	check("user", false, false)
	username, ismod = "bob", true
	check("moderator", true, false)
	username, ismod = "admin", true
	check("admin", true, true)
}

func TestAgo(t *testing.T) {
	cases := map[time.Duration]string{
		-5 * time.Minute:         "just now", // clock skew
		30 * time.Second:         "just now",
		3 * time.Minute:          "3m ago",
		59 * time.Minute:         "59m ago",
		2 * time.Hour:            "2h ago",
		47 * time.Hour:           "1d ago",
		29 * 24 * time.Hour:      "29d ago",
		150 * 24 * time.Hour:     "5mo ago",
		2 * 365 * 24 * time.Hour: "2y ago",
	}
	for d, want := range cases {
		if got := ago(d); got != want {
			t.Errorf("ago(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestLastActive(t *testing.T) {
	b := Board{Contents: [][]string{
		{"2026-11-25 09:00:00", "a", "alice"},
		{"not a time", "b", "mallory"},
		{"2026-11-25 21:30:00", "c", "bob"},
		{"2026-11-24 23:00:00", "d", "alice"},
	}}
	last, ok := b.lastActive()
	if want := time.Date(2026, 11, 25, 21, 30, 0, 0, time.Local); !ok || !last.Equal(want) {
		t.Fatalf("lastActive = %v %v, want %v", last, ok, want)
	}
	if _, ok := (Board{}).lastActive(); ok {
		t.Fatal("a board with no posts has no last activity")
	}
}
