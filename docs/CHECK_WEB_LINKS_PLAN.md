# Scoped deployed-site link checker plan

## Summary

Create a standalone Go command under `scripts/check-web-links/` that starts from one deployed project root, recursively discovers referenced pages and assets, and validates only URLs within the same origin and path boundary.

Example:

```bash
go run ./scripts/check-web-links -url="https://chengchuu.github.io/mazey/"
```

Keep the existing `scripts/crawl-web-naruto/` command unchanged.

## Command interface

- Require `-url` with an absolute HTTP or HTTPS project-root URL.
- Support `-timeout=15s` as the timeout for each request.
- Support `-concurrency=8` as the maximum number of simultaneous requests.
- Support `-maxURLs=10000` as the maximum number of in-scope URLs.
- Normalize the project-root path with a trailing slash.
- Match paths by segment. For example, `/mazey/` includes `/mazey/api/` but excludes `/mazey-other/`.
- Exit with status `0` when every discovered in-scope URL and fragment passes.
- Exit with status `1` when the check finds a broken URL or fragment, or when the check is incomplete.
- Exit with status `2` when command arguments are invalid.

## Discovery and validation

- Send GET requests and accept HTTP `2xx` responses.
- Follow redirects only while every hop remains within the configured origin and path.
- Report redirects that leave the configured scope as failures.
- Parse HTML references for:
  - Page links.
  - Stylesheets and scripts.
  - Images and `srcset` candidates.
  - Media sources and posters.
  - Icons, manifests, and sitemaps.
  - Iframes and embedded resources.
  - Metadata images.
  - Inline `<style>` elements and `style` attributes.
- Respect the HTML `<base href>` element when resolving relative references.
- Parse CSS `url(...)` and `@import` references with `github.com/tdewolff/parse/v2`. Add this module as a direct dependency.
- Parse standardized URL-bearing web-manifest fields, including icons, screenshots, shortcuts, and `start_url`.
- Parse sitemap indexes and URL sets through their `<loc>` elements.
- Fetch JavaScript files, but do not scan JavaScript source for imports, fetch calls, or dynamically generated URLs.
- Remove fragments when deduplicating network requests.
- Validate same-project HTML fragments against target `id` attributes and legacy named anchors.
- Preserve query strings when checking URLs.
- Skip external HTTP or HTTPS URLs without requesting them.
- Skip non-fetchable schemes such as `mailto:`, `tel:`, `javascript:`, `data:`, and `blob:`.
- Deduplicate normalized URLs and retain the first referring URL for diagnostics.
- Stop adding URLs when the check reaches `-maxURLs`, report incomplete coverage, and exit with status `1`.
- Do not retry failed requests in the first version.

## Reporting and documentation

- Print deterministic, sorted sections for checked, failed, and skipped URLs.
- Include the HTTP status or network or parser error for each failure.
- Include the first referring URL for each failure.
- End with counts for checked, passed, failed, skipped, and discovered URLs.
- Add `scripts/check-web-links/README.md` with:
  - Commands for the Mazey and gurl deployment examples.
  - Scope and path-matching behavior.
  - Flag defaults and exit statuses.
  - Static-discovery limitations.

## Implementation structure

- Keep `main.go` thin and call a testable `run` function that accepts arguments, standard output, and standard error, then returns an exit status.
- Use `net/http` for requests and redirect control.
- Use structured HTML, JSON, XML, and CSS parsers instead of regular expressions for document discovery.
- Use an in-memory work queue and a concurrency-safe set for discovered and checked URLs.
- Base relative references on the final in-scope response URL after redirects.
- Collect results during the crawl and sort them before rendering the console report.

## Test plan

Use `httptest.Server` fixtures to cover:

- HTML navigation and every supported asset attribute.
- Relative, root-relative, and `<base href>` URL resolution.
- Nested CSS references and `@import` rules.
- Web manifests, sitemap indexes, and URL sets.
- Duplicate URLs, query strings, and fragment deduplication.
- Same-path crawling and exclusion of external origins and sibling project paths.
- Valid and missing HTML fragments.
- HTTP `2xx`, `4xx`, and `5xx` responses.
- Request timeouts and network failures.
- In-scope redirects and scope-escaping redirects.
- Malformed manifests, sitemaps, and CSS.
- Maximum-URL failure behavior.
- Deterministic reporting and command exit statuses.

Run these validation commands:

```bash
go test ./scripts/check-web-links
go test ./...
go vet ./scripts/check-web-links
git diff --check
```

When network access is available, smoke-test the completed command against both example deployments:

```bash
go run ./scripts/check-web-links -url="https://chengchuu.github.io/mazey/"
go run ./scripts/check-web-links -url="https://chengchuu.github.io/gurl/"
```

## Assumptions

- One command invocation checks one deployed project.
- The checker validates statically discoverable resources.
- Runtime requests created by JavaScript are outside the first version.
- External links are reported as skipped and are never requested.
- The first version produces console output only.
