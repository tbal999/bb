package main

import (
	"bb/addons"
	bot "bb/urlparse"
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil" //deprecated will need to update this soon
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color" //to add colour to output
	"mvdan.cc/xurls/v2"      //to parse URLs

	"github.com/pterm/pterm" //to make terminal pretty
)

var (
	//Everything below does not need to be configured
	ismod          bool
	clear          map[string]func()               // create a map for storing clear funcs
	username       string                          // store the username of the individual currently using BB
	homefilepath   string                          // the path for where the bb config files are
	snapfilepath   string                          // the path for where the bb snapshot file is
	masterfilepath = "/home/" + admin + "/.bbmod/" // mods will be added to this file. Will also store the anonymous file.
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
	maximum        int //board scroll logic
	minimum        int //board scroll logic
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

//GENERAL FUNCTIONS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
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

//Clear screen
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

//Grab a timestamp.
func timestamp() string {
	t := time.Now()
	var x = t.Format("2006-01-02 15:04:05")
	return x
}

//create a new board for the global BB variable.
func newboard(title string, bb BB) {
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

//remove a board from BB at a specific index
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

//Does 2D string array already have a specific string in it?
func alreadyhas(s [][]string, st string) bool {
	for index := range s {
		for index2 := range s[index] {
			if s[index][index2] == st {
				return true
			}
		}
	}
	return false
}

func grabgeminiurl(input string) string {
	if strings.Contains(input, "gemini://") {
		rxRelaxed := xurls.Relaxed()
		astring := rxRelaxed.FindString(input)
		if astring != "" { //--------------------------GEM grap input print out a fancy Title
			return "gemi" + astring
		}
	}
	return ""
}

//Add a slice to a 2d slice
func add2slice(ax *[][]string, b []string) [][]string {
	a := *ax
	a = append(a, b)
	*ax = a
	return a
}

//Grab a list folders in home dir (to grab the Usernames)
func ufolderlist() []string {
	output := []string{}
	files, _ := ioutil.ReadDir("/home/")
	for _, f := range files {
		output = append(output, f.Name())
	}
	return output
}

//Refresh the BB data with any new inputs.
func rehash() {
	pin.Save()
	an.Save()
	bb = BB{} //Clear BB
	bb.Load() //Reload BB
}

//Personal METHODS//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
type Personal struct {
	Browser string
}

//Save file
func (p Personal) Save() {
	Base := &p
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(snapfilepath+pername, output, 0666)
	if err != nil {
		fmt.Println(err)
	}
}

//Load file
func (p *Personal) Load() {
	item := *p
	jsonFile, _ := ioutil.ReadFile(snapfilepath + pername)
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
	p := *P
	titletosave := ""
	datetosave := ""
	exists := false
	for index2 := range bb.B {
		if index2 == ix {
			titletosave = bb.B[index2].Title
			datetosave = bb.B[index2].Date
		}
	}
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
	p := *P
	titletodel := ""
	datetodel := ""
	for index2 := range bb.B {
		if index2 == ix {
			titletodel = bb.B[index2].Title
			datetodel = bb.B[index2].Date
			break
		}
	}
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

//Save mod file
func (p Pin) Save() {
	Base := &p
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(masterfilepath+pinname, output, 0666)
	if err != nil {
		fmt.Println(err)
	}
}

//Load pins
func (p *Pin) Load() {
	item := *p
	jsonFile, _ := ioutil.ReadFile(masterfilepath + pinname)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*p = item
}

///Lastfile & Last METHODS //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Last struct {
	Title string
	Date  string
}

//Save last file
func (l Last) Save() {
	Base := &l
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(snapfilepath+lastname, output, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

//Load mods
func (l *Last) Load() {
	item := *l
	jsonFile, _ := ioutil.ReadFile(snapfilepath + lastname)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*l = item
}

///Moderator & Mod METHODS //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//Only the administrator can be a moderator at the moment
type Mod struct {
	Name         []string
	Boardarchive []string
	Datearchive  []string
}

//Save mod file
func (m Mod) Save() {
	if ismod {
		Base := &m
		output, err := json.MarshalIndent(Base, "", "\t")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = ioutil.WriteFile(modfilepath+"mod.json", output, 0644)
		if err != nil {
			fmt.Println(err)
		}
	}
}

//Load mods
func (m *Mod) Load() {
	item := *m
	jsonFile, _ := ioutil.ReadFile(masterfilepath + "mod.json") //no need to worry about error
	_ = json.Unmarshal([]byte(jsonFile), &item)                 //but leaving it empty in case one day... i need to worry
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
		modp := "/home/" + homeuser + "/.bbmod/"
		learnfolder, err := ioutil.ReadDir(modp)
		if err == nil {
			for _, learnfile := range learnfolder {
				if learnfile.Name() == "mod.json" {
					mm := Mod{}
					jsonFile, _ := ioutil.ReadFile(modp + "mod.json") //no need to worry about error
					_ = json.Unmarshal([]byte(jsonFile), &mm)         //but leaving it empty in case one day... i need to worry
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

//Check if a board is on the archive list - this is alternative to delete. If a board is on board archive it won't load
func (m Mod) Check(b Board) bool {
	for index := range m.Boardarchive {
		if m.Boardarchive[index] == b.Title && m.Datearchive[index] == b.Date {
			return true
		}
	}
	return false
}

//Checks if current user is a mod
func (m Mod) IsMod() {
	for index := range m.Name {
		if m.Name[index] == username {
			ismod = true
		}
	}
}

//Checks if current user is a mod
func (m Mod) IsUserMod(uname string) bool {
	for index := range m.Name {
		if m.Name[index] == uname {
			return true
		}
	}
	return false
}

//Lets moderators archive a board by indoex
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

//Lets you add a mod if you are admin
func (m *Mod) AddMod(user string) {
	for index := range m.Name {
		if m.Name[index] == user {
			return
		}
	}
	m.Name = append(m.Name, user)
}

//Lets you add a mod if you are admin
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

//Save mod file
func (a Anon) Save() {
	Base := &a
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(masterfilepath+anonname, output, 0666)
	if err != nil {
		fmt.Println(err)
	}
}

//Load mods
func (a *Anon) Load() {
	item := *a
	jsonFile, _ := ioutil.ReadFile(masterfilepath + anonname)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*a = item
}

func (a *Anon) Add(title, date string, contents []string) {
	A := *a
	exists := false
	for index := range A.Title {
		if A.Title[index] == title && A.Date[index] == date {
			exists = true
			a.Board[index].Contents = append(a.Board[index].Contents, contents)
		}
	}
	if !exists {
		A.Title = append(A.Title, title)
		A.Date = append(A.Date, date)
		board := Anonboard{}
		board.Contents = append(board.Contents, contents)
		A.Board = append(A.Board, board)
	}
	*a = A
}

///Snapshot & Snapshot METHODS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//Snap is struct for storing snapshot of every board in BB
type Snap struct {
	Owner   []string
	Title   []string
	Date    []string
	Length  []int
	Checked []bool
}

//Save snapshot to json
func (s Snap) Save() {
	Base := &s
	output, err := json.MarshalIndent(Base, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(snapfilepath+snapname, output, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

//Load snapshot to json
func (s *Snap) Load() {
	item := *s
	jsonFile, _ := ioutil.ReadFile(snapfilepath + snapname)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*s = item
}

//If we've just read a board, we want to switch that board from 'not checked' to 'checked'.
func (s *Snap) Switch(title, date string) {
	S := *s
	for index := range S.Title {
		if S.Title[index] == title && S.Date[index] == date {
			S.Checked[index] = true
		}
	}
	*s = S
}

func (s Snap) Exists(title, date string) bool {
	for index := range s.Title {
		if s.Title[index] == title && s.Date[index] == date {
			return true
		}
	}
	return false
}

func (s Snap) Whatsnew() []string {
	news := []string{}
	for index := range s.Title {
		for index2 := range bb.B {
			if bb.B[index2].Title == s.Title[index] && bb.B[index2].Date == s.Date[index] && len(bb.B[index2].Contents) != s.Length[index] {
				news = append(news, bb.B[index2].Title)
			}
		}
	}
	return news
}

///BB & BB METHODS////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//BB is the main struct holding all boards.
type BB struct {
	B []Board
}

//Save a snapshot of the global BB obj to global Snap obj
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

//Load all the boards to BB
func (b *BB) Load() {
	an.Load()
	list := ufolderlist()
	for index := range list {
		b.collect(list[index])
	}
	b.anoncollect()
}

//Iterate through snapshot to check if something has changed (at BB level)
func (b BB) snapcheck(s Snap) bool {
	for index := range b.B {
		for index2 := range s.Title {
			if b.B[index].Title == s.Title[index2] && b.B[index].Date == s.Date[index2] {
				if len(b.B[index].Contents) != s.Length[index2] && !s.Checked[index2] {
					return true
				}
			}
		}
	}
	return false
}

func (b BB) loadpin(s Snap) []int {
	indexlist := []int{}
	for index := range b.B {
		for indexp := range pin.Title {
			if pin.Title[indexp] == b.B[index].Title && pin.Date[indexp] == b.B[index].Date {
				indexlist = append(indexlist, index)
				strindex := strconv.Itoa(index)
				repeat := false
				for index2 := range s.Title {
					if b.B[index].Title == s.Title[index2] && len(b.B[index].Contents) != s.Length[index2] && !s.Checked[index2] {
						color.Cyan(strindex + ") " + b.B[index].Title + " | author: " + b.B[index].Owner + " | " + b.B[index].Date + " **PINNED**")
						repeat = true
						break
					}
					if !s.Exists(b.B[index].Title, b.B[index].Date) {
						color.Green(strindex + ") " + b.B[index].Title + " | author: " + b.B[index].Owner + " | " + b.B[index].Date + " **PINNED**")
						repeat = true
						break
					}
				}
				if !repeat {
					fmt.Println(strindex + ") " + b.B[index].Title + " | author: " + b.B[index].Owner + " | " + b.B[index].Date + " **PINNED**")
				}
			} else {
				continue
			}
		}
	}
	fmt.Println("")
	return indexlist
}

func checkindexlist(list []int, index int) bool {
	for ix := range list {
		if list[ix] == index {
			return true
		}
	}
	return false
}

//Load all contents of BB to the screen.
func (b BB) loadall(s Snap, searchstring string) {
	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue))
	change := bb.snapcheck(aa)
	var truemin int
	var truemax int
	if change {
		pterm.Println(header.Sprint("-- " + boardtitle + " -- [NEW]"))
	} else {
		pterm.Println(header.Sprint("-- " + boardtitle + " --"))
	}
	indexlist := b.loadpin(s)
	if len(b.B) <= length {
		minimum = 0
		maximum = len(b.B)
	} else {
		minimum = len(b.B) - length
		maximum = len(b.B)
	}
	if searchstring != "" {
		truemin = 0
		truemax = len(b.B)
	} else {
		truemin = minimum - back
		if truemin < 0 {
			truemin = 0
		}
		truemax = maximum - back
	}
	for index := range b.B {
		if index >= truemin && index <= truemax && !checkindexlist(indexlist, index) && b.B[index].Date != "" {
			strindex := strconv.Itoa(index)
			repeat := false
			searching := false
			for index2 := range s.Title {
				if searchstring != "" {
					if !strings.Contains(b.B[index].Title, searchstring) && !strings.Contains(b.B[index].Date, searchstring) {
						searching = true
						break
					}
				}
				if b.B[index].Title == s.Title[index2] && len(b.B[index].Contents) != s.Length[index2] && !s.Checked[index2] {
					color.Cyan(strindex + ") " + b.B[index].Title + " | author: " + b.B[index].Owner + " | " + b.B[index].Date)
					repeat = true
					break
				}
				if !s.Exists(b.B[index].Title, b.B[index].Date) {
					color.Green(strindex + ") " + b.B[index].Title + " | author: " + b.B[index].Owner + " | " + b.B[index].Date)
					repeat = true
					break
				}

			}
			if !repeat && !searching {
				fmt.Println(strindex + ") " + b.B[index].Title + " | author: " + b.B[index].Owner + " | " + b.B[index].Date)
			}
		} else {
			continue
		}
	}
	if len(b.B) == 0 {
		fmt.Println("there are no boards")
	} else {
		fmt.Println("")
		fmt.Printf("BB Length: %d, range: %d to %d", len(b.B), truemin, truemax)
		fmt.Printf("\n\n")
	}
}

func (b BB) loadgem(ix, urlindex int) string {
	var link string
	for index := range b.B {
		if index == ix {
			for index2 := range b.B[index].Contents {
				if index2 == urlindex {
					if len(b.B[index].Contents[index2]) == 3 {
						link = grabgeminiurl(b.B[index].Contents[index2][1])
						break
					}
				}
			}
		}
	}
	return link
}

func (b BB) viewurl(ix int) bool {
	change := bb.snapcheck(aa)
	var real bool
	var truemin int
	var truemax int
	for index := range b.B {
		if index == ix && b.B[index].Owner != "" && b.B[index].Date != "" {
			aa.Switch(b.B[index].Title, b.B[index].Date)
			sort.Slice(b.B[index].Contents, func(i, j int) bool { return b.B[index].Contents[i][0] < b.B[index].Contents[j][0] })
			real = true
			ll.Title = b.B[index].Title
			ll.Date = b.B[index].Date
			ll.Save()
			if !change {
				pterm.DefaultSection.Println(b.B[index].Title + " | " + b.B[index].Owner + " ^new")
			} else {
				pterm.DefaultSection.Println(b.B[index].Title + " | " + b.B[index].Owner)
			}
			fmt.Println("")
			if len(b.B[index].Contents) <= length {
				minimum = 0
				maximum = len(b.B[index].Contents)
			} else {
				minimum = len(b.B[index].Contents) - length
				maximum = len(b.B[index].Contents)
			}
			truemin = minimum - back
			if truemin < 0 {
				truemin = 0
			}
			truemax = maximum - back
			for index2 := range b.B[index].Contents {
				if index2 >= truemin && index2 <= truemax {
					urlindex := strconv.Itoa(index2)
					if len(b.B[index].Contents[index2]) == 3 && grabgeminiurl(b.B[index].Contents[index2][1]) != "" {
						if strings.Contains(b.B[index].Contents[index2][1], "@"+username) {
							color.Cyan(urlindex + ") " + grabgeminiurl(b.B[index].Contents[index2][1]))
						} else {
							fmt.Println(urlindex + ") " + grabgeminiurl(b.B[index].Contents[index2][1]))
						}
					}
				} else {
					continue
				}
			}
		}
	}
	if !real {
		fmt.Println("invalid index")
		return false
	} else {
		fmt.Println("")
		fmt.Printf("Board length: %d, range: %d to %d", len(b.B[ix].Contents), truemin, truemax)
		fmt.Printf("\n\n")
	}
	return true
}

//Load specific board up.
func (b *BB) loadboard(ix int, searchstring string) bool {
	btp := &pterm.BasicTextPrinter{}
	bgp := btp.WithStyle(&pterm.ThemeDefault.SuccessMessageStyle)
	change := bb.snapcheck(aa)
	var real bool
	var truemin int
	var truemax int
	for index := range b.B {
		if index == ix && b.B[index].Owner != "" && b.B[index].Date != "" {
			aa.Switch(b.B[index].Title, b.B[index].Date)
			sort.Slice(b.B[index].Contents, func(i, j int) bool { return b.B[index].Contents[i][0] < b.B[index].Contents[j][0] })
			real = true
			ll.Title = b.B[index].Title
			ll.Date = b.B[index].Date
			ll.Save()
			if change {
				pterm.DefaultSection.Println(b.B[index].Title + " | " + b.B[index].Owner + " ^new")
			} else {
				pterm.DefaultSection.Println(b.B[index].Title + " | " + b.B[index].Owner)
			}
			fmt.Println("")
			if len(b.B[index].Contents) <= length {
				minimum = 0
				maximum = len(b.B[index].Contents)
			} else {
				minimum = len(b.B[index].Contents) - length
				maximum = len(b.B[index].Contents)
			}
			if searchstring != "" {
				truemin = 0
				truemax = len(b.B[index].Contents)
			} else {
				truemin = minimum - back
				if truemin < 0 {
					truemin = 0
				}
				truemax = maximum - back
			}
			for index2 := range b.B[index].Contents {
				if index2 >= truemin && index2 <= truemax {
					if len(b.B[index].Contents[index2]) == 3 {
						if searchstring != "" && !(strings.Contains(b.B[index].Contents[index2][1], searchstring)) && !(strings.Contains(b.B[index].Contents[index2][0], searchstring)) {
							continue
						}
						if strings.Contains(b.B[index].Contents[index2][1], "@"+username) {
							bgp.Println(b.B[index].Contents[index2][0] + " <" + b.B[index].Contents[index2][2] + "> " + b.B[index].Contents[index2][1])
						} else {
							btp.Println(b.B[index].Contents[index2][0] + " <" + b.B[index].Contents[index2][2] + "> " + b.B[index].Contents[index2][1])
						}
					}
				} else {
					continue
				}
			}
		}
	}
	if !real {
		fmt.Println("invalid index")
		return false
	} else {
		fmt.Println("")
		fmt.Printf("Board length: %d, range: %d to %d", len(b.B[ix].Contents), truemin, truemax)
		fmt.Printf("\n\n")
	}
	return true
}

//Delete a board in BB. Only works if nobody else has viewed the board. Only sudo / root can delete regardless of this.
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
			if B.B[index].Title == an.Title[index2] {
				for index3 := range an.Board[index2].Contents {
					B.B[index].Contents = append(B.B[index].Contents, an.Board[index2].Contents[index3])
				}
			}
		}
	}
	*b = B
}

//Collect all data from every user to load into BB - making sure to order boards by date created.
func (b *BB) collect(homeuser string) {
	ownercount := make(map[string][]string)
	homepath := "/home/" + homeuser + "/.bb/"
	B := *b
	learnfolder, err := ioutil.ReadDir(homepath)
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
						for index2 := range x.Contents {
							if !alreadyhas(B.B[index].Contents, x.Contents[index2][1]) {
								B.B[index].Contents = add2slice(&B.B[index].Contents, x.Contents[index2])
							}
						}
					}
				}
				for key, element := range ownercount {
					if len(element) > 1 {
						var users string
						for index := range element {
							if index != len(element) {
								users += element[index] + ","
							} else {
								users += element[index]
							}
						}
						fmt.Printf("Topic '%s' ownership has been tampered with %d times, potential owners are: %s\n", key, len(element)-1, users) //You could add a log here
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
	botindex := 0
	for index := range b.B {
		if b.B[index].Title == title && b.B[index].Date == date {
			botindex = index
			break
		}
	}
	if strings.Contains(input, "http://") || strings.Contains(input, "https://") {
		rxRelaxed := xurls.Relaxed()
		astring := rxRelaxed.FindString(input)
		if astring != "" { //--------------------------HTTP grap input print out a fancy Title
			httpTitle := bot.HTTPget(astring)
			b.addURLtitle(botindex, httpTitle, input, title, date, anon)
		}
	} else if strings.Contains(input, "gemini://") {
		rxRelaxed := xurls.Relaxed()
		astring := rxRelaxed.FindString(input)
		if astring != "" { //--------------------------GEM grap input print out a fancy Title
			httpTitle := bot.GEMGet("gemi" + astring)
			b.addURLtitle(botindex, httpTitle, input, title, date, anon)
		}
	} else {
		item := []string{}
		item = append(item, timestamp())
		if anon {
			item = append(item, addons.Parse(input))
			item = append(item, "???")
			an.Add(title, date, item)
		} else {
			item = append(item, addons.Parse(input))
			item = append(item, username)
			b.B[botindex].Contents = append(b.B[botindex].Contents, item)
		}
	}
	b.B[botindex].Save(b.B[botindex].Title)
}

//adds the URL title of a website to the end of a user's input.
func (b *BB) addURLtitle(botindex int, input, userinput, title, date string, anon bool) {
	B := *b
	item := []string{}
	item = append(item, timestamp())
	if anon {
		item = append(item, addons.Parse(userinput)+" | Title: "+input)
		item = append(item, "???")
		an.Add(title, date, item)
	} else {
		item = append(item, addons.Parse(userinput)+" | Title: "+input)
		item = append(item, username)
		B.B[botindex].Contents = append(B.B[botindex].Contents, item)
	}
	*b = B
}

//Board is the struct that holds all data about a specific board
type Board struct {
	Date     string
	Owner    string
	Title    string
	Contents [][]string
}

func (b *Board) Load(filename string) {
	item := *b
	jsonFile, _ := ioutil.ReadFile(filename)
	_ = json.Unmarshal([]byte(jsonFile), &item)
	*b = item
}

func (b Board) Delete(filename string) { //ONLY WORKS FOR ROOT AND SUDO USERS
	var afilepath string
	list := ufolderlist()
	for index := range list {
		afilepath = "/home/" + list[index] + "/.bb/"
		bb.collect(list[index])
		files, err := ioutil.ReadDir(afilepath)
		if err != nil {
			continue //Maybe this user has not actually used BB yet!
		}
		for _, f := range files {
			if strings.Split(f.Name(), ".")[0] == filename {
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

//Save the board
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
	err = ioutil.WriteFile(homefilepath+filename+".json", output, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

//View the entire BB
func ViewBB(search string, in *os.File, speed int) {
	if in == nil {
		in = os.Stdin
	}
	ticker := time.NewTicker(time.Duration(speed) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			{
				callclear()
				bb.loadall(aa, search)
				search = ""
				fmt.Printf("--enter 'h' for help--\n")
				fmt.Printf("Pick Index >> ")
				Scanner := bufio.NewScanner(in)
				Scanner.Scan()
				results := Scanner.Text()
				switch results {
				case "b":
					fmt.Printf("Enter client name/path > ")
					Scanner.Scan()
					per.Browser = Scanner.Text()
					per.Save()
					continue
				case "q":
					aa = bb.saveSnapshot() //Create snapshot of whatever BB currently is
					aa.Save()              //Save snapshot to file
					return
				case "r":
					aa = bb.saveSnapshot() //Create snapshot of whatever BB currently is
					aa.Save()              //Save snapshot to file
					rehash()
					continue
				case "w":
					back += length
					if len(bb.B) <= length {
						back = 0
					} else {
						if back > len(bb.B)-length {
							back = len(bb.B) - length
						}
					}
					continue
				case "s":
					back -= length
					if back < 0 {
						back = 0
					}
					continue
				case "h":
					callclear()
					fmt.Println(helpstring)
					//time.Sleep(2 * time.Duration(time.Second))
					Scanner.Scan()
					continue
				}
				if len(results) > 3 {
					if len(results) > 4 {
						if results[0:3] == "new" {
							newboard(results[4:], bb)
							continue
						}
						if results[0:3] == "del" {
							index, _ := strconv.Atoi(results[4:])
							bb.delboard(index)
							continue
						}
						if results[0:3] == "fil" {
							search = string(results[4:])
							continue
						}
					}
					if len(results) > 5 {
						if results[0:4] == "pin+" {
							index, _ := strconv.Atoi(string(results[5:]))
							pin.Add(index)
							continue
						}
						if results[0:4] == "pin-" {
							index, _ := strconv.Atoi(string(results[5:]))
							pin.Remove(index)
							continue
						}
					}
				} else {
					if results != "" {
						index, _ := strconv.Atoi(results)
						back = 0
						Viewboard(index, search, in)
						continue
					}
				}
			}
		}
	}
}

func Viewboard(index int, search string, in *os.File) {
	if in == nil {
		in = os.Stdin
	}
	Scanner := bufio.NewScanner(in)
	for {
		callclear()
		real := bb.loadboard(index, search)
		if !real {
			return
		}
		search = ""
		fmt.Printf("Add >> ")
		Scanner.Scan()
		if Scanner.Text() != "" {
			if len(Scanner.Text()) > 3 {
				if Scanner.Text()[0:3] == "fil" && len(Scanner.Text()) > 4 {
					search = string(Scanner.Text()[4:])
					continue
				}
				if Scanner.Text()[0:4] == "anon" && len(Scanner.Text()) > 5 {
					bb.addtoboard(Scanner.Text()[5:], ll.Title, ll.Date, true)
					rehash()
					continue
				}
			}
			if Scanner.Text() == "w" {
				back += length
				if len(bb.B[index].Contents) <= length {
					back = 0
				} else {
					if back > len(bb.B[index].Contents)-length {
						back = len(bb.B[index].Contents) - length
					}
				}
			}
			if Scanner.Text() == "h" || Scanner.Text() == "H" {
				callclear()
				fmt.Println(helpstring)
				fmt.Scanln()
				continue
			}
			if Scanner.Text() == "s" {
				back -= length
				if back < 0 {
					back = 0
				}
			}
			if Scanner.Text() == "l" {
				callclear()
				bb.viewurl(index)
				fmt.Printf("client + URL Index >> ")
				Scanner.Scan()
				urlindex, _ := strconv.Atoi(Scanner.Text())
				url := bb.loadgem(index, urlindex)
				intmux(url, per.Browser)
				fmt.Println("Opening browser. Press anything to continue.")
				fmt.Scanln()
				continue
			}
			if Scanner.Text() == "r" || Scanner.Text() == "fil" || Scanner.Text() == "anon" || Scanner.Text() == "w" || Scanner.Text() == "s" {
				rehash()
				continue
			} else if Scanner.Text() == "q" {
				back = 0
				break
			} else {
				bb.addtoboard(Scanner.Text(), ll.Title, ll.Date, false)
			}
		} else {
			back = 0
			break
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
