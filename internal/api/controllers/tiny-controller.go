package controllers

import (
	"errors"
	"net/http"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/tiny"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

func RedirectTiny(c *gin.Context) {
	per := persistence.GetTinyRepository()
	TinyKey := c.Param("key")
	if data, err := per.QueryOriLinkByTinyKey(TinyKey); err != nil {
		errStr := err.Error()
		if errStr == "" {
			errStr = "404 Link Not Found"
		}
		c.HTML(http.StatusNotFound, "index.tmpl", gin.H{
			"title": errStr,
		})
	} else {
		c.Redirect(http.StatusFound, data)
	}
}

func GetTiny(c *gin.Context) {
	per := persistence.GetTinyRepository()
	TinyKey := c.Query("tiny_key")
	if data, err := per.QueryOriLinkByTinyKey(TinyKey); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("data not found"))
	} else {
		c.JSON(http.StatusOK, gin.H{"ori_link": data})
	}
}

func CreateTiny(c *gin.Context) {
	type addParams struct {
		models.Tiny
		BaseUrl string `json:"base_url" form:"base_url"`
	}
	var tiny addParams
	var TinyLink string
	var baseUrl string
	var oneTime bool
	var err error
	s := persistence.GetTinyRepository()
	_ = c.BindJSON(&tiny)
	baseUrl = tiny.BaseUrl
	oneTime = tiny.OneTime
	if TinyLink, err = s.SaveOriLink(tiny.OriLink, baseUrl, oneTime); err != nil {
		http_err.NewError(c, http.StatusBadRequest, err)
	} else {
		c.JSON(http.StatusCreated, gin.H{
			"tiny_link": TinyLink,
			"data":      TinyLink,
			"errors":    []string{},
		})
	}
}
