package router

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesIncludesOnlyCurrentRoutes(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	app := gin.New()
	registerRoutes(app)

	registered := make(map[string]bool)
	for _, route := range app.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	currentRoutes := []string{
		"GET /docs/*any",
		"GET /",
		"GET /api/ping",
		"GET /api/index",
		"GET /api/gee/get-data-by-alias",
		"POST /api/gee/create-alias2data",
		"GET /api/gee/count-alias2data",
		"GET /api/gee/check",
		"POST /api/gee/webhook-message",
		"GET /api/gee/query-short-link",
		"POST /api/gee/generate-short-link",
		"GET /api/gee/get-tag-name",
		"GET /t/:key",
		"POST /server/mock",
		"GET /server/agent/record",
	}
	for _, route := range currentRoutes {
		if !registered[route] {
			t.Errorf("current route %q is not registered", route)
		}
	}

	legacyRoutes := []string{
		"POST /api/login",
		"POST /api/login/add",
		"GET /api/users",
		"POST /api/users",
		"GET /api/users/:id",
		"PUT /api/users/:id",
		"DELETE /api/users/:id",
	}
	for _, route := range legacyRoutes {
		if registered[route] {
			t.Errorf("legacy route %q is still registered", route)
		}
	}
}
