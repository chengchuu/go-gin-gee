package link

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestLinkPersistenceAndRequestBoundaries(t *testing.T) {
	parsed, err := schema.Parse(&Link{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"uk_link_key", "uk_link_dedup_hash"} {
		index := parsed.LookIndex(name)
		if index == nil || index.Class != "UNIQUE" || len(index.Fields) != 1 {
			t.Fatalf("invalid index %s: %#v", name, index)
		}
	}
	if field := parsed.LookUpField("DedupHash"); field.Size != 64 || !field.NotNull {
		t.Fatalf("fingerprint column: %#v", field)
	}
	if field := parsed.LookUpField("DirectRedirect"); !field.NotNull || field.DefaultValue != "false" {
		t.Fatalf("policy column: %#v", field)
	}
	if parsed.LookUpField("CommonAPIKey") != nil {
		t.Fatal("request secret must not be persisted")
	}
	var request CreateRequest
	if err := json.Unmarshal([]byte(`{"ori_link":"https://example.test","base_url":"https://short.test","one_time":true,"common_api_key":"secret","direct_redirect":true,"link_key":"forged","visit_count":99}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.OriginalURL != "https://example.test" || request.BaseURL != "https://short.test" || !request.OneTime || request.CommonAPIKey != "secret" {
		t.Fatalf("request: %#v", request)
	}
	if reflect.TypeOf(request).NumField() != 4 {
		t.Fatal("creation request must contain only caller-controlled fields")
	}
}
