package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bitfield/script"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
)

// Examples:
// go run scripts/batch-git-pull/main.go -path="/Users/Web/web"
// go run scripts/batch-git-pull/main.go -path="C:\Web\web"
// path required
func main() {
	logger.Println("Git Pull ...")
	placeholder := "unknown"

	// https://gobyexample.com/command-line-flags
	projectPath := flag.String("path", placeholder, "folder of projects")
	runCommands := flag.String("commands", "git pull", "commands")
	flag.Parse()

	logger.Println("projectPath:", *projectPath)
	logger.Println("runCommands:", *runCommands)

	if *projectPath == placeholder {
		logger.Fatal("path is required")
	}

	// helper to check if repo has a remote configured
	hasRemote := func(repoDir string) (string, error) {
		// Use git -C <repoDir> config --get remote.origin.url
		cmd := exec.Command("git", "-C", repoDir, "config", "--get", "remote.origin.url")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}

	if runtime.GOOS == "windows" {
		// Windows: use PowerShell; iterate .git folders under projectPath
		script.ListFiles(fmt.Sprintf(`%s\*\.git`, *projectPath)).FilterLine(func(s string) string {
			// compute repo dir (parent of .git)
			repoDir := filepath.Dir(s)
			logger.Info("found repo: %s", repoDir)

			remoteURL, err := hasRemote(repoDir)
			if err != nil || remoteURL == "" {
				logger.Info("no remote found for %s\n-- Skipping --", repoDir)
				return ""
			}
			logger.Info("remote for repo: %s", remoteURL)

			// Build PowerShell command lines; quote path to handle spaces
			cmdLines := fmt.Sprintf("Write-Output 'Path: %s'; ", repoDir)
			cmdLines += fmt.Sprintf("Set-Location -LiteralPath '%s'; ", repoDir)
			cmdLines += fmt.Sprintf("%s; ", *runCommands)
			cmdLines += "Write-Output '-- All Done in PowerShell --'; "

			cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				logger.Error("error running commands in %s: %v", repoDir, err)
			}
			logger.Info("result:\n%s", result)
			return ""
		}).Stdout()
	} else {
		// Unix-like: iterate .git folders under projectPath
		script.ListFiles(fmt.Sprintf("%s/*/.git", *projectPath)).FilterLine(func(s string) string {
			repoDir := filepath.Dir(s)
			logger.Info("found repo: %s", repoDir)

			remoteURL, err := hasRemote(repoDir)
			if err != nil || remoteURL == "" {
				logger.Info("no remote found for %s\n-- Skipping --", repoDir)
				return ""
			}
			logger.Info("remote for repo: %s", remoteURL)

			cmdLines := fmt.Sprintf("echo Path: %s;", repoDir)
			// quote repoDir to handle spaces
			cmdLines += fmt.Sprintf("cd '%s';", repoDir)
			cmdLines += fmt.Sprintf("%s;", *runCommands)
			cmdLines += "echo '-- All Done in Shell --';"

			cmd := exec.Command("/bin/sh", "-c", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				logger.Error("error running commands in %s: %v", repoDir, err)
			}
			logger.Info("result:\n%s", result)
			return ""
		}).Stdout()
	}

	logger.Println("Git Pull Done.")
}
