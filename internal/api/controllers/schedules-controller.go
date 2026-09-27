package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/chengchuu/asiatz"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/sites"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron/v2"
)

func CheckSitesHealth(c *gin.Context) {
	per := persistence.GetRobotRepository()
	webSites, err := getWebSites()
	if err != nil {
		logger.Warn("check: %s", err)
		http_err.NewError(c, http.StatusInternalServerError, err)
		return
	}
	message, err := per.ClearCheckResult(webSites)
	if err != nil {
		logger.Error("check: %s", err)
		http_err.NewError(c, http.StatusInternalServerError, err)
	} else {
		c.JSON(http.StatusOK, gin.H{"data": message})
	}
}

func RunCheck() {
	per := persistence.GetRobotRepository()
	ss, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		logger.Error("create health-check scheduler: %v", err)
		return
	}
	everyDayAtFn := func() {
		sites, err := getWebSites()
		if err != nil {
			logger.Warn("check: %s", err)
		} else {
			per.ClearCheckResult(sites)
		}
	}
	if _, err := addHealthCheckJob(ss, everyDayAtFn); err != nil {
		_ = ss.Shutdown()
		logger.Error("schedule health check: %v", err)
		return
	}
	ss.Start()
}

func addHealthCheckJob(scheduler gocron.Scheduler, task func()) (gocron.Job, error) {
	everyDayAtStr, err := asiatz.ShanghaiToUTC("10:00")
	if err != nil {
		return nil, err
	}
	at, err := time.Parse("15:04", everyDayAtStr)
	if err != nil {
		return nil, err
	}
	return scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(at.Hour()), uint(at.Minute()), 0))),
		gocron.NewTask(task),
	)
}

func getWebSites() (*[]models.WebSite, error) {
	conf := config.GetConfig()
	webSites := &conf.Data.Sites
	if len(*webSites) == 0 {
		return nil, errors.New("no sites")
	}
	return webSites, nil
}
