package main

import (
	"bb/addons"
	"bb/urlparse"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Board struct {
	BoardRef
	Owner    string
	Messages []Message // oldest first
}

// loadBoards collects every board from every user's .bb folder.
//
// A board exists only if its owner's own file is in the owner's home, and a message counts
// only if it is in its author's file (anon posts come from the shared anon file). Anyone can
// put anything in their own files, so this stops users inventing boards or messages in
// someone else's name.
func (a *App) loadBoards() {
	var boards []Board
	messages := map[BoardRef][]Message{}
	for _, user := range a.users() {
		dir := a.dir(user, boardsDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // not a bb user (yet)
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".json" {
				continue
			}
			f, err := readBoardFile(dir + e.Name())
			ref := BoardRef{f.Title, f.Date}
			if err != nil || ref.Title == "" || ref.Date == "" || a.mods.isArchived(ref) {
				continue
			}
			if f.Owner == user {
				boards = append(boards, Board{BoardRef: ref, Owner: user})
			}
			for _, m := range f.Messages {
				if m.Author == user {
					messages[ref] = append(messages[ref], m)
				}
			}
		}
	}
	for _, ab := range a.loadAnon() {
		messages[ab.BoardRef] = append(messages[ab.BoardRef], ab.Messages...)
	}
	for i := range boards {
		boards[i].Messages = messages[boards[i].BoardRef]
		sort.SliceStable(boards[i].Messages, func(x, y int) bool {
			return boards[i].Messages[x].Time < boards[i].Messages[y].Time
		})
	}
	sort.SliceStable(boards, func(i, j int) bool { return boards[i].Date < boards[j].Date })
	a.boards = boards
}

func (a *App) loadAnon() []anonBoard {
	data, err := readShared(a.sharedFile(anonName))
	if err != nil {
		return nil
	}
	boards, _ := parseAnon(data)
	return boards
}

// find returns the index of a board, or -1.
func (a *App) find(ref BoardRef) int {
	for i, b := range a.boards {
		if b.BoardRef == ref {
			return i
		}
	}
	return -1
}

func (a *App) board(ix int) (Board, error) {
	if ix < 0 || ix >= len(a.boards) {
		return Board{}, fmt.Errorf("there is no board %d", ix)
	}
	return a.boards[ix], nil
}

// createBoard starts a new board owned by the current user.
func (a *App) createBoard(title string) error {
	title = strings.TrimSpace(clean(title))
	if title == "" || strings.ContainsAny(title, "/\\") || strings.HasPrefix(title, ".") {
		return errors.New("board titles can't be empty, contain slashes, or start with '.'")
	}
	for _, b := range a.boards {
		if b.Title == title {
			return errors.New("there is already a board called " + title + " - please use a unique title")
		}
	}
	path := a.ownBoardFile(title)
	if _, err := os.Stat(path); err == nil { // e.g. one of theirs that was archived
		return errors.New("you already have a board file called " + title + " - please use another title")
	}
	f := boardFile{Title: title, Date: time.Now().Format("2006-01-02"), Owner: a.user}
	if err := writeJSON(path, f, 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, "saved "+title)
	a.loadBoards()
	return nil
}

// deleteBoard removes a board, but only for its owner and only while nobody else has posted
// on it (their copies live in their homes, which the owner can't touch). Moderators can
// archive any board instead.
func (a *App) deleteBoard(ix int) error {
	b, err := a.board(ix)
	if err != nil {
		return err
	}
	if b.Owner != a.user {
		return fmt.Errorf("you are not owner of topic %d. Ask a mod to archive if needed", ix)
	}
	for _, user := range a.users() {
		if user == a.user {
			continue
		}
		if f, err := readBoardFile(a.dir(user, boardsDir) + b.Title + ".json"); err == nil && f.Date == b.Date {
			return errors.New("others have posted on " + b.Title + ", so it can't be deleted. Ask a mod to archive it instead")
		}
	}
	if err := os.Remove(a.ownBoardFile(b.Title)); err != nil {
		return err
	}
	a.loadBoards()
	return nil
}

// post adds a message to a board, in the user's own file or, for anon, the shared anon file.
// Addons can rewrite the text, and a link gets its page title added.
func (a *App) post(ref BoardRef, input string, anon bool) error {
	ix := a.find(ref)
	if ix < 0 {
		return errNotFound
	}
	text := clean(addons.Parse(input))
	if link := firstURL(input, "http://", "https://", "gemini://"); link != "" {
		var title string
		if strings.HasPrefix(link, "gemini://") {
			title = urlparse.GEMGet(link)
		} else {
			title = urlparse.HTTPget(link)
		}
		if title = strings.TrimSpace(clean(title)); title != "" {
			text += " | Title: " + title
		}
	}
	msg := Message{Time: timestamp(), Text: text, Author: a.user}
	if anon {
		msg.Author = anonAuthor
		return updateShared(a.sharedFile(anonName), func(old []byte) ([]byte, error) {
			boards, err := parseAnon(old)
			if err != nil {
				return nil, fmt.Errorf("could not read the anon file, message not posted: %w", err)
			}
			for i := range boards {
				if boards[i].BoardRef == ref {
					boards[i].Messages = append(boards[i].Messages, msg)
					return formatAnon(boards)
				}
			}
			return formatAnon(append(boards, anonBoard{ref, []Message{msg}}))
		})
	}
	path := a.ownBoardFile(ref.Title)
	f, err := readBoardFile(path)
	if err != nil || f.Date != ref.Date { // first post here (or an old file for another board of that name)
		f = boardFile{Title: ref.Title, Date: ref.Date, Owner: a.boards[ix].Owner}
	}
	f.Messages = append(own(f.Messages, a.user), msg)
	return writeJSON(path, f, 0644)
}

// own keeps only the messages written by user.
func own(messages []Message, user string) []Message {
	var out []Message
	for _, m := range messages {
		if m.Author == user {
			out = append(out, m)
		}
	}
	return out
}

// lastActive returns the time of the newest message, or false if there are none.
// Timestamps that don't parse (hand-edited files) are ignored.
func (b Board) lastActive() (time.Time, bool) {
	var last time.Time
	for _, m := range b.Messages {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", m.Time, time.Local); err == nil && t.After(last) {
			last = t
		}
	}
	return last, !last.IsZero()
}
