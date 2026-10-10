package router

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"github.com/chengchuu/go-gin-gee/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestLinkCreationBaseURLError(t *testing.T) {
	for _, tc := range []struct {
		name, baseURL, body, message string
		failDB                       bool
		status                       int
	}{
		{"missing", "", `{"ori_link":"https://destination.test"}`, "BASE_URL is required. Configure Data.BaseURL or provide base_url in the request.", false, http.StatusBadRequest},
		{"override", "", `{"ori_link":"https://destination.test","base_url":"https://override.test"}`, "", false, http.StatusCreated},
		{"database failure", "https://short.test", `{"ori_link":"https://destination.test"}`, "unable to create short link", true, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, app := newLinkPolicyRouter(t)
			config.Config.Data.BaseURL = tc.baseURL
			if tc.failDB {
				if err := database.Callback().Create().Before("gorm:create").Register("test:creation_failure", func(tx *gorm.DB) {
					tx.AddError(errors.New("private SQL diagnostic"))
				}); err != nil {
					t.Fatal(err)
				}
			}
			request := httptest.NewRequest(http.MethodPost, "/api/gee/generate-short-link", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var payload struct {
				Code    int
				Message string
				Data    string
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if tc.status == http.StatusBadRequest {
				if payload.Code != http.StatusBadRequest || payload.Message != tc.message {
					t.Fatalf("unexpected error response: %s", response.Body.String())
				}
				var count int64
				if err := database.Model(&models.Link{}).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("failed request persisted rows: %d %v", count, err)
				}
			} else if !strings.HasPrefix(payload.Data, "https://override.test/t/") {
				t.Fatalf("unexpected URL: %s", payload.Data)
			}
		})
	}
}

