package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBatchPullUsesOnlyRepositoriesWithRemotes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell fixture; PowerShell runtime requires Windows")
	}
	root := t.TempDir()
	for _, dir := range []string{"bin", "repos/with remote/.git", "repos/no-remote/.git", "repos/not-a-repo"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// PATH contains only the stub: neither remote lookup nor pull can reach Git.
	stub := "#!/bin/sh\nif [ \"$1\" = -C ]; then\n case \"$2\" in */no-remote) exit 1;; esac\n echo https://example.invalid/fixture.git\n exit 0\nfi\nif [ \"$1\" = pull ]; then\n printf '%s\\n' \"$PWD\" >> \"$GEE_GIT_TRACE\"\n exit 0\nfi\nexit 2\n"
	if err := os.WriteFile(filepath.Join(root, "bin/git"), []byte(stub), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "bin"))
	trace := filepath.Join(root, "trace")
	t.Setenv("GEE_GIT_TRACE", trace)
	originalFlags, originalArgs := flag.CommandLine, os.Args
	t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
	flag.CommandLine = flag.NewFlagSet("fixture", flag.ContinueOnError)
	os.Args = []string{"fixture", "-path=" + filepath.Join(root, "repos")}
	main()
	got, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(filepath.Join(root, "repos/with remote"))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := filepath.EvalSymlinks(strings.TrimSuffix(string(got), "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if actual != want {
		t.Fatalf("pull trace %q, want one pull in %q", got, want)
	}
}
