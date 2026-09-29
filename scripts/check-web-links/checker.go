package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout     = 15 * time.Second
	defaultConcurrency = 8
	defaultMaxURLs     = 10000
)

type options struct {
	root           *url.URL
	timeout        time.Duration
	concurrency    int
	maxURLs        int
	showSkipped    bool
	checkFragments bool
	reportPath     string
	reportTypes    []string
}

type item struct {
	url      *url.URL
	referrer string
	hint     documentKind
}

type failure struct {
	url      string
	reason   string
	referrer string
}

type skip struct {
	url      string
	reason   string
	referrer string
}

type fragmentRef struct {
	fragment string
	referrer string
}

type crawler struct {
	opts    options
	client  *http.Client
	jobs    chan item
	work    sync.WaitGroup
	stdout  io.Writer
	printMu sync.Mutex

	mu              sync.Mutex
	discovered      map[string]item
	checked         map[string]bool
	types           map[string]string
	failures        map[string]failure
	skipped         map[string]skip
	anchors         map[string]map[string]struct{}
	fragments       map[string]map[string]fragmentRef
	incomplete      bool
	rootOrigin      string
	rootPath        string
	rootEscapedPath string
}

var errRedirectScope = errors.New("redirect leaves configured scope")

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	c := newCrawler(opts, stdout)
	c.crawl()
	snapshot := c.snapshot()
	reportConsole(stdout, snapshot, opts.showSkipped)
	if opts.reportPath != "" {
		content := renderFileReport(snapshot, opts)
		if err := writeReportAtomic(opts.reportPath, content); err != nil {
			fmt.Fprintf(stderr, "error: write report: %v\n", err)
			return 1
		}
	}
	if snapshot.hasFailures() {
		return 1
	}
	return 0
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	fs := flag.NewFlagSet("check-web-links", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rawURL := fs.String("url", "", "absolute HTTP or HTTPS project-root URL (required)")
	timeout := fs.Duration("timeout", defaultTimeout, "timeout for each request")
	concurrency := fs.Int("concurrency", defaultConcurrency, "maximum simultaneous requests")
	maxURLs := fs.Int("maxURLs", defaultMaxURLs, "maximum in-scope URLs")
	showSkipped := fs.Bool("showSkipped", false, "show individual skipped URLs")
	checkFragments := fs.Bool("checkFragments", false, "validate HTML fragment targets")
	reportPath := fs.String("reportPath", "", "optional deterministic report file path")
	rawReportTypes := fs.String("reportTypes", "", "comma-separated resource types to show in the file report")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	reportTypesSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "reportTypes" {
			reportTypesSet = true
		}
	})
	if fs.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *rawURL == "" {
		return options{}, errors.New("-url is required")
	}
	u, err := url.Parse(*rawURL)
	if err != nil || !u.IsAbs() || (!strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https")) || u.Host == "" {
		return options{}, errors.New("-url must be an absolute HTTP or HTTPS URL")
	}
	if u.User != nil {
		return options{}, errors.New("-url must not contain user information")
	}
	if *timeout <= 0 {
		return options{}, errors.New("-timeout must be greater than zero")
	}
	if *concurrency <= 0 {
		return options{}, errors.New("-concurrency must be greater than zero")
	}
	if *maxURLs <= 0 {
		return options{}, errors.New("-maxURLs must be greater than zero")
	}
	if reportTypesSet && *reportPath == "" {
		return options{}, errors.New("-reportTypes requires -reportPath")
	}
	reportTypes, err := parseReportTypes(*rawReportTypes)
	if err != nil {
		return options{}, err
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Fragment = ""
	u.Path = cleanRootPath(u.Path)
	u.RawPath = ""
	normalizeURL(u)
	return options{
		root: u, timeout: *timeout, concurrency: *concurrency, maxURLs: *maxURLs,
		showSkipped: *showSkipped, checkFragments: *checkFragments,
		reportPath: *reportPath, reportTypes: reportTypes,
	}, nil
}

func cleanRootPath(p string) string {
	p = path.Clean("/" + p)
	if p != "/" {
		p += "/"
	}
	return p
}

