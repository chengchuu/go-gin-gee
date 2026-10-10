package link

import (
	"time"

	"gorm.io/gorm"

	"github.com/chengchuu/go-gin-gee/internal/pkg/models"
)

type Link struct {
	models.Model
	OriginalURL string `gorm:"column:original_url;type:text;not null"`
	// Generated links use SHA-256; aliases stored on collision use fixed:<key>.
	DedupHash string `gorm:"column:dedup_hash;size:64;not null;uniqueIndex:uk_link_dedup_hash"`
	// NULL is allowed only while the creation transaction allocates the numeric ID.
	LinkKey        string `gorm:"column:link_key;size:32;default:null;uniqueIndex:uk_link_key"`
	DirectRedirect bool   `gorm:"column:direct_redirect;not null;default:false"`
	OneTime        bool   `gorm:"column:one_time;not null;default:false"`
	VisitCount     int    `gorm:"column:visit_count;type:int;not null;default:0"`
}

type SpecialLink struct {
	Key  string `json:"key"`
	Link string `json:"link"`
}

// CreateRequest contains only caller-controlled creation fields.
type CreateRequest struct {
	CommonAPIKey string `json:"common_api_key"`
	OriginalURL  string `json:"ori_link" binding:"required"`
	BaseURL      string `json:"base_url"`
	OneTime      bool   `json:"one_time"`
}

// Resolution is shared by persisted links and trusted configuration links.
type Resolution struct {
	OriginalURL    string
	DirectRedirect bool
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
