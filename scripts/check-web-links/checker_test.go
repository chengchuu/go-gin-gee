package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCrawlDiscoversSupportedDocumentsAndAttributes(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	requests := map[string]int{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.URL.RequestURI()]++
		mu.Unlock()
		switch r.URL.Path {
		case "/project/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!doctype html><html><head>
<base href="/project/base/">
<link rel="stylesheet" href="/project/css/main.css">
<link rel="manifest" href="/project/app.webmanifest">
<link rel="sitemap" href="/project/sitemap.xml">
<link rel="icon" href="/project/icon.ico">
<script src="/project/app.js"></script>
<meta property="og:image" content="/project/meta.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:type" content="image/png">
<style>.hero { background:url('/project/inline.png') }</style>
</head><body style="background:url(/project/attribute.png)">
<a href="page.html?x=1#target">page</a><a href="page.html?x=1#legacy">duplicate request</a>
<a href="`+server.URL+`/project-other/not-requested">sibling</a>
<a href="https://example.com/external">external</a><a href="mailto:test@example.com">mail</a>
<img src="/project/image.png" srcset="/project/image-2.png 2x, data:image/png;base64,AAAA 3x">
<picture><source src="/project/source.png" srcset="/project/source-2.png 2x"></picture>
<video src="/project/video.mp4" poster="/project/poster.jpg"></video>
<audio src="/project/audio.mp3"></audio><track src="/project/captions.vtt">
<iframe src="/project/frame.html"></iframe><embed src="/project/embed.bin">
<object data="/project/object.bin"></object><input type="image" src="/project/input.png">
</body></html>`)
		case "/project/base/page.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<h1 id="target">Target</h1><a name="legacy"></a><a href="/project/root.html">root</a>`)
		case "/project/frame.html", "/project/root.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<p id="ok">ok</p>`)
		case "/project/css/main.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, `@import "nested.css"; body{background:url('../css.png')}`)
		case "/project/css/nested.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, `.x{background:url(/project/nested.png)}`)
		case "/project/app.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			fmt.Fprint(w, `{"start_url":"/project/start.html","icons":[{"src":"/project/manifest-icon.png"}],"screenshots":[{"src":"/project/screenshot.png"}],"shortcuts":[{"url":"/project/shortcut.html","icons":[{"src":"/project/shortcut.png"}]}]}`)
		case "/project/start.html", "/project/shortcut.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<p>ok</p>`)
		case "/project/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, `<sitemapindex><sitemap><loc>%s/project/child-sitemap.xml</loc></sitemap></sitemapindex>`, server.URL)
		case "/project/child-sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, `<urlset><url><loc>%s/project/from-sitemap.html</loc></url></urlset>`, server.URL)
		case "/project/from-sitemap.html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<p>ok</p>`)
		case "/project-other/not-requested":
			t.Error("sibling project path must not be requested")
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{"-url=" + server.URL + "/project", "-concurrency=4", "-showSkipped", "-checkFragments"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr=%q, output:\n%s", code, stderr.String(), stdout.String())
	}
	for _, requestURI := range []string{
		"/project/", "/project/base/page.html?x=1", "/project/css/main.css", "/project/css/nested.css",
		"/project/css.png", "/project/nested.png", "/project/app.webmanifest", "/project/sitemap.xml",
		"/project/child-sitemap.xml", "/project/from-sitemap.html", "/project/frame.html", "/project/embed.bin",
		"/project/object.bin", "/project/meta.png", "/project/inline.png", "/project/attribute.png",
	} {
		mu.Lock()
		count := requests[requestURI]
		mu.Unlock()
		if count != 1 {
			t.Errorf("request count for %s = %d, want 1", requestURI, count)
		}
	}
	if !strings.Contains(stdout.String(), "https://example.com/external -- external origin") ||
		!strings.Contains(stdout.String(), "mailto:test@example.com -- non-fetchable scheme") ||
		!strings.Contains(stdout.String(), "/project-other/not-requested -- outside project path") {
		t.Errorf("skip report missing expected URLs:\n%s", stdout.String())
	}
	mu.Lock()
	if requests["/project/1200"] != 0 || requests["/project/image/png"] != 0 {
		t.Errorf("non-URL metadata companions were requested: %#v", requests)
	}
	mu.Unlock()
	assertSortedSection(t, stdout.String(), "Skipped URLs:", "Resource types:")
	wantTypes := "Resource types: AUDIO=1 CSS=2 HTML=7 IMAGE=15 JAVASCRIPT=1 MANIFEST=1 SITEMAP=2 UNKNOWN=3 VIDEO=1"
	if !strings.Contains(stdout.String(), wantTypes) {
		t.Errorf("resource type report missing %q:\n%s", wantTypes, stdout.String())
	}
}

