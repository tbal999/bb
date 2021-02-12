# bb for LINUX
Built this over the last 10 days to serve as a 
bulletin board for pubnix: https://heathens.club/


We wanted a custom board. So I built one. That's it!

Instructions:

```
(ROW 42 and 43 in bb.go)
	admin      = "username" //set this to your home folder name on your pubnix
	boardtitle = "Title of the board" //set this to whatever you want
```
Once you've done that, just build it & run it i.e
go build bb.go

Then run "./bb mod" to initialise all the folders.

You'll then have an up and running pubnix bulletin board. Just make sure all other users have access to the application. Maybe place it in the binary folder.

You may need to edit the permissions in the admin .bbmod folder so that bban0n.json & bbp1n.json are both rw-rw-rw.

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
	
for CHAT section:
	q - exits back to index section
	r - refresh the board you are on
	fil - filter chat by specific string e.g YYYY-MM or substring
	w - scroll up the board
	d - scroll down the board
	anon - make message anonymous
	anything else - types text to board
	nothing - also exits back to index section
	ctrl-c to quit
	
FYI:
	- Boards glow cyan when new content is posted
	- New boards glow green.
	- You can comment other people via @ sign i.e @person
		they will see message highlighted
	- If you are on a board and new content is posted on another board, 
		you'll see '^new' beside author name
```

