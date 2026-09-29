package persistence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/takuoki/clmconv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LinkRepository struct{}

var linkRepository = &LinkRepository{}

const cusConPrefix = "[Link]"

func GetLinkRepository() *LinkRepository { return linkRepository }

// The field names and order are part of the deduplication format. URLs remain exact.
func linkFingerprint(originalURL, baseURL string, oneTime, directRedirect bool) (string, error) {
	identity := struct {
		OriginalURL    string `json:"original_url"`
		BaseURL        string `json:"base_url"`
		OneTime        bool   `json:"one_time"`
		DirectRedirect bool   `json:"direct_redirect"`
	}{originalURL, baseURL, oneTime, directRedirect}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}

func (r *LinkRepository) SaveOriLink(originalURL, addBaseUrl string, oneTime, directRedirect bool) (string, error) {
	if err := checkDBDriver(); err != nil {
		return "", err
	}
	baseURL := config.GetConfig().Data.BaseURL
	if addBaseUrl != "" {
		baseURL = addBaseUrl
	}
	if baseURL == "" {
		return "", errors.New("BASE_URL is required")
	}
	hash, err := linkFingerprint(originalURL, baseURL, oneTime, directRedirect)
	if err != nil {
		return "", err
	}
	var record models.Link
	err = db.GetDB().Transaction(func(tx *gorm.DB) error {
		candidate := models.Link{OriginalURL: originalURL, DedupHash: hash, OneTime: oneTime, DirectRedirect: directRedirect}
		// Insert first: the unique constraint serializes concurrent identical requests.
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedup_hash"}}, DoNothing: true}).Create(&candidate).Error; err != nil {
			return err
		}
		if err := tx.Where("dedup_hash = ?", hash).First(&record).Error; err != nil {
			return err
		}
		if record.LinkKey != "" {
			return nil
		}
		converter := clmconv.New(clmconv.WithStartFromOne(), clmconv.WithLowercase())
		key := converter.Itoa(int(record.ID))
		// Generated keys contain no underscores. Suffixes avoid manually reserved keys
		// without creating dummy database records or changing the numeric ID.
		reserved := make(map[string]bool)
		for _, special := range config.GetConfig().Data.SpecialLinks {
			reserved[special.Key] = true
		}
		for reserved[key] {
			key += "_"
		}
		if len(key) > 32 {
			return errors.New("unable to allocate short-link key")
		}
		if err := tx.Model(&record).Update("link_key", key).Error; err != nil {
			return err
		}
		record.LinkKey = key
		return nil
	})
	if err != nil {
		return "", err
	}
	return r.BuildLink(baseURL, record.LinkKey), nil
}

func (r *LinkRepository) BuildLink(baseURL, linkKey string) string {
	return fmt.Sprintf("%s/t/%s", baseURL, linkKey)
}

func (r *LinkRepository) ResolveLink(linkKey string) (models.Resolution, error) {
	if linkKey == "" {
		return models.Resolution{}, errors.New("404 Link Not Found")
	}
	for _, special := range config.GetConfig().Data.SpecialLinks {
		if special.Key == linkKey {
			return models.Resolution{OriginalURL: special.Link, DirectRedirect: true}, nil
		}
	}
	var record models.Link
	notFound, err := First(&models.Link{LinkKey: linkKey}, &record, nil)
	if notFound {
		return models.Resolution{}, errors.New("404 Link Not Found")
	}
	if err != nil {
		return models.Resolution{}, errors.New("404 Link Not Available")
	}
	if record.OneTime && record.VisitCount > 0 {
		return models.Resolution{}, errors.New("404 Link Expired")
	}
	go r.RecordVisitCountByLinkKey(linkKey)
	return models.Resolution{OriginalURL: record.OriginalURL, DirectRedirect: record.DirectRedirect}, nil
}

func (r *LinkRepository) RecordVisitCountByLinkKey(linkKey string) (bool, error) {
	var record models.Link
	notFound, err := First(&models.Link{LinkKey: linkKey}, &record, nil)
	if notFound {
		return false, errors.New("link not found")
	}
	if err != nil {
		return false, err
	}
	record.VisitCount++
	if err := Updates(&record, &record); err != nil {
		return false, err
	}
	logger.Printf("%s Current Count: %d", cusConPrefix, record.VisitCount)
	return true, nil
}
