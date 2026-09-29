package controllers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/chengchuu/go-gin-gee/internal/api/auth"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

func RedirectLink(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	per := persistence.GetLinkRepository()
	linkKey := c.Param("link_key")
	if data, err := per.ResolveLink(linkKey); err != nil {
		renderLinkError(c, err)
	} else {
		destination := data.OriginalURL
		if !data.DirectRedirect {
			settings := config.GetConfig().Data
			warningURL, err := linkWarningURL(destination, settings.LinkRedirectPageURL)
			if err != nil {
				c.String(http.StatusServiceUnavailable, "Link warning service is unavailable.")
				return
			}
			destination = warningURL
		}
		c.Redirect(http.StatusFound, destination)
	}
}

// The warning service is configured by the application, never by request headers.
func linkWarningURL(destination, configuredURL string) (string, error) {
	warning, err := url.Parse(configuredURL)
	if err != nil || (warning.Scheme != "https" && warning.Scheme != "http") || warning.Hostname() == "" || warning.User != nil || warning.Fragment != "" {
		return "", errors.New("invalid warning URL")
	}
	query, err := url.ParseQuery(warning.RawQuery)
	if err != nil {
		return "", errors.New("invalid warning URL query")
	}
	query.Set("url", destination)
	warning.RawQuery = query.Encode()
	return warning.String(), nil
}

func renderLinkError(c *gin.Context, err error) {
	errStr := err.Error()
	if errStr == "" {
		errStr = "404 Link Not Found"
	}
	content := "This short link could not be opened. Check the link or contact the person who shared it."
	classname := "error"
	switch errStr {
	case "404 Link Not Found":
		content = "This short link could not be found."
		classname = "warn"
	case "404 Link Not Available":
		content = "This short link could not be opened."
	case "404 Link Expired":
		content = "This one-time link has already been used."
		classname = "warn"
	}
	c.HTML(http.StatusNotFound, "index.tmpl", gin.H{
		"title":     errStr,
		"content":   content,
		"classname": classname,
	})
}

func GetLink(c *gin.Context) {
	per := persistence.GetLinkRepository()
	linkKey := c.Query("link_key")
	if data, err := per.ResolveLink(linkKey); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("data not found"))
	} else {
		c.JSON(http.StatusOK, gin.H{"ori_link": data.OriginalURL})
	}
}

func CreateLink(c *gin.Context) {
	var record models.CreateRequest
	s := persistence.GetLinkRepository()
	if err := c.ShouldBindJSON(&record); err != nil {
		http_err.NewError(c, http.StatusBadRequest, errors.New("invalid link request"))
		return
	}
	direct := auth.ValidAPIKey(record.CommonAPIKey, config.GetConfig().Data.CommonAPIKeys)
	if generatedLink, err := s.SaveOriLink(record.OriginalURL, record.BaseURL, record.OneTime, direct); err != nil {
		http_err.NewError(c, http.StatusBadRequest, errors.New("unable to create short link"))
	} else {
		c.JSON(http.StatusCreated, gin.H{
			// Deprecated: use data. tiny_link remains an identical response alias.
			"tiny_link": generatedLink,
			"data":      generatedLink,
			"errors":    []string{},
		})
	}
}
