# bb for LINUX
Built this initially over 10 days to serve as a 
bulletin board for pubnix: https://heathens.club/

I have refactored it a bit since then & added some tests.

This is decentralized and works like most tildespace tools/apps/programs. 
Each user will have their data stored within their home folder, and bb
will iterate through all users to collate the messages for each board.

Your data will be read only for everybody else, so everybody can read eachother's data 
and enable the board to work!

Updates:

- 21/02/2021 add a gemini client link
- 06/03/2021 improved speed of viewbb / viewboard funcs
- 13/03/2021 replaced recursion with loop / continue to prevent possible memory issues
- 25/11/2026 bug fixes & cleanup: crash fixes (unreachable gemini links, malformed board files),
  anon posts no longer get overwritten, repeated messages no longer dropped, commands like `fil`/`anon`/`rev`
  need a space so "file..."/"anonymous..."/"review..." post normally, terminal escape codes are stripped,
  real tests, Go 1.21+

How BB looks:
```
-- Pubnix BB --
0) pinned board! | author: tom | 2021-02-12 | active 2d ago **PINNED**

3) testboard 3 | author: tom | 2021-02-12 | active 3m ago
2) testboard 2 | author: tom | 2021-02-12
1) testboard 1 | author: tom | 2021-02-12 | active 5mo ago

BB Length: 4, showing 0 to 3

--enter 'h' for help--
Pick Index >> 
```
How a board looks:
```
testboard 1 | tom

2021-02-12 14:32:05 <tom> Hello this is some text
2021-02-12 14:32:09 <tom> here is some more
2021-02-12 14:32:23 <test> here is some more text from a test user!

Board length: 3, showing 0 to 2

Add >> 

```

We wanted a custom board. So I built one. That's it! Might need to clean it up a bit as I built it pretty fast, but it works as is so that's a start.

Instructions:

```
(top of bb.go)
	admin      = "username" //set this to your home folder name on your pubnix
	boardtitle = "Title of the board" //set this to whatever you want
```
Once you've done that, just build it (needs Go 1.21+) & run it i.e
```
go build -o bb .     (or: make build)
make test            (vet + tests)
make smoke           (the above in a Linux container, plus real users admin/alice/bob
                      driving bb against each other - see test/smoke/smoke.sh)
make demo            (a shell on that fake pubnix with some boards to play with:
                      su - alice, then bb. tmux is there to be two users at once)
```

Then run "./bb init" once to initialise all the necessary folders.

You'll then have an up and running pubnix bulletin board. 

Then simply place the bb exec to your bin, and make the application accessible by all users. Ta-da, you have your own private bulletin board on your pubnix.

ARGS instructions:

```
Usage of ./bb:
  -a string
        Add input to board you were last accessing
  -c string
        Title of new board you want to create (one word only)
  -l int
        Index of board you want to load (default 99999)
  -p
        Print all boards
        No args for interactive mode
        Only 1 arg at a time allowed

  u     list boards with new posts since you last looked

For administration/moderation:

        bb mod             -- show your status and the list of moderators
        bb mod del index   -- archive a board at specific index (moderators)
                              (use standard -p arg to print board and select index)
        bb mod + user      -- make user a moderator (admin only)
        bb mod - user      -- remove user as a moderator (admin only)

```

Interactive Instructions:
```
for INDEX section:
	<number> - open the board at that index
	new - create a new board i.e 'new topictitle'
	del - delete a board by index i.e 'del 3'
		If nobody else has accessed it - you can delete it. 
	        Otherwise, you need superuser permission.
	fil - filter index by search string e.g 'fil 2021-02' or 'fil Title'
	pin+ - pin a board by index i.e 'pin+ 3' (moderators)
	pin- - unpin a board by index i.e 'pin- 3' (moderators)
	q - to quit, or use ctrl-c / ctrl-d
	r - refresh the index section
	w - scroll up the index
	s - scroll down the index
	b - choose gemini client (default=amfora)

for CHAT section:
	q - exits back to index section
	r - refresh the board you are on
	fil - filter chat by specific string e.g 'fil some text'
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
```

You can add additional interpretive logic in the addons.go file.
There's one toy example 'rev' where if you type 'rev ' in front of your message, the message will be reversed.

```
package addons

import "strings"

//You can add whatever you want here to interpret the input of the user. I've included an example

// simple reverse a string function
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Exported function - all user chat input is passed through this function. Feel free to edit.
func Parse(usertext string) (output string) {
	output = usertext
	switch {
	case strings.HasPrefix(usertext, "rev "):
		output = reverse(usertext[4:]) //if text starts with 'rev ' then you reverse the rest of the string
	}
	return output
}
```
