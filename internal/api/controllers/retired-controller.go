package controllers

import (
	"net/http"

	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

// RetiredAPI terminates requests to explicitly retired routes.
func RetiredAPI(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Abort()
	http_err.Failure(c, http.StatusGone, http_err.CodeAPIRetired, "This API has been retired.")
}
