# Extract Nginx Missing Files

Extract unique filesystem paths from Nginx error-log entries containing this exact error:

```text
open() "PATH" failed (2: No such file or directory)
```

Paths are written one per line in first-seen order. Other Nginx errors are ignored.

Use `-sort` to sort paths naturally, so a path containing `icon-2.png` appears before one containing `icon-10.png`.

Use `-since` to include entries from an inclusive local timestamp through the time when the command starts. Matching lines without a valid timestamp are skipped when this filter is active.

## Read and Write Files

```bash
go run ./scripts/extract-nginx-missing-files \
  -in ./log/nginx_error.log \
  -out ./log/missing-files.log \
  -sort
```

## Use a Pipeline

```bash
cat nginx-error.log | go run ./scripts/extract-nginx-missing-files
```

## Sort and Filter by Date

```bash
go run ./scripts/extract-nginx-missing-files \
  -in nginx-error.log \
  -out missing-files.log \
  -sort \
  -since="2026/09/09 09:00:00"
```

Omit `-in` to read from standard input. Omit `-out` to write to standard output.
