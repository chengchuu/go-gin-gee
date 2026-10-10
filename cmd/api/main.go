package main

import (
	_ "github.com/chengchuu/go-gin-gee/docs"
	"github.com/chengchuu/go-gin-gee/internal/api"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
)

// @Golang API
// @version 1.0
// @description API in Golang with Gin Framework

// @contact.name Cheng
// @contact.url https://github.com/chengchuu
// @contact.email mazeyqian@gmail.com

// @license.name MIT
// @license.url https://github.com/chengchuu/go-gin-gee/blob/main/LICENSE

// @BasePath /api

func main() {
	if err := api.Run(); err != nil {
		logger.Fatal("API stopped: %v", err)
	}
}
