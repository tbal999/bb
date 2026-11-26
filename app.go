package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Config is set once per install (see bb.go).
type Config struct {
	Admin    string // home folder name of the bb admin
	Title    string // shown at the top of the index
	HomeRoot string // folder holding every user's home folder, "/home/" (tests use a temp dir)
}

// App is one user's bb session: everything loaded from disk plus the terminal input.
type App struct {
	cfg  Config
	user string

	boards []Board           // every visible board, oldest first
	pins   []BoardRef        // pinned boards (shared)
	mods   moderation        // moderators and archived boards
	seen   map[BoardRef]int  // message counts when the user last looked (snapshot file)
	opened map[BoardRef]bool // boards opened this session
	prefs  prefs

	in *bufio.Scanner // nil when running a one-off command, so nothing waits for ENTER
}

func newApp(cfg Config, user string) *App {
	return &App{cfg: cfg, user: user, opened: map[BoardRef]bool{}}
}

// homeUser returns the folder name of a home directory directly inside root
// (/home/alice -> alice), or "" for anything else (/root, /var/home/alice, ...).
func homeUser(root, homeDir string) string {
	name := strings.TrimSuffix(strings.TrimPrefix(homeDir, root), "/")
	if !strings.HasPrefix(homeDir, root) || name == "" || strings.Contains(name, "/") {
		return ""
	}
	return name
}

// ---- paths ----

func (a *App) dir(user, sub string) string    { return a.cfg.HomeRoot + user + "/" + sub + "/" }
func (a *App) privateFile(name string) string { return a.dir(a.user, privateDir) + name }
func (a *App) sharedFile(name string) string  { return a.dir(a.cfg.Admin, modDir) + name }
func (a *App) ownBoardFile(title string) string {
	return a.dir(a.user, boardsDir) + title + ".json"
}

// users lists everyone with a home folder.
func (a *App) users() []string {
	entries, _ := os.ReadDir(a.cfg.HomeRoot)
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func (a *App) isAdmin() bool { return a.user == a.cfg.Admin }
func (a *App) isMod() bool   { return a.mods.isModerator(a.user) }

// initialised reports whether the admin has run 'bb init' (which creates the shared files).
func (a *App) initialised() bool {
	_, err := os.Stat(a.sharedFile(anonName))
	return err == nil
}

// setupDirs creates the user's folders. Boards and moderator lists must be readable by everyone.
func (a *App) setupDirs() {
	for _, d := range []struct {
		sub  string
		perm os.FileMode
	}{{boardsDir, 0755}, {modDir, 0755}, {privateDir, 0700}} {
		path := a.dir(a.user, d.sub)
		if err := os.Mkdir(path, d.perm); err == nil {
			_ = os.Chmod(path, d.perm) // beat the umask
		}
	}
}

// ---- loading ----

// load reads everything for a new session.
func (a *App) load() {
	a.prefs = prefs{Browser: "amfora"}
	_ = readJSON(a.privateFile(prefsName), &a.prefs)
	a.seen = readSnapshot(a.privateFile(snapName))
	a.refresh()
}

// refresh re-reads everything other users can change. It never writes.
func (a *App) refresh() {
	a.loadMods()
	a.loadPins()
	a.loadBoards()
}

// ---- what's new ----

// isNew reports whether a board appeared since the user last looked.
func (a *App) isNew(b Board) bool {
	_, known := a.seen[b.BoardRef]
	return !known
}

// unread reports whether a board has posts since the user last looked that they haven't opened yet.
func (a *App) unread(b Board) bool {
	n, known := a.seen[b.BoardRef]
	return known && n != len(b.Messages) && !a.opened[b.BoardRef]
}

func (a *App) anyUnread() bool {
	for _, b := range a.boards {
		if a.unread(b) {
			return true
		}
	}
	return false
}

// whatsNew lists the boards with posts since the user last looked.
func (a *App) whatsNew() []string {
	var titles []string
	for _, b := range a.boards {
		if a.unread(b) {
			titles = append(titles, b.Title)
		}
	}
	return titles
}

// saveSnapshot records the current boards as seen.
func (a *App) saveSnapshot() {
	var entries []snapEntry
	a.seen = map[BoardRef]int{}
	for _, b := range a.boards {
		entries = append(entries, snapEntry{b.BoardRef, len(b.Messages)})
		a.seen[b.BoardRef] = len(b.Messages)
	}
	a.opened = map[BoardRef]bool{}
	a.report(writeJSON(a.privateFile(snapName), struct{ Boards []snapEntry }{entries}, 0600))
}

// ---- terminal helpers ----

// readLine reads one line of input; false at end of input (ctrl-d).
func (a *App) readLine() (string, bool) {
	if a.in == nil || !a.in.Scan() {
		return "", false
	}
	return a.in.Text(), true
}

// pause waits for ENTER in interactive mode.
func (a *App) pause() {
	if a.in != nil {
		fmt.Fprintln(out, "Press ENTER to continue.")
		a.readLine()
	}
}

// report shows an error, pausing in interactive mode so it isn't cleared straight away.
func (a *App) report(err error) {
	if err != nil {
		fmt.Fprintln(out, err)
		a.pause()
	}
}

func timestamp() string { return time.Now().Format("2006-01-02 15:04:05") }

// now is the clock used for "active ... ago" (tests replace it).
var now = time.Now

var errNotFound = errors.New("board not found (it may have been deleted or archived)")

// out is where bb writes (tests silence it).
var out io.Writer = os.Stdout
