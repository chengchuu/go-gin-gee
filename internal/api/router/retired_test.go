package router

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/api/controllers"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	"github.com/gin-gonic/gin"
)

type retiredTestTransport struct{ calls atomic.Int32 }

func (transport *retiredTestTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	return nil, errors.New("outbound HTTP is forbidden in retirement tests")
}

func TestRetiredAPIRoutes(t *testing.T) {
	previousConfig, previousDB := config.Config, db.DB
	previousMode, previousWriter := gin.Mode(), gin.DefaultWriter
	previousTransport := http.DefaultTransport
	transport := &retiredTestTransport{}
	http.DefaultTransport = transport
	config.Config = &config.Configuration{}
	db.DB = nil
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		config.Config, db.DB = previousConfig, previousDB
		gin.SetMode(previousMode)
		gin.DefaultWriter = previousWriter
		http.DefaultTransport = previousTransport
	})
	app := Setup(io.Discard)
	app.POST("/test-retired", controllers.RetiredAPI, func(c *gin.Context) {
		t.Error("handler ran after terminal retirement response")
	})
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/gee/get-tag-name"},
		{http.MethodGet, "/api/gee/get-tag-name?namespace=ignored"},
		{http.MethodPost, "/test-retired"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			app.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != http.StatusGone {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json; charset=utf-8" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("headers = %v", response.Header())
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			want := map[string]interface{}{"code": float64(41001), "message": "This API has been retired.", "data": nil}
			if !reflect.DeepEqual(payload, want) {
				t.Fatalf("payload = %#v, want %#v", payload, want)
			}
		})
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/unknown"},
		{http.MethodPost, "/api/gee/get-tag-name"},
	} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != http.StatusNotFound || response.Body.String() != `{"message":"api not found"}` || response.Header().Get("Cache-Control") != "" {
			t.Fatalf("unexpected fallback: %d %s %v", response.Code, response.Body.String(), response.Header())
		}
	}
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	var ping map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &ping); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || !strings.HasPrefix(ping["message"], "pong/") || response.Header().Get("Cache-Control") != "" {
		t.Fatalf("unexpected ping response: %d %s", response.Code, response.Body.String())
	}
	if calls := transport.calls.Load(); calls != 0 {
		t.Fatalf("made %d outbound HTTP requests", calls)
	}
}
