package controllers

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/api/auth"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/kv"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
)

const kvAPIKeyHeader = "X-API-Key"

var kvKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)

type GetKVRequest struct {
	Key string `json:"key" binding:"required"`
}

type SetKVRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	ContentType string `json:"content_type" default:"text/plain"`
	Visibility  string `json:"visibility" default:"private" enums:"public,private"`
}

type IncrementKVRequest struct {
	Key   string `json:"key" binding:"required"`
	Delta *int64 `json:"delta"`
}

type KVGetData struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ContentType string    `json:"content_type"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type KVSetData struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ContentType string `json:"content_type"`
	Visibility  string `json:"visibility"`
	Created     bool   `json:"created"`
}

type KVIncrementData struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
	Delta int64  `json:"delta"`
}

// GetKV godoc
// @Summary Get a key-value entry
// @Description Public entries can be read without an API key. Private entries require X-API-Key.
// @Tags key-value
// @Accept json
// @Produce json
// @Param X-API-Key header string false "Required only for private entries"
// @Param request body GetKVRequest true "Key lookup"
// @Success 200 {object} http_err.APIResponse{data=KVGetData}
// @Failure 400 {object} http_err.APIResponse
// @Failure 404 {object} http_err.APIResponse
// @Failure 500 {object} http_err.APIResponse
// @Router /gee/kv/get [post]
func GetKV(c *gin.Context) {
	var request GetKVRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidRequest, "invalid request")
		return
	}
	request.Key = strings.TrimSpace(request.Key)
	if !validKVKey(request.Key) {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidKey, "invalid key")
		return
	}

	entry, err := persistence.GetKVRepository().Get(request.Key)
	if errors.Is(err, persistence.ErrKVNotFound) {
		http_err.Failure(c, http.StatusNotFound, http_err.CodeKeyNotFound, "key not found")
		return
	}
	if err != nil {
		logger.Error("key-value get failed")
		http_err.Failure(c, http.StatusInternalServerError, http_err.CodeInternalServerError, "internal server error")
		return
	}
	if entry.Visibility != "public" && !validKVAPIKey(c) {
		http_err.Failure(c, http.StatusNotFound, http_err.CodeKeyNotFound, "key not found")
		return
	}

	http_err.Success(c, http.StatusOK, KVGetData{
		Key:         entry.Key,
		Value:       entry.Value,
		ContentType: entry.ContentType,
		Visibility:  entry.Visibility,
		CreatedAt:   entry.CreatedAt,
		UpdatedAt:   entry.UpdatedAt,
	})
}

// SetKV godoc
// @Summary Create or replace a key-value entry
// @Description Requires a valid X-API-Key and uses upsert semantics.
// @Tags key-value
// @Accept json
// @Produce json
// @Param X-API-Key header string true "Key-value API key"
// @Param request body SetKVRequest true "Entry value and metadata"
// @Success 200 {object} http_err.APIResponse{data=KVSetData}
// @Success 201 {object} http_err.APIResponse{data=KVSetData}
// @Failure 400 {object} http_err.APIResponse
// @Failure 401 {object} http_err.APIResponse
// @Failure 403 {object} http_err.APIResponse
// @Failure 409 {object} http_err.APIResponse
// @Failure 500 {object} http_err.APIResponse
// @Router /gee/kv/set [post]
func SetKV(c *gin.Context) {
	if !authorizeKVWrite(c) {
		return
	}

	var request SetKVRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidRequest, "invalid request")
		return
	}
	request.Key = strings.TrimSpace(request.Key)
	if !validKVKey(request.Key) {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidKey, "invalid key")
		return
	}
	request.ContentType = strings.TrimSpace(request.ContentType)
	if request.ContentType == "" {
		request.ContentType = "text/plain"
	}
	request.Visibility = strings.TrimSpace(request.Visibility)
	if request.Visibility == "" {
		request.Visibility = "private"
	}
	if len(request.ContentType) > 64 || (request.Visibility != "public" && request.Visibility != "private") {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidValue, "invalid value")
		return
	}

	entry := &kv.Entry{
		Key:         request.Key,
		Value:       request.Value,
		ContentType: request.ContentType,
		Visibility:  request.Visibility,
	}
	created, err := persistence.GetKVRepository().Set(entry)
	if errors.Is(err, persistence.ErrKVIncompatible) {
		http_err.Failure(c, http.StatusConflict, http_err.CodeIncompatibleValueType, "incompatible value type")
		return
	}
	if err != nil {
		logger.Error("key-value set failed")
		http_err.Failure(c, http.StatusInternalServerError, http_err.CodeInternalServerError, "internal server error")
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	http_err.Success(c, status, KVSetData{
		Key:         entry.Key,
		Value:       entry.Value,
		ContentType: entry.ContentType,
		Visibility:  entry.Visibility,
		Created:     created,
	})
}

// IncrementKV godoc
// @Summary Atomically increment a counter
// @Description Requires a valid X-API-Key. Missing counters start at zero; omitted delta defaults to one.
// @Tags key-value
// @Accept json
// @Produce json
// @Param X-API-Key header string true "Key-value API key"
// @Param request body IncrementKVRequest true "Counter key and delta"
// @Success 200 {object} http_err.APIResponse{data=KVIncrementData}
// @Failure 400 {object} http_err.APIResponse
// @Failure 401 {object} http_err.APIResponse
// @Failure 403 {object} http_err.APIResponse
// @Failure 409 {object} http_err.APIResponse
// @Failure 500 {object} http_err.APIResponse
// @Router /gee/kv/increment [post]
func IncrementKV(c *gin.Context) {
	if !authorizeKVWrite(c) {
		return
	}

	var request IncrementKVRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidRequest, "invalid request")
		return
	}
	request.Key = strings.TrimSpace(request.Key)
	if !validKVKey(request.Key) {
		http_err.Failure(c, http.StatusBadRequest, http_err.CodeInvalidKey, "invalid key")
		return
	}
	delta := int64(1)
	if request.Delta != nil {
		delta = *request.Delta
	}

	value, err := persistence.GetKVRepository().Increment(request.Key, delta)
	if errors.Is(err, persistence.ErrKVIncompatible) {
		http_err.Failure(c, http.StatusConflict, http_err.CodeIncompatibleValueType, "incompatible value type")
		return
	}
	if err != nil {
		logger.Error("key-value increment failed")
		http_err.Failure(c, http.StatusInternalServerError, http_err.CodeInternalServerError, "internal server error")
		return
	}
	http_err.Success(c, http.StatusOK, KVIncrementData{Key: request.Key, Value: value, Delta: delta})
}

func validKVKey(key string) bool { return kvKeyPattern.MatchString(key) }

func validKVAPIKey(c *gin.Context) bool {
	conf := config.GetConfig()
	return conf != nil && auth.ValidAPIKey(c.GetHeader(kvAPIKeyHeader), conf.Data.KVAPIKeys)
}

func authorizeKVWrite(c *gin.Context) bool {
	input := c.GetHeader(kvAPIKeyHeader)
	if input == "" {
		http_err.Failure(c, http.StatusUnauthorized, http_err.CodeAPIKeyRequired, "API key required")
		return false
	}
	if !validKVAPIKey(c) {
		http_err.Failure(c, http.StatusForbidden, http_err.CodeAccessDenied, "access denied")
		return false
	}
	return true
}
