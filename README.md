# bb for LINUX
a pubix bulletin board in go

Built this over the last 10 days to serve as a 
bulletin board for pubnix: https://heathens.club/


We wanted a custom board. So I built one. That's it!

Instructions:

```
(ROW 42 and 43)
	admin      = "username" //set this to your home folder name on your pubnix
	boardtitle = "Title of the board" //set this to whatever you want
```
Once you've done that, just build it & run it i.e
go build bb.go

Then run "./bb mod" to initialise all the folders.

You'll then have an up and running pubnix bulletin board. Just make sure all other users have access to the application.