func TestSkippedURLsHiddenByDefault(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<a href="https://example.com/external">external</a>`)
	}))
	defer server.Close()

	var hidden bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/"}, &hidden, &bytes.Buffer{}); code != 0 {
		t.Fatalf("default run code = %d", code)
	}
	if strings.Contains(hidden.String(), "https://example.com/external") {
		t.Errorf("default report exposed skipped URL:\n%s", hidden.String())
	}
	if strings.Contains(hidden.String(), "Skipped URLs:") || !strings.Contains(hidden.String(), "skipped=1") {
		t.Errorf("default report exposed skipped section or omitted count:\n%s", hidden.String())
	}
	if strings.Contains(hidden.String(), "Failed URLs:") {
		t.Errorf("successful report exposed empty Failed URLs section:\n%s", hidden.String())
	}

	var shown bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/", "-showSkipped"}, &shown, &bytes.Buffer{}); code != 0 {
		t.Fatalf("show-skipped run code = %d", code)
	}
	if !strings.Contains(shown.String(), "https://example.com/external -- external origin") {
		t.Errorf("-showSkipped report omitted skipped URL:\n%s", shown.String())
	}
}

func TestFragmentsCheckedOnlyWhenEnabled(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/" {
			fmt.Fprint(w, `<a href="page#missing">missing fragment</a>`)
			return
		}
		fmt.Fprint(w, `<p id="present">page</p>`)
	}))
	defer server.Close()

	var unchecked bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/"}, &unchecked, &bytes.Buffer{}); code != 0 {
		t.Fatalf("default run code = %d, output:\n%s", code, unchecked.String())
	}
	if strings.Contains(unchecked.String(), "missing HTML fragment") {
		t.Errorf("default run checked fragment target:\n%s", unchecked.String())
	}

	var checked bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/", "-checkFragments"}, &checked, &bytes.Buffer{}); code != 1 {
		t.Fatalf("fragment-check run code = %d, output:\n%s", code, checked.String())
	}
	if !strings.Contains(checked.String(), "missing HTML fragment") {
		t.Errorf("-checkFragments omitted missing target:\n%s", checked.String())
	}
}

func TestFragmentsIgnoredForNonHTMLResources(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<link rel="stylesheet" href="site.css#theme"><script src="app.js#module"></script><img src="image.png#crop">`)
		case "/site.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, `body { color: black }`)
		case "/app.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprint(w, `console.log("ok")`)
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			fmt.Fprint(w, "not-a-real-image")
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/", "-checkFragments"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("run code = %d, output:\n%s", code, stdout.String())
	}
	if strings.Contains(stdout.String(), "fragment target") || strings.Contains(stdout.String(), "missing HTML fragment") {
		t.Errorf("non-HTML fragments were validated:\n%s", stdout.String())
	}
}

