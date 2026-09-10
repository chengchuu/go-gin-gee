# Extract Nginx Missing Files

Extract unique filesystem paths from Nginx error-log entries containing this exact error:

```text
open() "PATH" failed (2: No such file or directory)
```

Paths are written one per line in first-seen order. Other Nginx errors are ignored.

## Read and Write Files

```bash
go run ./scripts/extract-nginx-missing-files \
  -in nginx-error.log \
  -out missing-files.log
```

## Use a Pipeline

```bash
cat nginx-error.log | go run ./scripts/extract-nginx-missing-files
```

Omit `-in` to read from standard input. Omit `-out` to write to standard output.
