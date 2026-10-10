package persistence

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"testing"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
)

func TestLinkFingerprintContract(t *testing.T) {
	got, err := linkFingerprint("https://target.test/a?x=1&y=2#part", "https://short.test", false, true)
	if err != nil {
		t.Fatal(err)
	}
	// Pin the explicit serialization, including false fields and JSON escaping.
	encoded := `{"original_url":"https://target.test/a?x=1\u0026y=2#part","base_url":"https://short.test","one_time":false,"direct_redirect":true}`
	want := fmt.Sprintf("%x", sha256.Sum256([]byte(encoded)))
	if got != want {
		t.Fatalf("fingerprint = %s, want %s", got, want)
	}
}

func TestLinkCreationIdentity(t *testing.T) {
	database := newPersistenceDatabase(t)
	repository := &LinkRepository{}
	original := "https://target.test/路径?next=https%3A%2F%2Fnested.test%2F%3Fa%3D1#?base_url=literal"
	seen := map[string]bool{}
	for _, base := range []string{"https://example.test", "https://override.test"} {
		for _, once := range []bool{false, true} {
			for _, direct := range []bool{false, true} {
				generated, err := repository.SaveOriLink(original, base, once, direct)
				if err != nil {
					t.Fatal(err)
				}
				if seen[generated] {
					t.Fatalf("different identity reused %s", generated)
				}
				seen[generated] = true
				repeated, err := repository.SaveOriLink(original, base, once, direct)
				if err != nil || repeated != generated {
					t.Fatalf("repeat: %s %v", repeated, err)
				}
				var stored models.Link
				if err := database.Where("link_key = ?", strings.TrimPrefix(generated, base+"/t/")).First(&stored).Error; err != nil {
					t.Fatal(err)
				}
				if stored.OriginalURL != original || stored.DirectRedirect != direct || stored.OneTime != once || len(stored.DedupHash) != 64 {
					t.Fatalf("stored: %#v", stored)
				}
			}
		}
	}
	implicit, err := repository.SaveOriLink(original, "", false, false)
	if err != nil || !seen[implicit] {
		t.Fatalf("effective default not deduplicated: %s %v", implicit, err)
	}
	changed, err := repository.SaveOriLink(original+"x", "", false, false)
	if err != nil || seen[changed] {
		t.Fatalf("destination not included: %s %v", changed, err)
	}
	var stored models.Link
	if err := database.Where("link_key = ?", strings.TrimPrefix(implicit, "https://example.test/t/")).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&stored).Update("visit_count", 4).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.SaveOriLink(original, "", false, false); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&stored, stored.ID).Error; err != nil || stored.VisitCount != 4 {
		t.Fatalf("deduplication reset visits: %#v %v", stored, err)
	}
}

func TestLinkConcurrentCreation(t *testing.T) {
	database := newPersistenceDatabase(t)
	repository := &LinkRepository{}
	const workers = 8
	urls := make([]string, workers)
	errors := make([]error, workers)
	var group sync.WaitGroup
	for i := range urls {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			urls[i], errors[i] = repository.SaveOriLink("https://concurrent.test", "", false, true)
		}(i)
	}
	group.Wait()
	for i := range urls {
		if errors[i] != nil || urls[i] != urls[0] {
			t.Fatalf("creation %d: %s %v", i, urls[i], errors[i])
		}
	}
	var count int64
	if err := database.Model(&models.Link{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("rows: %d %v", count, err)
	}
	var record models.Link
	if err := database.First(&record).Error; err != nil || record.LinkKey == "" {
		t.Fatalf("incomplete creation: %#v %v", record, err)
	}
}
