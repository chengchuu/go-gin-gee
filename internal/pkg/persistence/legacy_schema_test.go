package persistence

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models/kv"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/tiny"
	"github.com/chengchuu/go-gin-gee/internal/testutil"
)

//go:embed testdata/gorm_v1_schema.sql
var legacySchema string

func TestLegacySchemaMigration(t *testing.T) {
	database, engine := testutil.OpenDatabase(t)
	schema := legacySchema
	// Translate SQLite-specific types to the GORM v1 server dialect types.
	// These server fixtures are representative schemas, not server-generated dumps.
	switch engine {
	case "postgres":
		schema = strings.ReplaceAll(schema, "integer primary key autoincrement", "bigserial primary key")
		schema = strings.ReplaceAll(schema, "datetime", "timestamp with time zone")
	case "mysql":
		schema = strings.ReplaceAll(schema, `"`, "`")
		schema = strings.ReplaceAll(schema, "integer primary key autoincrement", "bigint unsigned primary key auto_increment")
	}
	for _, statement := range strings.Split(schema, ";") {
		if strings.TrimSpace(statement) != "" {
			if err := database.Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	entry := kv.Entry{Key: "legacy", Value: "retained", ContentType: "text/plain", Visibility: "public"}
	counter := kv.Counter{Key: "legacy-counter", Value: 123, Visibility: "private"}
	link := tiny.Tiny{OriLink: "https://legacy.test", OriMd5: "legacy-hash", TinyKey: "legacy-key", OneTime: true, VisitCount: 7}
	for _, value := range []interface{}{&entry, &counter, &link} {
		if err := database.Create(value).Error; err != nil {
			t.Fatal(err)
		}
		// Capture the server's timestamp precision before migration.
		if err := database.First(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := database.AutoMigrate(&kv.Entry{}, &kv.Counter{}, &tiny.Tiny{}); err != nil {
			t.Fatal(err)
		}
		for _, index := range []struct {
			model interface{}
			name  string
		}{
			{&kv.Entry{}, "uk_kv_entries_key"},
			{&kv.Counter{}, "uk_kv_counters_key"},
			{&tiny.Tiny{}, "uk_tiny_ori_md5"},
			{&tiny.Tiny{}, "idx_tiny_key"},
		} {
			if !database.Migrator().HasIndex(index.model, index.name) {
				t.Errorf("migration lost index %s", index.name)
			}
		}
	}
	var storedEntry kv.Entry
	var storedCounter kv.Counter
	var storedLink tiny.Tiny
	for _, value := range []interface{}{&storedEntry, &storedCounter, &storedLink} {
		if err := database.First(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if storedEntry.ID != entry.ID || storedEntry.Value != entry.Value || storedEntry.Key != entry.Key || storedEntry.Visibility != entry.Visibility || !storedEntry.CreatedAt.Equal(entry.CreatedAt) {
		t.Fatalf("legacy entry changed: %#v", storedEntry)
	}
	if storedCounter.ID != counter.ID || storedCounter.Value != counter.Value || storedCounter.Key != counter.Key {
		t.Fatalf("legacy counter changed: %#v", storedCounter)
	}
	if storedLink.ID != link.ID || storedLink.OriLink != link.OriLink || storedLink.TinyKey != link.TinyKey || !storedLink.OneTime || storedLink.VisitCount != link.VisitCount {
		t.Fatalf("legacy link changed: %#v", storedLink)
	}
	if err := database.Create(&kv.Entry{Key: entry.Key, Value: "duplicate"}).Error; err == nil {
		t.Fatal("legacy uniqueness constraint no longer enforced")
	}
	newEntry := kv.Entry{Key: "new-after-migration", Value: "new"}
	if err := database.Create(&newEntry).Error; err != nil {
		t.Fatal(err)
	}
	if newEntry.ID <= entry.ID {
		t.Fatalf("generated ID %d did not advance past %d", newEntry.ID, entry.ID)
	}
}
