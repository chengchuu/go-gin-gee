package main

import (
	"os"
	"testing"
)

func TestMarkdownConversionAndClosingComment(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("data", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/md2td.md", []byte("Title\n\n<!-- ZH: 标题 -->\nBody\n"), 0600); err != nil {
		t.Fatal(err)
	}
	want := "/**\n * EN: Title\n * \n * ZH: 标题\n * Body\n */"
	for i := 0; i < 2; i++ {
		main()
		got, err := os.ReadFile("data/md2td.js")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("run %d: got %q, want %q", i, got, want)
		}
	}
}
