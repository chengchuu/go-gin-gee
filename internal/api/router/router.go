package router

import (
	"fmt"
	"io"
	"os"

	"github.com/chengchuu/go-gin-gee/internal/api/controllers"
	"github.com/chengchuu/go-gin-gee/internal/api/middlewares"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup() *gin.Engine {
	app := gin.New()

	// Get Config
	conf := config.GetConfig()
	// Logging to a file.
	if err := os.MkdirAll("./log", 0755); err != nil {
		logger.Println("mkdir err:", err)
	}
	// log/records
	agentRecordsPath := conf.Data.AgentRecordsPath
	if agentRecordsPath != "" {
		if err := os.MkdirAll(agentRecordsPath, 0755); err != nil {
			logger.Println("mkdir err:", err)
		}
	}
	// log/api.log
	f, err := os.Create("./log/api.log")
	if err != nil {
		logger.Println("create err:", err)
	}
	gin.DisableConsoleColor()
	gin.DefaultWriter = io.MultiWriter(f)

	// Middlewares
	app.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - - [%s] \"%s %s %s %d %s \" \" %s\" \" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	app.Use(gin.Recovery())
	switch conf.Data.EnableCORS {
	case "on":
		logger.Info("CORS enabled")
		app.Use(middlewares.CORS())
	case "off":
		logger.Info("CORS disabled")
		app.Use(middlewares.PreflightHandler())
	}
	app.Use(middlewares.LoggerHandler())
	app.NoRoute(middlewares.NoRouteHandler())
	registerRoutes(app)

	return app
}

func registerRoutes(app *gin.Engine) {
	// Routes
	// ================== Docs Routes
	app.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Static - begin
	templatePath := "data/index.tmpl"
	if _, err := os.Stat(templatePath); err != nil {
		if os.IsNotExist(err) {
			logger.Println("No template file found")
		} else {
			logger.Println("Error checking template file:", err)
		}
	} else {
		app.LoadHTMLFiles("data/index.tmpl")
	}
	// Static - end

	// Basic - begin
	app.GET("/", controllers.Index0310)
	app.GET("/api/ping", controllers.Ping)
	app.GET("/api/index", controllers.Index0920)
	// Basic - end

	// Gee - begin
	gee := app.Group("/api/gee")
	{
		gee.GET("/get-data-by-alias", controllers.GetDataByAlias)
		gee.POST("/create-alias2data", controllers.CreateAlias2data)
		gee.GET("/count-alias2data", controllers.CountAlias2data)
		gee.GET("/check", controllers.CheckSitesHealth)
		gee.POST("/webhook-message", controllers.SendDiscordMessage)
		gee.GET("/query-short-link", controllers.GetTiny)
		gee.POST("/generate-short-link", controllers.CreateTiny)
		gee.GET("/get-tag-name", controllers.GetTag)
	}
	// Gee - end

	// Tiny - begin
	app.GET("/t/:key", controllers.RedirectTiny)
	// Tiny - end

	// Server API - begin
	server := app.Group("/server")
	{
		// Agent
		// server.GET("/get", controllers.AgentGet)
		// server.POST("/post", controllers.AgentPost)
		// server.POST("/put", controllers.AgentPost)
		server.POST("/mock", controllers.AgentMock)
		server.GET("/agent/record", controllers.AgentRecord)
	}
	// Server API - end

}
