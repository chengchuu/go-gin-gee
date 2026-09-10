package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testLocation = time.FixedZone("test", 8*60*60)

func TestExtractMissingPaths(t *testing.T) {
	t.Parallel()
	input := strings.Join([]string{
		`2026/09/09 12:52:13 [error] 16#16: *12477 open() "/web/x/base/robots.txt" failed (2: No such file or directory), client: 115.231.78.4`,
		`2026/09/09 13:23:31 [error] 17#17: *12565 "/web/x/base/cgi-bin/luci/index.html" is not found (2: No such file or directory), client: 104.244.79.113`,
		`2026/09/09 13:22:07 [error] 17#17: *12564 access forbidden by rule, client: 113.219.202.141`,
		`2026/09/09 13:45:24 [error] 16#16: *12604 open() "/web/archives/asset/icon/read-192x192.png" failed (2: No such file or directory), client: 180.213.52.58`,
		`2026/09/09 13:45:25 [error] 16#16: *12605 open() "/web/archives/asset/icon/read-512x512.png" failed (2: No such file or directory), client: 180.213.52.15`,
		`2026/09/09 13:46:00 [error] open() "/web/中文 path/image.png" failed (2: No such file or directory)`,
		`2026/09/09 13:47:00 [error] open() "/web/denied.txt" failed (13: Permission denied), client: 127.0.0.1`,
		`malformed open() "" failed (2: No such file or directory)`,
		`2026/09/09 13:47:30 [error] reopen() "/web/reopened.txt" failed (2: No such file or directory)`,
		`2026/09/09 13:48:00 [error] open() "/web/archives/asset/icon/read-192x192.png" failed (2: No such file or directory), client: 127.0.0.1`,
	}, "\n")

	var output bytes.Buffer
	if err := extractMissingPaths(strings.NewReader(input), &output, extractOptions{}); err != nil {
		t.Fatalf("extractMissingPaths() error = %v", err)
	}
	want := strings.Join([]string{
		"/web/x/base/robots.txt",
		"/web/archives/asset/icon/read-192x192.png",
		"/web/archives/asset/icon/read-512x512.png",
		"/web/中文 path/image.png",
		"",
	}, "\n")
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestExtractMissingPathsHandlesLongLines(t *testing.T) {
	t.Parallel()
	input := strings.Repeat("x", 128*1024) + ` open() "/web/long-path.png" failed (2: No such file or directory)`
	var output bytes.Buffer
	if err := extractMissingPaths(strings.NewReader(input), &output, extractOptions{}); err != nil {
		t.Fatalf("extractMissingPaths() error = %v", err)
	}
	if output.String() != "/web/long-path.png\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestExtractMissingPathsAllowsEmptyAndUnmatchedInput(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "ordinary log line\n"} {
		var output bytes.Buffer
		if err := extractMissingPaths(strings.NewReader(input), &output, extractOptions{}); err != nil {
			t.Fatalf("extractMissingPaths(%q) error = %v", input, err)
		}
		if output.Len() != 0 {
			t.Fatalf("extractMissingPaths(%q) output = %q", input, output.String())
		}
	}
}

