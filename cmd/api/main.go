package main

import (
	_ "github.com/chengchuu/go-gin-gee/docs"
	"github.com/chengchuu/go-gin-gee/internal/api"
)

// @Golang API
// @version 1.0
// @description API in Golang with Gin Framework

// @contact.name Cheng
// @contact.url https://github.com/mazeyqian
// @contact.email mazeyqian@gmail.com

// @license.name MIT
// @license.url https://github.com/chengchuu/go-gin-gee/blob/main/LICENSE

// @BasePath /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
	api.Run()
}
