package models

import "time"

type Model struct {
	ID        uint64    `gorm:"column:id;primary_key;auto_increment" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null;index" json:"updated_at"`
}