func TestRunSupportsStdinAndStdout(t *testing.T) {
	t.Parallel()
	input := `open() "/web/image.png" failed (2: No such file or directory)`
	var stdout, stderr bytes.Buffer
	if code := run(nil, strings.NewReader(input), &stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != "/web/image.png\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExtractMissingPathsSortsNaturally(t *testing.T) {
	t.Parallel()
	input := strings.Join([]string{
		`open() "/web/icon-10.png" failed (2: No such file or directory)`,
		`open() "/web/icon-2.png" failed (2: No such file or directory)`,
		`open() "/web/icon-02.png" failed (2: No such file or directory)`,
		`open() "/web/icon-002.png" failed (2: No such file or directory)`,
		`open() "/web/icon-1.png" failed (2: No such file or directory)`,
	}, "\n")

	var output bytes.Buffer
	options := extractOptions{sortNaturally: true}
	if err := extractMissingPaths(strings.NewReader(input), &output, options); err != nil {
		t.Fatalf("extractMissingPaths() error = %v", err)
	}
	want := strings.Join([]string{
		"/web/icon-1.png",
		"/web/icon-002.png",
		"/web/icon-02.png",
		"/web/icon-2.png",
		"/web/icon-10.png",
		"",
	}, "\n")
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestExtractMissingPathsFiltersByDateBeforeDeduplication(t *testing.T) {
	t.Parallel()
	since := time.Date(2026, 9, 9, 9, 0, 0, 0, testLocation)
	until := time.Date(2026, 9, 9, 12, 0, 0, 0, testLocation)
	input := strings.Join([]string{
		`2026/09/09 08:59:59 [error] open() "/web/repeated.png" failed (2: No such file or directory)`,
		`2026/09/09 09:00:00 [error] open() "/web/start.png" failed (2: No such file or directory)`,
		`2026/09/09 10:00:00 [error] open() "/web/repeated.png" failed (2: No such file or directory)`,
		`2026/09/09 12:00:00 [error] open() "/web/end.png" failed (2: No such file or directory)`,
		`2026/09/09 12:00:01 [error] open() "/web/future.png" failed (2: No such file or directory)`,
		`invalid-date [error] open() "/web/invalid.png" failed (2: No such file or directory)`,
		`open() "/web/no-date.png" failed (2: No such file or directory)`,
	}, "\n")

	var output bytes.Buffer
	options := extractOptions{since: &since, until: until}
	if err := extractMissingPaths(strings.NewReader(input), &output, options); err != nil {
		t.Fatalf("extractMissingPaths() error = %v", err)
	}
	want := "/web/start.png\n/web/repeated.png\n/web/end.png\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestRunCombinesSinceAndNaturalSort(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, testLocation)
	input := strings.Join([]string{
		`2026/09/09 10:00:00 [error] open() "/web/icon-10.png" failed (2: No such file or directory)`,
		`2026/09/09 10:00:01 [error] open() "/web/icon-2.png" failed (2: No such file or directory)`,
	}, "\n")

	var stdout, stderr bytes.Buffer
	args := []string{"-sort", "-since", "2026/09/09 09:00:00"}
	if code := runAt(args, strings.NewReader(input), &stdout, &stderr, now); code != 0 {
		t.Fatalf("runAt() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != "/web/icon-2.png\n/web/icon-10.png\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunSupportsInputAndOutputFiles(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	inputPath := filepath.Join(directory, "nginx error.log")
	outputPath := filepath.Join(directory, "missing files.log")
	input := `open() "/web/file.png" failed (2: No such file or directory)`
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	if code := run([]string{"-in", inputPath, "-out", outputPath}, strings.NewReader("ignored"), io.Discard, &stderr); code != 0 {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "/web/file.png\n" {
		t.Fatalf("output file = %q", output)
	}
}

func TestRunExitCodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "unknown flag", args: []string{"-unknown"}, want: 2},
		{name: "positional argument", args: []string{"unexpected"}, want: 2},
		{name: "missing input", args: []string{"-in", filepath.Join(t.TempDir(), "missing.log")}, want: 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if code := run(test.args, strings.NewReader(""), io.Discard, io.Discard); code != test.want {
				t.Fatalf("run() code = %d, want %d", code, test.want)
			}
		})
	}
}

func TestRunRejectsInvalidSinceValues(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, testLocation)
	for _, since := range []string{
		"2026-09-09 09:00:00",
		"2026/09/09 12:00:01",
	} {
		var stderr bytes.Buffer
		code := runAt([]string{"-since", since}, strings.NewReader(""), io.Discard, &stderr, now)
		if code != 2 {
			t.Errorf("runAt(-since %q) code = %d, want 2", since, code)
		}
	}
}

func TestRunRejectsSameInputAndOutput(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nginx.log")
	content := []byte(`open() "/web/image.png" failed (2: No such file or directory)`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	if code := run([]string{"-in", path, "-out", path}, strings.NewReader(""), io.Discard, &stderr); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, content) {
		t.Fatalf("input was modified: got %q, want %q", after, content)
	}
}

func TestExtractMissingPathsReportsReadAndWriteFailures(t *testing.T) {
	t.Parallel()
	t.Run("read", func(t *testing.T) {
		reader := io.MultiReader(strings.NewReader("ordinary line\n"), errorReader{})
		if err := extractMissingPaths(reader, io.Discard, extractOptions{}); err == nil {
			t.Fatal("extractMissingPaths() error = nil")
		}
	})
	t.Run("write", func(t *testing.T) {
		input := strings.NewReader(`open() "/web/image.png" failed (2: No such file or directory)`)
		if err := extractMissingPaths(input, errorWriter{}, extractOptions{}); err == nil {
			t.Fatal("extractMissingPaths() error = nil")
		}
	})
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
