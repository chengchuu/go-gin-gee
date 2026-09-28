package persistence

import (
	"testing"

	"github.com/takuoki/clmconv"
)

func TestShortKeyAlphabetBoundaries(t *testing.T) {
	converter := clmconv.New(clmconv.WithStartFromOne(), clmconv.WithLowercase())
	for _, tc := range []struct {
		id  int
		key string
	}{
		{0, ""}, {1, "a"}, {25, "y"}, {26, "z"}, {27, "aa"},
		{52, "az"}, {53, "ba"}, {676, "yz"}, {677, "za"},
		{702, "zz"}, {703, "aaa"}, {18278, "zzz"}, {18279, "aaaa"},
	} {
		if got := converter.Itoa(tc.id); got != tc.key {
			t.Errorf("ID %d: got %q, want %q", tc.id, got, tc.key)
		}
	}
}