func newLinkPolicyRouter(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	database, engine := testutil.OpenDatabase(t)
	if err := database.AutoMigrate(&models.Link{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousConfig, previousMode := db.DB, config.Config, gin.Mode()
	db.DB = database
	config.Config = &config.Configuration{Database: config.DatabaseConfiguration{Driver: engine}, Data: config.DataConfiguration{BaseURL: "https://short.test", LinkRedirectPageURL: "https://i.mazey.net/pages/redirect/", CommonAPIKeys: []string{"", "valid-key"}}}
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { db.DB, config.Config = previousDB, previousConfig; gin.SetMode(previousMode) })
	app := gin.New()
	registerRoutes(app)
	app.LoadHTMLFiles("../../../assets/data/index.tmpl")
	return database, app
}

func createPolicyLink(t *testing.T, app *gin.Engine, key string, once bool) string {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"ori_link": "https://destination.test/路径?next=https%3A%2F%2Fnested.test%2F%3Fx%3D1&y=2#fragment",
		"base_url": "https://override.test", "one_time": once, "common_api_key": key,
		"direct_redirect": true, "link_key": "forged", "visit_count": 99,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/gee/generate-short-link", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data  string `json:"data"`
		Alias string `json:"tiny_link"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(payload.Data, "https://override.test/t/") || payload.Alias != payload.Data {
		t.Fatalf("override/response: %#v", payload)
	}
	return strings.TrimPrefix(payload.Data, "https://override.test")
}

func waitLinkVisit(t *testing.T, database *gorm.DB, key string, count int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var stored models.Link
		if err := database.Where("link_key = ?", key).First(&stored).Error; err != nil {
			t.Fatal(err)
		}
		if stored.VisitCount == count {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("asynchronous visit was not recorded")
}

func TestLinkCreationAuthorizationPersists(t *testing.T) {
	for _, order := range [][]string{{"", "invalid", "valid-key"}, {"valid-key", "invalid", ""}} {
		t.Run(strings.Join(order, "/"), func(t *testing.T) {
			database, app := newLinkPolicyRouter(t)
			paths := map[bool]string{}
			for _, key := range order {
				path := createPolicyLink(t, app, key, false)
				direct := key == "valid-key"
				if previous := paths[direct]; previous != "" && previous != path {
					t.Fatal("same policy not deduplicated")
				}
				paths[direct] = path
				var stored models.Link
				if err := database.Where("link_key = ?", strings.TrimPrefix(path, "/t/")).First(&stored).Error; err != nil {
					t.Fatal(err)
				}
				if stored.DirectRedirect != direct || stored.VisitCount != 0 || stored.LinkKey == "forged" {
					t.Fatalf("client overrode server fields: %#v", stored)
				}
			}
			if paths[false] == paths[true] {
				t.Fatal("direct and warning policies share a link")
			}
			// Revocation does not change existing policies; neither does an opening header.
			config.Config.Data.CommonAPIKeys = nil
			for direct, path := range paths {
				request := httptest.NewRequest(http.MethodGet, path, nil)
				request.RemoteAddr = "127.0.0.1:12345"
				request.Header.Set("X-Link-Warning-URL", "https://attacker.test/")
				request.Header.Set("X-Common-API-Key", "valid-key")
				response := httptest.NewRecorder()
				app.ServeHTTP(response, request)
				waitLinkVisit(t, database, strings.TrimPrefix(path, "/t/"), 1)
				if response.Code != http.StatusFound || response.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("redirect: %d %#v", response.Code, response.Header())
				}
				var stored models.Link
				database.Where("link_key = ?", strings.TrimPrefix(path, "/t/")).First(&stored)
				location := response.Header().Get("Location")
				if direct {
					// Gin escapes Unicode in Location, so compare decoded URL values.
					got, err := url.Parse(location)
					want, _ := url.Parse(stored.OriginalURL)
					if err != nil || got.Scheme != want.Scheme || got.Host != want.Host || got.Path != want.Path || got.RawQuery != want.RawQuery || got.Fragment != want.Fragment {
						t.Fatalf("direct location: %s", location)
					}
				} else {
					got, err := url.Parse(location)
					if err != nil || got.Scheme != "https" || got.Host != "i.mazey.net" || got.Path != "/pages/redirect/" || len(got.Query()["url"]) != 1 || got.Query().Get("url") != stored.OriginalURL {
						t.Fatalf("warning location: %s", location)
					}
				}
			}
		})
	}
}

func TestOneTimeWarningConsumesAtResolution(t *testing.T) {
	database, app := newLinkPolicyRouter(t)
	path := createPolicyLink(t, app, "", true)
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("X-Link-Warning-URL", "https://attacker.test/")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	waitLinkVisit(t, database, strings.TrimPrefix(path, "/t/"), 1)
	if response.Code != http.StatusFound {
		t.Fatalf("first visit: %d", response.Code)
	}
	response = httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "404 Link Expired") {
		t.Fatalf("second visit: %d %s", response.Code, response.Body.String())
	}
	if repeated := createPolicyLink(t, app, "invalid", true); repeated != path {
		t.Fatal("consumed link was replaced")
	}
}

func TestWarningConfigurationFailsClosed(t *testing.T) {
	database, app := newLinkPolicyRouter(t)
	path := createPolicyLink(t, app, "", false)
	for i, warning := range []string{"", "javascript:alert(1)", "/redirect/"} {
		config.Config.Data.LinkRedirectPageURL = warning
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("X-Link-Warning-URL", "https://must-not-fallback.test/")
		response := httptest.NewRecorder()
		app.ServeHTTP(response, request)
		waitLinkVisit(t, database, strings.TrimPrefix(path, "/t/"), i+1)
		if response.Code != http.StatusServiceUnavailable || response.Header().Get("Location") != "" {
			t.Fatalf("unsafe fallback: %d %#v", response.Code, response.Header())
		}
	}
}

func TestWarningURLIndependentOfCORS(t *testing.T) {
	for _, cors := range []string{"on", "off", ""} {
		t.Run(cors, func(t *testing.T) {
			database, app := newLinkPolicyRouter(t)
			config.Config.Data.EnableCORS = cors
			path := createPolicyLink(t, app, "", false)
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.RemoteAddr = "192.0.2.1:1234"
			request.Header.Add("X-Link-Warning-URL", "https://attacker.test/")
			request.Header.Add("X-Link-Warning-URL", "invalid")
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			waitLinkVisit(t, database, strings.TrimPrefix(path, "/t/"), 1)
			location, err := url.Parse(response.Header().Get("Location"))
			if response.Code != http.StatusFound || err != nil || location.Host != "i.mazey.net" {
				t.Fatalf("warning URL tied to CORS or headers: %d %q %v", response.Code, response.Header().Get("Location"), err)
			}
		})
	}
}

func TestMalformedCreationDoesNotPersist(t *testing.T) {
	database, app := newLinkPolicyRouter(t)
	for _, body := range []string{`{`, `{}`, `{"ori_link":false}`, `{"ori_link":"https://target.test","common_api_key":123}`, `{"ori_link":"https://target.test","common_api_key":true}`, `{"ori_link":"https://target.test","common_api_key":[]}`, `{"ori_link":"https://target.test","common_api_key":{}}`} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/gee/generate-short-link", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid input: %d", response.Code)
		}
	}
	var count int64
	database.Model(&models.Link{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid request created a row")
	}
}

func TestCreationBodyOwnsAuthorization(t *testing.T) {
	database, app := newLinkPolicyRouter(t)
	config.Config.Data.CommonAPIKeys = []string{"first-secret", "second-secret"}
	paths := map[bool]string{}
	for _, tc := range []struct {
		name, body, header string
		direct             bool
	}{
		{"missing", "", "", false},
		{"header only", "", "first-secret", false},
		{"empty", `,"common_api_key":""`, "first-secret", false},
		{"invalid", `,"common_api_key":"invalid-secret"`, "first-secret", false},
		{"valid", `,"common_api_key":"first-secret"`, "", true},
		{"other valid", `,"common_api_key":"second-secret"`, "invalid-secret", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"ori_link":"https://target.test","direct_redirect":true` + tc.body + `}`
			request := httptest.NewRequest(http.MethodPost, "/api/gee/generate-short-link", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Common-API-Key", tc.header)
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "secret") || strings.Contains(response.Body.String(), "common_api_key") {
				t.Fatalf("creation response: %d %s", response.Code, response.Body.String())
			}
			var payload struct{ Data string }
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if previous := paths[tc.direct]; previous != "" && previous != payload.Data {
				t.Fatal("raw credential affected deduplication")
			}
			paths[tc.direct] = payload.Data
			var stored models.Link
			key := strings.TrimPrefix(payload.Data, "https://short.test/t/")
			if err := database.Where("link_key = ?", key).First(&stored).Error; err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(stored)
			if err != nil {
				t.Fatal(err)
			}
			if stored.DirectRedirect != tc.direct || strings.Contains(string(encoded), "secret") {
				t.Fatalf("stored policy or secret: %s", encoded)
			}
		})
	}
	if paths[true] == paths[false] {
		t.Fatal("policies share a link")
	}
}
