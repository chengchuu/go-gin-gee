package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	parsepkg "github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
	"golang.org/x/net/html"
)

type documentKind int

const (
	kindUnknown documentKind = iota
	kindHTML
	kindCSS
	kindManifest
	kindSitemap
)

type reference struct {
	value string
	base  *url.URL
	hint  documentKind
}

func detectKind(hint documentKind, contentType, pathname string) documentKind {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch mediaType {
	case "text/html", "application/xhtml+xml":
		return kindHTML
	case "text/css":
		return kindCSS
	case "application/manifest+json":
		return kindManifest
	case "application/json":
		if hint == kindManifest {
			return kindManifest
		}
		return kindUnknown
	case "application/xml", "text/xml", "application/rss+xml", "application/atom+xml":
		if hint == kindSitemap || isSitemapPath(pathname) {
			return kindSitemap
		}
		return kindUnknown
	}
	if mediaType != "" {
		return kindUnknown
	}
	if hint != kindUnknown {
		return hint
	}
	switch strings.ToLower(path.Ext(pathname)) {
	case ".html", ".htm":
		return kindHTML
	case ".css":
		return kindCSS
	case ".webmanifest":
		return kindManifest
	case ".xml":
		if !isSitemapPath(pathname) {
			return kindUnknown
		}
		return kindSitemap
	}
	return kindUnknown
}

func isSitemapPath(pathname string) bool {
	name := strings.ToLower(path.Base(pathname))
	return path.Ext(name) == ".xml" && strings.Contains(name, "sitemap")
}

func classifyExpectedResource(hint documentKind, pathname string) string {
	pathKind := detectKind(kindUnknown, "", pathname)
	if resourceType := classifyResource(pathKind, "", pathname); resourceType != "UNKNOWN" {
		return resourceType
	}
	return classifyResource(hint, "", pathname)
}

