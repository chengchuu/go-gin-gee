package controllers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/chengchuu/asiatz"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/mazeyqian/go-gin-gee/internal/pkg/config"
	models "github.com/mazeyqian/go-gin-gee/internal/pkg/models/sites"
	"github.com/mazeyqian/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/mazeyqian/go-gin-gee/pkg/http-err"
)

func CheckSitesHealth(c *gin.Context) {
	per := persistence.GetRobotRepository()
	webSites, err := getWebSites()
	if err != nil {
		log.Println("error:", err)
		http_err.NewError(c, http.StatusInternalServerError, err)
		return
	}
	markdown, err := per.ClearCheckResult(webSites)
	if err != nil {
		log.Println("error:", err)
		http_err.NewError(c, http.StatusInternalServerError, err)
	} else {
		c.JSON(http.StatusOK, gin.H{"data": *markdown})
	}
}

func RunCheck() {
	per := persistence.GetRobotRepository()
	// https://github.com/go-co-op/gocron
	// https://pkg.go.dev/time#Location
	UTC, _ := time.LoadLocation("UTC")
	ss := gocron.NewScheduler(UTC)
	everyDayAtStr, _ := asiatz.ShanghaiToUTC("10:00")
	everyDayAtFn := func() {
		sites, err := getWebSites()
		if err != nil {
			log.Println("error:", err)
		} else {
			per.ClearCheckResult(sites)
		}
	}
	ss.Every(1).Day().At(everyDayAtStr).Do(everyDayAtFn)
	ss.StartAsync()
}

func getWebSites() (*[]models.WebSite, error) {
	conf := config.GetConfig()
	webSites := &conf.Data.Sites
	if len(*webSites) == 0 {
		return nil, errors.New("no sites")
	}
	return webSites, nil
}
