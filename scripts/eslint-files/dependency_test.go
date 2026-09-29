package main

import (
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestFileExclusionPreservesOrderAndDuplicates(t *testing.T) {
	files := []string{"b.js", "keep.ts", "a.js", "b.js"}
	got := lo.Filter(files, func(s string, _ int) bool { return !lo.Contains([]string{"keep.ts"}, s) })
	if want := []string{"b.js", "a.js", "b.js"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
