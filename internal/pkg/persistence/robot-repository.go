package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/sites"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	wxworkbot "github.com/vimsucks/wxwork-bot-go"
)

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
		}
		if status.Code == resCode {
			healthySites = append(healthySites, status)
		} else {
			failSites = append(failSites, SiteStatus{status.Name, resCode, url})
		}
	}
	return &healthySites, &failSites, nil
}

func (r *Sites) ClearCheckResult(WebSites *[]models.WebSite) (*wxworkbot.Markdown, error) {
	sucessNames := []string{}
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

	lo.ForEach(*healthySites, func(site SiteStatus, _ int) {
		sucessNames = append(sucessNames, site.Name)
	})
	// Sort Success Names
	sort.Strings(sucessNames)
	mdStr := "Health Check Result:\n"
	lo.ForEach(sucessNames, func(name string, _ int) {
		mdStr += fmt.Sprintf("<font color=\"info\">%s OK</font>\n", name)
	})
	lo.ForEach(*failSites, func(site SiteStatus, _ int) {
		siteLink, _ := lo.FindKeyBy(ss.List, func(k string, v SiteStatus) bool {
			return v.Name == site.Name
		})
		mdStr += fmt.Sprintf(
			"<font color=\"warning\">%s FAIL</font>\n"+
				"Error Code: %d\n"+
				"Link: [%s](%s)\n",
			site.Name,
			site.Code,
			siteLink,
			siteLink,
		)
	})
	mdStr += fmt.Sprintf("<font color=\"comment\">*%s%d*</font>", "Sum: ", len(*healthySites)+len(*failSites))
	sA := GetAlias2dataRepository()
	data, err := sA.Get("WECOM_ROBOT_CHECK")
	wxworkRobotKey := ""
	if err != nil {
		logger.Error("error: %v", err)
		conf := config.GetConfig()
		wxworkRobotKey = conf.Data.WeComRobotCheck
	} else {
		wxworkRobotKey = data.Data
	}
	logger.Println("Robot wxworkRobotKey:", wxworkRobotKey)
	if wxworkRobotKey == "" {
		return nil, errors.New("wecom robot key is empty")
	}
	// https://github.com/vimsucks/wxwork-bot-go
	bot := wxworkbot.New(wxworkRobotKey)
	markdown := wxworkbot.Markdown{
		Content: mdStr,
	}
	err = bot.Send(markdown)
	if err != nil {
		logger.Error("error: %v", err)
	}
	return &markdown, nil
}
