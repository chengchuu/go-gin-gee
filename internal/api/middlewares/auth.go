package middlewares

import (
	"net/http"

	"github.com/chengchuu/go-gin-gee/pkg/crypto"
	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader("authorization")
		if !crypto.ValidateToken(authorizationHeader) {
			code := http.StatusUnauthorized
			c.AbortWithStatusJSON(code, gin.H{"code": code, "message": "unauthorized"})
			return
		} else {
			c.Next()
		}
	}
}
