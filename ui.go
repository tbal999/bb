package main

// The interactive screens: the index of boards, and a single board.

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

const pageSize = 20 // boards or messages shown at once

const helpText = `
===BB HELP===
for INDEX section:
	<number> - open the board at that index
	new - create a new board i.e 'new topictitle'
	del - delete a board by index i.e 'del 3'
		Only your own boards, and only while nobody else has posted on them.
	fil - filter index by search string e.g YYYY-MM or Title
	q - to quit, or use ctrl-c / ctrl-d
	r - refresh the index section
	w - scroll up the index
	s - scroll down the index
	b - choose gemini client (default=amfora)

for CHAT section:
	q - exits back to index section
	r - refresh the board you are on
	fil - filter chat by specific string e.g YYYY-MM or substring
	w - scroll up the board
	s - scroll down the board
	l - visit a gemini url via client
	anon - make message anonymous i.e 'anon hello'
	rev - reverses your text i.e 'rev hello'
	anything else - types text to board
	nothing - also exits back to index section
	ctrl-c to quit

FYI:
	- For gemini client functionality you need to run bb inside tmux
	- Boards glow cyan when new content is posted
	- New boards glow green.
	- The date is when a board was created; "active ... ago" is its newest post
	- You can comment other people via @ sign i.e @person
		they will see message highlighted
	- If you are on a board and new content is posted on another board,
		you'll see '^new' beside author name
`

const modHelp = `
MODERATOR commands (you are a moderator):
	pin+ / pin- - pin or unpin a board by index (in the INDEX section)
	bb mod del index - archive a board for everyone (from the shell; see bb -p for indexes)
	bb mod list - list archived boards
	bb mod restore n - restore a board you archived (n from bb mod list)
	bb mod - show the list of moderators
`

const adminHelp = `
ADMIN commands (you are the admin):
	bb init - set up the shared files (once)
	bb mod + user - make user a moderator
	bb mod - user - remove user as a moderator
`

func init() { pterm.EnableOutput() }

// help is the 'h' screen, with the extra commands moderators and the admin can use.
func (a *App) help() string {
	text := helpText
	if a.isMod() {
		text += modHelp
	}
	if a.isAdmin() {
		text += adminHelp
	}
	return text
}

// interactive runs the index screen until the user quits.
func (a *App) interactive(in io.Reader) {
	a.in = bufio.NewScanner(in) // one scanner for the whole session, or piped input loses lines
	scroll, search := 0, ""
	for {
		clearScreen()
		a.printIndex(search, scroll)
		search = ""
		fmt.Fprintf(out, "--enter 'h' for help--\nPick Index >> ")
		line, ok := a.readLine()
		if !ok { // ctrl-d
			a.saveSnapshot()
			return
		}
		cmd := strings.TrimSpace(line)
		switch {
		case cmd == "q":
			a.saveSnapshot()
			return
		case cmd == "r":
			a.saveSnapshot()
			a.refresh()
		case cmd == "w":
			scroll = min(scroll+pageSize, max(0, len(a.boards)-pageSize))
		case cmd == "s":
			scroll = max(0, scroll-pageSize)
		case cmd == "h":
			clearScreen()
			fmt.Fprintln(out, a.help())
			a.pause()
		case cmd == "b":
			fmt.Fprintf(out, "Enter client name/path > ")
			if browser, ok := a.readLine(); ok && strings.TrimSpace(browser) != "" {
				a.prefs.Browser = strings.TrimSpace(browser)
				a.report(writeJSON(a.privateFile(prefsName), a.prefs, 0600))
			}
		default:
			if rest, ok := argument(cmd, "new"); ok {
				a.report(a.createBoard(rest))
			} else if n, ok := numberArgument(cmd, "del"); ok {
				a.report(a.deleteBoard(n))
			} else if rest, ok := argument(cmd, "fil"); ok {
				search = rest
			} else if n, ok := numberArgument(cmd, "pin+"); ok {
				a.report(a.setPin(n, true))
			} else if n, ok := numberArgument(cmd, "pin-"); ok {
				a.report(a.setPin(n, false))
			} else if n, err := strconv.Atoi(cmd); err == nil { // anything else that isn't a number is ignored
				if b, err := a.board(n); err == nil {
					a.boardScreen(b.BoardRef)
				}
			}
		}
	}
}

// boardScreen shows one board and posts what the user types, until they go back.
func (a *App) boardScreen(ref BoardRef) {
	scroll, search := 0, ""
	defer a.refresh()
	for {
		clearScreen()
		if !a.printBoard(ref, search, scroll, false) {
			a.pause()
			return
		}
		search = ""
		fmt.Fprintf(out, "Add >> ")
		line, ok := a.readLine()
		if !ok {
			return
		}
		cmd := strings.TrimSpace(line)
		// Commands only match as a whole word, so messages like "file...", "anonymous..." or "review..." post normally.
		switch {
		case cmd == "" || cmd == "q":
			return
		case cmd == "w":
			n := len(a.boards[a.find(ref)].Messages)
			scroll = min(scroll+pageSize, max(0, n-pageSize))
		case cmd == "s":
			scroll = max(0, scroll-pageSize)
		case cmd == "h" || cmd == "H":
			clearScreen()
			fmt.Fprintln(out, a.help())
			a.pause()
			continue
		case cmd == "l":
			a.openLink(ref)
			continue
		case cmd == "r" || cmd == "fil" || cmd == "anon":
			// just refresh
		default:
			if rest, ok := argument(line, "fil"); ok {
				search = rest
				continue
			} else if rest, ok := argument(line, "anon"); ok {
				a.report(a.post(ref, rest, true))
			} else {
				a.report(a.post(ref, line, false))
			}
		}
		a.refresh()
	}
}

