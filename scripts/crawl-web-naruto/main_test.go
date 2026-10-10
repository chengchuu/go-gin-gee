package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
)

func TestCrawlLocalDiscoveryAndDeduplication(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<a href="http://%s/child">first</a><a href="http://%s/child">duplicate</a><a href="http://%s/missing">missing</a><a href="https://example.invalid/outside">external</a>`, r.Host, r.Host, r.Host)
		case "/child":
			fmt.Fprint(w, "<p>child</p>")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	previousArgs, previousFlags := os.Args, flag.CommandLine
	os.Args = []string{"crawl-test", "-allowedDomain=" + parsed.Hostname(), "-firstURL=" + server.URL + "/"}
	flag.CommandLine = flag.NewFlagSet("crawl-test", flag.ContinueOnError)
	t.Cleanup(func() { os.Args, flag.CommandLine = previousArgs, previousFlags })
	main()
	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{"/", "/child", "/missing"} {
		if hits[path] != 1 {
			t.Errorf("requests to %q = %d, want 1", path, hits[path])
		}
	}
}
