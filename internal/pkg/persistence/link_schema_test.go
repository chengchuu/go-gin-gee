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
	link := models.Link{OriLink: "https://example.test/retained", OriMd5: "retained-hash", LinkKey: "retained-key", OneTime: true, VisitCount: 7}
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
		for _, name := range []string{"uk_link_ori_md5", "idx_link_key"} {
			if !database.Migrator().HasIndex("gee_link", name) {
				t.Errorf("missing index %s", name)
			}
		}
		var stored models.Link
		if err := database.Table("gee_link").First(&stored, link.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.ID != link.ID || stored.OriLink != link.OriLink || stored.OriMd5 != link.OriMd5 || stored.LinkKey != link.LinkKey || !stored.OneTime || stored.VisitCount != link.VisitCount || !stored.CreatedAt.Equal(link.CreatedAt) || !stored.UpdatedAt.Equal(link.UpdatedAt) {
			t.Fatalf("migration changed link: %#v", stored)
		}
	}
	duplicate := models.Link{OriLink: "https://example.test/duplicate", OriMd5: link.OriMd5, LinkKey: "duplicate"}
	if err := database.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate link hash was accepted")
	}
	next := models.Link{OriLink: "https://example.test/next", OriMd5: "next-hash", LinkKey: "next-key"}
	if err := database.Create(&next).Error; err != nil {
		t.Fatal(err)
	}
	if next.ID <= link.ID {
		t.Fatalf("generated ID %d did not advance past %d", next.ID, link.ID)
	}
}
