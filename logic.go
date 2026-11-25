package main

import (
	"bb/addons"
	bot "bb/urlparse"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color" //to add colour to output

	"github.com/pterm/pterm" //to make terminal pretty
)

var (
	//Everything below does not need to be configured
	ismod          bool
	clear          map[string]func()               // create a map for storing clear funcs
	username       string                          // store the username of the individual currently using BB
	homefilepath   string                          // the path for where the bb config files are
	snapfilepath   string                          // the path for where the bb snapshot file is
	homeroot       = "/home/"                      // folder holding every user's home folder (tests point this at a temp dir)
	masterfilepath = homeroot + admin + "/.bbmod/" // mods will be added to this file. Will also store the anonymous file.
	modfilepath    string
	aa             Snap     // Snapshot object
	bb             BB       // BB object
	mm             Mod      // Moderator object
	an             Anon     // Anon object
	ll             Last     // Last board object
	pin            Pin      // Pin board object
	per            Personal //Personal variables object
	snapname       = "bbsn4p.json"
	lastname       = "bbl4st.json"
	anonname       = "bban0n.json"
	pinname        = "bbp1n.json"
	pername        = "bbp3r.json"
	back           int //board scroll logic
	length         = 20
	helpstring     = `
===BB HELP===
for INDEX section:
	new - create a new board i.e 'new topictitle'
	del - delete a board by index. 
		If nobody else has accessed it - you can delete it. 
	        Otherwise, you need superuser permission.
	fil - filter index by search string e.g YYYY-MM or Title
	pin+ - pin a board by index
	pin- - unpin a board by index
	q - to quit, or use ctrl-c
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
	anon - make message anonymous
	rev - reverses your text
	anything else - types text to board
	nothing - also exits back to index section
	ctrl-c to quit
	
FYI:
	- For gemini client functionality you need to run bb inside tmux
	- Boards glow cyan when new content is posted
	- New boards glow green.
	- You can comment other people via @ sign i.e @person
		they will see message highlighted
	- If you are on a board and new content is posted on another board, 
		you'll see '^new' beside author name
	
PRESS ENTER TO CONTINUE...
					`
)

// GENERAL FUNCTIONS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func init() { //runs at start pre-main to initialize some things.
	pterm.EnableOutput()
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux - maybe one day we'll add windows functionality. who knows.
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func check(username string) bool {
	if username == admin {
		return true
	}

	if _, err := os.Stat(homeroot + admin + "/.bb"); os.IsNotExist(err) {
		fmt.Printf("the admin: '%s' has not yet ran / initiated bb.\n", admin)
		return false
	}

	return true
}

// Work out where every bb file lives for the admin and the current user.
func setPaths() {
	masterfilepath = homeroot + admin + "/.bbmod/"
	homefilepath = homeroot + username + "/.bb/"
	snapfilepath = homeroot + username + "/.bbsn/"
	modfilepath = homeroot + username + "/.bbmod/"
}

// Clear screen
func callclear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	}
	//No need to panic.
}

func intmux(link, client string) {
	muxcmd := exec.Command("tmux", "split", "-h", client, link) //Linux
	err := muxcmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

// Grab a timestamp.
func timestamp() string {
	t := time.Now()
	var x = t.Format("2006-01-02 15:04:05")
	return x
}

// create a new board for the global BB variable.
func newboard(title string, bb BB) {
	title = strings.TrimSpace(clean(title))
	if title == "" || strings.ContainsAny(title, "/\\") || strings.HasPrefix(title, ".") {
		fmt.Println("Board titles can't be empty, contain slashes, or start with '.'. Press enter to continue")
		fmt.Scanln()
		return
	}
	for index := range bb.B {
		if bb.B[index].Title == title {
			fmt.Println("Please use unique title. Press enter to continue")
			fmt.Scanln()
			return
		}
	}
	board := Board{}
	board.Title = title
	board.Owner = username
	t := time.Now()
	var x = t.Format("2006-01-02")
	board.Date = x
	board.Save(title)
	fmt.Println("saved " + title)
}

// remove a board from BB at a specific index
func remove(slice []Board, s int) []Board {
	if len(slice) <= 1 {
		return []Board{}
	}
	return append(slice[:s], slice[s+1:]...)
}

func modremove(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// Add a slice to a 2d slice
func add2slice(ax *[][]string, b []string) [][]string {
	a := *ax
	a = append(a, b)
	*ax = a
	return a
}

// Grab a list folders in home dir (to grab the Usernames)
func ufolderlist() []string {
	output := []string{}
	files, _ := os.ReadDir(homeroot)
	for _, f := range files {
		output = append(output, f.Name())
	}
	return output
}

// Refresh the BB data with any new inputs.
// This only reads. The shared pin and anon files are written at the moment they change,
// so a refresh never overwrites what other users have saved since we last loaded them.
func rehash() {
	pin.Load()
	bb = BB{} //Clear BB
	bb.Load() //Reload BB
}

// Personal METHODS//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type Personal struct {
	Browser string
}

// Save file
func (p Personal) Save() {
	Base := &p
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(snapfilepath+pername, output, 0666)
	if err != nil {
		fmt.Println(err)
	}
}

// Load file
func (p *Personal) Load() {
	item := *p
	jsonFile, _ := os.ReadFile(snapfilepath + pername)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	if item.Browser == "" {
		item.Browser = "amfora"
	}
	*p = item
}

///Pin board METHODS //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Pin struct {
	Title []string
	Date  []string
}

