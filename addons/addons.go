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

//Exported function - all user chat input is passed through this function. Feel free to edit.
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
