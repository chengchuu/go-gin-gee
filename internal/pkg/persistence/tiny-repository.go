package persistence

import (
	"errors"
	"fmt"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/tiny"
	"github.com/chengchuu/go-gin-gee/pkg/helpers"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/chengchuu/gurl"
	"github.com/takuoki/clmconv"
)

type TinyRepository struct{}

var tinyRepository *TinyRepository

const cusConPrefix = "[Tiny]"

func GetTinyRepository() *TinyRepository {
	if tinyRepository == nil {
		tinyRepository = &TinyRepository{}
	}
	return tinyRepository
}

func (r *TinyRepository) SaveOriLink(OriLink string, addBaseUrl string, oneTime bool) (string, error) {
	var err error
	var tiny models.Tiny
	var linkForEncode string
	if addBaseUrl != "" {
		linkForEncode, err = gurl.SetHashParam(OriLink, "base_url", addBaseUrl)
		if err != nil {
			return "", err
		}
	} else {
		linkForEncode = OriLink
	}
	OriMd5 := helpers.ConvertStringToMD5Hash(linkForEncode)
	baseUrl := config.GetConfig().Data.BaseURL
	if addBaseUrl != "" {
		baseUrl = addBaseUrl
	}
	if baseUrl == "" {
		return "", errors.New("BASE_URL is required")
	}
	data, err := r.QueryOriLinkByOriMd5(OriMd5)
	if err != nil {
		return "", err
	}
	if data != nil {
		return r.BuildTinyLink(baseUrl, data.TinyKey), nil
	}
	tiny.OriLink = OriLink
	tiny.OriMd5 = OriMd5
	err = Create(&tiny)
	if err != nil {
		return "", err
	}
	TinyId := tiny.ID
	// https://github.com/takuoki/clmconv
	converter := clmconv.New(clmconv.WithStartFromOne(), clmconv.WithLowercase())
	TinyKey := converter.Itoa(int(TinyId))
	// Compare
	specialLinks := config.GetConfig().Data.SpecialLinks
	if len(specialLinks) > 0 {
		for _, v := range specialLinks {
			if v.Key == TinyKey {
				logger.Printf("%s Key(%s) is already in use", cusConPrefix, TinyKey)
				tiny.OriLink = v.Link
				tiny.OriMd5 = helpers.ConvertStringToMD5Hash(v.Link)
				tiny.TinyKey = TinyKey
				err = Save(&tiny)
				if err != nil {
					return "", err
				}
				return r.SaveOriLink(OriLink, addBaseUrl, oneTime)
			}
		}
	}
	_, err = r.SaveTinyKey(TinyId, TinyKey, oneTime)
	if err != nil {
		return "", err
	}
	tiny.TinyKey = TinyKey
	return r.BuildTinyLink(baseUrl, tiny.TinyKey), err
}

func (r *TinyRepository) BuildTinyLink(baseUrl string, TinyKey string) string {
	return fmt.Sprintf("%s/t/%s", baseUrl, TinyKey)
}

func (r *TinyRepository) QueryOriLinkByTinyKey(TinyKey string) (string, error) {
	var tiny models.Tiny
	var err error
	specialLinks := config.GetConfig().Data.SpecialLinks
	if len(specialLinks) > 0 {
		for _, v := range specialLinks {
			if v.Key == TinyKey {
				logger.Printf("%s Key(%s) is found in special links(%s)", cusConPrefix, TinyKey, v.Link)
				return v.Link, err
			}
		}
	}
	where := models.Tiny{}
	where.TinyKey = TinyKey
	notFound, err := First(&where, &tiny, []string{})
	logger.Printf("%s Is this key NotFound in DB: %t", cusConPrefix, notFound)
	if notFound {
		err = nil
		return "", errors.New("404 Link Not Found")
	}
	if err != nil {
		logger.Error("error: %v", err)
		return "", errors.New("404 Link Not Available")
	}
	if tiny.OneTime && tiny.VisitCount > 0 {
		return "", errors.New("404 Link Expired")
	}
	go r.RecordVisitCountByTinyKey(TinyKey)
	return tiny.OriLink, err
}

func (r *TinyRepository) RecordVisitCountByTinyKey(TinyKey string) (bool, error) {
	var tiny models.Tiny
	var err error
	where := models.Tiny{}
	where.TinyKey = TinyKey
	notFound, err := First(&where, &tiny, []string{})
	if notFound {
		err = nil
		return false, errors.New("link not found")
	}
	if err != nil {
		return false, err
	}
	tiny.VisitCount = tiny.VisitCount + 1
	err = Updates(&where, &tiny)
	if err != nil {
		return false, err
	}
	logger.Printf("%s Current Count: %d", cusConPrefix, tiny.VisitCount)
	return true, err
}

func (r *TinyRepository) QueryOriLinkByOriMd5(OriMd5 string) (*models.Tiny, error) {
	var tiny models.Tiny
	if OriMd5 == "" {
		return nil, errors.New("OriMd5 is required")
	}
	where := models.Tiny{}
	where.OriMd5 = OriMd5
	notFound, err := First(&where, &tiny, []string{})
	logger.Printf("Check if the link is NotFound in DB: %t", notFound)
	if notFound {
		err = nil
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tiny, err
}

func (r *TinyRepository) SaveTinyKey(TinyId uint64, TinyKey string, oneTime bool) (bool, error) {
	var tiny models.Tiny
	var err error
	where := models.Tiny{}
	where.ID = TinyId
	tiny.TinyKey = TinyKey
	tiny.OneTime = oneTime
	err = Updates(&where, &tiny)
	if err != nil {
		return false, err
	}
	return true, err
}
