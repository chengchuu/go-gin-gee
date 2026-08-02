# dedupe-decode

A small Go utility script to:

1. Remove duplicate lines
2. Decode URL-encoded path segments (for example, `%E6%96%B0%E9%A1%B5%E9%9D%A2` → `新页面`)
3. Filter out unwanted URLs:
   - lines starting with `//`
   - lines ending with dynamic extensions `.php` or `.jsp` (case-insensitive)

---

## File Location

- Script: `scripts/dedupe-decode/main.go`

---

## Requirements

- Go 1.18+ (recommended)

Check your Go version:

```bash
go version
```

---

## Usage

Run from project root:

```bash
go run scripts/dedupe-decode/main.go -in scripts/dedupe-decode/in.example.txt -out scripts/dedupe-decode/out1.example.txt
```

Or use stdin/stdout:

```bash
cat scripts/dedupe-decode/in.example.txt | go run scripts/dedupe-decode/main.go > scripts/dedupe-decode/out2.example.txt
```

---

## Flags

- `-in` (optional): input file path
  - If omitted, reads from `stdin`
- `-out` (optional): output file path
  - If omitted, writes to `stdout`

Examples:

```bash
# file -> file
go run scripts/dedupe-decode/main.go -in input.txt -out output.txt

# stdin -> stdout
cat input.txt | go run scripts/dedupe-decode/main.go
```

---

## What the script does

For each input line:

1. `TrimSpace` and skip empty lines
2. Decode each URL path segment safely using `url.PathUnescape`
   - Keeps `/` separators unchanged
3. Skip line if:
   - starts with `//`
   - file extension is `.php` or `.jsp` (after removing query/fragment)
4. Deduplicate while preserving first-seen order
5. Write remaining lines to output

---

## Filtering rules

### 1) Skip protocol-relative URLs

Skipped if line starts with:

- `//`

Example skipped:

- `//cdn.example.com/a.js`

### 2) Skip dynamic page extensions

Skipped if decoded path ends with:

- `.php`
- `.jsp`

Case-insensitive, and still skipped with query/fragment:

- `/a/b/index.php`
- `/a/b/index.php?x=1`
- `/a/b/index.JSP#hash`

---

## Example

Input:

```plain
/asset/default/123/%E6%96%B0%E9%A1%B5%E9%9D%A2.svg
/asset/default/123/%E6%96%B0%E9%A1%B5%E9%9D%A2.svg
//cdn.example.com/test.js
/asset/read/index.php?from=abc
/asset/de/%E5%B9%B2%E9%A5%AD%20(2).svg
```

Output:

```plain
/asset/default/123/新页面.svg
/asset/de/干饭 (2).svg
```

---

## Notes

- Deduplication happens **after decoding**.
- If decoding of a segment fails, the original segment is kept.
- Output order is stable (first occurrence wins).

---

## Troubleshooting

- **`failed to open input file`**
  Check the `-in` path and run command from the correct working directory.
- **`failed to create output file`**
  Check parent directory exists and you have write permission.
- **Unexpected empty output**
  Your lines may be filtered by `//`, `.php`, or `.jsp` rules.
