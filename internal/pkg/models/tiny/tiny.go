package tiny

import (
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Tiny struct {
	models.Model
	OriLink    string `gorm:"column:ori_link;not null;size:350" json:"ori_link" form:"ori_link"`
	OriMd5     string `gorm:"column:ori_md5;not null;size:40" json:"ori_md5" form:"ori_md5"`
	TinyKey    string `gorm:"column:tiny_key;not null;size:20" json:"tiny_key" form:"tiny_key"`
	TinyLink   string `gorm:"column:tiny_link;not null;size:30" json:"tiny_link" form:"tiny_link"`
	OneTime    bool   `gorm:"column:one_time;not null;default:false" json:"one_time" form:"one_time"`
	VisitCount int    `gorm:"column:visit_count;not null;default:0" json:"visit_count" form:"visit_count"`
}

type SpecialLink struct {
	Key  string `json:"key"`
	Link string `json:"link"`
}

func (Tiny) TableName() string {
	return "tiny"
}

func (m *Tiny) BeforeCreate() error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

func (m *Tiny) BeforeUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
