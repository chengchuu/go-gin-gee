package alias2data

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Alias2data struct {
	models.Model
	Alias  string `gorm:"column:alias;not null;size:50" json:"alias" form:"alias"`
	Data   string `gorm:"column:data;not null;size:255" json:"data" form:"data"`
	Type   string `gorm:"column:type;not null;size:20" json:"type" form:"type"`
	Public bool   `gorm:"column:public;not null;default:true" json:"public" form:"public"`
}

func (m *Alias2data) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *Alias2data) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
