package persistence

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/sites"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

const (
	discordWebhookIDAlias    = "WEBHOOK_ID"
	discordWebhookTokenAlias = "WEBHOOK_TOKEN"
)

var discordWebhookBaseURL = "https://discord.com/api/webhooks"

type Sites struct {
	List map[string]SiteStatus
}

type SiteStatus struct {
	Name string
	Code int
	Link string
}

type ReportData struct {
	Timestamp    string
	HealthyCount int
	FailedCount  int
	TotalCount   int
	HealthySites []SiteStatus
	FailedSites  []SiteStatus
}

type DiscordMessage struct {
	Content string `json:"content"`
}

// Return value
var robotRepository *Sites

func GetRobotRepository() *Sites {
	if robotRepository == nil {
		robotRepository = &Sites{}
	}
	return robotRepository
}

func (r *Sites) getWebSiteStatus() (*[]SiteStatus, *[]SiteStatus, error) {
	// http://c.biancheng.net/view/32.html
	healthySites := []SiteStatus{}
	failSites := []SiteStatus{}
	client := resty.New().
		SetTimeout(5 * time.Second).
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))
	// https://github.com/go-resty/resty/blob/master/redirect.go
	for url, status := range r.List {
		resCode := 0
		resp, err := client.R().
			SetDoNotParseResponse(true).
			Get(url)
		if err != nil {
			logger.Error("error: %v", err)
			resCode = 0
		} else {
			resCode = resp.StatusCode()
			if resp.RawBody() != nil {
				resp.RawBody().Close()
			}
		}
		if status.Code == resCode {
			healthySites = append(healthySites, status)
		} else {
			failSites = append(failSites, SiteStatus{status.Name, resCode, url})
		}
	}
	return &healthySites, &failSites, nil
}

func (r *Sites) ClearCheckResult(WebSites *[]models.WebSite) (*DiscordMessage, error) {
	reportData := ReportData{
		Timestamp:    "",
		HealthyCount: 0,
		FailedCount:  0,
		TotalCount:   0,
		HealthySites: []SiteStatus{},
		FailedSites:  []SiteStatus{},
	}
	logDir := "log"
	ss := r
	ss.List = map[string]SiteStatus{}
	if len(*WebSites) > 0 {
		for _, site := range *WebSites {
			ss.List[site.Link] = SiteStatus{site.Name, site.Code, site.Link}
		}
	} else {
		return nil, errors.New("WebSites is empty")
	}
	healthySites, failSites, err := ss.getWebSiteStatus()
	if err != nil {
		logger.Error("error: %v", err)
	}
	// Prepare Report Data
	reportData.Timestamp = time.Now().Format("2006-01-02 15:04:05")
	reportData.HealthySites = *healthySites
	reportData.HealthyCount = len(*healthySites)
	reportData.FailedSites = *failSites
	reportData.FailedCount = len(*failSites)
	reportData.TotalCount = len(*healthySites) + len(*failSites)
	// Parse template
	tmpl, err := template.New("report").Parse(HTMLTemplate)
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(logDir, "robot.html")
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// Execute template
	if err := tmpl.Execute(file, reportData); err != nil {
		return nil, err
	}

	mdStr := buildHealthCheckMarkdown(ss, healthySites, failSites)

	webhookID, webhookToken, err := getDiscordWebhookConfig()
	if err != nil {
		return nil, err
	}
	webhookURL := buildDiscordWebhookURL(webhookID, webhookToken)
	message := DiscordMessage{
		Content: mdStr,
	}
	if err := sendDiscordWebhook(webhookURL, message); err != nil {
		logger.Error("error: %v", err)
		return nil, err
	}
	return &message, nil
}

func buildHealthCheckMarkdown(ss *Sites, healthySites, failSites *[]SiteStatus) string {
	sucessNames := []string{}
	lo.ForEach(*healthySites, func(site SiteStatus, _ int) {
		sucessNames = append(sucessNames, site.Name)
	})
	// Sort Success Names
	sort.Strings(sucessNames)
	mdStr := "Health Check Result:\n"
	lo.ForEach(sucessNames, func(name string, _ int) {
		mdStr += fmt.Sprintf("**%s OK**\n", name)
	})
	lo.ForEach(*failSites, func(site SiteStatus, _ int) {
		siteLink, _ := lo.FindKeyBy(ss.List, func(k string, v SiteStatus) bool {
			return v.Name == site.Name
		})
		mdStr += fmt.Sprintf(
			"**%s FAIL**\n"+
				"Error Code: %d\n"+
				"Link: [%s](%s)\n",
			site.Name,
			site.Code,
			siteLink,
			siteLink,
		)
	})
	mdStr += fmt.Sprintf("*%s%d*", "Sum: ", len(*healthySites)+len(*failSites))
	return mdStr
}

func getDiscordWebhookConfig() (string, string, error) {
	webhookID := ""
	webhookToken := ""
	conf := config.GetConfig()
	if conf != nil {
		webhookID = conf.Data.WebhookID
		webhookToken = conf.Data.WebhookToken
	}

	if conf != nil {
		sA := GetAlias2dataRepository()
		if webhookID == "" {
			data, err := sA.Get(discordWebhookIDAlias)
			if err != nil {
				logger.Error("error: %v", err)
			} else {
				webhookID = data.Data
			}
		}
		if webhookToken == "" {
			data, err := sA.Get(discordWebhookTokenAlias)
			if err != nil {
				logger.Error("error: %v", err)
			} else {
				webhookToken = data.Data
			}
		}
	}

	if webhookID == "" || webhookToken == "" {
		return "", "", errors.New("discord webhook id or token is empty")
	}
	return webhookID, webhookToken, nil
}

func buildDiscordWebhookURL(webhookID, webhookToken string) string {
	return fmt.Sprintf(
		"%s/%s/%s",
		strings.TrimRight(discordWebhookBaseURL, "/"),
		webhookID,
		webhookToken,
	)
}

func sendDiscordWebhook(webhookURL string, message DiscordMessage) error {
	client := resty.New().
		SetTimeout(5 * time.Second)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(message).
		Post(webhookURL)
	if err != nil {
		return err
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}
