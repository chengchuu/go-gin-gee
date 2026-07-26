package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/kv"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

const kvTestAPIKey = "gee_kv_test"

func TestGetKV(t *testing.T) {
	app, database := newKVTestApp(t)
	seedKVEntry(t, database, "site.public", "Gee Service", "public")
	seedKVEntry(t, database, "site.private", "secret", "private")
	seedKVEntry(t, database, "site.invalid-visibility", "must stay hidden", "unexpected")

	tests := []struct {
		name     string
		path     string
		body     string
		apiKey   string
		status   int
		code     int
		dataNil  bool
		wantData map[string]interface{}
	}{
		{name: "public key", path: "/api/gee/kv/get", body: `{"key":"site.public"}`, status: http.StatusOK, code: 0, wantData: map[string]interface{}{"key": "site.public", "value": "Gee Service", "visibility": "public"}},
		{name: "private key authorized", path: "/api/gee/kv/get", body: `{"key":"site.private"}`, apiKey: kvTestAPIKey, status: http.StatusOK, code: 0, wantData: map[string]interface{}{"key": "site.private", "value": "secret", "visibility": "private"}},
		{name: "private key concealed", path: "/api/gee/kv/get", body: `{"key":"site.private"}`, status: http.StatusNotFound, code: 40401, dataNil: true},
		{name: "unknown visibility concealed", path: "/api/gee/kv/get", body: `{"key":"site.invalid-visibility"}`, status: http.StatusNotFound, code: 40401, dataNil: true},
		{name: "unknown key", path: "/api/gee/kv/get", body: `{"key":"missing"}`, status: http.StatusNotFound, code: 40401, dataNil: true},
		{name: "missing key", path: "/api/gee/kv/get", body: `{}`, status: http.StatusBadRequest, code: 40001, dataNil: true},
		{name: "malformed json", path: "/api/gee/kv/get", body: `{`, status: http.StatusBadRequest, code: 40001, dataNil: true},
		{name: "invalid key", path: "/api/gee/kv/get", body: `{"key":"Site Title"}`, status: http.StatusBadRequest, code: 40002, dataNil: true},
		{name: "query key ignored", path: "/api/gee/kv/get?key=site.public", body: `{}`, status: http.StatusBadRequest, code: 40001, dataNil: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performKVRequest(app, test.path, test.body, test.apiKey)
			payload := assertKVEnvelope(t, response, test.status, test.code)
			if test.dataNil && payload["data"] != nil {
				t.Fatalf("data = %#v, want nil", payload["data"])
			}
			if test.wantData != nil {
				data, ok := payload["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("data = %#v, want object", payload["data"])
				}
				for key, want := range test.wantData {
					if data[key] != want {
						t.Errorf("data[%q] = %#v, want %#v", key, data[key], want)
					}
				}
			}
		})
	}
}

func TestSetKV(t *testing.T) {
	app, _ := newKVTestApp(t)

	created := performKVRequest(app, "/api/gee/kv/set", `{"key":"site.title","value":"Gee Service"}`, kvTestAPIKey)
	createdPayload := assertKVEnvelope(t, created, http.StatusCreated, 0)
	createdData := createdPayload["data"].(map[string]interface{})
	if createdData["created"] != true || createdData["visibility"] != "private" || createdData["content_type"] != "text/plain" {
		t.Fatalf("unexpected create data: %#v", createdData)
	}

	updated := performKVRequest(app, "/api/gee/kv/set", `{"key":"site.title","value":"Updated Service","content_type":"text/markdown","visibility":"public"}`, kvTestAPIKey)
	updatedPayload := assertKVEnvelope(t, updated, http.StatusOK, 0)
	updatedData := updatedPayload["data"].(map[string]interface{})
	if updatedData["created"] != false || updatedData["value"] != "Updated Service" || updatedData["visibility"] != "public" {
		t.Fatalf("unexpected update data: %#v", updatedData)
	}

	tests := []struct {
		name   string
		body   string
		apiKey string
		status int
		code   int
	}{
		{name: "invalid visibility", body: `{"key":"site.x","visibility":"shared"}`, apiKey: kvTestAPIKey, status: http.StatusBadRequest, code: 40003},
		{name: "missing key", body: `{"value":"x"}`, apiKey: kvTestAPIKey, status: http.StatusBadRequest, code: 40001},
		{name: "oversized key", body: fmt.Sprintf(`{"key":"%s"}`, strings.Repeat("a", 129)), apiKey: kvTestAPIKey, status: http.StatusBadRequest, code: 40002},
		{name: "missing api key", body: `{"key":"site.x"}`, status: http.StatusUnauthorized, code: 40101},
		{name: "invalid api key", body: `{"key":"site.x"}`, apiKey: "wrong", status: http.StatusForbidden, code: 40301},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performKVRequest(app, "/api/gee/kv/set", test.body, test.apiKey)
			payload := assertKVEnvelope(t, response, test.status, test.code)
			if payload["data"] != nil {
				t.Fatalf("data = %#v, want nil", payload["data"])
			}
		})
	}
}

