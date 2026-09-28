package persistence

import (
	"strings"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/kv"
	modelsLink "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/internal/testutil"
	"github.com/chengchuu/go-gin-gee/pkg/helpers"
	"gorm.io/gorm"
)

// These tests capture persistence contracts before the ORM migration. Each test
// owns a disposable database; no configured application database is opened.
func newPersistenceDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, engine := testutil.OpenDatabase(t)
	if err := database.AutoMigrate(&kv.Entry{}, &kv.Counter{}, &modelsLink.Link{}); err != nil {
		_ = closeTestDatabase(database)
		t.Fatal(err)
	}
	previousDB, previousConfig := db.DB, config.Config
	db.DB = database
	config.Config = &config.Configuration{
		Database: config.DatabaseConfiguration{Driver: engine},
		Data:     config.DataConfiguration{BaseURL: "https://example.test"},
	}
	t.Cleanup(func() {
		db.DB, config.Config = previousDB, previousConfig
		if err := closeTestDatabase(database); err != nil {
			t.Error(err)
		}
	})
	return database
}

func TestPersistenceMigrationPreservesRowsAndUniqueKeys(t *testing.T) {
	database := newPersistenceDatabase(t)
	entry := kv.Entry{Key: "retained", Value: "original"}
	if err := database.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.ID == 0 || entry.CreatedAt.IsZero() || entry.UpdatedAt.IsZero() {
		t.Fatalf("missing generated fields: %#v", entry)
	}
	// Compare persisted precision, which differs between database engines.
	if err := database.First(&entry, entry.ID).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := database.AutoMigrate(&kv.Entry{}, &kv.Counter{}, &modelsLink.Link{}); err != nil {
			t.Fatal(err)
		}
	}
	var retained kv.Entry
	if err := database.First(&retained, entry.ID).Error; err != nil {
		t.Fatal(err)
	}
	if retained.Key != entry.Key || retained.Value != entry.Value || !retained.CreatedAt.Equal(entry.CreatedAt) {
		t.Fatalf("migration changed stored data: %#v", retained)
	}
	if err := database.Create(&kv.Entry{Key: entry.Key, Value: "duplicate"}).Error; err == nil {
		t.Fatal("duplicate entry key was accepted")
	}
	counter := kv.Counter{Key: "counter"}
	if err := database.Create(&counter).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&kv.Counter{Key: counter.Key}).Error; err == nil {
		t.Fatal("duplicate counter key was accepted")
	}
}

func TestPersistenceRollbackAndZeroValueUpdates(t *testing.T) {
	database := newPersistenceDatabase(t)
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	if err := tx.Create(&kv.Entry{Key: "rolled-back", Value: "temporary"}).Error; err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if _, err := GetKVRepository().Get("rolled-back"); err != ErrKVNotFound {
		t.Fatalf("rolled-back row: %v", err)
	}
	entry := &kv.Entry{Key: "empty", Value: "before", Visibility: "public"}
	if created, err := GetKVRepository().Set(entry); err != nil || !created {
		t.Fatalf("create: created=%t err=%v", created, err)
	}
	entry.Value = ""
	if created, err := GetKVRepository().Set(entry); err != nil || created {
		t.Fatalf("update: created=%t err=%v", created, err)
	}
	stored, err := GetKVRepository().Get(entry.Key)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Value != "" || stored.ID != entry.ID {
		t.Fatalf("empty update not preserved: %#v", stored)
	}
}

func TestLinkPersistenceDeduplicationVisitsAndSpecialLinks(t *testing.T) {
	database := newPersistenceDatabase(t)
	repository := &LinkRepository{}
	link, err := repository.SaveOriLink("https://target.test/page", "", true)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := repository.SaveOriLink("https://target.test/page", "", true)
	if err != nil || duplicate != link {
		t.Fatalf("deduplication: link=%q err=%v", duplicate, err)
	}
	var stored modelsLink.Link
	if err := database.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ID == 0 || stored.LinkKey == "" || !stored.OneTime || stored.VisitCount != 0 || stored.CreatedAt.IsZero() {
		t.Fatalf("incorrect persisted short link: %#v", stored)
	}
	other := modelsLink.Link{OriMd5: "other", OriLink: "https://other.test", LinkKey: "other"}
	if err := database.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RecordVisitCountByLinkKey(stored.LinkKey); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&other, other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if other.VisitCount != 0 {
		t.Fatalf("unrelated link visit count = %d, want 0", other.VisitCount)
	}
	if err := database.First(&stored, stored.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.VisitCount != 1 {
		t.Fatalf("visit count = %d, want 1", stored.VisitCount)
	}
	if _, err := repository.QueryOriLinkByLinkKey(stored.LinkKey); err == nil || err.Error() != "404 Link Expired" {
		t.Fatalf("one-time expiration: %v", err)
	}
	config.Config.Data.SpecialLinks = []modelsLink.SpecialLink{{Key: stored.LinkKey, Link: "https://special.test"}}
	if resolved, err := repository.QueryOriLinkByLinkKey(stored.LinkKey); err != nil || resolved != "https://special.test" {
		t.Fatalf("special-link precedence: link=%q err=%v", resolved, err)
	}
	if err := database.Create(&modelsLink.Link{OriMd5: stored.OriMd5, OriLink: "duplicate"}).Error; err == nil {
		t.Fatal("duplicate short-link hash was accepted")
	}
}

func closeTestDatabase(database *gorm.DB) error {
	pool, err := database.DB()
	if err != nil {
		return err
	}
	return pool.Close()
}

func TestLinkBaseURLHashCompatibility(t *testing.T) {
	for _, test := range []struct{ original, hashed string }{
		{"https://target.test/page", "https://target.test/page#?base_url=https://short.test"},
		{"https://target.test/page?x=1#/view?mode=full", "https://target.test/page?x=1#/view?mode=full&base_url=https://short.test"},
		{"https://target.test/page#?base_url=https://old.test", "https://target.test/page#?base_url=https://short.test"},
	} {
		t.Run(test.original, func(t *testing.T) {
			newPersistenceDatabase(t)
			repository := &LinkRepository{}
			link, err := repository.SaveOriLink(test.original, "https://short.test", false)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(link, "https://short.test/t/") {
				t.Fatalf("base URL override lost: %q", link)
			}
			stored, err := repository.QueryOriLinkByOriMd5(helpers.ConvertStringToMD5Hash(test.hashed))
			if err != nil || stored == nil || stored.OriLink != test.original {
				t.Fatalf("persisted hash or original URL changed: stored=%#v err=%v", stored, err)
			}
			again, err := repository.SaveOriLink(test.original, "https://short.test", false)
			if err != nil || again != link {
				t.Fatalf("deduplication changed: link=%q err=%v", again, err)
			}
		})
	}
}
