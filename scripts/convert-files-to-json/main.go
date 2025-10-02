package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitfield/script"
	"github.com/chengchuu/go-gin-gee/internal/pkg/constants"
)

type GamecenterH5 struct {
	Bundleurl   string `json:"bundleurl"`
	Degradeurl  string `json:"degradeurl"`
	Viewurl     string `json:"viewurl"`
	Manifesturl string `json:"manifesturl"`
}

// Examples:
// go run scripts/transfer-files-to-json/main.go -path="./dist" -projects="placeholder" -baseurl="//example.com/static/" -outfile="./out.json"
func main() {
	log.Println(constants.StartMsgStr)
	projectPath := flag.String("path", "./dist", "folder of projects")
	assignedProjects := flag.String("projects", "placeholder", "assigned projects")
	baseUrl := flag.String("baseurl", "//example.com/static/", "domain and path")
	outFile := flag.String("outfile", "./out.json", "out file")
	flag.Parse()
	log.Println("projectPath:", *projectPath)
	log.Println("assignedProjects:", *assignedProjects)
	log.Println("baseUrl:", *baseUrl)
	log.Println("outFile:", *outFile)
	gh := &GamecenterH5{}
	domain := *baseUrl
	projects := []string{
		"placeholder",
	}
	regexStr := "^.+("
	for _, v := range projects {
		regexStr += fmt.Sprintf("%s|", v)
	}
	regexStr += fmt.Sprintf("%s).+$", *assignedProjects)
	log.Println("regexStr:", regexStr)
	regex := regexp.MustCompile(regexStr)
	script.ListFiles(*projectPath).MatchRegexp(regex).FilterLine(func(s string) string {
		fileName := filepath.Base(s)
		link := domain + fileName
		if strings.Contains(link, ".template.html") {
			gh.Viewurl = link
		}
		if strings.Contains(link, ".degrade.html") {
			gh.Degradeurl = link
		}
		if strings.Contains(link, ".client.") {
			gh.Manifesturl = link
		}
		if strings.Contains(link, ".server.") {
			gh.Bundleurl = link
		}
		return link
	}).Stdout()
	log.Println("gh:", *gh)
	file, _ := json.Marshal(*gh)
	_ = ioutil.WriteFile(*outFile, file, 0644)
	log.Println(constants.EndMsgStr)
}
