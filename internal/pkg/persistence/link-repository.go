package persistence

import (
	"errors"
	"fmt"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/pkg/helpers"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/chengchuu/gurl"
	"github.com/takuoki/clmconv"
)

type LinkRepository struct{}

var linkRepository *LinkRepository

const cusConPrefix = "[Link]"

func GetLinkRepository() *LinkRepository {
	if linkRepository == nil {
		linkRepository = &LinkRepository{}
	}
	return linkRepository
}

func (r *LinkRepository) SaveOriLink(OriLink string, addBaseUrl string, oneTime bool) (string, error) {
	var err error
	var record models.Link
	var linkForEncode string
	baseUrl := config.GetConfig().Data.BaseURL
	if addBaseUrl != "" {
		baseUrl = addBaseUrl
		linkForEncode, err = gurl.SetHashParam(OriLink, "base_url", addBaseUrl)
		if err != nil {
			return "", err
		}
	} else {
		linkForEncode = OriLink
	}
	if baseUrl == "" {
		return "", errors.New("BASE_URL is required")
	}
	OriMd5 := helpers.ConvertStringToMD5Hash(linkForEncode)
	data, err := r.QueryOriLinkByOriMd5(OriMd5)
	if err != nil {
		return "", err
	}
	if data != nil {
		return r.BuildLink(baseUrl, data.LinkKey), nil
	}
	record.OriLink = OriLink
	record.OriMd5 = OriMd5
	err = Create(&record)
	if err != nil {
		return "", err
	}
	linkID := record.ID
	// https://github.com/takuoki/clmconv
	converter := clmconv.New(clmconv.WithStartFromOne(), clmconv.WithLowercase())
	linkKey := converter.Itoa(int(linkID))
	// Compare
	specialLinks := config.GetConfig().Data.SpecialLinks
	if len(specialLinks) > 0 {
		for _, v := range specialLinks {
			if v.Key == linkKey {
				logger.Printf("%s Key(%s) is already in use", cusConPrefix, linkKey)
				record.OriLink = v.Link
				record.OriMd5 = helpers.ConvertStringToMD5Hash(v.Link)
				record.LinkKey = linkKey
				err = Save(&record)
				if err != nil {
					return "", err
				}
				return r.SaveOriLink(OriLink, addBaseUrl, oneTime)
			}
		}
	}
	_, err = r.SaveLinkKey(linkID, linkKey, oneTime)
	if err != nil {
		return "", err
	}
	record.LinkKey = linkKey
	return r.BuildLink(baseUrl, record.LinkKey), err
}

func (r *LinkRepository) BuildLink(baseUrl string, linkKey string) string {
	return fmt.Sprintf("%s/t/%s", baseUrl, linkKey)
}

func (r *LinkRepository) QueryOriLinkByLinkKey(linkKey string) (string, error) {
	if linkKey == "" {
		return "", errors.New("404 Link Not Found")
	}
	var record models.Link
	var err error
	specialLinks := config.GetConfig().Data.SpecialLinks
	if len(specialLinks) > 0 {
		for _, v := range specialLinks {
			if v.Key == linkKey {
				logger.Printf("%s Key(%s) is found in special links(%s)", cusConPrefix, linkKey, v.Link)
				return v.Link, err
			}
		}
	}
	where := models.Link{}
	where.LinkKey = linkKey
	notFound, err := First(&where, &record, []string{})
	logger.Printf("%s Is this key NotFound in DB: %t", cusConPrefix, notFound)
	if notFound {
		err = nil
		return "", errors.New("404 Link Not Found")
	}
	if err != nil {
		logger.Error("error: %v", err)
		return "", errors.New("404 Link Not Available")
	}
	if record.OneTime && record.VisitCount > 0 {
		return "", errors.New("404 Link Expired")
	}
	go r.RecordVisitCountByLinkKey(linkKey)
	return record.OriLink, err
}

func (r *LinkRepository) RecordVisitCountByLinkKey(linkKey string) (bool, error) {
	var record models.Link
	var err error
	where := models.Link{}
	where.LinkKey = linkKey
	notFound, err := First(&where, &record, []string{})
	if notFound {
		err = nil
		return false, errors.New("link not found")
	}
	if err != nil {
		return false, err
	}
	record.VisitCount = record.VisitCount + 1
	err = Updates(&record, &record)
	if err != nil {
		return false, err
	}
	logger.Printf("%s Current Count: %d", cusConPrefix, record.VisitCount)
	return true, err
}

func (r *LinkRepository) QueryOriLinkByOriMd5(OriMd5 string) (*models.Link, error) {
	var record models.Link
	if OriMd5 == "" {
		return nil, errors.New("OriMd5 is required")
	}
	where := models.Link{}
	where.OriMd5 = OriMd5
	notFound, err := First(&where, &record, []string{})
	logger.Printf("Check if the link is NotFound in DB: %t", notFound)
	if notFound {
		err = nil
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, err
}

func (r *LinkRepository) SaveLinkKey(linkID uint64, linkKey string, oneTime bool) (bool, error) {
	var record models.Link
	var err error
	where := models.Link{}
	where.ID = linkID
	record.LinkKey = linkKey
	record.OneTime = oneTime
	err = Updates(&where, &record)
	if err != nil {
		return false, err
	}
	return true, err
}
