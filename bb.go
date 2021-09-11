package main

/*
   Copyright (C) 2021 - tom@fern91.com

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.
   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.
   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

var (
	//THESE TWO VARS MUST BE CONFIGURED
	admin      = "tom" ////////////////////////// the username for administrator. As administrator you MUST run "./bb init" to ensure BB is set up correctly.
	boardtitle = "heathens.club bb"
)

func check(username string) bool {
	if username == admin {
		return true
	}

	if _, err := os.Stat("/home/" + admin + "/.bb"); os.IsNotExist(err) {
		fmt.Printf("the admin: '%s' has not yet ran / initiated bb.\n", admin)
		return false
	}

	return true
}

func main() { //Main entry function where flag vars are set up.d
	initiateBB(os.Args)
}

func initiateBB(input []string) {
	u, err := user.Current()
	if err != nil {
		fmt.Println(err)
	}
	var username string
	dirs := strings.Split(u.HomeDir, "/")
	if len(dirs) == 3 {
		username = dirs[2]
	}
	if !check(username) {
		return
	}
	pin.Load()
	an.Load()
	mm.Load()
	mm.IsMod()
	homefilepath = "/home/" + username + "/.bb/"
	snapfilepath = "/home/" + username + "/.bbsn/"
	modfilepath = "/home/" + username + "/.bbmod/"
	err = os.Mkdir("/home/"+username+"/.bb", 0777)
	if err != nil {
		fmt.Println(err)
	}
	err = os.Mkdir("/home/"+username+"/.bbsn", 0777)
	if err != nil {
		fmt.Println(err)
	}
	err = os.Mkdir("/home/"+username+"/.bbmod/", 0777)
	if err != nil {
		fmt.Println(err)
	}
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
					savedUmask := syscall.Umask(0)
					err := os.Mkdir("/home/"+username+"/.bbmod", 0777)
					if err != nil {
						fmt.Println(err)
					}
					mm.AddMod(username)
					mm.IsMod()
					mm.Save()
					pin.Save()
					an.Save()
					_ = syscall.Umask(savedUmask) // Return the umask to the original
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
