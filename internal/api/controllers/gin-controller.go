package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Ping(c *gin.Context) {
	t := time.Now()
	// https://stackoverflow.com/questions/33119748/convert-time-time-to-string
	ret := "pong/v1.0.0/" + t.Format("2006-01-02 15:04:05")
	c.JSON(200, gin.H{
		"message": ret,
	})
}

func Index0310(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome",
	})
}

func Index0920(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Main Website",
	})
}
