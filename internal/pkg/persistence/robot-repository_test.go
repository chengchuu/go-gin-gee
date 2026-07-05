package persistence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
)

func TestBuildHealthCheckMarkdownPreservesContent(t *testing.T) {
	sites := &Sites{
		List: map[string]SiteStatus{
			"https://alpha.example": {Name: "Alpha", Code: 200, Link: "https://alpha.example"},
			"https://beta.example":  {Name: "Beta", Code: 200, Link: "https://beta.example"},
		},
	}
	healthySites := []SiteStatus{
		{Name: "Alpha", Code: 200, Link: "https://alpha.example"},
	}
	failedSites := []SiteStatus{
		{Name: "Beta", Code: 503, Link: "https://beta.example"},
	}

	got := buildHealthCheckMarkdown(sites, &healthySites, &failedSites)
	want := "Health Check Result:\n" +
		"Alpha OK\n" +
		"Beta FAIL\n" +
		"Error Code: 503\n" +
		"Link: <https://beta.example>\n" +
		"All: 2 | Passed: 1 | Failed: 1"

	if got != want {
		t.Fatalf("buildHealthCheckMarkdown() = %q, want %q", got, want)
	}
}

func TestBuildHealthCheckMarkdownLimitsPassedSites(t *testing.T) {
	sites := &Sites{List: map[string]SiteStatus{}}
	healthySites := []SiteStatus{
		{Name: "Delta", Code: 200, Link: "https://delta.example"},
		{Name: "Alpha", Code: 200, Link: "https://alpha.example"},
		{Name: "Charlie", Code: 200, Link: "https://charlie.example"},
		{Name: "Bravo", Code: 200, Link: "https://bravo.example"},
	}
	failedSites := []SiteStatus{}

	got := buildHealthCheckMarkdown(sites, &healthySites, &failedSites)
	want := "Health Check Result:\n" +
		"Alpha OK\n" +
		"Bravo OK\n" +
		"Charlie OK\n" +
		"All: 4 | Passed: 4 | Failed: 0"

	if got != want {
		t.Fatalf("buildHealthCheckMarkdown() = %q, want %q", got, want)
	}
	if strings.Contains(got, "Delta OK") {
		t.Fatalf("buildHealthCheckMarkdown() included more than %d passed sites: %q", displayedPassedSitesLimit, got)
	}
}

func TestGetWebSiteStatusUsesNoRedirectPolicyForExpected301(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect-301", "/redirect-200":
			http.Redirect(w, r, "/ok", http.StatusMovedPermanently)
		case "/ok":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sites := &Sites{
		List: map[string]SiteStatus{
			server.URL + "/redirect-301": {Name: "Redirect Expected", Code: http.StatusMovedPermanently},
			server.URL + "/redirect-200": {Name: "Redirect Followed", Code: http.StatusOK},
		},
	}

	healthySites, failSites, err := sites.getWebSiteStatus()
	if err != nil {
		t.Fatalf("getWebSiteStatus() error = %v", err)
	}
	if len(*failSites) != 0 {
		t.Fatalf("failSites = %+v, want empty", *failSites)
	}
	if len(*healthySites) != 2 {
		t.Fatalf("healthySites length = %d, want 2", len(*healthySites))
	}
}

func TestSendDiscordWebhookSendsExpectedRequestAndAccepts204(t *testing.T) {
	expectedContent := "Health Check Result:\nAlpha OK\nAll: 1 | Passed: 1 | Failed: 0"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/webhooks/test-id/test-token" {
			t.Errorf("path = %s, want /api/webhooks/test-id/test-token", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("content-type = %s, want application/json", got)
		}

		var payload DiscordMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if payload.Content != expectedContent {
			t.Errorf("payload content = %q, want %q", payload.Content, expectedContent)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	oldBaseURL := discordWebhookBaseURL
	discordWebhookBaseURL = server.URL + "/api/webhooks"
	defer func() { discordWebhookBaseURL = oldBaseURL }()

	webhookURL := buildDiscordWebhookURL("test-id", "test-token")
	err := sendDiscordWebhook(webhookURL, DiscordMessage{Content: expectedContent})
	if err != nil {
		t.Fatalf("sendDiscordWebhook() error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestSendDiscordWebhookReturnsBodyForNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid webhook", http.StatusBadRequest)
	}))
	defer server.Close()

	err := sendDiscordWebhook(server.URL, DiscordMessage{Content: "Health Check Result:\n*Sum: 0*"})
	if err == nil {
		t.Fatal("sendDiscordWebhook() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("error = %q, want status 400", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid webhook") {
		t.Fatalf("error = %q, want response body", err.Error())
	}
}

func TestGetDiscordWebhookConfigFromConfig(t *testing.T) {
	oldConfig := config.Config
	config.Config = &config.Configuration{
		Data: config.DataConfiguration{
			WebhookID:    "config-id",
			WebhookToken: "config-token",
		},
	}
	defer func() { config.Config = oldConfig }()

	webhookID, webhookToken, err := getDiscordWebhookConfig()
	if err != nil {
		t.Fatalf("getDiscordWebhookConfig() error = %v", err)
	}
	if webhookID != "config-id" {
		t.Fatalf("webhookID = %q, want config-id", webhookID)
	}
	if webhookToken != "config-token" {
		t.Fatalf("webhookToken = %q, want config-token", webhookToken)
	}
}

func TestGetDiscordWebhookConfigMissing(t *testing.T) {
	oldConfig := config.Config
	config.Config = &config.Configuration{}
	defer func() { config.Config = oldConfig }()

	_, _, err := getDiscordWebhookConfig()
	if err == nil {
		t.Fatal("getDiscordWebhookConfig() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "discord webhook id or token is empty") {
		t.Fatalf("error = %q, want missing webhook config error", err.Error())
	}
}
