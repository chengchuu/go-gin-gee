package models

import "time"

type Model struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at;not null;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;index" json:"updated_at"`
}