func TestIncrementKV(t *testing.T) {
	app, _ := newKVTestApp(t)

	assertIncrement := func(body string, want int64, delta int64) {
		t.Helper()
		response := performKVRequest(app, "/api/gee/kv/increment", body, kvTestAPIKey)
		payload := assertKVEnvelope(t, response, http.StatusOK, 0)
		data := payload["data"].(map[string]interface{})
		if data["value"] != float64(want) || data["delta"] != float64(delta) {
			t.Fatalf("increment data = %#v, want value=%d delta=%d", data, want, delta)
		}
	}

	assertIncrement(`{"key":"page.views"}`, 1, 1)
	assertIncrement(`{"key":"page.views","delta":0}`, 1, 0)
	assertIncrement(`{"key":"page.views","delta":5}`, 6, 5)
	assertIncrement(`{"key":"page.views","delta":-2}`, 4, -2)

	missing := performKVRequest(app, "/api/gee/kv/increment", `{"key":"another"}`, "")
	assertKVEnvelope(t, missing, http.StatusUnauthorized, 40101)
	invalid := performKVRequest(app, "/api/gee/kv/increment", `{"key":"another"}`, "wrong")
	assertKVEnvelope(t, invalid, http.StatusForbidden, 40301)
	missingKey := performKVRequest(app, "/api/gee/kv/increment", `{}`, kvTestAPIKey)
	assertKVEnvelope(t, missingKey, http.StatusBadRequest, 40001)
	invalidKey := performKVRequest(app, "/api/gee/kv/increment", `{"key":"Page Views"}`, kvTestAPIKey)
	assertKVEnvelope(t, invalidKey, http.StatusBadRequest, 40002)

	set := performKVRequest(app, "/api/gee/kv/set", `{"key":"not.counter","value":"12"}`, kvTestAPIKey)
	assertKVEnvelope(t, set, http.StatusCreated, 0)
	incompatible := performKVRequest(app, "/api/gee/kv/increment", `{"key":"not.counter"}`, kvTestAPIKey)
	payload := assertKVEnvelope(t, incompatible, http.StatusConflict, 40901)
	if payload["data"] != nil {
		t.Fatalf("failure data = %#v, want nil", payload["data"])
	}
}

