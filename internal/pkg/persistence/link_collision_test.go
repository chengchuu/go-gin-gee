package persistence

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/takuoki/clmconv"
	"gorm.io/gorm"
)

func collisionRows(t *testing.T, database *gorm.DB) []models.Link {
	t.Helper()
	var rows []models.Link
	if err := database.Order("id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestLinkLazySpecialCollisions(t *testing.T) {
	database := newPersistenceDatabase(t)
	config.Config.Data.BaseURL = "https://i.mazey.net"
	config.Config.Data.SpecialLinks = []models.SpecialLink{
		{Key: "a", Link: "http://blog.mazey.net/"},
		{Key: "b", Link: "https://blog.mazey.net/"},
		{Key: "m", Link: "https://docs.test/"},
		{Key: "dl", Link: "https://sdk.test/#/README.md"},
	}
	repository := GetLinkRepository()
	resolved, err := repository.ResolveLink("dl")
	if err != nil || resolved.OriginalURL != "https://sdk.test/#/README.md" || !resolved.DirectRedirect {
		t.Fatalf("config resolution: %#v %v", resolved, err)
	}
	if len(collisionRows(t, database)) != 0 {
		t.Fatal("resolution created rows")
	}
	generated, err := repository.SaveOriLink("https://ordinary.test", "", true, false)
	if err != nil || generated != "https://i.mazey.net/t/c" {
		t.Fatalf("first link: %s %v", generated, err)
	}
	rows := collisionRows(t, database)
	if len(rows) != 3 {
		t.Fatalf("rows: %#v", rows)
	}
	for i, key := range []string{"a", "b"} {
		row := rows[i]
		if row.ID != uint64(i+1) || row.LinkKey != key || row.DedupHash != "fixed:"+key || row.OriginalURL != config.Config.Data.SpecialLinks[i].Link || !row.DirectRedirect || row.OneTime || row.VisitCount != 0 {
			t.Fatalf("special row: %#v", row)
		}
	}
	if rows[2].ID != 3 || rows[2].LinkKey != "c" || !rows[2].OneTime || rows[2].DirectRedirect || rows[2].OriginalURL != "https://ordinary.test" {
		t.Fatalf("ordinary row: %#v", rows[2])
	}
	// A newly configured alias must not cause deduplication to overwrite an existing row.
	config.Config.Data.SpecialLinks = append(config.Config.Data.SpecialLinks, models.SpecialLink{Key: "c", Link: "https://override.test"})
	repeated, err := repository.SaveOriLink("https://ordinary.test", "", true, false)
	if err != nil || repeated != generated || !reflect.DeepEqual(rows, collisionRows(t, database)) {
		t.Fatal("deduplication changed records")
	}
	config.Config.Data.SpecialLinks[0].Link = "https://changed.test"
	resolved, err = repository.ResolveLink("a")
	if err != nil || resolved.OriginalURL != "https://changed.test" || !resolved.DirectRedirect {
		t.Fatalf("config did not win: %#v %v", resolved, err)
	}
}

func TestLinkManySpecialCollisionsConcurrent(t *testing.T) {
	database := newPersistenceDatabase(t)
	converter := clmconv.New(clmconv.WithStartFromOne(), clmconv.WithLowercase())
	const reserved = 40
	for id := 1; id <= reserved; id++ {
		config.Config.Data.SpecialLinks = append(config.Config.Data.SpecialLinks, models.SpecialLink{Key: converter.Itoa(id), Link: "https://shared.test"})
	}
	const workers = 8
	urls, errs := make([]string, workers), make([]error, workers)
	var group sync.WaitGroup
	for i := range urls {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			urls[i], errs[i] = GetLinkRepository().SaveOriLink("https://shared.test", "", false, true)
		}(i)
	}
	group.Wait()
	want := "https://example.test/t/" + converter.Itoa(reserved+1)
	for i := range urls {
		if errs[i] != nil || urls[i] != want {
			t.Fatalf("creation: %s %v", urls[i], errs[i])
		}
	}
	rows := collisionRows(t, database)
	if len(rows) != reserved+1 {
		t.Fatalf("row count: %d", len(rows))
	}
	for _, row := range rows {
		if row.LinkKey == "" {
			t.Fatal("incomplete key committed")
		}
	}
}

func TestLinkSpecialCollisionRollback(t *testing.T) {
	for _, failAt := range []int{1, 3} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			database := newPersistenceDatabase(t)
			config.Config.Data.SpecialLinks = []models.SpecialLink{{Key: "a", Link: "https://a.test"}, {Key: "b", Link: "https://b.test"}}
			failure := errors.New("allocation failed")
			updates := 0
			if err := database.Callback().Update().Before("gorm:update").Register("test:allocation_failure", func(tx *gorm.DB) {
				updates++
				if updates == failAt {
					tx.AddError(failure)
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer database.Callback().Update().Remove("test:allocation_failure")
			if _, err := GetLinkRepository().SaveOriLink("https://ordinary.test", "", false, false); !errors.Is(err, failure) {
				t.Fatalf("failure: %v", err)
			}
			if len(collisionRows(t, database)) != 0 {
				t.Fatal("partial transaction committed")
			}
		})
	}
}
