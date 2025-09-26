package main

import (
	"fmt"
	"os/exec"

	"github.com/chengchuu/go-gin-gee/internal/pkg/constants"
)

func main() {
	fmt.Println("Restart...")
	cmdLines := constants.ScriptStartMsg
	cmdLines += `cd /web/go-gin-gee;`
	cmdLines += `git checkout master;`
	cmdLines += `git pull;`
	cmdLines += `GOOS=linux GOARCH=amd64 go build -o dist/startupjapan-linux-amd64 cmd/startupjapan/main.go;`
	// Another way to restart: cmdLines += `supervisorctl restart api;`
	cmdLines += `systemctl restart supervisord;`
	cmdLines += `supervisorctl status;`
	cmdLines += constants.ScriptEndMsg
	cmd := exec.Command("/bin/sh", "-c", cmdLines)
	result, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("result: %s", result)
}