func TestIncrementKVConcurrentRequestsDoNotLoseUpdates(t *testing.T) {
	app, _ := newKVTestApp(t)
	const requests = 40

	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, requests)
	valuesChannel := make(chan int, requests)
	for index := 0; index < requests; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			response := performKVRequest(app, "/api/gee/kv/increment", `{"key":"concurrent.views"}`, kvTestAPIKey)
			if response.Code != http.StatusOK {
				errorsChannel <- fmt.Errorf("status = %d, body = %s", response.Code, response.Body.String())
				return
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				errorsChannel <- fmt.Errorf("decode response: %w", err)
				return
			}
			data, ok := payload["data"].(map[string]interface{})
			if !ok {
				errorsChannel <- fmt.Errorf("data = %#v, want object", payload["data"])
				return
			}
			valuesChannel <- int(data["value"].(float64))
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	close(valuesChannel)
	for err := range errorsChannel {
		t.Error(err)
	}
	seenValues := make(map[int]bool, requests)
	for value := range valuesChannel {
		seenValues[value] = true
	}
	for value := 1; value <= requests; value++ {
		if !seenValues[value] {
			t.Errorf("concurrent responses did not include value %d; got %#v", value, seenValues)
		}
	}

	response := performKVRequest(app, "/api/gee/kv/increment", `{"key":"concurrent.views","delta":0}`, kvTestAPIKey)
	payload := assertKVEnvelope(t, response, http.StatusOK, 0)
	data := payload["data"].(map[string]interface{})
	if data["value"] != float64(requests) {
		t.Fatalf("value = %#v, want %d", data["value"], requests)
	}
}

func TestIncrementKVConcurrentDistinctCounters(t *testing.T) {
	app, _ := newKVTestApp(t)
	const requests = 40

	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, requests)
	for index := 0; index < requests; index++ {
		waitGroup.Add(1)
		go func(counter int) {
			defer waitGroup.Done()
			body := fmt.Sprintf(`{"key":"concurrent.counter-%d"}`, counter)
			response := performKVRequest(app, "/api/gee/kv/increment", body, kvTestAPIKey)
			if response.Code != http.StatusOK {
				errorsChannel <- fmt.Errorf("counter %d: status = %d, body = %s", counter, response.Code, response.Body.String())
			}
		}(index)
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Error(err)
	}
}

func TestSetKVConcurrentRequestsPreserveUpsertSemantics(t *testing.T) {
	app, database := newKVTestApp(t)
	const requests = 20

	var waitGroup sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, requests)
	for index := 0; index < requests; index++ {
		waitGroup.Add(1)
		go func(value int) {
			defer waitGroup.Done()
			body := fmt.Sprintf(`{"key":"concurrent.setting","value":"%d"}`, value)
			responses <- performKVRequest(app, "/api/gee/kv/set", body, kvTestAPIKey)
		}(index)
	}
	waitGroup.Wait()
	close(responses)

	createdCount := 0
	for response := range responses {
		if response.Code != http.StatusOK && response.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		data := payload["data"].(map[string]interface{})
		if data["created"] == true {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created responses = %d, want exactly 1", createdCount)
	}

	var entryCount int
	if err := database.Model(&kv.Entry{}).Where(map[string]interface{}{"key": "concurrent.setting"}).Count(&entryCount).Error; err != nil {
		t.Fatal(err)
	}
	if entryCount != 1 {
		t.Fatalf("stored entries = %d, want 1", entryCount)
	}
}

func TestSetAndIncrementCannotClaimTheSameKey(t *testing.T) {
	app, database := newKVTestApp(t)
	const attempts = 20

	for attempt := 0; attempt < attempts; attempt++ {
		key := fmt.Sprintf("type-race.%d", attempt)
		start := make(chan struct{})
		responses := make(chan *httptest.ResponseRecorder, 2)
		var waitGroup sync.WaitGroup
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			<-start
			body := fmt.Sprintf(`{"key":"%s","value":"text"}`, key)
			responses <- performKVRequest(app, "/api/gee/kv/set", body, kvTestAPIKey)
		}()
		go func() {
			defer waitGroup.Done()
			<-start
			body := fmt.Sprintf(`{"key":"%s"}`, key)
			responses <- performKVRequest(app, "/api/gee/kv/increment", body, kvTestAPIKey)
		}()
		close(start)
		waitGroup.Wait()
		close(responses)

		successCount := 0
		conflictCount := 0
		for response := range responses {
			switch response.Code {
			case http.StatusOK, http.StatusCreated:
				successCount++
			case http.StatusConflict:
				conflictCount++
			default:
				t.Fatalf("key %q returned status %d: %s", key, response.Code, response.Body.String())
			}
		}
		if successCount != 1 || conflictCount != 1 {
			t.Fatalf("key %q responses: success=%d conflict=%d, want 1 each", key, successCount, conflictCount)
		}

		var entryCount int
		var counterCount int
		if err := database.Model(&kv.Entry{}).Where(map[string]interface{}{"key": key}).Count(&entryCount).Error; err != nil {
			t.Fatal(err)
		}
		if err := database.Model(&kv.Counter{}).Where(map[string]interface{}{"key": key}).Count(&counterCount).Error; err != nil {
			t.Fatal(err)
		}
		if entryCount+counterCount != 1 {
			t.Fatalf("key %q stored entry=%d counter=%d, want exactly one type", key, entryCount, counterCount)
		}
	}
}

