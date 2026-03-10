package config

import (
	"encoding/json"
	"flag"

	modelsS "github.com/chengchuu/go-gin-gee/internal/pkg/models/sites"
	modelsT "github.com/chengchuu/go-gin-gee/internal/pkg/models/tiny"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var Config *Configuration

type Configuration struct {
	Server   ServerConfiguration
	Database DatabaseConfiguration
	Data     DataConfiguration
}

type ServerConfiguration struct {
	Port   string
	Secret string
	Mode   string
}

type DatabaseConfiguration struct {
	Driver       string
	Dbname       string
	Username     string
	Password     string
	Host         string
	Port         string
	MaxLifetime  int
	MaxOpenConns int
	MaxIdleConns int
}

type DataConfiguration struct {
	EnableCORS       string
	WeComRobotCheck  string
	BaseURL          string
	AgentRecordsPath string
	Sites            []modelsS.WebSite
	SpecialLinks     []modelsT.SpecialLink
}

// SetupDB initialize configuration
func Setup() {
	var configuration *Configuration
	var err error

	// Flags
	flag.String("config-path", "data/config.json", "path of configuration")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Environment variables
	// Development: macOS, export WECOM_ROBOT_CHECK="x-x-x"
	viper.AutomaticEnv()
	// Default value
	viper.SetDefault("EnableCORS", "")
	viper.SetDefault("WECOM_ROBOT_CHECK", "")
	viper.SetDefault("BASE_URL", "")
	viper.SetDefault("CONFIG_DATA_SITES", "")
	viper.SetDefault("CONFIG_TYPE", "json")

	// Configuration File
	configPath := viper.GetString("config-path")
	configType := viper.GetString("CONFIG_TYPE")
	viper.SetConfigFile(configPath)
	viper.SetConfigType(configType)
	// Read the configuration file
	if err = viper.ReadInConfig(); err != nil {
		logger.Println("No configuration file found, using default configuration")
		configuration = &Configuration{}
	} else {
		err = viper.Unmarshal(&configuration)
		if err != nil {
			logger.Fatal("Unable to decode into struct, %v", err)
		}
	}

	// Environment variables&Configuration File
	enableCORS := viper.GetString("EnableCORS")
	if enableCORS != "" {
		configuration.Data.EnableCORS = enableCORS
	}
	weComRobotCheck := viper.GetString("WECOM_ROBOT_CHECK")
	if weComRobotCheck != "" {
		configuration.Data.WeComRobotCheck = weComRobotCheck
	}
	configDataSites := viper.GetString("CONFIG_DATA_SITES")
	if configDataSites != "" {
		err = json.Unmarshal([]byte(configDataSites), &configuration.Data.Sites)
		if err != nil {
			logger.Println("error:", err)
		}
	}
	baseURL := viper.GetString("BASE_URL")
	if baseURL != "" {
		configuration.Data.BaseURL = baseURL
	}

	// Fallback
	if configuration.Server.Port == "" {
		configuration.Server.Port = "3000"
	}
	if configuration.Server.Secret == "" {
		configuration.Server.Secret = "wednov23rd2022"
	}
	if configuration.Server.Mode == "" {
		configuration.Server.Mode = "release"
	}

	Config = configuration
}

// GetConfig helps you to get configuration data
func GetConfig() *Configuration {
	if gin.IsDebugging() {
		logger.Info("Config: %+v", Config)
	}
	return Config
}
