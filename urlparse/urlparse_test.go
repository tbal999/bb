package urlparse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGEMGetUnreachableDoesNotPanic(t *testing.T) {
	start := time.Now()
	if got := GEMGet("gemini://127.0.0.1:1/"); got != "" { // nothing listens on port 1
		t.Fatalf("want empty title, got %q", got)
	}
	if time.Since(start) > 2*fetchTimeout {
		t.Fatal("unreachable host was not abandoned in time")
	}
}

func TestHTTPgetTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Hello bb</title></head></html>")
	}))
	defer srv.Close()
	if got := HTTPget(srv.URL); got != "Hello bb" {
		t.Fatalf("want %q, got %q", "Hello bb", got)
	}
}

func TestResolveRelativeRedirect(t *testing.T) {
	if got := resolve("gemini://a.org/dir/page", "../other"); got != "gemini://a.org/other" {
		t.Fatalf("got %q", got)
	}
	if got := resolve("gemini://a.org/x", "gemini://b.org/y"); got != "gemini://b.org/y" {
		t.Fatalf("got %q", got)
	}
}
