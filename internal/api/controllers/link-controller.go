package controllers

import (
	"errors"
	"net/http"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

func RedirectLink(c *gin.Context) {
	per := persistence.GetLinkRepository()
	linkKey := c.Param("link_key")
	if data, err := per.QueryOriLinkByLinkKey(linkKey); err != nil {
		renderLinkError(c, err)
	} else {
		c.Redirect(http.StatusFound, data)
	}
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
	if data, err := per.QueryOriLinkByLinkKey(linkKey); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("data not found"))
	} else {
		c.JSON(http.StatusOK, gin.H{"ori_link": data})
	}
}

func CreateLink(c *gin.Context) {
	type addParams struct {
		models.Link
		BaseUrl string `json:"base_url" form:"base_url"`
	}
	var record addParams
	var generatedLink string
	var baseUrl string
	var oneTime bool
	var err error
	s := persistence.GetLinkRepository()
	_ = c.BindJSON(&record)
	baseUrl = record.BaseUrl
	oneTime = record.OneTime
	if generatedLink, err = s.SaveOriLink(record.OriLink, baseUrl, oneTime); err != nil {
		http_err.NewError(c, http.StatusBadRequest, err)
	} else {
		c.JSON(http.StatusCreated, gin.H{
			// Deprecated: use data. tiny_link remains an identical response alias.
			"tiny_link": generatedLink,
			"data":      generatedLink,
			"errors":    []string{},
		})
	}
}
