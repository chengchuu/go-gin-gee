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
	if containsHeader(allowedHeaders, "X-Common-API-Key") {
		t.Fatal("obsolete common API key header is allowed by CORS")
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

func TestCORSJSONPreflight(t *testing.T) {
	app := gin.New()
	app.Use(CORS())
	app.POST("/api/gee/generate-short-link", func(c *gin.Context) {
		t.Fatal("preflight reached the creation handler")
	})
	request := httptest.NewRequest(http.MethodOptions, "/api/gee/generate-short-link", nil)
	request.Header.Set("Origin", "https://pages.test")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "*" || !containsHeader(response.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("invalid preflight: %d %#v", response.Code, response.Header())
	}
	for _, header := range []string{"Content-Type"} {
		if !containsHeader(response.Header().Get("Access-Control-Allow-Headers"), header) {
			t.Fatalf("preflight does not allow %s", header)
		}
	}
}

func TestProxyPreflightLeavesCORSHeadersToNginx(t *testing.T) {
	app := gin.New()
	app.Use(PreflightHandler())
	request := httptest.NewRequest(http.MethodOptions, "/api/gee/generate-short-link", nil)
	request.Header.Set("Origin", "https://pages.test")
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status: %d", response.Code)
	}
	for _, header := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Access-Control-Allow-Methods", "Access-Control-Allow-Credentials"} {
		if response.Header().Get(header) != "" {
			t.Fatalf("Go emitted proxy-owned header %s", header)
		}
	}
}