func classifyResource(kind documentKind, contentType, pathname string) string {
	switch kind {
	case kindHTML:
		return "HTML"
	case kindCSS:
		return "CSS"
	case kindManifest:
		return "MANIFEST"
	case kindSitemap:
		return "SITEMAP"
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch {
	case mediaType == "application/javascript", mediaType == "text/javascript",
		mediaType == "application/ecmascript", mediaType == "text/ecmascript", mediaType == "application/x-javascript":
		return "JAVASCRIPT"
	case mediaType == "application/json", strings.HasSuffix(mediaType, "+json"):
		return "JSON"
	case strings.HasPrefix(mediaType, "image/"):
		return "IMAGE"
	case mediaType == "application/xml", mediaType == "text/xml", strings.HasSuffix(mediaType, "+xml"):
		return "XML"
	case strings.HasPrefix(mediaType, "font/") || mediaType == "application/font-woff" || mediaType == "application/vnd.ms-fontobject":
		return "FONT"
	case strings.HasPrefix(mediaType, "audio/"):
		return "AUDIO"
	case strings.HasPrefix(mediaType, "video/"):
		return "VIDEO"
	case mediaType == "application/pdf":
		return "PDF"
	case strings.HasPrefix(mediaType, "text/"):
		return "TEXT"
	}
	switch strings.ToLower(path.Ext(pathname)) {
	case ".js", ".mjs", ".cjs":
		return "JAVASCRIPT"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".svg", ".ico":
		return "IMAGE"
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return "FONT"
	case ".mp3", ".wav", ".ogg", ".m4a", ".aac", ".flac":
		return "AUDIO"
	case ".mp4", ".webm", ".mov", ".m4v":
		return "VIDEO"
	case ".json":
		return "JSON"
	case ".xml":
		return "XML"
	case ".pdf":
		return "PDF"
	}
	if contentType == "" {
		return "UNKNOWN"
	}
	return "OTHER"
}

func parseDocument(kind documentKind, body []byte, finalURL *url.URL, collectAnchors bool) ([]reference, map[string]struct{}, error) {
	switch kind {
	case kindHTML:
		return parseHTML(body, finalURL, collectAnchors)
	case kindCSS:
		refs, err := parseCSS(body, finalURL)
		return refs, nil, err
	case kindManifest:
		refs, err := parseManifest(body, finalURL)
		return refs, nil, err
	case kindSitemap:
		refs, err := parseSitemap(body, finalURL)
		return refs, nil, err
	default:
		return nil, nil, nil
	}
}

func parseHTML(body []byte, responseURL *url.URL, collectAnchors bool) ([]reference, map[string]struct{}, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	base := responseURL
	baseFound := false
	var baseErr error
	var findBase func(*html.Node)
	findBase = func(n *html.Node) {
		if baseFound {
			return
		}
		if n.Type == html.ElementNode && n.Data == "base" {
			if raw, ok := attr(n, "href"); ok {
				baseFound = true
				if parsed, parseErr := url.Parse(strings.TrimSpace(raw)); parseErr == nil {
					base = responseURL.ResolveReference(parsed)
				} else {
					baseErr = fmt.Errorf("invalid base URL: %w", parseErr)
				}
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			findBase(child)
		}
	}
	findBase(doc)
	if baseErr != nil {
		return nil, nil, baseErr
	}

	refs := make([]reference, 0)
	anchors := make(map[string]struct{})
	var discoveryErr error
	add := func(raw string, hint documentKind) {
		if strings.TrimSpace(raw) != "" {
			refs = append(refs, reference{value: raw, base: base, hint: hint})
		}
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if collectAnchors {
				if id, ok := attr(n, "id"); ok && id != "" {
					anchors[id] = struct{}{}
				}
				if n.Data == "a" {
					if name, ok := attr(n, "name"); ok && name != "" {
						anchors[name] = struct{}{}
					}
				}
			}
			if raw, ok := attr(n, "style"); ok {
				cssRefs, cssErr := parseCSS([]byte(raw), base)
				if cssErr != nil && discoveryErr == nil {
					discoveryErr = fmt.Errorf("inline style: %w", cssErr)
				} else if cssErr == nil {
					refs = append(refs, cssRefs...)
				}
			}
			switch n.Data {
			case "a", "area":
				addAttr(n, "href", kindHTML, add)
			case "link":
				hint := kindUnknown
				rel, _ := attr(n, "rel")
				for _, token := range strings.Fields(strings.ToLower(rel)) {
					if token == "stylesheet" {
						hint = kindCSS
					}
					if token == "manifest" {
						hint = kindManifest
					}
					if token == "sitemap" {
						hint = kindSitemap
					}
				}
				addAttr(n, "href", hint, add)
			case "script", "iframe", "embed", "track", "input":
				addAttr(n, "src", kindUnknown, add)
			case "img", "source":
				addAttr(n, "src", kindUnknown, add)
				addSrcset(n, add)
			case "video":
				addAttr(n, "src", kindUnknown, add)
				addAttr(n, "poster", kindUnknown, add)
			case "audio":
				addAttr(n, "src", kindUnknown, add)
			case "object":
				addAttr(n, "data", kindUnknown, add)
			case "meta":
				key, _ := attr(n, "property")
				if key == "" {
					key, _ = attr(n, "name")
				}
				if key == "" {
					key, _ = attr(n, "itemprop")
				}
				key = strings.ToLower(key)
				if isMetadataImageField(key) {
					addAttr(n, "content", kindUnknown, add)
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "style" && n.FirstChild != nil {
			var text bytes.Buffer
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.TextNode {
					text.WriteString(child.Data)
				}
			}
			cssRefs, cssErr := parseCSS(text.Bytes(), base)
			if cssErr != nil && discoveryErr == nil {
				discoveryErr = fmt.Errorf("style element: %w", cssErr)
			} else if cssErr == nil {
				refs = append(refs, cssRefs...)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return refs, anchors, discoveryErr
}

func isMetadataImageField(key string) bool {
	switch key {
	case "og:image", "og:image:url", "og:image:secure_url",
		"twitter:image", "twitter:image:src", "image", "thumbnail",
		"thumbnailurl", "contenturl", "msapplication-tileimage":
		return true
	default:
		return false
	}
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val, true
		}
	}
	return "", false
}

func addAttr(n *html.Node, key string, hint documentKind, add func(string, documentKind)) {
	if raw, ok := attr(n, key); ok {
		add(raw, hint)
	}
}

func addSrcset(n *html.Node, add func(string, documentKind)) {
	raw, ok := attr(n, "srcset")
	if !ok {
		return
	}
	for _, candidate := range parseSrcset(raw) {
		add(candidate, kindUnknown)
	}
}

// parseSrcset follows the candidate-separation parts of the HTML algorithm. In
// particular, it does not mistake the comma inside a data URL for a separator.
func parseSrcset(raw string) []string {
	var values []string
	for i := 0; i < len(raw); {
		for i < len(raw) && (raw[i] == ',' || isASCIISpace(raw[i])) {
			i++
		}
		start := i
		for i < len(raw) && !isASCIISpace(raw[i]) {
			i++
		}
		candidate := strings.TrimRight(raw[start:i], ",")
		if candidate != "" {
			values = append(values, candidate)
		}
		depth := 0
		for i < len(raw) {
			switch raw[i] {
			case '(':
				depth++
			case ')':
				if depth > 0 {
					depth--
				}
			case ',':
				if depth == 0 {
					i++
					goto nextCandidate
				}
			}
			i++
		}
	nextCandidate:
	}
	return values
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func parseCSS(body []byte, base *url.URL) ([]reference, error) {
	lexer := css.NewLexer(parsepkg.NewInput(bytes.NewReader(body)))
	refs := make([]reference, 0)
	wantImport := false
	for {
		tt, token := lexer.Next()
		if tt == css.ErrorToken {
			if errors.Is(lexer.Err(), io.EOF) {
				break
			}
			return nil, lexer.Err()
		}
		if tt == css.BadURLToken || tt == css.BadStringToken {
			return nil, fmt.Errorf("malformed CSS %s", tt.String())
		}
		if tt == css.AtKeywordToken {
			wantImport = strings.EqualFold(string(token), "@import")
			continue
		}
		if tt == css.URLToken {
			raw, err := cssURLValue(string(token))
			if err != nil {
				return nil, err
			}
			refs = append(refs, reference{value: raw, base: base, hint: func() documentKind {
				if wantImport {
					return kindCSS
				}
				return kindUnknown
			}()})
			wantImport = false
			continue
		}
		if wantImport && tt == css.StringToken {
			raw, err := cssStringValue(string(token))
			if err != nil {
				return nil, err
			}
			refs = append(refs, reference{value: raw, base: base, hint: kindCSS})
			wantImport = false
		} else if wantImport && tt != css.WhitespaceToken && tt != css.CommentToken {
			wantImport = false
		}
	}
	return refs, nil
}

func cssURLValue(token string) (string, error) {
	i := strings.IndexByte(token, '(')
	if i < 0 || !strings.HasSuffix(token, ")") {
		return "", errors.New("malformed CSS URL")
	}
	raw := strings.TrimSpace(strings.TrimSuffix(token[i+1:], ")"))
	if len(raw) > 0 && (raw[0] == '\'' || raw[0] == '"') {
		return cssStringValue(raw)
	}
	return decodeCSSValue(raw)
}

func cssStringValue(raw string) (string, error) {
	if len(raw) < 2 || (raw[0] != '\'' && raw[0] != '"') || raw[len(raw)-1] != raw[0] {
		return "", errors.New("malformed CSS string")
	}
	return decodeCSSValue(raw[1 : len(raw)-1])
}

func decodeCSSValue(raw string) (string, error) {
	var decoded strings.Builder
	decoded.Grow(len(raw))
	for i := 0; i < len(raw); {
		if raw[i] != '\\' {
			decoded.WriteByte(raw[i])
			i++
			continue
		}
		i++
		if i == len(raw) {
			return "", errors.New("malformed CSS escape")
		}
		switch raw[i] {
		case '\n', '\f':
			i++
			continue
		case '\r':
			i++
			if i < len(raw) && raw[i] == '\n' {
				i++
			}
			continue
		}
		if isHexDigit(raw[i]) {
			start := i
			for i < len(raw) && i-start < 6 && isHexDigit(raw[i]) {
				i++
			}
			value, err := strconv.ParseUint(raw[start:i], 16, 32)
			if err != nil {
				return "", fmt.Errorf("malformed CSS escape: %w", err)
			}
			r := rune(value)
			if r == 0 || r > utf8.MaxRune || (r >= 0xD800 && r <= 0xDFFF) {
				r = utf8.RuneError
			}
			decoded.WriteRune(r)
			if i < len(raw) && isASCIISpace(raw[i]) {
				if raw[i] == '\r' && i+1 < len(raw) && raw[i+1] == '\n' {
					i++
				}
				i++
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(raw[i:])
		decoded.WriteRune(r)
		i += size
	}
	return decoded.String(), nil
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

type manifestDocument struct {
	StartURL    string             `json:"start_url"`
	Icons       []manifestResource `json:"icons"`
	Screenshots []manifestResource `json:"screenshots"`
	Shortcuts   []manifestShortcut `json:"shortcuts"`
}
type manifestResource struct {
	Src string `json:"src"`
}
type manifestShortcut struct {
	URL   string             `json:"url"`
	Icons []manifestResource `json:"icons"`
}

func parseManifest(body []byte, base *url.URL) ([]reference, error) {
	var manifest manifestDocument
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&manifest); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("multiple JSON values")
	}
	refs := []reference{}
	add := func(raw string) {
		if raw != "" {
			refs = append(refs, reference{value: raw, base: base})
		}
	}
	add(manifest.StartURL)
	for _, resource := range manifest.Icons {
		add(resource.Src)
	}
	for _, resource := range manifest.Screenshots {
		add(resource.Src)
	}
	for _, shortcut := range manifest.Shortcuts {
		add(shortcut.URL)
		for _, icon := range shortcut.Icons {
			add(icon.Src)
		}
	}
	return refs, nil
}

func parseSitemap(body []byte, base *url.URL) ([]reference, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	refs := []reference{}
	root := ""
	for {
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if root == "" {
			root = strings.ToLower(start.Name.Local)
			if root != "urlset" && root != "sitemapindex" {
				return nil, fmt.Errorf("unexpected sitemap root element %q", start.Name.Local)
			}
		}
		if strings.ToLower(start.Name.Local) != "loc" {
			continue
		}
		var loc string
		if err := dec.DecodeElement(&loc, &start); err != nil {
			return nil, err
		}
		if strings.TrimSpace(loc) != "" {
			hint := kindUnknown
			if root == "sitemapindex" {
				hint = kindSitemap
			}
			refs = append(refs, reference{value: strings.TrimSpace(loc), base: base, hint: hint})
		}
	}
	if root == "" {
		return nil, errors.New("empty sitemap")
	}
	return refs, nil
}
