package main

// Everything bb keeps on disk, and how it is read and written.
//
// Each user has three folders in their home:
//   .bb/     one <title>.json per board they created or posted on (only their own messages)
//   .bbsn/   private: what they had seen last time, the last board opened, preferences
//   .bbmod/  moderators keep their archive list here; the admin's also holds the
//            shared, world-writable anon and pin files
//
// The file names predate this code and stay as they are so existing installs keep working.
// Files written before the 2026 cleanup used parallel lists and [time, text, author] arrays;
// they are still read, and are rewritten in the current format the next time they are saved.

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

const (
	boardsDir  = ".bb"
	privateDir = ".bbsn"
	modDir     = ".bbmod"
	snapName   = "bbsn4p.json"
	lastName   = "bbl4st.json"
	prefsName  = "bbp3r.json"
	anonName   = "bban0n.json"
	pinName    = "bbp1n.json"
	modName    = "mod.json"

	anonAuthor = "???"
)

// BoardRef identifies a board. Titles can repeat across dates, so both are needed.
type BoardRef struct {
	Title string
	Date  string
}

type Message struct {
	Time   string // "2006-01-02 15:04:05", local time of the poster
	Text   string
	Author string
}

// decodeMessages reads messages in either format, skipping any that are malformed and
// cleaning terminal escape codes out of the rest. Board files live in other users' homes,
// so nothing in them is trusted.
func decodeMessages(raws []json.RawMessage) []Message {
	var out []Message
	for _, raw := range raws {
		var m Message
		var row []string
		if json.Unmarshal(raw, &row) == nil { // pre-2026: [time, text, author]
			if len(row) != 3 {
				continue
			}
			m = Message{Time: row[0], Text: row[1], Author: row[2]}
		} else if json.Unmarshal(raw, &m) != nil {
			continue
		}
		out = append(out, Message{Time: clean(m.Time), Text: clean(m.Text), Author: clean(m.Author)})
	}
	return out
}

// ---- board files ----

type boardFile struct {
	Title    string
	Date     string
	Owner    string
	Messages []Message
}

func readBoardFile(path string) (boardFile, error) {
	var raw struct {
		Title, Date, Owner string
		Messages           []json.RawMessage
		Contents           []json.RawMessage // pre-2026 name
	}
	if err := readJSON(path, &raw); err != nil {
		return boardFile{}, err
	}
	return boardFile{
		Title:    clean(raw.Title),
		Date:     clean(raw.Date),
		Owner:    clean(raw.Owner),
		Messages: append(decodeMessages(raw.Contents), decodeMessages(raw.Messages)...),
	}, nil
}

// ---- shared anon file ----

type anonBoard struct {
	BoardRef
	Messages []Message
}

func parseAnon(data []byte) ([]anonBoard, error) {
	var raw struct {
		Boards []struct {
			Title, Date string
			Messages    []json.RawMessage
		}
		// pre-2026: parallel lists
		Title []string
		Date  []string
		Board []struct{ Contents []json.RawMessage }
	}
	if err := unmarshalOrEmpty(data, &raw); err != nil {
		return nil, err
	}
	var out []anonBoard
	for i := 0; i < len(raw.Title) && i < len(raw.Date) && i < len(raw.Board); i++ {
		out = append(out, anonBoard{BoardRef{clean(raw.Title[i]), clean(raw.Date[i])}, decodeMessages(raw.Board[i].Contents)})
	}
	for _, b := range raw.Boards {
		out = append(out, anonBoard{BoardRef{clean(b.Title), clean(b.Date)}, decodeMessages(b.Messages)})
	}
	for i := range out {
		for j := range out[i].Messages {
			out[i].Messages[j].Author = anonAuthor // the file is world-writable: never trust a name in it
		}
	}
	return out, nil
}

func formatAnon(boards []anonBoard) ([]byte, error) {
	return json.MarshalIndent(struct{ Boards []anonBoard }{boards}, "", "\t")
}

// ---- shared pin file ----

func parsePins(data []byte) ([]BoardRef, error) {
	var raw struct {
		Pins []BoardRef
		// pre-2026: parallel lists
		Title []string
		Date  []string
	}
	if err := unmarshalOrEmpty(data, &raw); err != nil {
		return nil, err
	}
	pins := zipRefs(raw.Title, raw.Date)
	for _, p := range raw.Pins {
		pins = append(pins, BoardRef{clean(p.Title), clean(p.Date)})
	}
	return pins, nil
}

func formatPins(pins []BoardRef) ([]byte, error) {
	return json.MarshalIndent(struct{ Pins []BoardRef }{pins}, "", "\t")
}

// ---- moderator files ----

// modFile is one moderator's file. Only the admin's Moderators list counts.
type modFile struct {
	Moderators []string   `json:",omitempty"`
	Archived   []BoardRef `json:",omitempty"`
}

func readModFile(path string) (modFile, error) {
	var raw struct {
		Moderators []string
		Archived   []BoardRef
		// pre-2026
		Name         []string
		Boardarchive []string
		Datearchive  []string
	}
	if err := readJSON(path, &raw); err != nil {
		return modFile{}, err
	}
	m := modFile{Archived: zipRefs(raw.Boardarchive, raw.Datearchive)}
	for _, name := range append(raw.Name, raw.Moderators...) {
		m.Moderators = appendUnique(m.Moderators, name)
	}
	for _, ref := range raw.Archived {
		m.Archived = append(m.Archived, ref)
	}
	return m, nil
}

// ---- private files ----

type snapEntry struct {
	BoardRef
	Length int
}

// readSnapshot returns how many messages each board had when the user last looked.
func readSnapshot(path string) map[BoardRef]int {
	var raw struct {
		Boards []snapEntry
		// pre-2026: parallel lists
		Title  []string
		Date   []string
		Length []int
	}
	seen := map[BoardRef]int{}
	if readJSON(path, &raw) != nil {
		return seen
	}
	for i := 0; i < len(raw.Title) && i < len(raw.Date) && i < len(raw.Length); i++ {
		seen[BoardRef{raw.Title[i], raw.Date[i]}] = raw.Length[i]
	}
	for _, e := range raw.Boards {
		seen[e.BoardRef] = e.Length
	}
	return seen
}

type prefs struct {
	Browser string // gemini client opened by the 'l' command
}

// ---- helpers ----

func zipRefs(titles, dates []string) []BoardRef {
	var refs []BoardRef
	for i := 0; i < len(titles) && i < len(dates); i++ {
		refs = append(refs, BoardRef{clean(titles[i]), clean(dates[i])})
	}
	return refs
}

func appendUnique(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

func unmarshalOrEmpty(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// writeJSON writes v to path with exactly perm, whatever the user's umask: files other
// users must read (boards, moderator lists) are unreadable to them if the umask is 077.
func writeJSON(path string, v any, perm os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}

// readShared reads a shared file under a shared lock, so it is never seen half-written.
func readShared(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := lockFile(f, false); err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}

// updateShared changes a shared file under an exclusive lock: read, change, write back.
// Two users posting at the same moment queue up instead of overwriting each other.
func updateShared(path string, change func(old []byte) ([]byte, error)) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("bb is not set up yet - ask the admin to run 'bb init'")
	}
	if err != nil {
		return err
	}
	defer f.Close()
	if err := lockFile(f, true); err != nil {
		return err
	}
	old, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	data, err := change(old)
	if err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err = f.WriteAt(data, 0)
	return err
}
