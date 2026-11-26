package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
)

const usage = `Usage:
  bb                  interactive mode (h for help)
  bb -p               print all boards
  bb -c title         create a board
  bb -l index         print a board (and make it the one -a posts to)
  bb -a text          add text to the board you last opened
  bb u                list boards with new posts since you last looked

Moderators:
  bb mod              show your status and the list of moderators
  bb mod del index    archive a board (see bb -p for indexes)
  bb mod list         list archived boards
  bb mod restore n    restore a board you archived (n from bb mod list)

Admin:
  bb init             set up bb (once)
  bb mod + user       make user a moderator
  bb mod - user       remove user as a moderator
`

// run is bb's entry point: it works out who is running it and does what args ask.
func run(cfg Config, args []string, stdin io.Reader) int {
	u, err := user.Current()
	if err != nil {
		fmt.Fprintln(out, err)
		return 1
	}
	name := homeUser(cfg.HomeRoot, u.HomeDir)
	if name == "" {
		// Without a /home/<name> folder there is nowhere to store posts, so they would silently vanish.
		fmt.Fprintf(out, "bb needs your home folder to be %s<username>, but yours is %s (root can't use bb - su to a user first).\n", cfg.HomeRoot, u.HomeDir)
		return 1
	}
	a := newApp(cfg, name)
	if !a.isAdmin() && !a.initialised() {
		fmt.Fprintf(out, "the admin: '%s' has not yet ran / initiated bb.\n", cfg.Admin)
		return 1
	}
	a.setupDirs()
	a.load()
	return a.command(args, stdin)
}

// command runs one bb command line (without the program name).
func (a *App) command(args []string, stdin io.Reader) int {
	fs := flag.NewFlagSet("bb", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	printAll := fs.Bool("p", false, "")
	create := fs.String("c", "", "")
	open := fs.Int("l", 0, "")
	add := fs.String("a", "", "")
	if err := fs.Parse(args); err != nil {
		return usageError(err.Error())
	}
	var set []string
	fs.Visit(func(f *flag.Flag) { set = append(set, f.Name) })
	rest := fs.Args()
	if len(set) > 1 || (len(set) == 1 && len(rest) > 0) {
		return usageError("use one option or command at a time")
	}

	if len(set) == 1 {
		switch set[0] {
		case "p":
			if *printAll {
				a.printIndex("", 0)
				a.saveSnapshot()
			}
		case "c":
			return result(a.createBoard(*create))
		case "l":
			b, err := a.board(*open)
			if err != nil {
				return result(err)
			}
			a.printBoard(b.BoardRef, "", 0, false)
		case "a":
			var last BoardRef
			_ = readJSON(a.privateFile(lastName), &last)
			return result(a.post(last, *add, false))
		}
		return 0
	}

	switch {
	case len(rest) == 0:
		a.interactive(stdin)
	case len(rest) == 1 && rest[0] == "u":
		if news := a.whatsNew(); len(news) == 0 {
			fmt.Fprintln(out, "No updates")
		} else {
			fmt.Fprintln(out, "Board updates on: "+strings.Join(news, ", "))
		}
	case len(rest) == 1 && rest[0] == "init":
		return result(a.initialise())
	case rest[0] == "mod":
		return a.modCommand(rest[1:])
	default:
		return usageError("unknown command: " + strings.Join(rest, " "))
	}
	return 0
}

func (a *App) modCommand(args []string) int {
	switch {
	case len(args) == 0:
		if a.isAdmin() {
			fmt.Fprint(out, "###you are admin###"+adminHelp)
		}
		if a.isMod() {
			fmt.Fprint(out, "you are a bb moderator"+modHelp)
		} else {
			fmt.Fprintln(out, "you are not a moderator")
		}
		fmt.Fprintln(out, "Moderators:", a.mods.moderators)
	case len(args) == 1 && args[0] == "list":
		if !a.isMod() {
			return result(fmt.Errorf("only moderators can list archived boards"))
		}
		a.printArchived()
	case len(args) == 2 && (args[0] == "del" || args[0] == "restore"):
		n, err := strconv.Atoi(args[1])
		if err != nil {
			return usageError("expected a number, got " + args[1])
		}
		if args[0] == "del" {
			return result(a.archive(n))
		}
		return result(a.restore(n))
	case len(args) == 2 && (args[0] == "+" || args[0] == "-"):
		return result(a.setModerator(args[1], args[0] == "+"))
	default:
		return usageError("unknown mod command: " + strings.Join(args, " "))
	}
	return 0
}

func result(err error) int {
	if err != nil {
		fmt.Fprintln(out, err)
		return 1
	}
	return 0
}

func usageError(msg string) int {
	fmt.Fprintln(os.Stderr, "bb: "+msg)
	fmt.Fprint(os.Stderr, usage)
	return 2
}
