package kv

import (
	"time"

	"gorm.io/gorm"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Counter struct {
	models.Model
	Key        string `gorm:"column:key;size:128;not null;uniqueIndex:uk_kv_counters_key" json:"key"`
	Value      int64  `gorm:"column:value;not null;default:0" json:"value"`
	Visibility string `gorm:"column:visibility;size:16;not null;default:'private'" json:"visibility"`
}

func (Counter) TableName() string { return "kv_counters" }

func (counter *Counter) BeforeCreate(tx *gorm.DB) error {
	counter.CreatedAt = time.Now().UTC()
	counter.UpdatedAt = counter.CreatedAt
	return nil
}

func (counter *Counter) BeforeUpdate(tx *gorm.DB) error {
	counter.UpdatedAt = time.Now().UTC()
	return nil
}
