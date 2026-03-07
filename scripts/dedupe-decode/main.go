package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

func main() {
	// Usage:
	//   go run scripts/dedupe-decode/main.go -in scripts/dedupe-decode/in.secret.txt -out scripts/dedupe-decode/out1.secret.txt
	//   cat scripts/dedupe-decode/in.secret.txt | go run scripts/dedupe-decode/main.go > scripts/dedupe-decode/out2.secret.txt
	//
	// If -in is not provided, read from stdin.
	// If -out is not provided, write to stdout.
	inPath := flag.String("in", "", "input file path (optional, default: stdin)")
	outPath := flag.String("out", "", "output file path (optional, default: stdout)")
	flag.Parse()

	var scanner *bufio.Scanner
	if *inPath != "" {
		f, err := os.Open(*inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open input file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	var out *os.File
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	writer := bufio.NewWriter(out)
	defer writer.Flush()

	seen := make(map[string]struct{})

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		decoded := decodePathSegments(line)

		if _, exists := seen[decoded]; exists {
			continue
		}
		seen[decoded] = struct{}{}

		fmt.Fprintln(writer, decoded)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}
}

// decodePathSegments decodes each path segment separately so "/" stays untouched.
// It also handles %20 etc.
func decodePathSegments(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		if decoded, err := url.PathUnescape(parts[i]); err == nil {
			parts[i] = decoded
		}
	}
	return strings.Join(parts, "/")
}
