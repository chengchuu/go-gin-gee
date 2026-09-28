package link

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm/schema"
)

func TestLinkKeyMapping(t *testing.T) {
	model := Link{LinkKey: "abc"}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["link_key"] != "abc" {
		t.Fatalf("JSON = %s", encoded)
	}
	if _, exists := fields["tiny_key"]; exists {
		t.Fatal("old JSON key exposed")
	}
	var decoded Link
	if err := json.Unmarshal([]byte(`{"link_key":"json-key"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.LinkKey != "json-key" {
		t.Fatalf("decoded = %#v", decoded)
	}
	var form Link
	if err := binding.MapFormWithTag(&form, map[string][]string{"link_key": {"form-key"}, "tiny_key": {"old-key"}}, "form"); err != nil {
		t.Fatal(err)
	}
	if form.LinkKey != "form-key" {
		t.Fatalf("form = %#v", form)
	}
	if _, exists := reflect.TypeOf(model).FieldByName("TinyKey"); exists {
		t.Fatal("obsolete Go field exists")
	}
	parsed, err := schema.Parse(&model, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := parsed.LookUpField("LinkKey")
	if field.DBName != "link_key" || field.Size != 32 || !field.NotNull {
		t.Fatalf("field = %#v", field)
	}
	index := parsed.LookIndex("idx_link_key")
	if index == nil || len(index.Fields) != 1 || index.Fields[0].DBName != "link_key" {
		t.Fatalf("index = %#v", index)
	}
}
