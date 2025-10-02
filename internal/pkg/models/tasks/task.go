package tasks

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/users"
)

type Task struct {
	models.Model
	Name   string     `gorm:"column:name;not null;" json:"name" form:"name"`
	Text   string     `gorm:"column:text;not null;" json:"text" form:"text"`
	UserID uint64     `gorm:"column:user_id;unique_index:user_id;not null;" json:"user_id" form:"user_id"`
	User   users.User `json:"user"`
}

func (m *Task) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *Task) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