func (P *Pin) Add(ix int) {
	if ix < 0 || ix >= len(bb.B) {
		return
	}
	if err := P.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Println("could not read the pin file, not pinning:", err)
		return
	}
	p := *P
	titletosave := bb.B[ix].Title
	datetosave := bb.B[ix].Date
	exists := false
	for index := range p.Title {
		if p.Title[index] == titletosave && p.Date[index] == datetosave {
			exists = true
		}
	}
	if !exists {
		p.Title = append(p.Title, titletosave)
		p.Date = append(p.Date, datetosave)
	}
	*P = p
	P.Save()
}

func (P *Pin) Remove(ix int) {
	if ix < 0 || ix >= len(bb.B) {
		return
	}
	if err := P.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Println("could not read the pin file, not unpinning:", err)
		return
	}
	p := *P
	titletodel := bb.B[ix].Title
	datetodel := bb.B[ix].Date
	for index := range p.Title {
		if p.Title[index] == titletodel && p.Date[index] == datetodel {
			p.Title = modremove(p.Title, index)
			p.Date = modremove(p.Date, index)
			break
		}
	}
	*P = p
	P.Save()
}

// Save mod file
func (p Pin) Save() {
	Base := &p
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(masterfilepath+pinname, output, 0666)
	if err != nil {
		fmt.Println(err)
	}
}

// Load pins. The pin file is shared and world-writable, so mismatched lists are trimmed rather than trusted.
// On error the pins already in memory are kept.
func (p *Pin) Load() error {
	jsonFile, err := os.ReadFile(masterfilepath + pinname)
	if err != nil {
		return err
	}
	item := Pin{}
	if err := json.Unmarshal(jsonFile, &item); err != nil {
		return err
	}
	n := len(item.Title)
	if len(item.Date) < n {
		n = len(item.Date)
	}
	item.Title, item.Date = item.Title[:n], item.Date[:n]
	*p = item
	return nil
}

///Lastfile & Last METHODS //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Last struct {
	Title string
	Date  string
}