// openLink lists the board's gemini links and opens the chosen one in a tmux pane.
func (a *App) openLink(ref BoardRef) {
	clearScreen()
	if !a.printBoard(ref, "", 0, true) {
		return
	}
	fmt.Fprintf(out, "client + URL Index >> ")
	line, ok := a.readLine()
	if !ok {
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		return
	}
	messages := a.boards[a.find(ref)].Messages
	if n < 0 || n >= len(messages) || grabgeminiurl(messages[n].Text) == "" {
		fmt.Fprintln(out, "No gemini link at that index.")
	} else if err := exec.Command("tmux", "split", "-h", a.prefs.Browser, grabgeminiurl(messages[n].Text)).Run(); err != nil {
		fmt.Fprintln(out, "could not open it (bb needs to run inside tmux):", err)
	} else {
		fmt.Fprintln(out, "Opening browser.")
	}
	a.pause()
}

// printIndex prints the pinned boards, then a page of the rest, newest first.
func (a *App) printIndex(search string, scroll int) {
	heading := "-- " + a.cfg.Title + " --"
	if a.anyUnread() {
		heading += " [NEW]"
	}
	pterm.Println(pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).Sprint(heading))
	for i, b := range a.boards {
		if a.isPinned(b.BoardRef) {
			a.printBoardLine(i, b, " **PINNED**")
		}
	}
	fmt.Fprintln(out, "")
	if len(a.boards) == 0 {
		fmt.Fprintln(out, "there are no boards")
		return
	}
	first, last := pageRange(len(a.boards), scroll, search != "")
	for i := last; i >= first; i-- {
		b := a.boards[i]
		if a.isPinned(b.BoardRef) {
			continue
		}
		if search != "" && !strings.Contains(b.Title, search) && !strings.Contains(b.Date, search) {
			continue
		}
		a.printBoardLine(i, b, "")
	}
	fmt.Fprintf(out, "\nBB Length: %d, showing %d to %d\n\n", len(a.boards), first, last)
}

// printBoardLine prints one index entry: green if the board is new since the user last looked,
// cyan if it has unread posts.
func (a *App) printBoardLine(index int, b Board, suffix string) {
	line := strconv.Itoa(index) + ") " + b.Title + " | author: " + b.Owner + " | " + b.Date
	if last, ok := b.lastActive(); ok {
		line += " | active " + ago(now().Sub(last))
	}
	line += suffix
	switch {
	case a.isNew(b):
		color.Green(line)
	case a.unread(b):
		color.Cyan(line)
	default:
		fmt.Fprintln(out, line)
	}
}

// printBoard prints a page of a board's messages (or just its gemini links), marks it as read
// and remembers it as the board 'bb -a' posts to. False if the board has gone.
func (a *App) printBoard(ref BoardRef, search string, scroll int, linksOnly bool) bool {
	ix := a.find(ref)
	if ix < 0 {
		fmt.Fprintln(out, errNotFound)
		return false
	}
	b := a.boards[ix]
	heading := b.Title + " | " + b.Owner
	if a.anyUnread() {
		heading += " ^new"
	}
	a.opened[ref] = true
	a.report(writeJSON(a.privateFile(lastName), ref, 0600))
	pterm.DefaultSection.Println(heading)
	fmt.Fprintln(out, "")

	plain := &pterm.BasicTextPrinter{}
	highlight := plain.WithStyle(&pterm.ThemeDefault.SuccessMessageStyle)
	first, last := pageRange(len(b.Messages), scroll, search != "")
	for i := first; i <= last; i++ {
		m := b.Messages[i]
		mentioned := strings.Contains(m.Text, "@"+a.user)
		if linksOnly {
			if link := grabgeminiurl(m.Text); link != "" {
				line := strconv.Itoa(i) + ") " + link
				if mentioned {
					color.Cyan(line)
				} else {
					fmt.Fprintln(out, line)
				}
			}
			continue
		}
		if search != "" && !strings.Contains(m.Text, search) && !strings.Contains(m.Time, search) {
			continue
		}
		line := m.Time + " <" + m.Author + "> " + m.Text
		if mentioned {
			highlight.Println(line)
		} else {
			plain.Println(line)
		}
	}
	fmt.Fprintf(out, "\nBoard length: %d, showing %d to %d\n\n", len(b.Messages), first, last)
	return true
}

// pageRange returns the first and last index to show from n items (newest last),
// scrolled back by scroll items. A search shows everything.
func pageRange(n, scroll int, searching bool) (first, last int) {
	if searching || n <= pageSize {
		return 0, n - 1
	}
	last = max(n-1-scroll, pageSize-1)
	return last - pageSize + 1, last
}

// ago formats a duration as a short "3m ago" style age.
func ago(d time.Duration) string {
	const day = 24 * time.Hour
	switch {
	case d < time.Minute: // also covers clock skew putting a post slightly in the future
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m ago"
	case d < day:
		return strconv.Itoa(int(d/time.Hour)) + "h ago"
	case d < 30*day:
		return strconv.Itoa(int(d/day)) + "d ago"
	case d < 365*day:
		return strconv.Itoa(int(d/(30*day))) + "mo ago"
	default:
		return strconv.Itoa(int(d/(365*day))) + "y ago"
	}
}

// argument returns what follows "word " in cmd, if anything does.
func argument(cmd, word string) (string, bool) {
	if !strings.HasPrefix(cmd, word+" ") {
		return "", false
	}
	rest := strings.TrimSpace(cmd[len(word)+1:])
	return rest, rest != ""
}

func numberArgument(cmd, word string) (int, bool) {
	rest, ok := argument(cmd, word)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

func clearScreen() {
	if runtime.GOOS != "linux" {
		return
	}
	cmd := exec.Command("clear")
	cmd.Stdout = out
	_ = cmd.Run()
}