func TestFailuresFragmentsRedirectsStatusesAndParsers(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/project/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<a href="good#present">ok</a><a href="good#missing">bad fragment</a><a href="missing">404</a><a href="error">500</a><a href="redirect-in">in</a><a href="redirect-out">out</a><link rel="manifest" href="bad.webmanifest"><link rel="stylesheet" href="bad.css"><link rel="sitemap" href="bad.xml">`)
		case "/project/good", "/project/final":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<p id="present">ok</p>`)
		case "/project/missing":
			http.NotFound(w, r)
		case "/project/error":
			http.Error(w, "error", http.StatusInternalServerError)
		case "/project/redirect-in":
			http.Redirect(w, r, "/project/final", http.StatusFound)
		case "/project/redirect-out":
			http.Redirect(w, r, "/outside", http.StatusFound)
		case "/project/bad.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			fmt.Fprint(w, `{bad`)
		case "/project/bad.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, `body{background:url(bad value)}`)
		case "/project/bad.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<urlset><url><loc>broken`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	code := run([]string{"-url=" + server.URL + "/project/", "-checkFragments"}, &stdout, &bytes.Buffer{})
	if code != 1 {
		t.Fatalf("run() code = %d, want 1\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "Failed URLs:") {
		t.Errorf("failure report omitted Failed URLs section:\n%s", stdout.String())
	}
	for _, want := range []string{"missing HTML fragment", "HTTP 404 Not Found", "HTTP 500 Internal Server Error", "redirect leaves configured scope", "parser error:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestTimeoutNetworkFailureAndURLLimit(t *testing.T) {
	t.Parallel()
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(40 * time.Millisecond)
			fmt.Fprint(w, "ok")
		}))
		defer server.Close()
		var stdout bytes.Buffer
		if code := run([]string{"-url=" + server.URL + "/", "-timeout=5ms"}, &stdout, &bytes.Buffer{}); code != 1 || !strings.Contains(stdout.String(), "network error:") {
			t.Fatalf("timeout code/output = %d/%q", code, stdout.String())
		}
	})
	t.Run("network", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		rawURL := server.URL
		server.Close()
		var stdout bytes.Buffer
		if code := run([]string{"-url=" + rawURL + "/"}, &stdout, &bytes.Buffer{}); code != 1 || !strings.Contains(stdout.String(), "network error:") {
			t.Fatalf("network code/output = %d/%q", code, stdout.String())
		}
	})
	t.Run("limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<a href="a">a</a><a href="b">b</a>`)
		}))
		defer server.Close()
		var stdout bytes.Buffer
		if code := run([]string{"-url=" + server.URL + "/", "-maxURLs=2"}, &stdout, &bytes.Buffer{}); code != 1 || !strings.Contains(stdout.String(), "coverage is incomplete") {
			t.Fatalf("limit code/output = %d/%q", code, stdout.String())
		}
	})
}