// Save last file
func (l Last) Save() {
	Base := &l
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(snapfilepath+lastname, output, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// Load mods
func (l *Last) Load() {
	item := *l
	jsonFile, _ := os.ReadFile(snapfilepath + lastname)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*l = item
}

///Moderator & Mod METHODS //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Only the administrator can be a moderator at the moment
type Mod struct {
	Name         []string
	Boardarchive []string
	Datearchive  []string
}

// Save mod file
func (m Mod) Save() {
	if ismod {
		Base := &m
		output, err := json.MarshalIndent(Base, "", "\t")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = os.WriteFile(modfilepath+"mod.json", output, 0644)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Load mods
func (m *Mod) Load() {
	item := *m
	jsonFile, _ := os.ReadFile(masterfilepath + "mod.json") //no need to worry about error
	_ = json.Unmarshal([]byte(jsonFile), &item)             //but leaving it empty in case one day... i need to worry
	item.Collect()
	*m = item
}

func (m *Mod) Collect() {
	list := ufolderlist()
	for index := range list {
		m.collect(list[index])
	}
}

func (m *Mod) collect(homeuser string) {
	if m.IsUserMod(homeuser) {
		modp := homeroot + homeuser + "/.bbmod/"
		learnfolder, err := os.ReadDir(modp)
		if err == nil {
			for _, learnfile := range learnfolder {
				if learnfile.Name() == "mod.json" {
					mm := Mod{}
					jsonFile, _ := os.ReadFile(modp + "mod.json") //no need to worry about error
					_ = json.Unmarshal([]byte(jsonFile), &mm)     //but leaving it empty in case one day... i need to worry
					for index := range mm.Boardarchive {
						exists := false
						for index2 := range m.Boardarchive {
							if m.Boardarchive[index2] == mm.Boardarchive[index] && m.Datearchive[index2] == mm.Datearchive[index] {
								exists = true
							}
						}
						if !exists {
							m.Boardarchive = append(m.Boardarchive, mm.Boardarchive[index])
							m.Datearchive = append(m.Datearchive, mm.Datearchive[index])
						}
					}
				}
			}
		}
	}
}

// Check if a board is on the archive list - this is alternative to delete. If a board is on board archive it won't load
func (m Mod) Check(b Board) bool {
	for index := range m.Boardarchive {
		if m.Boardarchive[index] == b.Title && m.Datearchive[index] == b.Date {
			return true
		}
	}
	return false
}

// Checks if current user is a mod
func (m Mod) IsMod() {
	for index := range m.Name {
		if m.Name[index] == username {
			ismod = true
		}
	}
}

// Checks if current user is a mod
func (m Mod) IsUserMod(uname string) bool {
	for index := range m.Name {
		if m.Name[index] == uname {
			return true
		}
	}
	return false
}

// Lets moderators archive a board by indoex
func (m *Mod) Archive(item int) {
	for index := range bb.B {
		if index == item {
			m.Boardarchive = append(m.Boardarchive, bb.B[index].Title)
			m.Datearchive = append(m.Datearchive, bb.B[index].Date)
			fmt.Println(bb.B[index].Title + " archived")
			return
		}
	}
}

// Lets you add a mod if you are admin
func (m *Mod) AddMod(user string) {
	for index := range m.Name {
		if m.Name[index] == user {
			return
		}
	}
	m.Name = append(m.Name, user)
}

// Lets you add a mod if you are admin
func (m *Mod) RemoveMod(user string) {
	for index := range m.Name {
		if m.Name[index] == user {
			m.Name = modremove(m.Name, index)
			break
		}
	}
}

///Anon & Anon METHODS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Anon struct {
	Title []string
	Date  []string
	Board []Anonboard
}

type Anonboard struct {
	Contents [][]string
}

// Save mod file
func (a Anon) Save() {
	Base := &a
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(masterfilepath+anonname, output, 0666)
	if err != nil {
		fmt.Println(err)
	}
}

// Load anon posts. The anon file is shared and world-writable, so its contents are checked rather than trusted.
// On error the posts already in memory are kept.
func (a *Anon) Load() error {
	jsonFile, err := os.ReadFile(masterfilepath + anonname)
	if err != nil {
		return err
	}
	item := Anon{}
	if err := json.Unmarshal(jsonFile, &item); err != nil {
		return err
	}
	n := len(item.Title)
	if len(item.Date) < n {
		n = len(item.Date)
	}
	if len(item.Board) < n {
		n = len(item.Board)
	}
	item.Title, item.Date, item.Board = item.Title[:n], item.Date[:n], item.Board[:n]
	for index := range item.Board {
		item.Board[index].Contents = validRows(item.Board[index].Contents)
	}
	*a = item
	return nil
}

func (a *Anon) Add(title, date string, contents []string) {
	for index := range a.Title {
		if a.Title[index] == title && a.Date[index] == date {
			a.Board[index].Contents = append(a.Board[index].Contents, contents)
			return
		}
	}
	a.Title = append(a.Title, title)
	a.Date = append(a.Date, date)
	a.Board = append(a.Board, Anonboard{Contents: [][]string{contents}})
}

// Post an anonymous message. Re-reads the shared file first so posts other users made
// since we loaded it are kept, then writes it straight back.
func postAnon(title, date string, item []string) {
	if err := an.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Println("could not read the anon file, message not posted:", err)
		return
	}
	an.Add(title, date, item)
	an.Save()
}

///Snapshot & Snapshot METHODS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Snap is struct for storing snapshot of every board in BB
type Snap struct {
	Owner   []string
	Title   []string
	Date    []string
	Length  []int
	Checked []bool
}

// Save snapshot to json
func (s Snap) Save() {
	Base := &s
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(snapfilepath+snapname, output, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// Load snapshot to json
func (s *Snap) Load() {
	item := *s
	jsonFile, _ := os.ReadFile(snapfilepath + snapname)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*s = item
}

// If we've just read a board, we want to switch that board from 'not checked' to 'checked'.
func (s *Snap) Switch(title, date string) {
	if i := s.index(title, date); i >= 0 {
		s.Checked[i] = true
	}
}

// Whatsnew lists the boards with posts since the snapshot was saved.
func (s Snap) Whatsnew() []string {
	news := []string{}
	for _, board := range bb.B {
		if s.unread(board) {
			news = append(news, board.Title)
		}
	}
	return news
}

///BB & BB METHODS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// BB is the main struct holding all boards.
type BB struct {
	B []Board
}

// Save a snapshot of the global BB obj to global Snap obj
func (b BB) saveSnapshot() Snap {
	s := Snap{}
	for index := range b.B {
		s.Title = append(s.Title, b.B[index].Title)
		s.Owner = append(s.Owner, b.B[index].Owner)
		s.Length = append(s.Length, len(b.B[index].Contents))
		s.Date = append(s.Date, b.B[index].Date)
		s.Checked = append(s.Checked, false)
	}
	return s
}

// Load all the boards to BB
func (b *BB) Load() {
	an.Load()
	list := ufolderlist()
	for index := range list {
		b.collect(list[index])
	}
	b.anoncollect()
}

// index finds a board in the snapshot, or returns -1.
func (s Snap) index(title, date string) int {
	for i := range s.Title {
		if i < len(s.Date) && i < len(s.Length) && i < len(s.Checked) && s.Title[i] == title && s.Date[i] == date {
			return i
		}
	}
	return -1
}

// unread reports whether a board has posts that arrived since the snapshot and haven't been viewed yet.
func (s Snap) unread(b Board) bool {
	i := s.index(b.Title, b.Date)
	return i >= 0 && len(b.Contents) != s.Length[i] && !s.Checked[i]
}

// Iterate through snapshot to check if something has changed (at BB level)
func (b BB) snapcheck(s Snap) bool {
	for _, board := range b.B {
		if s.unread(board) {
			return true
		}
	}
	return false
}

// pageRange returns the first and last index to show from n items (newest last),
// scrolled back by the global `back`. A search shows everything.
func pageRange(n int, searching bool) (first, last int) {
	if searching || n <= length {
		return 0, n - 1
	}
	last = n - 1 - back
	if last < length-1 {
		last = length - 1
	}
	return last - length + 1, last
}

// printBoardLine prints one index entry: green if the board is new since the last visit,
// cyan if it has unread posts.
func printBoardLine(index int, b Board, s Snap, suffix string) {
	line := strconv.Itoa(index) + ") " + b.Title + " | author: " + b.Owner + " | " + b.Date + suffix
	switch {
	case s.index(b.Title, b.Date) < 0:
		color.Green(line)
	case s.unread(b):
		color.Cyan(line)
	default:
		fmt.Println(line)
	}
}

// loadpin prints the pinned boards and returns their indexes.
func (b BB) loadpin(s Snap) map[int]bool {
	pinned := map[int]bool{}
	for index, board := range b.B {
		for i := range pin.Title {
			if pin.Title[i] == board.Title && pin.Date[i] == board.Date {
				pinned[index] = true
				printBoardLine(index, board, s, " **PINNED**")
				break
			}
		}
	}
	fmt.Println("")
	return pinned
}

// Load all contents of BB to the screen.
func (b BB) loadall(s Snap, searchstring string) {
	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue))
	heading := "-- " + boardtitle + " --"
	if b.snapcheck(s) {
		heading += " [NEW]"
	}
	pterm.Println(header.Sprint(heading))
	pinned := b.loadpin(s)
	if len(b.B) == 0 {
		fmt.Println("there are no boards")
		return
	}
	first, last := pageRange(len(b.B), searchstring != "")
	for index := last; index >= first; index-- {
		board := b.B[index]
		if pinned[index] || board.Date == "" {
			continue
		}
		if searchstring != "" && !strings.Contains(board.Title, searchstring) && !strings.Contains(board.Date, searchstring) {
			continue
		}
		printBoardLine(index, board, s, "")
	}
	fmt.Printf("\nBB Length: %d, showing %d to %d\n\n", len(b.B), first, last)
}

// loadgem returns the gemini link in message urlindex of board ix, or "".
func (b BB) loadgem(ix, urlindex int) string {
	if ix < 0 || ix >= len(b.B) || urlindex < 0 || urlindex >= len(b.B[ix].Contents) {
		return ""
	}
	return grabgeminiurl(b.B[ix].Contents[urlindex][1])
}

// Load specific board up.
func (b *BB) loadboard(ix int, searchstring string) bool {
	return b.showBoard(ix, searchstring, false)
}

// List the gemini links on a board, numbered by message, for the 'l' command.
func (b *BB) viewurl(ix int) bool {
	return b.showBoard(ix, "", true)
}

// showBoard prints board ix (or only its gemini links), marks it as read and
// remembers it as the board new messages are added to.
func (b *BB) showBoard(ix int, searchstring string, linksOnly bool) bool {
	if ix < 0 || ix >= len(b.B) || b.B[ix].Owner == "" || b.B[ix].Date == "" {
		fmt.Println("invalid index")
		return false
	}
	change := b.snapcheck(aa)
	board := &b.B[ix]
	aa.Switch(board.Title, board.Date)
	sort.SliceStable(board.Contents, func(i, j int) bool { return board.Contents[i][0] < board.Contents[j][0] })
	ll.Title, ll.Date = board.Title, board.Date
	ll.Save()
	heading := board.Title + " | " + board.Owner
	if change {
		heading += " ^new"
	}
	pterm.DefaultSection.Println(heading)
	fmt.Println("")

	btp := &pterm.BasicTextPrinter{}
	highlight := btp.WithStyle(&pterm.ThemeDefault.SuccessMessageStyle)
	first, last := pageRange(len(board.Contents), searchstring != "")
	for i := first; i <= last; i++ {
		row := board.Contents[i] // always [timestamp, text, author] - see validRows
		mentioned := strings.Contains(row[1], "@"+username)
		if linksOnly {
			if link := grabgeminiurl(row[1]); link != "" {
				line := strconv.Itoa(i) + ") " + link
				if mentioned {
					color.Cyan(line)
				} else {
					fmt.Println(line)
				}
			}
			continue
		}
		if searchstring != "" && !strings.Contains(row[1], searchstring) && !strings.Contains(row[0], searchstring) {
			continue
		}
		line := row[0] + " <" + row[2] + "> " + row[1]
		if mentioned {
			highlight.Println(line)
		} else {
			btp.Println(line)
		}
	}
	fmt.Printf("\nBoard length: %d, showing %d to %d\n\n", len(board.Contents), first, last)
	return true
}

// Delete a board in BB. Only works if nobody else has viewed the board. Only sudo / root can delete regardless of this.
func (b *BB) delboard(i int) {
	var real bool
	for index := range b.B {
		if index == i && b.B[index].Owner == username {
			real = true
			b.B[index].Delete(b.B[index].Title)
			b.B = remove(b.B, index)
		}
	}
	if !real {
		index := strconv.Itoa(i)
		fmt.Println("If index " + index + " exists, you are not owner of topic. Ask a mod to archive if needed.")
		fmt.Println("Press ENTER to continue.")
		fmt.Scanln()
	}
}

func (b *BB) anoncollect() {
	B := *b
	for index := range B.B {
		for index2 := range an.Title {
			if B.B[index].Title == an.Title[index2] && B.B[index].Date == an.Date[index2] {
				for _, row := range an.Board[index2].Contents {
					if !hasRow(B.B[index].Contents, row) {
						B.B[index].Contents = append(B.B[index].Contents, row)
					}
				}
			}
		}
	}
	*b = B
}

// Collect all data from every user to load into BB - making sure to order boards by date created.
func (b *BB) collect(homeuser string) {
	ownercount := make(map[string][]string)
	homepath := homeroot + homeuser + "/.bb/"
	B := *b
	learnfolder, err := os.ReadDir(homepath)
	if err == nil {
		for _, learnfile := range learnfolder {
			Ex := filepath.Ext(learnfile.Name())
			if Ex == ".json" {
				x := Board{}
				x.Load(homepath + learnfile.Name())
				archiveboard := mm.Check(x)
				if archiveboard {
					continue
				}
				chk := false
				for index := range B.B {
					if B.B[index].Title == x.Title && B.B[index].Date == x.Date {
						ownercount[x.Title] = append(ownercount[x.Title], x.Owner)
						chk = true
						// Skip only exact repeats (same time, text and author): different users,
						// or the same user twice, can post the same text.
						for index2 := range x.Contents {
							if !hasRow(B.B[index].Contents, x.Contents[index2]) {
								B.B[index].Contents = add2slice(&B.B[index].Contents, x.Contents[index2])
							}
						}
					}
				}
				for key, element := range ownercount {
					if len(element) > 1 {
						fmt.Printf("Topic '%s' ownership has been tampered with %d times, potential owners are: %s\n", key, len(element)-1, strings.Join(element, ", ")) //You could add a log here
						fmt.Scanln()
					}
				}
				if !chk {
					B.B = append(B.B, x)
				}
			}
		}
	}
	sort.SliceStable(B.B, func(i, j int) bool {
		return B.B[i].Date < B.B[j].Date
	})
	*b = B
}

// BB ADD TO BOARD AND 'BOT' LOGIC - i.e BB can interpret input and add additional content / do actions on back of it.
func (b *BB) addtoboard(input, title, date string, anon bool) {
	botindex := -1
	for index := range b.B {
		if b.B[index].Title == title && b.B[index].Date == date {
			botindex = index
			break
		}
	}
	if botindex < 0 {
		fmt.Println("board not found (it may have been deleted or archived) - open a board first. Message not posted.")
		return
	}
	text := clean(addons.Parse(input))
	if link := firstURL(input, "http://", "https://", "gemini://"); link != "" { // add the page's title to the message
		var linkTitle string
		if strings.HasPrefix(link, "gemini://") {
			linkTitle = bot.GEMGet(link)
		} else {
			linkTitle = bot.HTTPget(link)
		}
		if linkTitle = strings.TrimSpace(clean(linkTitle)); linkTitle != "" {
			text += " | Title: " + linkTitle
		}
	}
	if anon {
		postAnon(title, date, []string{timestamp(), text, "???"})
		return
	}
	b.B[botindex].Contents = append(b.B[botindex].Contents, []string{timestamp(), text, username})
	b.B[botindex].Save(b.B[botindex].Title)
}

// Board is the struct that holds all data about a specific board
type Board struct {
	Date     string
	Owner    string
	Title    string
	Contents [][]string
}

// Load a board file. These live in other users' home folders, so malformed rows are dropped
// and all text is cleaned of terminal escape codes before anyone sees it.
func (b *Board) Load(filename string) {
	item := *b
	jsonFile, _ := os.ReadFile(filename)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	item.Title = clean(item.Title)
	item.Owner = clean(item.Owner)
	item.Date = clean(item.Date)
	item.Contents = validRows(item.Contents)
	*b = item
}

func (b Board) Delete(filename string) { //ONLY WORKS FOR ROOT AND SUDO USERS
	var afilepath string
	list := ufolderlist()
	for index := range list {
		afilepath = homeroot + list[index] + "/.bb/"
		bb.collect(list[index])
		files, err := os.ReadDir(afilepath)
		if err != nil {
			continue //Maybe this user has not actually used BB yet!
		}
		for _, f := range files {
			if f.Name() == filename+".json" { // titles may contain dots, so match the whole name
				e := os.Remove(afilepath + f.Name())
				if e != nil {
					fmt.Println("Cannot delete. Ask a moderator to archive if needed.")
					fmt.Println("Press ENTER to continue.")
					fmt.Scanln()
					return
				}
			}
		}
	}
}

// Save the board
func (b Board) Save(filename string) {
	Base := &b
	tempboard := Board{}
	tempboard.Title = Base.Title
	tempboard.Owner = Base.Owner
	tempboard.Date = Base.Date
	for index := range Base.Contents {
		if len(Base.Contents[index]) == 3 {
			if Base.Contents[index][2] == username {
				tempboard.Contents = append(tempboard.Contents, Base.Contents[index])
			}
		}
	}
	output, err := json.MarshalIndent(&tempboard, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile(homefilepath+filename+".json", output, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// View the entire BB
func ViewBB(search string, in *os.File, speed int) {
	if in == nil {
		in = os.Stdin
	}
	// One scanner for the whole session (shared with Viewboard): a new scanner per prompt
	// throws away whatever the old one had buffered, which loses lines of piped input.
	scanner := bufio.NewScanner(in)
	ticker := time.NewTicker(time.Duration(speed) * time.Millisecond)
	defer ticker.Stop()
	quit := func() {
		aa = bb.saveSnapshot() //Create snapshot of whatever BB currently is
		aa.Save()              //Save snapshot to file
	}
	for range ticker.C {
		callclear()
		bb.loadall(aa, search)
		search = ""
		fmt.Printf("--enter 'h' for help--\n")
		fmt.Printf("Pick Index >> ")
		if !scanner.Scan() { // end of input (ctrl-d) - previously this redrew forever
			quit()
			return
		}
		results := strings.TrimSpace(scanner.Text())
		switch {
		case results == "b":
			fmt.Printf("Enter client name/path > ")
			if scanner.Scan() {
				per.Browser = strings.TrimSpace(scanner.Text())
				per.Save()
			}
		case results == "q":
			quit()
			return
		case results == "r":
			quit()
			rehash()
		case results == "w":
			back += length
			if len(bb.B) <= length {
				back = 0
			} else if back > len(bb.B)-length {
				back = len(bb.B) - length
			}
		case results == "s":
			back -= length
			if back < 0 {
				back = 0
			}
		case results == "h":
			callclear()
			fmt.Println(helpstring)
			scanner.Scan()
		case strings.HasPrefix(results, "new "):
			newboard(results[4:], bb)
		case strings.HasPrefix(results, "del "):
			if index, err := strconv.Atoi(strings.TrimSpace(results[4:])); err == nil {
				bb.delboard(index)
			}
		case strings.HasPrefix(results, "fil "):
			search = strings.TrimSpace(results[4:])
		case strings.HasPrefix(results, "pin+ "):
			if index, err := strconv.Atoi(strings.TrimSpace(results[5:])); err == nil && ismod {
				pin.Add(index)
			}
		case strings.HasPrefix(results, "pin- "):
			if index, err := strconv.Atoi(strings.TrimSpace(results[5:])); err == nil && ismod {
				pin.Remove(index)
			}
		default:
			if index, err := strconv.Atoi(results); err == nil { // anything else that isn't a number is ignored
				back = 0
				Viewboard(index, search, scanner)
			}
		}
	}
}

func Viewboard(index int, search string, scanner *bufio.Scanner) {
	for {
		callclear()
		real := bb.loadboard(index, search)
		if !real {
			return
		}
		search = ""
		fmt.Printf("Add >> ")
		if !scanner.Scan() { // end of input
			break
		}
		text := scanner.Text()
		cmd := strings.TrimSpace(text)
		// Commands only match as a whole word, so messages like "file...", "anonymous..." or "review..." post normally.
		switch {
		case cmd == "" || cmd == "q":
			back = 0
			rehash()
			return
		case strings.HasPrefix(text, "fil ") && strings.TrimSpace(text[4:]) != "":
			search = strings.TrimSpace(text[4:])
			continue
		case strings.HasPrefix(text, "anon ") && strings.TrimSpace(text[5:]) != "":
			bb.addtoboard(text[5:], ll.Title, ll.Date, true)
		case cmd == "w":
			back += length
			if len(bb.B[index].Contents) <= length {
				back = 0
			} else if back > len(bb.B[index].Contents)-length {
				back = len(bb.B[index].Contents) - length
			}
		case cmd == "s":
			back -= length
			if back < 0 {
				back = 0
			}
		case cmd == "h" || cmd == "H":
			callclear()
			fmt.Println(helpstring)
			scanner.Scan()
			continue
		case cmd == "l":
			callclear()
			bb.viewurl(index)
			fmt.Printf("client + URL Index >> ")
			if !scanner.Scan() {
				break
			}
			if urlindex, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil {
				if url := bb.loadgem(index, urlindex); url != "" {
					intmux(url, per.Browser)
					fmt.Println("Opening browser. Press ENTER to continue.")
				} else {
					fmt.Println("No gemini link at that index. Press ENTER to continue.")
				}
				scanner.Scan()
			}
			continue
		case cmd == "r" || cmd == "fil" || cmd == "anon":
			// just refresh
		default:
			bb.addtoboard(text, ll.Title, ll.Date, false)
		}
		rehash()
	}
	rehash()
}

func switchstring(allPtr *bool, savePtr, addPtr *string, loadPtr *int) string {
	var out string
	if allPtr != nil && *allPtr { //ynnn
		out += "y"
	} else {
		out += "n"
	}
	if savePtr != nil && *savePtr != "" { //nynn
		out += "y"
	} else {
		out += "n"
	}
	if loadPtr != nil && *loadPtr != 99999 { //nnyn
		out += "y"
	} else {
		out += "n"
	}
	if addPtr != nil && *addPtr != "" { //nnny
		out += "y"
	} else {
		out += "n"
	}
	return out
}

func initiateBB(input []string) {
	u, err := user.Current()
	if err != nil {
		fmt.Println(err)
		return
	}
	dirs := strings.Split(u.HomeDir, "/")
	if len(dirs) == 3 {
		username = dirs[2]
	}
	if !check(username) {
		return
	}
	setPaths()
	pin.Load()
	an.Load()
	mm.Load()
	mm.IsMod()
	// These usually exist already; errors are ignored on purpose.
	_ = os.Mkdir(homefilepath, 0777)
	_ = os.Mkdir(snapfilepath, 0777)
	_ = os.Mkdir(modfilepath, 0777)
	per.Load()
	bb.Load()
	aa.Load()
	additionalargs := "u - get list of updated boards\nmod - access moderation mode"
	allPtr := flag.Bool("p", false, "\nPrint all boards\nNo args for interactive mode\nAdditional args:\n"+additionalargs)
	savePtr := flag.String("c", ``, "Title of new board you want to create (one word only)")
	loadPtr := flag.Int("l", 99999, "Index of board you want to load")
	addPtr := flag.String("a", ``, "Add input to board you were last accessing")
	flag.Parse()
	var switchboard = switchstring(allPtr, savePtr, addPtr, loadPtr)
	switch switchboard {
	case "ynnn":
		bb.loadall(aa, "")
		aa = bb.saveSnapshot()
		aa.Save()
	case "nynn":
		newboard(*savePtr, bb)
	case "nnyn":
		bb.loadboard(*loadPtr, "")
	case "nnny":
		ll.Load()
		bb.addtoboard(*addPtr, ll.Title, ll.Date, false)
	default:
		if len(input) == 2 {
			if input[1] == "u" {
				newstuff := aa.Whatsnew()
				if len(newstuff) == 0 {
					fmt.Printf("No updates")
				} else {
					fmt.Printf("Board updates on: ")
					for index := range newstuff {
						fmt.Printf("%s", newstuff[index])
						if index != len(newstuff)-1 {
							fmt.Printf(", ")
						}
					}
				}
				fmt.Printf("\n")
			}
			if input[1] == "init" { //for ADMIN only - do this once at the start of BB
				if username == admin {
					savedUmask := setUmask(0)
					err := os.Mkdir(homeroot+username+"/.bbmod", 0777)
					if err != nil {
						fmt.Println(err)
					}
					mm.AddMod(username)
					mm.IsMod()
					mm.Save()
					pin.Save()
					an.Save()
					_ = setUmask(savedUmask) // Return the umask to the original
					fmt.Println("the files have now either been created & bb is initiated - or they already existed!")
				} else {
					fmt.Println("you tried to initiate bb - but you are not the admin. The admin is: " + admin)
				}
			}
			if input[1] == "mod" {
				if username == admin {
					fmt.Println("###you are admin###")
					fmt.Println("additional args:")
					fmt.Println("init (initialise BB)")
					fmt.Println("mod + (add user as mod)")
					fmt.Println("mod - (remove user as mod)")
					fmt.Println("")
				}
				if ismod {
					fmt.Println("you are a bb moderator")
					fmt.Println(`
mod args:
	mod del index -- delete/archive a board at specific index
	(use standard -p arg to print board and select index)
	`)
				} else {
					fmt.Println("you are not a moderator")
				}
				fmt.Println("Moderators:")
				fmt.Println(mm.Name)
			}
		}
		if len(input) == 4 {
			if input[2] == "del" && ismod {
				bb.Load()
				ix, _ := strconv.Atoi(input[3])
				mm.Archive(ix)
				mm.Save()
				mm.Load()
			}
			if os.Args[2] == "+" && username == admin {
				mm.AddMod(input[3])
				mm.Save()
			}
			if os.Args[2] == "-" && username == admin {
				mm.RemoveMod(input[3])
				mm.Save()
			}
		}
	}
	if len(os.Args) == 1 {
		ViewBB("", nil, 10)
	}
}
