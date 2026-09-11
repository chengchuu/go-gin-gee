package api

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

func Run() error {
	logger.Init()
	// Set the timezone to UTC
	// https://www.zeitverschiebung.net/en/timezone/asia--shanghai
	os.Setenv("TZ", "UTC")
	setConfiguration()
	conf := config.GetConfig()
	return withAccessLog("./log", func(accessLog io.Writer) error {
		// Run before the API starts, after the required access log is open.
		if len(conf.Data.Sites) > 0 {
			controllers.RunCheck()
		} else {
			fmt.Println("No sites found, unnecessary to run check")
		}
		web := router.Setup(accessLog)
		fmt.Println("API Running on port " + conf.Server.Port)
		fmt.Println("==================>")
		return web.Run(":" + conf.Server.Port)
	})
}

// withAccessLog owns the request log for the duration of the server lifecycle.
func withAccessLog(directory string, serve func(io.Writer) error) (result error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("create access log directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(directory, "api.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("open access log: %w", err)
	}
	previousWriter := gin.DefaultWriter
	defer func() {
		gin.DefaultWriter = previousWriter
		if err := file.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close access log: %w", err))
		}
	}()
	return serve(file)
}
