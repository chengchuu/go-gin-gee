package middlewares

import (
	"net/http"
	"time"

	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
)

// NoMethodHandler
func NoMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(405, gin.H{"message": "metodo no permitido"})
	}
}

// NoRouteHandler
func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		logger.Error("path not found: %s", path)
		if len(path) > 5 && path[:5] == "/api/" {
			c.JSON(http.StatusNotFound, gin.H{"message": "api not found"})
		} else {
			c.HTML(http.StatusNotFound, "index.tmpl", gin.H{
				"title": "404 Page Not Found",
			})
		}
	}
}

func LoggerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Println("Request URL:", c.Request.URL)
		t := time.Now()
		// before request
		c.Next()
		// after request
		latency := time.Since(t)
		logger.Println("Consume Time:", latency)
		// access the status we are sending
		status := c.Writer.Status()
		logger.Println("StatusCode:", status)
	}
}
