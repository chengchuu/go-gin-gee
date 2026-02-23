package api

import (
	"fmt"
	"os"

	"github.com/chengchuu/go-gin-gee/internal/api/controllers"
	"github.com/chengchuu/go-gin-gee/internal/api/router"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
)

func setConfiguration() {
	config.Setup()
	db.SetupDB()
	gin.SetMode(config.GetConfig().Server.Mode)
}

func Run() {
	logger.Init()
	// Set the timezone to UTC
	// https://www.zeitverschiebung.net/en/timezone/asia--shanghai
	os.Setenv("TZ", "UTC")
	setConfiguration()
	conf := config.GetConfig()
	// log.Println("Config:", conf)
	// Run before the API starts
	if len(conf.Data.Sites) > 0 {
		controllers.RunCheck()
	} else {
		fmt.Println("No sites found, unnecessary to run check")
	}
	web := router.Setup()
	fmt.Println("API Running on port " + conf.Server.Port)
	fmt.Println("==================>")
	_ = web.Run(":" + conf.Server.Port)
}
