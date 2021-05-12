package urlparse

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	gemini "github.com/makeworld-the-better-one/go-gemini"
	"golang.org/x/net/html"
)

func HTTPget(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return "Nothing, because there was a response error!"
	}
	// do this now so it won't be forgotten
	defer resp.Body.Close()
	// reads html as a slice of bytes
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "Nothing, because there was a html parse error!"
	}
	// show the HTML code as a string %s
	return getTitle(string(html))
}

func getTitle(HTMLString string) (title string) {
	r := strings.NewReader(HTMLString)
	z := html.NewTokenizer(r)
	var i int
	for {
		tt := z.Next()
		i++
		if i > 100 {
			return
		}
		switch {
		case tt == html.ErrorToken:
			return
		case tt == html.StartTagToken:
			t := z.Token()

			if t.Data != "title" {
				continue
			}
			tt := z.Next()

			if tt == html.TextToken {
				t := z.Token()
				title = t.Data
				return
			}
		}
	}
}

func gemgrabtitle(url string) (int, string, string) {
	var title string
	a := gemini.Client{}
	response, err := a.Fetch(url)
	if err != nil {
		fmt.Println(err)
	}
	err = response.SetReadTimeout(5)
	if err != nil {
		fmt.Println(err)
	}
	statuscode := response.Status
	meta := response.Meta
	data, _ := ioutil.ReadAll(response.Body)
	bodydata := strings.Split(string(data), "\n")
	for index := range bodydata {
		if len(bodydata[index]) > 0 {
			if string(bodydata[index][0]) == "#" {
				title = string(bodydata[index])
				break
			}
		}
	}
	return statuscode, meta, title
}

func GEMGet(url string) string {
	iter := 0
	maxNumber := 2
	for {
		statuscode, meta, title := gemgrabtitle(url)
		switch statuscode {
		case 31:
			url = meta
			iter++
			if iter >= maxNumber {
				return "Too many redirects"
			}
		default:
			return title
		}
	}
}
