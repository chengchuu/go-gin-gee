package main

import (
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"

	"github.com/bitfield/script"
	"github.com/chengchuu/go-gin-gee/internal/pkg/constants"
	"github.com/chengchuu/go-gin-gee/pkg/logger"
)

// Examples:
// go run scripts/batch-git-pull/main.go -path="/Users/Web/web"
// go run scripts/batch-git-pull/main.go -path="C:\Web\web"
// go run scripts/batch-git-pull/main.go -path="C:\Web\web" -projects="placeholder1|placeholder2"
// path required
// projects optional
func main() {
	logger.Println("Git Pull ...")
	placeholder := "unknown"
	// https://gobyexample.com/command-line-flags
	projectPath := flag.String("path", placeholder, "folder of projects")
	assignedProjects := flag.String("projects", ".", "assigned projects")
	runCommands := flag.String("commands", "git pull;", "commands")
	flag.Parse()
	logger.Println("projectPath:", *projectPath)
	logger.Println("assignedProjects:", *assignedProjects)
	logger.Println("runCommands:", *runCommands)
	if *projectPath == placeholder {
		logger.Fatal("path is required")
	}
	projects := []string{
		"placeholder",
	}
	regexStr := "^.+("
	for _, v := range projects {
		regexStr += fmt.Sprintf("%s|", v)
	}
	// Example: ^.+(placeholder|.)$
	regexStr += fmt.Sprintf("%s)\\/\\.git$", *assignedProjects)
	logger.Info("regexStr: %s", regexStr)
	regex := regexp.MustCompile(regexStr)
	if runtime.GOOS == "windows" {
		script.ListFiles(fmt.Sprintf("%s\\*\\.git", *projectPath)).FilterLine(func(s string) string {
			// Build PowerShell command lines; use semicolons and quote paths to handle spaces
			cmdLines := fmt.Sprintf("Write-Output 'Path: %s'; ", s)
			// change directory into the .git folder then go up one level to the repo root
			cmdLines += fmt.Sprintf("cd '%s'; ", s)
			cmdLines += "cd ..; "
			cmdLines += fmt.Sprintf("%s ", *runCommands)
			cmdLines += "Write-Output '-- All Done in PowerShell --'; "
			cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				logger.Println("error:", err)
			}
			logger.Printf("result:\n%s", result)
			return ""
		}).Stdout()
	} else {
		script.ListFiles(fmt.Sprintf("%s/*/.git", *projectPath)).MatchRegexp(regex).FilterLine(func(s string) string {
			cmdLines := constants.ScriptStartMsg
			cmdLines += fmt.Sprintf("echo Path: %s;", s)
			cmdLines += fmt.Sprintf("cd %s;", s)
			cmdLines += `cd ../;`
			cmdLines += *runCommands
			cmdLines += constants.ScriptEndMsg
			cmd := exec.Command("/bin/sh", "-c", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				logger.Println("error:", err)
			}
			logger.Printf("result: %s", result)
			return ""
		}).Stdout()
	}
	logger.Println("Git Pull Done.")
}
