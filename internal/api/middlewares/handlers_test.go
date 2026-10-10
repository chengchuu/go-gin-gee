package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNoRoutePageClassname(t *testing.T) {
	app := gin.New()
	app.LoadHTMLFiles("../../../assets/data/index.tmpl")
	app.NoRoute(NoRouteHandler())
	for _, path := range []string{"/missing-page", "/api/missing"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", w.Code)
			}
			if path == "/api/missing" {
				if w.Body.String() != `{"message":"api not found"}` || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
					t.Fatalf("unexpected API response: %s", w.Body.String())
				}
			} else if !strings.Contains(w.Body.String(), `<main class="base base-warn"`) || !strings.Contains(w.Body.String(), `<h1 id="title">404 Page Not Found</h1>`) {
				t.Fatal("HTML 404 missing warning class or existing title")
			}
		})
	}
}
