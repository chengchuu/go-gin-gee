package db

import (
	"fmt"
	"log"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/kv"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/link"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type Database struct {
	*gorm.DB
}

// SetupDB opens a database and saves the reference to `Database` struct.
func SetupDB() error {
	configuration := config.GetConfig()

	driver := configuration.Database.Driver
	database := configuration.Database.Dbname
	username := configuration.Database.Username
	password := configuration.Database.Password
	host := configuration.Database.Host
	port := configuration.Database.Port

	if driver == "" {
		log.Println("No database driver specified")
		return nil
	}

	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open("./" + database + ".db")
	case "postgres":
		dialector = postgres.Open("host=" + host + " port=" + port + " user=" + username + " dbname=" + database + " sslmode=disable password=" + password)
	case "mysql":
		dialector = mysql.Open(username + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8&parseTime=True&loc=Local")
	default:
		return fmt.Errorf("unsupported database driver")
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	pool, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database connection pool: %w", err)
	}
	pool.SetMaxIdleConns(configuration.Database.MaxIdleConns)
	pool.SetMaxOpenConns(configuration.Database.MaxOpenConns)
	pool.SetConnMaxLifetime(time.Duration(configuration.Database.MaxLifetime) * time.Second)
	if err := db.AutoMigrate(&kv.Entry{}, &kv.Counter{}, &link.Link{}); err != nil {
		_ = pool.Close()
		return fmt.Errorf("migrate database: %w", err)
	}
	DB = db
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
