package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestLinkHTTPContract(t *testing.T) {
	database, engine := testutil.OpenDatabase(t)
	if err := database.AutoMigrate(&models.Link{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousConfig, previousMode := db.DB, config.Config, gin.Mode()
	db.DB = database
	config.Config = &config.Configuration{
		Database: config.DatabaseConfiguration{Driver: engine},
		Data:     config.DataConfiguration{BaseURL: "https://example.test", SpecialLinks: []models.SpecialLink{{Key: "special", Link: "https://destination.test/page"}}},
	}
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { db.DB, config.Config = previousDB, previousConfig; gin.SetMode(previousMode) })
	app := gin.New()
	registerRoutes(app)
	app.LoadHTMLFiles("../../../assets/data/index.tmpl")
	var generated string
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/gee/generate-short-link", strings.NewReader(`{"ori_link":"https://original.test/page"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", response.Code, response.Body.String())
		}
		var payload struct {
			Alias  string   `json:"tiny_link"`
			Data   string   `json:"data"`
			Errors []string `json:"errors"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
			t.Fatal(err)
		}
		if _, exists := fields["link"]; exists {
			t.Fatal("response contains removed link field")
		}
		var stored models.Link
		if err := database.Where("original_url = ?", "https://original.test/page").First(&stored).Error; err != nil {
			t.Fatal(err)
		}
		if stored.LinkKey == "" || payload.Data != "https://example.test/t/"+stored.LinkKey || payload.Alias != payload.Data || payload.Errors == nil || len(payload.Errors) != 0 {
			t.Fatalf("unexpected response: %s", response.Body.String())
		}
		if i > 0 && generated != payload.Data {
			t.Fatal("deduplication changed generated URL")
		}
		generated = payload.Data
	}
	// Configured aliases resolve without asynchronous database visit writes.
	for _, tc := range []struct {
		path     string
		status   int
		redirect bool
	}{
		{"/api/gee/query-short-link?link_key=special", http.StatusOK, false},
		{"/api/gee/query-short-link?link_key=special&tiny_key=ignored", http.StatusOK, false},
		{"/api/gee/query-short-link?tiny_key=special", http.StatusNotFound, false},
		{"/api/gee/query-short-link?link_key=", http.StatusNotFound, false},
		{"/api/gee/query-short-link?link_key=missing", http.StatusNotFound, false},
		{"/t/special", http.StatusFound, true},
		{"/t/missing", http.StatusNotFound, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if tc.redirect && response.Header().Get("Location") != "https://destination.test/page" {
				t.Fatalf("location = %q", response.Header().Get("Location"))
			}
			if tc.redirect && response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("redirect must not be cached")
			}
			if tc.status == http.StatusOK {
				var payload map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if payload["ori_link"] != "https://destination.test/page" {
					t.Fatalf("payload = %#v", payload)
				}
			}
		})
	}
}
