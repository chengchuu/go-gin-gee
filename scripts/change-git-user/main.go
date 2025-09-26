package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/bitfield/script"
	"github.com/chengchuu/go-gin-gee/internal/pkg/constants"
)

func main() {
	log.Println("Change git user...")
	// Examples:
	// go run scripts/change-git-user/main.go -path="/Users/mazey/Web/Mazey" -username="Mazey" -useremail="mazey@mazey.net"
	// Air Example: "/Volumes/ProjectX/Example"
	// "example@example.net" "Example Na"
	// Pro Mazey: "/Users/mazey/Web/Mazey"
	// "mazey@mazey.net" "Mazey"
	placeholder := "unknown"
	absolutePath := flag.String("path", placeholder, "path of projects")
	userName := flag.String("username", placeholder, "user name")
	userEmail := flag.String("useremail", placeholder, "user email")
	flag.Parse()
	log.Println("absolutePath:", *absolutePath)
	log.Println("userName:", *userName)
	log.Println("userEmail:", *userEmail)
	if *absolutePath == placeholder || *userName == placeholder || *userEmail == placeholder {
		log.Panicln("params is required")
	}
	// https://bitfieldconsulting.com/golang/scripting
	// https://pkg.go.dev/github.com/bitfield/script#ListFiles
	// https://pkg.go.dev/runtime#pkg-constants
	if runtime.GOOS == "windows" {
		script.ListFiles(fmt.Sprintf("%s\\*\\.git", *absolutePath)).FilterLine(func(s string) string {
			cmdLines := constants.ScriptStartMsgInWin + " && "
			cmdLines += fmt.Sprintf("echo Path: %s && ", s)
			cmdLines += fmt.Sprintf("cd %s && ", s)
			cmdLines += fmt.Sprintf(`git config user.name %s && `, *userName)
			cmdLines += fmt.Sprintf(`git config user.email %s && `, *userEmail)
			cmdLines += "echo All done in Windows CMD. && "
			cmdLines += constants.ScriptEndMsgInWin
			// https://stackoverflow.com/questions/13008255/how-to-execute-a-simple-windows-command-in-golang
			cmd := exec.Command("cmd", "/C", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("error:", err)
			}
			log.Printf("result: %s", result)
			return ""
		}).Stdout()
	} else {
		script.ListFiles(fmt.Sprintf("%s/*/.git", *absolutePath)).FilterLine(func(s string) string {
			cmdLines := constants.ScriptStartMsg
			// https://pkg.go.dev/fmt#Sprintf
			cmdLines += fmt.Sprintf("echo Path: %s;", s)
			cmdLines += fmt.Sprintf("cd %s;", s)
			cmdLines += fmt.Sprintf(`git config user.name "%s";`, *userName)
			cmdLines += fmt.Sprintf(`git config user.email "%s";`, *userEmail)
			cmdLines += constants.ScriptEndMsg
			cmd := exec.Command("/bin/sh", "-c", cmdLines)
			result, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("error:", err)
			}
			log.Printf("result: %s", result)
			return ""
		}).Stdout()
	}
}
