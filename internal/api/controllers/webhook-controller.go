package controllers

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
)

const webhookAPIKeyHeader = "X-Webhook-API-Key"

func SendDiscordMessage(c *gin.Context) {
	conf := config.GetConfig()
	if !isWebhookAPIEnabled(conf) {
		http_err.NewError(c, http.StatusNotFound, errors.New("webhook api is disabled"))
		return
	}
	if !isValidWebhookAPIKey(c.GetHeader(webhookAPIKeyHeader), conf.Data.WebhookAPIKeys) {
		http_err.NewError(c, http.StatusUnauthorized, errors.New("invalid webhook api key"))
		return
	}

	var message persistence.DiscordMessage
	if err := c.ShouldBindJSON(&message); err != nil {
		http_err.NewError(c, http.StatusBadRequest, errors.New("invalid json body"))
		return
	}
	message.Content = strings.TrimSpace(message.Content)
	if message.Content == "" {
		http_err.NewError(c, http.StatusBadRequest, errors.New("content is required"))
		return
	}

	per := persistence.GetRobotRepository()
	if err := per.SendDiscordMessage(message); err != nil {
		if persistence.IsDiscordWebhookConfigMissing(err) {
			http_err.NewError(c, http.StatusServiceUnavailable, errors.New("discord webhook is not configured"))
			return
		}
		logger.Error("send discord message failed: %v", err)
		http_err.NewError(c, http.StatusBadGateway, errors.New("unable to send discord message"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    message,
	})
}

func isWebhookAPIEnabled(conf *config.Configuration) bool {
	return conf != nil && conf.Data.EnableWebhookAPI == "on"
}

func isValidWebhookAPIKey(input string, keys []string) bool {
	if input == "" {
		return false
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(input), []byte(key)) == 1 {
			return true
		}
	}
	return false
}
