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
// go run scripts/batch-git-pull/main.go -path="/Users/mazey/Web/Mazey"
// go run scripts/batch-git-pull/main.go -path="/Users/mazey/Web/Rabbit" -projects="placeholder"
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
	regex := regexp.MustCompile(regexStr)
	if runtime.GOOS == "windows" {
		script.ListFiles(fmt.Sprintf("%s\\*\\.git", *projectPath)).FilterLine(func(s string) string {
			cmdLines := constants.ScriptStartMsgInWin + " && "
			cmdLines += fmt.Sprintf("echo Path: %s && ", s)
			// https://stackoverflow.com/questions/607670/windows-shell-command-to-get-the-full-path-to-the-current-directory
			cmdLines += fmt.Sprintf("cd %s && ", s)
			cmdLines += `cd ../ && `
			cmdLines += `git pull && `
			cmdLines += "echo All done in Windows CMD. && "
			cmdLines += constants.ScriptEndMsgInWin
			cmd := exec.Command("cmd", "/C", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				logger.Println("error:", err)
			}
			logger.Printf("result: %s", result)
			return ""
		}).Stdout()
	} else {
		script.ListFiles(fmt.Sprintf("%s/*/.git", *projectPath)).MatchRegexp(regex).FilterLine(func(s string) string {
			cmdLines := constants.ScriptStartMsg
			cmdLines += fmt.Sprintf("echo Path: %s;", s)
			cmdLines += fmt.Sprintf("cd %s;", s)
			// Control the branch: cmdLines += `git checkout master;`
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
