package http_err

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponseDataPreservesNaturalJSONTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		data interface{}
		want interface{}
	}{
		{name: "no result", data: nil, want: nil},
		{name: "empty collection", data: []string{}, want: []interface{}{}},
		{name: "scalar", data: true, want: true},
		{name: "object", data: map[string]string{}, want: map[string]interface{}{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			Success(context, http.StatusOK, test.data)
			var payload map[string]interface{}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload) != 3 || payload["code"] != float64(0) || payload["message"] != "success" {
				t.Fatalf("unexpected envelope: %#v", payload)
			}
			if !reflect.DeepEqual(payload["data"], test.want) {
				t.Fatalf("data = %#v (%T), want %#v (%T)", payload["data"], payload["data"], test.want, test.want)
			}
		})
	}
}

func TestFailureUsesNullData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	Failure(context, http.StatusBadRequest, CodeInvalidRequest, "invalid request")

	var payload map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 3 || payload["data"] != nil || payload["code"] == float64(0) {
		t.Fatalf("unexpected failure envelope: %#v", payload)
	}
}