func TestKVDoesNotExposeInternalDatabaseErrors(t *testing.T) {
	app, database := newKVTestApp(t)
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	response := performKVRequest(app, "/api/gee/kv/get", `{"key":"site.title"}`, "")
	payload := assertKVEnvelope(t, response, http.StatusInternalServerError, 50001)
	if payload["message"] != "internal server error" || payload["data"] != nil {
		t.Fatalf("unexpected internal-error response: %#v", payload)
	}
	if strings.Contains(strings.ToLower(response.Body.String()), "database") || strings.Contains(strings.ToLower(response.Body.String()), "sql") {
		t.Fatalf("response exposes internal details: %s", response.Body.String())
	}
}

func newKVTestApp(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	databaseName := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	database, err := gorm.Open("sqlite3", "file:"+databaseName+"?mode=memory&cache=shared&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	database.LogMode(false)
	database.DB().SetMaxOpenConns(8)
	if err = database.AutoMigrate(&kv.Entry{}, &kv.Counter{}).Error; err != nil {
		t.Fatal(err)
	}

	oldDB := db.DB
	oldConfig := config.Config
	db.DB = database
	config.Config = &config.Configuration{
		Database: config.DatabaseConfiguration{Driver: "sqlite"},
		Data:     config.DataConfiguration{KVAPIKeys: []string{kvTestAPIKey}},
	}
	t.Cleanup(func() {
		db.DB = oldDB
		config.Config = oldConfig
		_ = database.Close()
	})

	app := gin.New()
	app.POST("/api/gee/kv/get", GetKV)
	app.POST("/api/gee/kv/set", SetKV)
	app.POST("/api/gee/kv/increment", IncrementKV)
	return app, database
}

func seedKVEntry(t *testing.T, database *gorm.DB, key, value, visibility string) {
	t.Helper()
	entry := kv.Entry{Key: key, Value: value, ContentType: "text/plain", Visibility: visibility}
	if err := database.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}
}

func performKVRequest(app *gin.Engine, path, body, apiKey string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set(kvAPIKeyHeader, apiKey)
	}
	app.ServeHTTP(response, request)
	return response
}

func assertKVEnvelope(t *testing.T, response *httptest.ResponseRecorder, wantStatus, wantCode int) map[string]interface{} {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatus, response.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON response: %v; body = %s", err, response.Body.String())
	}
	if len(payload) != 3 {
		t.Fatalf("top-level fields = %#v, want exactly code, message, data", payload)
	}
	for _, field := range []string{"code", "message", "data"} {
		if _, ok := payload[field]; !ok {
			t.Fatalf("missing top-level field %q in %#v", field, payload)
		}
	}
	if _, ok := payload["alias"]; ok {
		t.Fatal("response contains obsolete alias field")
	}
	if _, ok := payload["count"]; ok {
		t.Fatal("response contains obsolete count field")
	}
	if payload["code"] != float64(wantCode) {
		t.Fatalf("code = %#v, want %d", payload["code"], wantCode)
	}
	if wantCode == 0 && payload["message"] != "success" {
		t.Fatalf("message = %#v, want success", payload["message"])
	}
	if wantCode != 0 && payload["data"] != nil {
		t.Fatalf("failure data = %#v, want nil", payload["data"])
	}
	return payload
}
