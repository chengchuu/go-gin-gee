package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsAuthorizationHeader(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	app := gin.New()
	app.Use(CORS())
	app.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	app.ServeHTTP(recorder, request)

	allowedHeaders := recorder.Header().Get("Access-Control-Allow-Headers")
	if !containsHeader(allowedHeaders, "Authorization") {
		t.Fatalf("Access-Control-Allow-Headers = %q, want Authorization", allowedHeaders)
	}
}

func containsHeader(headerList, target string) bool {
	for _, header := range strings.Split(headerList, ",") {
		if strings.EqualFold(strings.TrimSpace(header), target) {
			return true
		}
	}
	return false
}
