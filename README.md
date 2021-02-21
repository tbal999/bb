# bb for LINUX
Built this initially over 10 days to serve as a 
bulletin board for pubnix: https://heathens.club/

How BB looks:
```
== Pubnix BB ==
0) pinned board! | author: tom | 2021-02-12 **PINNED**

1) testboard 1 | author: tom | 2021-02-12
2) testboard 2 | author: tom | 2021-02-12
3) testboard 3 | author: tom | 2021-02-12

BB Length: 4, range: 0 to 4

--enter 'h' for help--
Pick Index >> 
```
How a board looks:
```
testboard 1 | tom

2021-02-12 14:32:05 | <tom> Hello this is some text
2021-02-12 14:32:09 | <tom> here is some more
2021-02-12 14:32:23 | <test> here is some more text from a test user!

Board length: 4, range: 0 to 4

Add >> 

```

We wanted a custom board. So I built one. That's it! Might need to clean it up a bit as I built it pretty fast, but it works as is so that's a start.

Instructions:

```
(ROW 42 and 43 in bb.go)
	admin      = "username" //set this to your home folder name on your pubnix
	boardtitle = "Title of the board" //set this to whatever you want
```
Once you've done that, just build it & run it i.e
go build bb.go

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
	
For administration/moderation:
./bb mod args:

        mod archive index -- archive a board at specific index.
        (use standard -p arg to print board and select index)
        mod p             -- print out list of archived boards

```

Interactive Instructions:
```
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
	d - scroll down the index
	b - choose gemini client (default=NONE)

for CHAT section:
	q - exits back to index section
	r - refresh the board you are on
	fil - filter chat by specific string e.g YYYY-MM or substring
	w - scroll up the board
	d - scroll down the board
	l - visit a gemini url via client
	anon - make message anonymous
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
```

You can add additional interpretive logic in the addons.go file.
There's one toy example 'rev' where if you type 'rev' in front of your message, the message will be reversed.

```
package addons

//You can add whatever you want here to interpret the input of the user. I've included an example

//simple reverse a string function
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func Parse(usertext string) (output string) {
	output = usertext
	if len(usertext) > 4 {
		switch usertext[0:3] {
		case "rev":
			output = reverse(usertext[4:]) //if text starts with 'rev' then you reverse the rest of the string
		}
	}
	return output
}
```
