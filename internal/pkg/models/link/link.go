package link

import (
	"time"

	"gorm.io/gorm"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Link struct {
	models.Model
	OriLink    string `gorm:"column:ori_link;type:text;not null" json:"ori_link" form:"ori_link"`
	OriMd5     string `gorm:"column:ori_md5;size:32;not null;uniqueIndex:uk_link_ori_md5" json:"ori_md5" form:"ori_md5"`
	LinkKey    string `gorm:"column:link_key;size:32;not null;index:idx_link_key" json:"link_key" form:"link_key"`
	OneTime    bool   `gorm:"column:one_time;not null;default:false" json:"one_time" form:"one_time"`
	VisitCount int    `gorm:"column:visit_count;type:int;not null;default:0" json:"visit_count" form:"visit_count"`
}

type SpecialLink struct {
	Key  string `json:"key"`
	Link string `json:"link"`
}

func (Link) TableName() string {
	return "gee_link"
}

func (m *Link) BeforeCreate(tx *gorm.DB) error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *Link) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}
