// Package testutil provides isolated fixtures for repository tests.
package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenDatabase creates a database owned by one test. Server fixtures connect
// only to loopback and create a random database; they never read app config or
// accept a database name/DSN from the environment. A selected engine must work:
// missing credentials, connectivity, or privileges fail rather than skip.
func OpenDatabase(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	engine := os.Getenv("GEE_TEST_DB_ENGINE")
	if engine == "" {
		engine = "sqlite"
	}
	var dialector gorm.Dialector
	if engine == "sqlite" {
		dialector = sqlite.Open(filepath.Join(t.TempDir(), "fixture.db") + "?_busy_timeout=5000")
	} else {
		password, ok := os.LookupEnv("GEE_TEST_DB_PASSWORD")
		if !ok || password == "" {
			t.Fatal("GEE_TEST_DB_PASSWORD is required for server database tests")
		}
		port, err := strconv.Atoi(os.Getenv("GEE_TEST_DB_PORT"))
		if err != nil || port < 1 || port > 65535 {
			t.Fatal("GEE_TEST_DB_PORT must be a valid loopback port")
		}
		var token [12]byte
		if _, err := rand.Read(token[:]); err != nil {
			t.Fatal(err)
		}
		name := "gee_test_" + hex.EncodeToString(token[:])
		var admin *sql.DB
		var quotedName string
		switch engine {
		case "mysql":
			cfg := mysqldriver.NewConfig()
			cfg.User, cfg.Passwd = "root", password
			cfg.Net, cfg.Addr = "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			cfg.ParseTime = true
			admin, err = sql.Open("mysql", cfg.FormatDSN())
			cfg.DBName = name
			dialector = mysql.Open(cfg.FormatDSN())
			quotedName = "`" + name + "`"
		case "postgres":
			// pgx accepts URI-escaped credentials through its URL parser.
			dsn := postgresURL(port, password, "postgres")
			admin, err = sql.Open("pgx", dsn)
			dialector = postgres.Open(postgresURL(port, password, name))
			quotedName = `"` + name + `"`
		default:
			t.Fatalf("unsupported GEE_TEST_DB_ENGINE %q", engine)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := admin.Exec("CREATE DATABASE " + quotedName); err != nil {
			_ = admin.Close()
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := admin.Exec("DROP DATABASE " + quotedName); err != nil {
				t.Errorf("drop owned test database: %v", err)
			}
			_ = admin.Close()
		})
	}
	database, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	versionQuery := "SELECT version()"
	if engine == "sqlite" {
		versionQuery = "SELECT sqlite_version()"
	}
	var version string
	if err := database.Raw(versionQuery).Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("database engine=%s version=%s", engine, version)
	return database, engine
}

func postgresURL(port int, password, database string) string {
	return fmt.Sprintf("postgres://%s@127.0.0.1:%d/%s?sslmode=disable", url.UserPassword("postgres", password).String(), port, database)
}
