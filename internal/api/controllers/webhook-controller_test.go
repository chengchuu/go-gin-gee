package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/api/auth"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

func TestSendDiscordMessageDisabled(t *testing.T) {
	withTestConfig(&config.Configuration{
		Data: config.DataConfiguration{
			EnableWebhookAPI: "off",
		},
	}, func() {
		w := performSendDiscordMessageRequest("", `{"content":"hello"}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestSendDiscordMessageRequiresValidAPIKey(t *testing.T) {
	withTestConfig(&config.Configuration{
		Data: config.DataConfiguration{
			EnableWebhookAPI: "on",
			WebhookAPIKeys:   []string{"gee_webhook_test"},
		},
	}, func() {
		missingKey := performSendDiscordMessageRequest("", `{"content":"hello"}`)
		if missingKey.Code != http.StatusUnauthorized {
			t.Fatalf("missing key status = %d, want %d", missingKey.Code, http.StatusUnauthorized)
		}

		invalidKey := performSendDiscordMessageRequest("wrong", `{"content":"hello"}`)
		if invalidKey.Code != http.StatusUnauthorized {
			t.Fatalf("invalid key status = %d, want %d", invalidKey.Code, http.StatusUnauthorized)
		}
	})
}

func TestSendDiscordMessageValidatesBody(t *testing.T) {
	withTestConfig(&config.Configuration{
		Data: config.DataConfiguration{
			EnableWebhookAPI: "on",
			WebhookAPIKeys:   []string{"gee_webhook_test"},
		},
	}, func() {
		w := performSendDiscordMessageRequest("gee_webhook_test", `{"content":"   "}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestSendDiscordMessageDoesNotAuthorizeEmptyConfiguredKeys(t *testing.T) {
	if auth.ValidAPIKey("", []string{""}) {
		t.Fatal("empty input and empty configured key should not authorize")
	}
	if auth.ValidAPIKey("anything", []string{""}) {
		t.Fatal("empty configured key should not authorize any input")
	}
}

func TestSendDiscordMessageReturnsServiceUnavailableWhenWebhookMissing(t *testing.T) {
	withTestConfig(&config.Configuration{
		Data: config.DataConfiguration{
			EnableWebhookAPI: "on",
			WebhookAPIKeys:   []string{"gee_webhook_test"},
		},
	}, func() {
		w := performSendDiscordMessageRequest("gee_webhook_test", `{"content":"hello"}`)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
		}
	})
}

func performSendDiscordMessageRequest(apiKey, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/gee/webhook-message", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set(webhookAPIKeyHeader, apiKey)
	}
	c.Request = req
	SendDiscordMessage(c)
	return w
}

func withTestConfig(conf *config.Configuration, fn func()) {
	oldConfig := config.Config
	config.Config = conf
	defer func() { config.Config = oldConfig }()
	fn()
}
