package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

func TestMain(m *testing.M) {
	// bb prints a lot; keep test output readable.
	out = io.Discard
	color.Output = io.Discard
	pterm.DisableOutput()
	os.Exit(m.Run())
}

// pubnix is a fake /home in a temp dir. Each as() is a separate user session.
type pubnix struct {
	t    *testing.T
	root string
}

func newPubnix(t *testing.T) *pubnix {
	p := &pubnix{t: t, root: t.TempDir()}
	if err := p.as("admin").initialise(); err != nil {
		t.Fatal(err)
	}
	return p
}

func (p *pubnix) as(user string) *App {
	p.t.Helper()
	if err := os.MkdirAll(filepath.Join(p.root, user), 0755); err != nil {
		p.t.Fatal(err)
	}
	a := newApp(Config{Admin: "admin", Title: "test bb", HomeRoot: p.root + "/"}, user)
	a.setupDirs()
	a.load()
	return a
}

func (p *pubnix) write(user, rel, content string) {
	p.t.Helper()
	path := filepath.Join(p.root, user, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		p.t.Fatal(err)
	}
}

func (p *pubnix) read(user, rel string) string {
	data, _ := os.ReadFile(filepath.Join(p.root, user, rel))
	return string(data)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// texts counts the messages on a board by "author: text".
func texts(a *App, title string) map[string]int {
	got := map[string]int{}
	for _, b := range a.boards {
		if b.Title == title {
			for _, m := range b.Messages {
				got[m.Author+": "+m.Text]++
			}
		}
	}
	return got
}

func titles(a *App) []string {
	var out []string
	for _, b := range a.boards {
		out = append(out, b.Title)
	}
	return out
}

func refOf(t *testing.T, a *App, title string) BoardRef {
	t.Helper()
	for _, b := range a.boards {
		if b.Title == title {
			return b.BoardRef
		}
	}
	t.Fatalf("no board %q in %v", title, titles(a))
	return BoardRef{}
}

func TestPostsMergeAcrossUsers(t *testing.T) {
	p := newPubnix(t)
	alice, bob := p.as("alice"), p.as("bob")
	must(t, alice.createBoard("general"))
	bob.refresh()
	general := refOf(t, alice, "general")
	must(t, alice.post(general, "lol", false))
	must(t, alice.post(general, "lol", false)) // the same text twice is two messages
	must(t, bob.post(general, "lol", false))
	must(t, bob.post(general, "alice", false)) // text matching a username used to be dropped

	alice.refresh()
	got := texts(alice, "general")
	if got["alice: lol"] != 2 || got["bob: lol"] != 1 || got["bob: alice"] != 1 {
		t.Fatalf("messages lost while merging: %v", got)
	}
	alice.refresh()
	if n := len(alice.boards[0].Messages); n != 4 {
		t.Fatalf("refresh duplicated messages: %d", n)
	}
}

// Files written by bb before the 2026 cleanup must still load, and get rewritten in the new format.
func TestOldFormatFilesStillWork(t *testing.T) {
	p := newPubnix(t)
	p.write("admin", ".bbmod/mod.json", `{"Name":["admin","bob"],"Boardarchive":["old"],"Datearchive":["2021-01-01"]}`)
	p.write("admin", ".bbmod/bban0n.json", `{"Title":["general"],"Date":["2021-02-12"],"Board":[{"Contents":[["2021-02-12 11:00:00","psst","???"]]}]}`)
	p.write("admin", ".bbmod/bbp1n.json", `{"Title":["general","gone"],"Date":["2021-02-12"]}`)
	p.write("alice", ".bb/general.json", `{"Date":"2021-02-12","Owner":"alice","Title":"general",
		"Contents":[["2021-02-12 10:00:00","hi","alice"],[],["only one field"]]}`)
	p.write("alice", ".bb/old.json", `{"Date":"2021-01-01","Owner":"alice","Title":"old","Contents":[]}`)
	p.write("bob", ".bb/general.json", `{"Date":"2021-02-12","Owner":"alice","Title":"general","Contents":[["2021-02-12 10:30:00","hey","bob"]]}`)
	p.write("alice", ".bbsn/bbsn4p.json", `{"Owner":["alice"],"Title":["general"],"Date":["2021-02-12"],"Length":[1],"Checked":[false]}`)
	p.write("alice", ".bbsn/bbl4st.json", `{"Title":"general","Date":"2021-02-12"}`)

	alice := p.as("alice")
	if got := strings.Join(titles(alice), ","); got != "general" {
		t.Fatalf("want only general (old is archived), got %s", got)
	}
	got := texts(alice, "general")
	if len(alice.boards[0].Messages) != 3 || got["alice: hi"] != 1 || got["bob: hey"] != 1 || got["???: psst"] != 1 {
		t.Fatalf("old messages not loaded: %v", got)
	}
	general := alice.boards[0].BoardRef
	if !alice.isPinned(general) || !alice.mods.isModerator("bob") || !alice.mods.isArchived(BoardRef{"old", "2021-01-01"}) {
		t.Fatalf("old pins/moderators/archive not loaded: pins %v mods %+v", alice.pins, alice.mods)
	}
	if news := alice.whatsNew(); len(news) != 1 || news[0] != "general" {
		t.Fatalf("old snapshot not used: whatsNew = %v", news)
	}

	// -a posts to the board in the old "last board" file, and rewrites alice's file in the new format.
	if code := alice.command([]string{"-a", "new post"}, nil); code != 0 {
		t.Fatalf("bb -a failed: %d", code)
	}
	must(t, alice.post(general, "secret", true))
	if f := p.read("alice", ".bb/general.json"); strings.Contains(f, "Contents") || !strings.Contains(f, `"Text": "hi"`) || !strings.Contains(f, "new post") {
		t.Fatalf("board file not converted:\n%s", f)
	}
	if f := p.read("admin", ".bbmod/bban0n.json"); !strings.Contains(f, `"Boards"`) || !strings.Contains(f, "psst") || !strings.Contains(f, "secret") {
		t.Fatalf("anon file not converted:\n%s", f)
	}
}

func TestHostileFilesAreContained(t *testing.T) {
	p := newPubnix(t)
	alice := p.as("alice")
	must(t, alice.createBoard("general"))
	date := alice.boards[0].Date
	p.write("bob", ".bb/general.json", `{"Title":"general","Owner":"alice","Date":"`+date+`","Messages":[
		[], ["one field"], {"Time":3},
		["`+date+` 10:00:00","\u001b]0;pwned\u0007\u001b[2Jgotcha","bob"],
		["`+date+` 10:01:00","I agree with bob","alice"]]}`) // bob signing as alice
	p.write("bob", ".bb/fake.json", `{"Title":"fake","Owner":"alice","Date":"`+date+`","Messages":[]}`) // a board "by alice"
	p.write("bob", ".bb/broken.json", `{not json`)
	p.write("admin", ".bbmod/bban0n.json", `{"Boards":[{"Title":"general","Date":"`+date+`","Messages":[["`+date+` 11:00:00","signed","alice"]]}]}`)

	alice.refresh()
	if got := strings.Join(titles(alice), ","); got != "general" {
		t.Fatalf("fake board shown: %s", got)
	}
	got := texts(alice, "general")
	if len(got) != 2 || got["bob: gotcha"] != 1 || got["???: signed"] != 1 {
		t.Fatalf("want bob's cleaned message and the anon one (forced to ???), got %v", got)
	}
}

func TestAnonPostsFromTwoSessions(t *testing.T) {
	p := newPubnix(t)
	alice, bob := p.as("alice"), p.as("bob")
	must(t, alice.createBoard("general"))
	bob.refresh()
	general := refOf(t, alice, "general")
	must(t, bob.post(general, "from bob", true))
	alice.refresh() // used to write alice's stale copy over bob's post
	must(t, alice.post(general, "from alice", true))
	bob.refresh()
	if got := texts(bob, "general"); got["???: from bob"] != 1 || got["???: from alice"] != 1 {
		t.Fatalf("anon posts lost: %v", got)
	}
}

func TestPostToMissingBoard(t *testing.T) {
	alice := newPubnix(t).as("alice")
	if err := alice.post(BoardRef{"nope", "2021-01-01"}, "hi", false); err != errNotFound {
		t.Fatalf("want errNotFound, got %v", err)
	}
	if code := alice.command([]string{"-a", "hi"}, nil); code != 1 { // no last board yet
		t.Fatalf("bb -a with no last board: want exit 1, got %d", code)
	}
}

func TestDeleteRules(t *testing.T) {
	p := newPubnix(t)
	alice, bob := p.as("alice"), p.as("bob")
	must(t, alice.createBoard("v1.2")) // dots in titles used to break delete
	must(t, alice.deleteBoard(0))
	if len(alice.boards) != 0 || p.read("alice", ".bb/v1.2.json") != "" {
		t.Fatal("owner could not delete an untouched board")
	}

	must(t, alice.createBoard("general"))
	bob.refresh()
	must(t, bob.post(bob.boards[0].BoardRef, "hi", false))
	if err := bob.deleteBoard(0); err == nil || !strings.Contains(err.Error(), "not owner") {
		t.Fatalf("non-owner delete: %v", err)
	}
	if err := alice.deleteBoard(0); err == nil || !strings.Contains(err.Error(), "others have posted") {
		t.Fatalf("delete with others' posts should be refused, got %v", err)
	}
	if p.read("alice", ".bb/general.json") == "" { // used to delete alice's copy, then fail on bob's
		t.Fatal("refused delete still removed the owner's file")
	}
}

func TestArchiveAndRestore(t *testing.T) {
	p := newPubnix(t)
	admin, alice := p.as("admin"), p.as("alice")
	must(t, alice.createBoard("general"))
	if err := alice.archive(0); err == nil {
		t.Fatal("non-moderator archived a board")
	}
	if err := alice.setModerator("alice", true); err == nil {
		t.Fatal("non-admin made a moderator")
	}
	must(t, admin.setModerator("bob", true))
	bob := p.as("bob")
	must(t, bob.archive(0))
	if len(p.as("alice").boards) != 0 {
		t.Fatal("archived board still shown")
	}
	admin.refresh()
	if err := admin.restore(0); err == nil || !strings.Contains(err.Error(), "archived by bob") {
		t.Fatalf("admin restored bob's archive: %v", err)
	}
	must(t, bob.restore(0))
	if len(p.as("alice").boards) != 1 {
		t.Fatal("restored board not shown")
	}
	must(t, admin.setModerator("bob", false))
	if p.as("bob").isMod() {
		t.Fatal("bob still a moderator")
	}
}

func TestPins(t *testing.T) {
	p := newPubnix(t)
	admin, alice := p.as("admin"), p.as("alice")
	must(t, alice.createBoard("general"))
	admin.refresh()
	if err := alice.setPin(0, true); err == nil {
		t.Fatal("non-moderator pinned")
	}
	must(t, admin.setPin(0, true))
	must(t, admin.setPin(0, true)) // pinning twice keeps one pin
	alice.refresh()
	if len(alice.pins) != 1 || !alice.isPinned(alice.boards[0].BoardRef) {
		t.Fatalf("pins: %v", alice.pins)
	}
	must(t, admin.setPin(0, false))
	alice.refresh()
	if len(alice.pins) != 0 {
		t.Fatalf("unpin failed: %v", alice.pins)
	}
}

func TestCommandLine(t *testing.T) {
	alice := newPubnix(t).as("alice")
	cases := []struct {
		args []string
		want int
	}{
		{[]string{"-p", "-c", "x"}, 2}, // one at a time
		{[]string{"-p", "u"}, 2},
		{[]string{"bogus"}, 2},
		{[]string{"-z"}, 2},
		{[]string{"x", "del", "1"}, 2}, // used to archive, like "mod del 1"
		{[]string{"mod", "del", "one"}, 2},
		{[]string{"mod", "+", "bob"}, 1}, // not the admin
		{[]string{"mod", "list"}, 1},     // not a moderator
		{[]string{"-l", "5"}, 1},
		{[]string{"-c", "topic"}, 0},
		{[]string{"-l", "0"}, 0},
		{[]string{"-p"}, 0},
		{[]string{"u"}, 0},
		{[]string{"mod"}, 0},
	}
	for _, c := range cases {
		if got := alice.command(c.args, nil); got != c.want {
			t.Errorf("bb %s: exit %d, want %d", strings.Join(c.args, " "), got, c.want)
		}
	}
}

// Drives a whole interactive session from piped input, ending with EOF instead of 'q'.
func TestInteractiveSession(t *testing.T) {
	p := newPubnix(t)
	alice := p.as("alice")
	must(t, alice.createBoard("general"))
	script := "new second\n" + // index 1 (newer)
		"0\n" + // open "general"
		"file is broken\n" + // used to be swallowed as a filter
		"anonymous coward here\n" + // used to be posted anonymously as "mous coward here"
		"review this\n" + // used to be reversed
		"anon secret\n" +
		"rev olleh\n" +
		"\n" // back to the index, then EOF (ctrl-d)
	done := make(chan struct{})
	go func() {
		defer close(done)
		alice.interactive(strings.NewReader(script))
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("interactive mode did not return at end of input")
	}
	got := texts(alice, "general")
	for _, want := range []string{"alice: file is broken", "alice: anonymous coward here", "alice: review this", "???: secret", "alice: hello"} {
		if got[want] != 1 {
			t.Errorf("missing %q; board has %v", want, got)
		}
	}
	if strings.Join(titles(alice), ",") != "general,second" {
		t.Errorf("'new second' did not create a board: %v", titles(alice))
	}
	if !strings.Contains(p.read("alice", ".bbsn/bbsn4p.json"), "general") {
		t.Error("snapshot not saved on exit")
	}
}

func TestHelpShowsModAndAdminCommandsOnlyToThem(t *testing.T) {
	p := newPubnix(t)
	must(t, p.as("admin").setModerator("bob", true))
	for _, c := range []struct {
		user           string
		mod, adminHelp bool
	}{{"alice", false, false}, {"bob", true, false}, {"admin", true, true}} {
		h := p.as(c.user).help()
		if strings.Contains(h, "MODERATOR commands") != c.mod || strings.Contains(h, "ADMIN commands") != c.adminHelp {
			t.Errorf("%s: wrong help sections", c.user)
		}
	}
}

func TestPageRange(t *testing.T) {
	cases := []struct {
		n, scroll   int
		search      bool
		first, last int
	}{
		{n: 0, first: 0, last: -1},
		{n: 4, first: 0, last: 3},
		{n: 25, first: 5, last: 24}, // exactly a page, not a page + 1
		{n: 25, scroll: 5, first: 0, last: 19},
		{n: 25, scroll: 99, first: 0, last: 19},
		{n: 25, scroll: 5, search: true, first: 0, last: 24},
	}
	for _, c := range cases {
		if first, last := pageRange(c.n, c.scroll, c.search); first != c.first || last != c.last {
			t.Errorf("pageRange(%d, %d, %v) = %d..%d, want %d..%d", c.n, c.scroll, c.search, first, last, c.first, c.last)
		}
	}
}

func TestAgo(t *testing.T) {
	cases := map[time.Duration]string{
		-5 * time.Minute:         "just now", // clock skew
		30 * time.Second:         "just now",
		3 * time.Minute:          "3m ago",
		2 * time.Hour:            "2h ago",
		47 * time.Hour:           "1d ago",
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
	b := Board{Messages: []Message{
		{Time: "2026-11-25 09:00:00"}, {Time: "not a time"}, {Time: "2026-11-25 21:30:00"}, {Time: "2026-11-24 23:00:00"},
	}}
	last, ok := b.lastActive()
	if want := time.Date(2026, 11, 25, 21, 30, 0, 0, time.Local); !ok || !last.Equal(want) {
		t.Fatalf("lastActive = %v %v, want %v", last, ok, want)
	}
	if _, ok := (Board{}).lastActive(); ok {
		t.Fatal("a board with no posts has no last activity")
	}
}

func TestHomeUser(t *testing.T) {
	cases := map[string]string{
		"/home/alice": "alice", "/home/alice/": "alice",
		"/root": "", "/home/": "", "/var/home/alice": "", "/home/club/alice": "",
	}
	for in, want := range cases {
		if got := homeUser("/home/", in); got != want {
			t.Errorf("homeUser(%q) = %q, want %q", in, got, want)
		}
	}
}
