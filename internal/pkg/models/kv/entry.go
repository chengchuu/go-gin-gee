package kv

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Entry struct {
	models.Model
	Key         string `gorm:"column:key;size:128;not null;unique_index:uk_kv_entries_key" json:"key"`
	Value       string `gorm:"column:value;type:text;not null" json:"value"`
	ContentType string `gorm:"column:content_type;size:64;not null;default:'text/plain'" json:"content_type"`
	Visibility  string `gorm:"column:visibility;size:16;not null;default:'private'" json:"visibility"`
}

func (Entry) TableName() string { return "kv_entries" }

func (entry *Entry) BeforeCreate() error {
	entry.CreatedAt = time.Now().UTC()
	entry.UpdatedAt = entry.CreatedAt
	return nil
}

func (entry *Entry) BeforeUpdate() error {
	entry.UpdatedAt = time.Now().UTC()
	return nil
}