func newCrawler(opts options, stdout io.Writer) *crawler {
	c := &crawler{
		opts:            opts,
		stdout:          stdout,
		jobs:            make(chan item, opts.maxURLs),
		discovered:      make(map[string]item),
		checked:         make(map[string]bool),
		types:           make(map[string]string),
		failures:        make(map[string]failure),
		skipped:         make(map[string]skip),
		anchors:         make(map[string]map[string]struct{}),
		fragments:       make(map[string]map[string]fragmentRef),
		rootOrigin:      origin(opts.root),
		rootPath:        opts.root.Path,
		rootEscapedPath: opts.root.EscapedPath(),
	}
	c.client = &http.Client{
		Timeout: opts.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !c.inScope(req.URL) {
				return fmt.Errorf("%w: %s", errRedirectScope, req.URL.String())
			}
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}
	return c
}

func (c *crawler) crawl() {
	c.printMu.Lock()
	fmt.Fprintln(c.stdout, "Checked URLs:")
	c.printMu.Unlock()
	for i := 0; i < c.opts.concurrency; i++ {
		go c.worker()
	}
	c.addResolved(c.opts.root, "(root)", kindUnknown)
	c.work.Wait()
	close(c.jobs)
	if c.opts.checkFragments {
		c.validateFragments()
	}
}

func (c *crawler) worker() {
	for it := range c.jobs {
		c.fetch(it)
		c.work.Done()
	}
}

func (c *crawler) fetch(it item) {
	key := networkKey(it.url)
	req, err := http.NewRequest(http.MethodGet, key, nil)
	if err != nil {
		c.addFailure(key, err.Error(), it.referrer)
		return
	}
	resp, err := c.client.Do(req)
	c.recordChecked(key)
	if err != nil {
		c.recordType(key, classifyExpectedResource(it.hint, it.url.Path))
		if resp != nil {
			resp.Body.Close()
		}
		reason := "network error: " + err.Error()
		if errors.Is(err, errRedirectScope) {
			reason = err.Error()
		}
		c.addFailure(key, reason, it.referrer)
		return
	}
	defer resp.Body.Close()
	finalURL := resp.Request.URL
	kind := detectKind(it.hint, resp.Header.Get("Content-Type"), finalURL.Path)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.recordType(key, classifyExpectedResource(it.hint, finalURL.Path))
		c.addFailure(key, fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)), it.referrer)
		return
	}
	c.recordType(key, classifyResource(kind, resp.Header.Get("Content-Type"), finalURL.Path))
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.addFailure(key, "network error: "+err.Error(), it.referrer)
		return
	}
	refs, anchors, err := parseDocument(kind, body, finalURL, c.opts.checkFragments)
	if err != nil {
		c.addFailure(key, "parser error: "+err.Error(), it.referrer)
		return
	}
	if kind == kindHTML && c.opts.checkFragments {
		c.mu.Lock()
		c.anchors[key] = anchors
		c.anchors[networkKey(finalURL)] = anchors
		c.mu.Unlock()
	}
	for _, ref := range refs {
		c.discover(ref.value, ref.base, key, ref.hint)
	}
}

func (c *crawler) recordChecked(raw string) {
	c.mu.Lock()
	c.checked[raw] = true
	c.mu.Unlock()

	c.printMu.Lock()
	fmt.Fprintf(c.stdout, "  %s\n", raw)
	c.printMu.Unlock()
}

func (c *crawler) discover(raw string, base *url.URL, referrer string, hint documentKind) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "#" {
		return
	}
	u, err := url.Parse(raw)
	if err != nil {
		c.addFailure(raw, "invalid URL: "+err.Error(), referrer)
		return
	}
	if u.Scheme != "" && !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		c.addSkip(raw, "non-fetchable scheme", referrer)
		return
	}
	u = base.ResolveReference(u)
	normalizeURL(u)
	if !c.inScope(u) {
		reason := "external origin"
		if origin(u) == c.rootOrigin {
			reason = "outside project path"
		}
		c.addSkip(u.String(), reason, referrer)
		return
	}
	fragment := u.Fragment
	u.Fragment = ""
	key := networkKey(u)
	if fragment != "" && c.opts.checkFragments {
		c.mu.Lock()
		if c.fragments[key] == nil {
			c.fragments[key] = make(map[string]fragmentRef)
		}
		if _, ok := c.fragments[key][fragment]; !ok {
			c.fragments[key][fragment] = fragmentRef{fragment: fragment, referrer: referrer}
		}
		c.mu.Unlock()
	}
	c.addResolved(u, referrer, hint)
}

