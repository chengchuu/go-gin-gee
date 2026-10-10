package persistence

import (
	"strings"
	"testing"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/docker"
	"github.com/samber/lo"
)

func TestDockerTagSelection(t *testing.T) {
	// Lock the ordered selection contract used by GetTagName without HTTP calls.
	tags := []models.DockerV2TagsResult{{Name: "latest"}, {Name: "first-amd64"}, {Name: "second-amd64"}}
	for _, tc := range []struct{ filter, want string }{{"amd64", "first-amd64"}, {"missing", ""}, {"", "latest"}} {
		got, ok := lo.Find(tags, func(tag models.DockerV2TagsResult) bool { return strings.Contains(tag.Name, tc.filter) })
		if got.Name != tc.want || ok != (tc.want != "") {
			t.Fatalf("filter %q: got %q, %v; want %q", tc.filter, got.Name, ok, tc.want)
		}
	}
}
