package alias2data

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Alias2data struct {
	models.Model
	Alias  string `gorm:"column:alias;size:100;not null;unique_index:uk_alias2data_alias" json:"alias" form:"alias"`
	Data   string `gorm:"column:data;type:text;not null" json:"data" form:"data"`
	Type   string `gorm:"column:type;size:30;not null;default:''" json:"type" form:"type"`
	Public bool   `gorm:"column:public;not null;default:true" json:"public" form:"public"`
}

func (Alias2data) TableName() string {
	return "gee_alias2data"
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
