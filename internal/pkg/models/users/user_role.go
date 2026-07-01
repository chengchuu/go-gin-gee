package users

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type UserRole struct {
	models.Model
	UserID   uint64 `gorm:"column:user_id;not null;unique_index:uk_user_roles_user_id" json:"user_id"`
	RoleName string `gorm:"column:role_name;type:varchar(50);not null;default:''" json:"role_name"`
}

func (m *UserRole) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *UserRole) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
