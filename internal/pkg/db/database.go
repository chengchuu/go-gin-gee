package db

import (
	"fmt"
	"log"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/alias2data"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/tasks"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/tiny"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/users"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var (
	DB  *gorm.DB
	err error
)

type Database struct {
	*gorm.DB
}

// SetupDB opens a database and saves the reference to `Database` struct.
func SetupDB() {
	var db = DB

	configuration := config.GetConfig()

	driver := configuration.Database.Driver
	database := configuration.Database.Dbname
	username := configuration.Database.Username
	password := configuration.Database.Password
	host := configuration.Database.Host
	port := configuration.Database.Port

	if driver == "" {
		log.Println("No database driver specified")
		return
	}

	if driver == "sqlite" { // SQLITE
		db, err = gorm.Open("sqlite3", "./"+database+".db")
		if err != nil {
			fmt.Println("db err: ", err)
		}
	} else if driver == "postgres" { // POSTGRES
		db, err = gorm.Open("postgres", "host="+host+" port="+port+" user="+username+" dbname="+database+"  sslmode=disable password="+password)
		if err != nil {
			fmt.Println("db err: ", err)
		}
	} else if driver == "mysql" { // MYSQL
		db, err = gorm.Open("mysql", username+":"+password+"@tcp("+host+":"+port+")/"+database+"?charset=utf8&parseTime=True&loc=Local")
		// db, err = gorm.Open("mysql", username+":"+password+"@tcp("+host+":"+port+")/"+database+"?charset=utf8&parseTime=True&loc=Asia%2FShanghai")
		if err != nil {
			fmt.Println("db err: ", err)
		}
	}

	// Change this to true if you want to see SQL queries
	db.LogMode(false)
	db.DB().SetMaxIdleConns(configuration.Database.MaxIdleConns)
	db.DB().SetMaxOpenConns(configuration.Database.MaxOpenConns)
	db.DB().SetConnMaxLifetime(time.Duration(configuration.Database.MaxLifetime) * time.Second)
	DB = db
	migration()
}

// Auto migrate project models
func migration() {
	DB.AutoMigrate(&users.User{})
	DB.AutoMigrate(&users.UserRole{})
	DB.AutoMigrate(&tasks.Task{})
	DB.AutoMigrate(&alias2data.Alias2data{})
	DB.AutoMigrate(&tiny.Tiny{})
}

func GetDB() *gorm.DB {
	return DB
}
