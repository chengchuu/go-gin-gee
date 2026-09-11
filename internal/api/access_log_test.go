package api

import (
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/chengchuu/go-gin-gee/internal/api/router"
	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

func TestAccessLogAppendAndRequestFormat(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "api.log")
	if err := os.WriteFile(logPath, []byte("previous entry\n"), 0600); err != nil {
		t.Fatal(err)
	}
	previousConfig, previousMode := config.Config, gin.Mode()
	previousWriter, previousErrorWriter := gin.DefaultWriter, gin.DefaultErrorWriter
	t.Cleanup(func() {
		config.Config = previousConfig
		gin.SetMode(previousMode)
		gin.DefaultWriter, gin.DefaultErrorWriter = previousWriter, previousErrorWriter
	})
	config.Config = &config.Configuration{}
	gin.SetMode(gin.TestMode)
	for i := 0; i < 2; i++ {
		var file *os.File
		err := withAccessLog(directory, func(writer io.Writer) error {
			file = writer.(*os.File)
			app := router.Setup(writer)
			request := httptest.NewRequest("GET", "/api/ping", nil)
			request.RemoteAddr = "192.0.2.1:1234"
			request.Header.Set("User-Agent", "access-log-test")
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != 200 || !strings.Contains(response.Body.String(), "pong/v1.0.0/") {
				t.Fatalf("unexpected ping response: %d", response.Code)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if gin.DefaultWriter != previousWriter || gin.DefaultErrorWriter != previousErrorWriter {
			t.Fatal("Gin writers were not preserved")
		}
		if _, err := file.WriteString("after close"); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("log was not closed: %v", err)
		}
	}
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(contents), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "previous entry" {
		t.Fatalf("expected original entry and two appended requests, got %d lines", len(lines))
	}
	format := regexp.MustCompile(`^192\.0\.2\.1 - - \[\d{2}/[A-Za-z]{3}/\d{4}:\d{2}:\d{2}:\d{2} [+-]\d{4}\] "GET /api/ping HTTP/1\.1 200 [^ ]+ " " access-log-test" " "$`)
	for _, line := range lines[1:] {
		if !format.MatchString(line) {
			t.Fatal("request log format changed")
		}
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("existing permissions changed: %v", info.Mode())
	}
}

func TestAccessLogClosesOnInitializationPanic(t *testing.T) {
	previous := gin.DefaultWriter
	var file *os.File
	func() {
		defer func() {
			if recover() != "initialization failed" {
				t.Fatal("initialization panic was not preserved")
			}
		}()
		_ = withAccessLog(t.TempDir(), func(writer io.Writer) error {
			file = writer.(*os.File)
			gin.DefaultWriter = writer
			panic("initialization failed")
		})
	}()
	if gin.DefaultWriter != previous {
		t.Fatal("writer not restored after initialization panic")
	}
	if _, err := file.WriteString("after panic"); !errors.Is(err, os.ErrClosed) {
		t.Fatal("log remained open after initialization panic")
	}
}

func TestAccessLogCreatesMissingFile(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "log")
	if err := withAccessLog(directory, func(writer io.Writer) error {
		_, err := io.WriteString(writer, "new entry\n")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(directory, "api.log"))
	if err != nil || string(contents) != "new entry\n" {
		t.Fatalf("missing log was not created: %v", err)
	}
	info, err := os.Stat(filepath.Join(directory, "api.log"))
	if err != nil || info.Mode().Perm()&0111 != 0 {
		t.Fatal("new log must have non-executable permissions")
	}
}

func TestAccessLogRejectsFilesystemConflicts(t *testing.T) {
	for _, conflict := range []string{"directory", "file"} {
		t.Run(conflict, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "log")
			if conflict == "directory" {
				if err := os.WriteFile(directory, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.MkdirAll(filepath.Join(directory, "api.log"), 0755); err != nil {
				t.Fatal(err)
			}
			previous := gin.DefaultWriter
			err := withAccessLog(directory, func(io.Writer) error {
				t.Fatal("startup continued after log failure")
				return nil
			})
			if err == nil || gin.DefaultWriter != previous {
				t.Fatal("log failure must preserve the writer and return an error")
			}
		})
	}
}

func TestAccessLogPreservesPrimaryAndCloseErrors(t *testing.T) {
	primary := errors.New("server failure")
	for _, closeEarly := range []bool{false, true} {
		previous := gin.DefaultWriter
		var file *os.File
		err := withAccessLog(t.TempDir(), func(writer io.Writer) error {
			file = writer.(*os.File)
			gin.DefaultWriter = writer
			if closeEarly {
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			return primary
		})
		if !errors.Is(err, primary) || gin.DefaultWriter != previous {
			t.Fatal("primary error or previous writer lost")
		}
		if closeEarly && !errors.Is(err, os.ErrClosed) {
			t.Fatal("close failure was not reported")
		}
		if _, err := file.WriteString("after failure"); !errors.Is(err, os.ErrClosed) {
			t.Fatal("log remained open after failure")
		}
	}
}
