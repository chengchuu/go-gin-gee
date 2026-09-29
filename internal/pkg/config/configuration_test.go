package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestConfigurationPrecedence(t *testing.T) {
	for _, override := range []bool{false, true} {
		name := "file"
		if override {
			name = "environment"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(`{"Server":{"Port":"4321"},"Database":{"Driver":"sqlite","MaxOpenConns":5},"Data":{"BaseURL":"https://file.test","WEBHOOK_ID":"file-id","WEBHOOK_TOKEN":"file-token","KVAPIKeys":["file-key"],"Sites":[{"Name":"file-site","Url":"https://file.test"}],"SpecialLinks":[{"Key":"reserved","Link":"https://special.test"}]}}`), 0600); err != nil {
				t.Fatal(err)
			}
			prepareConfigurationTest(t, path)
			if override {
				t.Setenv("BASE_URL", "https://env.test")
				t.Setenv("WEBHOOK_ID", "env-id")
				t.Setenv("WEBHOOK_TOKEN", "env-token")
				t.Setenv("ENABLECORS", "on")
				t.Setenv("CONFIG_DATA_SITES", `[{"Name":"env-site","Url":"https://env.test"}]`)
			}
			Setup()
			got := GetConfig()
			if got.Server.Port != "4321" || got.Server.Mode != "release" || got.Database.Driver != "sqlite" || got.Database.MaxOpenConns != 5 {
				t.Fatalf("file/default settings changed: server=%+v database=%+v", got.Server, got.Database)
			}
			wantURL, wantID, wantToken, wantSite := "https://file.test", "file-id", "file-token", "file-site"
			if override {
				wantURL, wantID, wantToken, wantSite = "https://env.test", "env-id", "env-token", "env-site"
				if got.Data.EnableCORS != "on" {
					t.Fatal("environment CORS override was lost")
				}
			}
			if got.Data.BaseURL != wantURL || got.Data.WebhookID != wantID || got.Data.WebhookToken != wantToken {
				t.Fatal("configuration precedence or mapstructure keys changed")
			}
			if len(got.Data.Sites) != 1 || got.Data.Sites[0].Name != wantSite || len(got.Data.KVAPIKeys) != 1 || got.Data.KVAPIKeys[0] != "file-key" || len(got.Data.SpecialLinks) != 1 || got.Data.SpecialLinks[0].Key != "reserved" {
				t.Fatal("configuration collections changed")
			}
		})
	}
}

func TestConfigurationMissingFileDefaults(t *testing.T) {
	prepareConfigurationTest(t, filepath.Join(t.TempDir(), "missing.json"))
	Setup()
	if len(Config.Data.CommonAPIKeys) != 0 {
		t.Fatal("common keys must default to empty")
	}
	if Config.Server.Port != "3000" || Config.Server.Mode != "release" || Config.Database.Driver != "" {
		t.Fatalf("fallback settings changed: %+v", Config)
	}
}

func TestCommonAPIKeysConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"Data":{"CommonAPIKeys":["first","second"],"WebhookAPIKeys":["webhook"],"KVAPIKeys":["kv"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	prepareConfigurationTest(t, path)
	Setup()
	if len(Config.Data.CommonAPIKeys) != 2 || Config.Data.CommonAPIKeys[0] != "first" || Config.Data.CommonAPIKeys[1] != "second" || Config.Data.WebhookAPIKeys[0] != "webhook" || Config.Data.KVAPIKeys[0] != "kv" {
		t.Fatal("independent API key lists were not loaded")
	}
}

func TestWarningConfigurationIndependentOfCORS(t *testing.T) {
	for _, cors := range []string{"on", "off"} {
		t.Run(cors, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			content := `{"Data":{"LinkRedirectPageURL":"https://warning.test/","EnableCORS":"` + cors + `"}}`
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			prepareConfigurationTest(t, path)
			Setup()
			if Config.Data.LinkRedirectPageURL != "https://warning.test/" || Config.Data.EnableCORS != cors {
				t.Fatal("warning configuration changed with CORS")
			}
		})
	}
}

func prepareConfigurationTest(t *testing.T, path string) {
	t.Helper()
	previousArgs, previousFlags, previousPFlags, previousConfig := os.Args, flag.CommandLine, pflag.CommandLine, Config
	os.Args = []string{"configuration-test", "--config-path", path}
	flag.CommandLine = flag.NewFlagSet("configuration-test", flag.ContinueOnError)
	pflag.CommandLine = pflag.NewFlagSet("configuration-test", pflag.ContinueOnError)
	viper.Reset()
	for _, key := range []string{"BASE_URL", "WEBHOOK_ID", "WEBHOOK_TOKEN", "ENABLECORS", "CONFIG_DATA_SITES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CONFIG_TYPE", "json")
	// A changed flag must take precedence over this environment value.
	t.Setenv("CONFIG-PATH", "ignored-by-explicit-flag.json")
	t.Cleanup(func() {
		os.Args, flag.CommandLine, pflag.CommandLine, Config = previousArgs, previousFlags, previousPFlags, previousConfig
		viper.Reset()
	})
}
