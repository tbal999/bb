package converter

import (
	"strings"
)

/*
For a couple of hours I used the 'original' dataset which proved too limiting.
But there was already too much data in the app, so this converter was necessary.
*/

type ORIGINAL struct {
	Date     string   `json:Date`
	Owner    string   `json:Owner`
	Title    string   `json:Title`
	Contents []string `json:Contents`
}

type NEW1 struct {
	Date     string     `json:Date`
	Owner    string     `json:Owner`
	Title    string     `json:Title`
	Contents [][]string `json:Contents`
}

func (o ORIGINAL) Convert() NEW1 {
	a := NEW1{}
	a.Date = o.Date
	a.Title = o.Title
	a.Owner = o.Owner
	for index := range o.Contents {
		tobeadded := []string{}
		item0 := strings.Split(o.Contents[index], " | ")[0]
		item1 := strings.Split(o.Contents[index], " | ")[1]
		tobeadded = append(tobeadded, item0)
		tobeadded = append(tobeadded, item1)
		a.Contents = append(a.Contents, tobeadded)
	}
	return a
}
