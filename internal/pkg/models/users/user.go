package users

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type User struct {
	models.Model
	Username  string   `gorm:"column:username;type:varchar(80);not null;unique_index:uk_users_username" json:"username" form:"username"`
	Firstname string   `gorm:"column:firstname;type:varchar(80);not null;default:''" json:"firstname" form:"firstname"`
	Lastname  string   `gorm:"column:lastname;type:varchar(80);not null;default:''" json:"lastname" form:"lastname"`
	Hash      string   `gorm:"column:hash;type:varchar(100);not null" json:"hash"`
	Role      UserRole `gorm:"foreignkey:UserID;association_foreignkey:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"role"`
}

func (m *User) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *User) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
