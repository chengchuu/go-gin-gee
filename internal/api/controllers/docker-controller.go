package controllers

import (
	"net/http"

	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

func GetTag(c *gin.Context) {
	rep := persistence.GetDockerRepository()
	tagName, err := rep.GetTagName("mazeyqian", "go-gin-gee", "api")
	if err != nil {
		http_err.NewError(c, http.StatusBadRequest, err)
	} else {
		c.JSON(http.StatusOK, gin.H{"tagName": tagName})
	}
}
