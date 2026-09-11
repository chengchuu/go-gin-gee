# Scoped deployed-site link checker

`check-web-links` recursively checks statically referenced pages and assets in
one deployed project. It does not change or replace `crawl-web-naruto`.

## Usage

Check the Mazey deployment:

```bash
go run ./scripts/check-web-links -url="https://chengchuu.github.io/mazey/"
```

Check the gurl deployment:

```bash
go run ./scripts/check-web-links -url="https://chengchuu.github.io/gurl/"
```

Write a deterministic report containing only HTML and CSS checked-URL details:

```bash
go run ./scripts/check-web-links \
  -url="https://chengchuu.github.io/mazey/" \
  -reportPath="reports/mazey-links.txt" \
  -reportTypes="HTML,CSS"
```

The project-root URL must be absolute and use HTTP or HTTPS. Its path is
normalized to end in `/`. Scope is matched by path segment: a root of
`/mazey/` includes `/mazey/api/`, but excludes `/mazey-other/`. External URLs
and same-origin URLs outside that path are reported as skipped and are not
requested. Redirects must remain within the same scope at every hop.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-url` | none | Required deployed project-root URL |
| `-timeout` | `15s` | Timeout for each HTTP request |
| `-concurrency` | `8` | Maximum simultaneous requests |
| `-maxURLs` | `10000` | Maximum number of discovered in-scope URLs |
| `-showSkipped` | `false` | Print individual skipped URLs |
| `-checkFragments` | `false` | Validate HTML `id` and legacy named-anchor targets |
| `-reportPath` | none | Optional deterministic UTF-8 text report path |
| `-reportTypes` | all | Comma-separated resource types included in report URL details |

The command exits with status `0` when all discovered in-scope URLs and HTML
fragments pass, `1` when a URL or fragment fails or coverage is incomplete,
and `2` for invalid command arguments.

The `Checked URLs:` title is printed when the crawl starts, and each checked
URL is printed synchronously as its request completes. With concurrent
requests, these live entries follow completion order. Deterministically sorted
failed and skipped sections are followed by one resource-type line such as
`Resource types: CSS=21 HTML=42` and the final summary. The
`Skipped URLs:` section is hidden by default; pass `-showSkipped` to print the
section and its individual URL diagnostics. The summary always includes the
skipped count.

The `Failed URLs:` section is printed only when failures exist. The summary
always includes the failed count.

When `-reportPath` is provided, the command writes a post-crawl report grouped
by resource type with alphabetically sorted groups and URLs. The parent
directory must already exist. The report is written to a temporary sibling and
atomically replaces an existing file only after rendering succeeds.

`-reportTypes` accepts case-insensitive, comma-separated values and requires
`-reportPath`. It filters only checked-URL detail groups in the file; crawling,
validation, console output, failures, type counts, summary values, and exit
status remain complete. Supported values are `AUDIO`, `CSS`, `FONT`, `HTML`,
`IMAGE`, `JAVASCRIPT`, `JSON`, `MANIFEST`, `OTHER`, `PDF`, `SITEMAP`, `TEXT`,
`UNKNOWN`, `VIDEO`, and `XML`. Omit the flag or pass an empty value to include
all types.

Fragment targets are not checked by default. Pass `-checkFragments` to collect
HTML `id` attributes and legacy named anchors and report missing targets.
Fragments on CSS, JavaScript, images, and other non-HTML resources are never
validated as anchor targets.

## Discovery limits

The checker discovers references in HTML attributes and inline CSS, CSS
`url(...)` and `@import` rules, web manifests, and sitemap `<loc>` elements.
It fetches JavaScript but does not inspect JavaScript imports, `fetch` calls,
or runtime-generated URLs. It does not execute pages, retry requests, or find
resources that are only created dynamically. External and non-fetchable URLs
such as `mailto:`, `data:`, and `javascript:` are only reported as skipped.