func (c *crawler) addResolved(u *url.URL, referrer string, hint documentKind) {
	key := networkKey(u)
	c.mu.Lock()
	if _, exists := c.discovered[key]; exists {
		c.mu.Unlock()
		return
	}
	if len(c.discovered) >= c.opts.maxURLs {
		c.incomplete = true
		if _, exists := c.failures["coverage"]; !exists {
			c.failures["coverage"] = failure{url: key, reason: fmt.Sprintf("maximum URL limit reached (%d); coverage is incomplete", c.opts.maxURLs), referrer: referrer}
		}
		c.mu.Unlock()
		return
	}
	it := item{url: cloneURL(u), referrer: referrer, hint: hint}
	c.discovered[key] = it
	c.work.Add(1)
	c.mu.Unlock()
	c.jobs <- it
}

func (c *crawler) inScope(u *url.URL) bool {
	if u == nil || origin(u) != c.rootOrigin {
		return false
	}
	return pathWithinRoot(u.Path, c.rootPath) && pathWithinRoot(u.EscapedPath(), c.rootEscapedPath)
}

func pathWithinRoot(candidate, root string) bool {
	if root == "/" {
		return strings.HasPrefix(candidate, "/")
	}
	candidate = path.Clean("/" + candidate)
	return candidate == strings.TrimSuffix(root, "/") || strings.HasPrefix(candidate, root)
}

func (c *crawler) validateFragments() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for target, refs := range c.fragments {
		if c.types[target] != "HTML" {
			continue
		}
		anchors, parsed := c.anchors[target]
		for fragment, ref := range refs {
			if !parsed {
				c.failures[target+"#"+url.PathEscape(fragment)] = failure{url: target + "#" + fragment, reason: "fragment target is not a parsed HTML document", referrer: ref.referrer}
				continue
			}
			if _, ok := anchors[fragment]; !ok {
				c.failures[target+"#"+url.PathEscape(fragment)] = failure{url: target + "#" + fragment, reason: "missing HTML fragment", referrer: ref.referrer}
			}
		}
	}
}

func (c *crawler) addFailure(raw, reason, referrer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.failures[raw]; !ok {
		c.failures[raw] = failure{url: raw, reason: reason, referrer: referrer}
	}
}

func (c *crawler) addSkip(raw, reason, referrer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.skipped[raw]; !ok {
		c.skipped[raw] = skip{url: raw, reason: reason, referrer: referrer}
	}
}

func (c *crawler) recordType(raw, resourceType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.types[raw] = resourceType
}

func (c *crawler) snapshot() crawlSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	snapshot := crawlSnapshot{
		root:       c.opts.root.String(),
		checked:    sortedKeys(c.checked),
		types:      make(map[string]string, len(c.types)),
		discovered: len(c.discovered),
		incomplete: c.incomplete,
	}
	for raw, resourceType := range c.types {
		snapshot.types[raw] = resourceType
	}
	failures := make([]failure, 0, len(c.failures))
	for _, f := range c.failures {
		failures = append(failures, f)
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].url < failures[j].url })
	skipped := make([]skip, 0, len(c.skipped))
	for _, s := range c.skipped {
		skipped = append(skipped, s)
	}
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].url < skipped[j].url })
	snapshot.failures = failures
	snapshot.skipped = skipped
	return snapshot
}

func origin(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	hostname := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return scheme + "://" + host
}
func normalizeURL(u *url.URL) {
	u.Scheme = strings.ToLower(u.Scheme)
	hostname := strings.ToLower(u.Hostname())
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		u.Host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		u.Host = "[" + hostname + "]"
	} else {
		u.Host = hostname
	}
	if u.Path == "" {
		u.Path = "/"
		u.RawPath = ""
	}
}

func networkKey(u *url.URL) string {
	v := cloneURL(u)
	v.Fragment = ""
	normalizeURL(v)
	return v.String()
}
func cloneURL(u *url.URL) *url.URL { v := *u; return &v }
func stripFragment(raw string) string {
	if i := strings.IndexByte(raw, '#'); i >= 0 {
		return raw[:i]
	}
	return raw
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