func TestArgumentsAndDeterministicReporting(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		nil, {"-url=ftp://example.com/"}, {"-url=https://example.com/", "-timeout=0"},
		{"-url=https://example.com/", "-concurrency=0"}, {"-url=https://example.com/", "-maxURLs=0"},
	} {
		if code := run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
			t.Errorf("run(%q) code = %d, want 2", args, code)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/project/" {
			fmt.Fprint(w, `<a href="z">z</a><a href="a">a</a>`)
		}
	}))
	defer server.Close()
	var first, second bytes.Buffer
	if run([]string{"-url=" + server.URL + "/project/", "-concurrency=1"}, &first, &bytes.Buffer{}) != 0 || run([]string{"-url=" + server.URL + "/project/", "-concurrency=1"}, &second, &bytes.Buffer{}) != 0 {
		t.Fatal("deterministic fixture unexpectedly failed")
	}
	if first.String() != second.String() {
		t.Errorf("reports differ:\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}
}

func TestCheckedURLsStreamWhileCrawlRuns(t *testing.T) {
	t.Parallel()
	requestStarted := make(chan struct{})
	releaseResponse := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseResponse
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<p>ok</p>`)
	}))
	defer server.Close()

	output := &synchronizedBuffer{}
	done := make(chan int, 1)
	go func() {
		done <- run([]string{"-url=" + server.URL + "/"}, output, &bytes.Buffer{})
	}()
	<-requestStarted
	if got := output.String(); got != "Checked URLs:\n" {
		t.Fatalf("output before request completion = %q", got)
	}
	close(releaseResponse)
	if code := <-done; code != 0 {
		t.Fatalf("run code = %d", code)
	}
	if !strings.Contains(output.String(), "  "+server.URL+"/\n") {
		t.Errorf("completed URL was not streamed:\n%s", output.String())
	}
}

func TestReportFileDefaultAndFilteredContent(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<link rel="stylesheet" href="z.css"><link rel="stylesheet" href="a.css"><script src="app.js"></script><script src="failed.js"></script><a href="https://example.com/external">external</a>`)
		case "/a.css", "/z.css":
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprint(w, `body { color: black }`)
		case "/app.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprint(w, `console.log("ok")`)
		case "/failed.js":
			w.Header().Set("Content-Type", "text/javascript")
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	unusedPath := filepath.Join(dir, "unused.txt")
	if code := run([]string{"-url=" + server.URL + "/"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("run without report code = %d", code)
	}
	if _, err := os.Stat(unusedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("report created without -reportPath: %v", err)
	}

	allPath := filepath.Join(dir, "all.txt")
	if code := run([]string{"-url=" + server.URL + "/", "-reportPath=" + allPath, "-showSkipped"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("all-types report code = %d", code)
	}
	all := readTestFile(t, allPath)
	allAgainPath := filepath.Join(dir, "all-again.txt")
	if code := run([]string{"-url=" + server.URL + "/", "-reportPath=" + allAgainPath, "-showSkipped"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("second all-types report code = %d", code)
	}
	if allAgain := readTestFile(t, allAgainPath); allAgain != all {
		t.Errorf("reports from identical concurrent crawls differ:\nfirst:\n%s\nsecond:\n%s", all, allAgain)
	}
	for _, want := range []string{
		"Project root: " + server.URL + "/",
		"Report types: ALL",
		"  CSS (2):\n    " + server.URL + "/a.css\n    " + server.URL + "/z.css",
		"  HTML (1):\n    " + server.URL + "/",
		"  JAVASCRIPT (2):\n    " + server.URL + "/app.js\n    " + server.URL + "/failed.js",
		"Failed URLs:\n  " + server.URL + "/failed.js -- HTTP 404 Not Found",
		"Skipped URLs:\n  https://example.com/external -- external origin",
		"Resource types: CSS=2 HTML=1 JAVASCRIPT=2",
		"Summary: checked=5 passed=4 failed=1 skipped=1 discovered=5",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("all-types report missing %q:\n%s", want, all)
		}
	}
	assertOrderedText(t, all, "  CSS (2):", "  HTML (1):", "  JAVASCRIPT (2):", "Failed URLs:", "Skipped URLs:", "Resource types:", "Summary:")

	filteredPath := filepath.Join(dir, "filtered.txt")
	var console bytes.Buffer
	args := []string{"-url=" + server.URL + "/", "-reportPath=" + filteredPath, "-reportTypes= html, CSS, css "}
	if code := run(args, &console, &bytes.Buffer{}); code != 1 {
		t.Fatalf("filtered report code = %d", code)
	}
	filtered := readTestFile(t, filteredPath)
	if !strings.Contains(filtered, "Report types: CSS,HTML") || !strings.Contains(filtered, "  CSS (2):") || !strings.Contains(filtered, "  HTML (1):") {
		t.Errorf("filtered report omitted selected groups:\n%s", filtered)
	}
	if strings.Contains(filtered, "  JAVASCRIPT (2):") {
		t.Errorf("filtered report included excluded checked group:\n%s", filtered)
	}
	for _, want := range []string{
		server.URL + "/failed.js -- HTTP 404 Not Found",
		"Resource types: CSS=2 HTML=1 JAVASCRIPT=2",
		"Summary: checked=5 passed=4 failed=1 skipped=1 discovered=5",
	} {
		if !strings.Contains(filtered, want) {
			t.Errorf("filter hid required report content %q:\n%s", want, filtered)
		}
	}
	if strings.Contains(filtered, "Skipped URLs:") {
		t.Errorf("filtered report showed skipped section without -showSkipped:\n%s", filtered)
	}
	for _, raw := range []string{"/app.js", "/failed.js"} {
		if !strings.Contains(console.String(), server.URL+raw) {
			t.Errorf("type filter changed console output for %s:\n%s", raw, console.String())
		}
	}
}

func TestURLNormalizationAndEscapedPathScope(t *testing.T) {
	t.Parallel()
	defaultPort, err := url.Parse("HTTPS://EXAMPLE.COM:443")
	if err != nil {
		t.Fatal(err)
	}
	if got := networkKey(defaultPort); got != "https://example.com/" {
		t.Fatalf("normalized default-port URL = %q", got)
	}
	uppercaseOpts, err := parseOptions([]string{"-url=HTTPS://EXAMPLE.COM/project/"}, io.Discard)
	if err != nil {
		t.Fatalf("case-insensitive HTTP scheme was rejected: %v", err)
	}
	if got := uppercaseOpts.root.String(); got != "https://example.com/project/" {
		t.Fatalf("normalized uppercase root = %q", got)
	}

	opts, err := parseOptions([]string{"-url=https://example.com/project/"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	c := newCrawler(opts, io.Discard)
	inside, _ := url.Parse("https://example.com/project/page.html")
	escapedSibling, _ := url.Parse("https://example.com/project%2Fother")
	escapedTraversal, _ := url.Parse("https://example.com/project/%2e%2e/outside")
	if !c.inScope(inside) {
		t.Error("ordinary project child was out of scope")
	}
	if c.inScope(escapedSibling) {
		t.Error("encoded slash bypassed project path-segment scope")
	}
	if c.inScope(escapedTraversal) {
		t.Error("encoded dot segment bypassed project path scope")
	}
}

func TestSameOriginURLWithoutPathDeduplicatesRoot(t *testing.T) {
	t.Parallel()
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<a href="%s">root without slash</a>`, server.URL)
	}))
	defer server.Close()

	var stdout bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("run code = %d, output:\n%s", code, stdout.String())
	}
	if requests != 1 {
		t.Fatalf("root request count = %d, want 1", requests)
	}
	if !strings.Contains(stdout.String(), "skipped=0 discovered=1") {
		t.Errorf("root URL was skipped or rediscovered:\n%s", stdout.String())
	}
}

func TestGenericXMLIsNotParsedAsSitemap(t *testing.T) {
	t.Parallel()
	missingRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<a href="data.xml">data</a>`)
		case "/data.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<config><loc>/missing</loc></config>`)
		case "/missing":
			missingRequested = true
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	if code := run([]string{"-url=" + server.URL + "/"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("run code = %d, output:\n%s", code, stdout.String())
	}
	if missingRequested {
		t.Error("generic XML <loc> was crawled as a sitemap")
	}
	if !strings.Contains(stdout.String(), "Resource types: HTML=1 XML=1") {
		t.Errorf("generic XML was misclassified:\n%s", stdout.String())
	}
}

func TestCSSReferencesDecodeEscapes(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://example.com/assets/site.css")
	refs, err := parseCSS([]byte(`@import "th\65 me.css"; body { background: url(im\61 ge.png) }`), base)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0].value != "theme.css" || refs[1].value != "image.png" {
		t.Fatalf("decoded CSS references = %#v", refs)
	}
}

func TestCSSRejectsUnterminatedReferences(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://example.com/assets/site.css")
	for _, cssText := range []string{`@import "unterminated`, `body { background: url(unclosed }`} {
		if _, err := parseCSS([]byte(cssText), base); err == nil {
			t.Errorf("parseCSS(%q) accepted an unterminated reference", cssText)
		}
	}
}

func TestManifestIgnoresNonFetchableURLMetadata(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://example.com/app.webmanifest")
	body := []byte(`{
  "id":"/app-id","scope":"/scope/","start_url":"/start",
  "icons":[{"src":"/icon.png"}],"screenshots":[{"src":"/shot.png"}],
  "shortcuts":[{"url":"/shortcut","icons":[{"src":"/shortcut.png"}]}],
  "share_target":{"action":"/share"},
  "protocol_handlers":[{"url":"/handle?url=%s"}],
  "file_handlers":[{"action":"/open","icons":[{"src":"/file.png"}]}],
  "related_applications":[{"url":"/store"}],
  "scope_extensions":[{"origin":"https://other.example"}]
}`)
	refs, err := parseManifest(body, base)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, ref.value)
	}
	want := []string{"/start", "/icon.png", "/shot.png", "/shortcut", "/shortcut.png"}
	if strings.Join(values, "|") != strings.Join(want, "|") {
		t.Fatalf("manifest references = %v, want %v", values, want)
	}
}

func TestResourceClassificationUsesIntendedFailureType(t *testing.T) {
	t.Parallel()
	if got := classifyExpectedResource(kindUnknown, "/assets/app.js"); got != "JAVASCRIPT" {
		t.Errorf("failed JavaScript classification = %q", got)
	}
	if got := classifyExpectedResource(kindHTML, "/documents/guide.pdf"); got != "PDF" {
		t.Errorf("failed linked PDF classification = %q", got)
	}
	if got := classifyResource(kindUnknown, "image/svg+xml", "/icon.svg"); got != "IMAGE" {
		t.Errorf("SVG classification = %q", got)
	}
}

func TestReportArgumentsAndWriteFailures(t *testing.T) {
	t.Parallel()
	t.Run("types require path", func(t *testing.T) {
		var stderr bytes.Buffer
		code := run([]string{"-url=https://example.com/", "-reportTypes=HTML"}, &bytes.Buffer{}, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "-reportTypes requires -reportPath") {
			t.Fatalf("code/stderr = %d/%q", code, stderr.String())
		}
	})
	t.Run("unknown type", func(t *testing.T) {
		var stderr bytes.Buffer
		code := run([]string{"-url=https://example.com/", "-reportPath=report.txt", "-reportTypes=HTML,nope"}, &bytes.Buffer{}, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "unknown report type(s): NOPE") || !strings.Contains(stderr.String(), "supported types: AUDIO, CSS") {
			t.Fatalf("code/stderr = %d/%q", code, stderr.String())
		}
	})
	t.Run("missing parent", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<p>ok</p>`)
		}))
		defer server.Close()
		path := filepath.Join(t.TempDir(), "missing", "report.txt")
		var stderr bytes.Buffer
		code := run([]string{"-url=" + server.URL + "/", "-reportPath=" + path}, &bytes.Buffer{}, &stderr)
		if code != 1 || !strings.Contains(stderr.String(), "error: write report:") {
			t.Fatalf("code/stderr = %d/%q", code, stderr.String())
		}
	})
}

func TestAtomicReportReplacementAndFailureCleanup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte("old report"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeReportAtomic(path, []byte("new report")); err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, path); got != "new report" {
		t.Fatalf("replacement content = %q", got)
	}
	assertNoReportTemps(t, dir)

	if err := os.WriteFile(path, []byte("preserve me"), 0o644); err != nil {
		t.Fatal(err)
	}
	renameErr := errors.New("forced rename failure")
	err := writeReportAtomicWithRename(path, []byte("discard me"), func(string, string) error { return renameErr })
	if !errors.Is(err, renameErr) {
		t.Fatalf("rename error = %v, want %v", err, renameErr)
	}
	if got := readTestFile(t, path); got != "preserve me" {
		t.Fatalf("failed replacement changed existing report: %q", got)
	}
	assertNoReportTemps(t, dir)
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertNoReportTemps(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".report.txt.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary report files remain: %v", matches)
	}
}

func assertOrderedText(t *testing.T, text string, values ...string) {
	t.Helper()
	position := -1
	for _, value := range values {
		next := strings.Index(text, value)
		if next < 0 || next <= position {
			t.Fatalf("%q is missing or out of order in:\n%s", value, text)
		}
		position = next
	}
}

type synchronizedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func assertSortedSection(t *testing.T, output, start, end string) {
	t.Helper()
	left := strings.Index(output, start)
	right := strings.Index(output, end)
	if left < 0 || right < left {
		t.Fatalf("section %q..%q not found", start, end)
	}
	lines := strings.Split(strings.TrimSpace(output[left+len(start):right]), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	actual := append([]string(nil), lines...)
	sort.Strings(lines)
	if strings.Join(actual, "\n") != strings.Join(lines, "\n") {
		t.Errorf("section %q is not sorted: %q", start, actual)
	}
}
