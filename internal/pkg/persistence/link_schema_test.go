package persistence

import (
	"testing"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
)

func TestLinkSchemaAndRepeatedMigration(t *testing.T) {
	database := newPersistenceDatabase(t)
	assertColumns := func() {
		t.Helper()
		if !database.Migrator().HasColumn("gee_link", "link_key") || database.Migrator().HasColumn("gee_link", "tiny_key") {
			t.Fatal("expected link_key column without tiny_key")
		}
	}
	assertColumns()
	if !database.Migrator().HasTable("gee_link") {
		t.Fatal("link table was not created")
	}
	if database.Migrator().HasTable("gee_tiny") {
		t.Fatal("obsolete link table was created")
	}
	link := models.Link{OriginalURL: "https://example.test/retained", DedupHash: "retained-hash", LinkKey: "retained-key", DirectRedirect: true, OneTime: true, VisitCount: 7}
	if err := database.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	// Read back engine-specific timestamp precision before comparing migrations.
	if err := database.First(&link, link.ID).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := database.AutoMigrate(&models.Link{}); err != nil {
			t.Fatal(err)
		}
		assertColumns()
		for _, name := range []string{"uk_link_dedup_hash", "uk_link_key"} {
			if !database.Migrator().HasIndex("gee_link", name) {
				t.Errorf("missing index %s", name)
			}
		}
		var stored models.Link
		if err := database.Table("gee_link").First(&stored, link.ID).Error; err != nil {
			t.Fatal(err)
		}
		if !stored.DirectRedirect {
			t.Fatal("migration lost redirect policy")
		}
		if stored.ID != link.ID || stored.OriginalURL != link.OriginalURL || stored.DedupHash != link.DedupHash || stored.LinkKey != link.LinkKey || !stored.OneTime || stored.VisitCount != link.VisitCount || !stored.CreatedAt.Equal(link.CreatedAt) || !stored.UpdatedAt.Equal(link.UpdatedAt) {
			t.Fatalf("migration changed link: %#v", stored)
		}
	}
	duplicate := models.Link{OriginalURL: "https://example.test/duplicate", DedupHash: link.DedupHash, LinkKey: "duplicate"}
	if err := database.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate link hash was accepted")
	}
	duplicateKey := models.Link{OriginalURL: "https://example.test/other", DedupHash: "other-hash", LinkKey: link.LinkKey}
	if err := database.Create(&duplicateKey).Error; err == nil {
		t.Fatal("duplicate short key was accepted")
	}
	next := models.Link{OriginalURL: "https://example.test/next", DedupHash: "next-hash", LinkKey: "next-key"}
	if err := database.Create(&next).Error; err != nil {
		t.Fatal(err)
	}
	if next.ID <= link.ID {
		t.Fatalf("generated ID %d did not advance past %d", next.ID, link.ID)
	}
}
