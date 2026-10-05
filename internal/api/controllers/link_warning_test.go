package controllers

import (
	"net/url"
	"testing"
)

func TestLinkRedirectPageURLBoundary(t *testing.T) {
	destination := "ftp://example.test/路径?a=1&next=https%3A%2F%2Fnested.test%2F%3Fx%3D1#part"
	for _, tc := range []struct {
		name, configured string
		valid            bool
	}{
		{"https", "https://warning.test/redirect/", true},
		{"http", "http://warning.test/redirect/", true},
		{"existing query", "https://warning.test/redirect/?lang=en&url=old&url=duplicate", true},
		{"missing", "", false},
		{"relative", "/redirect/", false},
		{"script", "javascript:alert(1)", false},
		{"credentials", "https://user:password@warning.test/", false},
		{"fragment", "https://warning.test/#part", false},
		{"bad query", "https://warning.test/?bad=%xx", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := linkWarningURL(destination, tc.configured)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t, result=%q err=%v", tc.valid, got, err)
			}
			if tc.valid {
				parsed, err := url.Parse(got)
				if err != nil || parsed.Host != "warning.test" || len(parsed.Query()["url"]) != 1 || parsed.Query().Get("url") != destination {
					t.Fatalf("destination changed: %q %v", got, err)
				}
				if tc.name == "existing query" && parsed.Query().Get("lang") != "en" {
					t.Fatal("existing query was lost")
				}
			}
		})
	}
}
