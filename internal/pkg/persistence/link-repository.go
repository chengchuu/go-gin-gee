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

var ErrBaseURLRequired = errors.New("BASE_URL is required")

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
		return "", ErrBaseURLRequired
	}
	hash, err := linkFingerprint(originalURL, baseURL, oneTime, directRedirect)
	if err != nil {
		return "", err
	}
	var record models.Link
	err = db.GetDB().Transaction(func(tx *gorm.DB) error {
		converter := clmconv.New(clmconv.WithStartFromOne(), clmconv.WithLowercase())
		for {
			candidate := models.Link{OriginalURL: originalURL, DedupHash: hash, OneTime: oneTime, DirectRedirect: directRedirect}
			// Insert first so concurrent identical requests serialize on the unique hash.
			insert := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedup_hash"}}, DoNothing: true}).Create(&candidate)
			if insert.Error != nil {
				return insert.Error
			}
			record = models.Link{}
			if err := tx.Where("dedup_hash = ?", hash).First(&record).Error; err != nil {
				return err
			}
			if record.LinkKey != "" {
				return nil
			}
			if record.ID != candidate.ID || insert.RowsAffected != 1 {
				return errors.New("existing link has no allocated key")
			}
			key := converter.Itoa(int(record.ID))
			if destination, reserved := specialLinkDestination(key); reserved {
				// Free the request fingerprint before retrying with the next ID.
				if err := tx.Model(&record).Updates(map[string]interface{}{
					"original_url":    destination,
					"dedup_hash":      "fixed:" + key,
					"link_key":        key,
					"direct_redirect": true,
					"one_time":        false,
				}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&record).Update("link_key", key).Error; err != nil {
				return err
			}
			record.LinkKey = key
			return nil
		}
	})
	if err != nil {
		return "", err
	}
	return r.BuildLink(baseURL, record.LinkKey), nil
}

func (r *LinkRepository) BuildLink(baseURL, linkKey string) string {
	return fmt.Sprintf("%s/t/%s", baseURL, linkKey)
}

func specialLinkDestination(key string) (string, bool) {
	for _, special := range config.GetConfig().Data.SpecialLinks {
		if special.Key == key {
			return special.Link, true
		}
	}
	return "", false
}

func (r *LinkRepository) ResolveLink(linkKey string) (models.Resolution, error) {
	if linkKey == "" {
		return models.Resolution{}, errors.New("404 Link Not Found")
	}
	if destination, found := specialLinkDestination(linkKey); found {
		return models.Resolution{OriginalURL: destination, DirectRedirect: true}, nil
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
