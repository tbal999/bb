package urlparse

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	gemini "github.com/makeworld-the-better-one/go-gemini"
	"golang.org/x/net/html"
)

const (
	fetchTimeout = 5 * time.Second
	maxBodyBytes = 1 << 20 // only the <title> is needed - never read more than 1MB
)

var httpClient = &http.Client{Timeout: fetchTimeout}

var geminiClient = &gemini.Client{ConnectTimeout: fetchTimeout, ReadTimeout: fetchTimeout}

func HTTPget(url string) string {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "Nothing, because there was a response error!"
	}
	// do this now so it won't be forgotten
	defer resp.Body.Close()
	// reads html as a slice of bytes
	html, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
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

// gemgrabtitle fetches a gemini page and returns its status, meta and first heading.
// An unreachable server returns status 0 and an empty title.
func gemgrabtitle(url string) (int, string, string) {
	response, err := geminiClient.Fetch(url)
	if err != nil {
		return 0, "", ""
	}
	defer response.Body.Close()
	var title string
	data, _ := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") {
			title = line
			break
		}
	}
	return response.Status, response.Meta, title
}

func GEMGet(rawURL string) string {
	const maxRedirects = 2
	for iter := 0; ; iter++ {
		statuscode, meta, title := gemgrabtitle(rawURL)
		if statuscode != 30 && statuscode != 31 { // 30 = temporary, 31 = permanent redirect
			return title
		}
		if iter >= maxRedirects {
			return "Too many redirects"
		}
		rawURL = resolve(rawURL, meta)
	}
}

// resolve handles redirects to relative URLs.
func resolve(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}
