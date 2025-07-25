package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/gocolly/colly/v2"
)

// Examples:
// go run scripts/crawl-web-headings/main.go -allowedDomain="example.com" -firstURL="http://example.com/first-page.html"
func main() {
	allowedDomain := flag.String("allowedDomain", "", "Allowed Domain")
	firstURL := flag.String("firstURL", "", "First URL to visit")
	flag.Parse()
	log.Printf("Allowed Domain: %s", *allowedDomain)
	log.Printf("First URL: %s", *firstURL)

	// Article navigation and related articles
	ignoreTitles := []string{
		"文章导航",
		"相关文章",
		"条评论",
	}
	// Visited URLs
	visitedURLs := make(map[string]bool)

	// Colly
	// Create a new Colly Collector
	c := colly.NewCollector(
		colly.AllowedDomains(*allowedDomain), // 限制爬取的域名
	)

	// Find each `<h2>` tag and print its content
	c.OnHTML("h2", func(e *colly.HTMLElement) {
		thatTitle := e.Text
		// Ignore specific titles
		for _, title := range ignoreTitles {
			// if thatTitle == title {
			// 	return
			// }
			if strings.Contains(thatTitle, title) {
				fmt.Println("Ignore title:", thatTitle)
				return
			}
		}
		// Ignore the title including "条评论"
		// if len(thatTitle) > 6 && strings.Contains(thatTitle, "条评论") {
		// 	fmt.Println("Ignore title:", thatTitle)
		// 	return
		// }
		fmt.Println("Title found:", thatTitle)
	})

	// Handle errors during the request
	c.OnError(func(r *colly.Response, err error) {
		errURL := r.Request.URL.String()
		log.Printf("Error occurred on URL %s: %v", errURL, err)
	})

	// Handle URLs found on the page
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		URL := e.Attr("href")
		// Visit URL found on page
		if visitedURLs[URL] {
			return
		}
		// Handle the URL end wiht .html
		// if len(URL) < 5 || URL[len(URL)-5:] != ".html" {
		// 	fmt.Println("Ignore link:", URL)
		// 	return
		// }
		fmt.Println("Next page found:", URL)
		visitedURLs[URL] = true
		c.Visit(e.Request.AbsoluteURL(URL))
	})

	// Visit the first URL
	err := c.Visit(*firstURL)
	if err != nil {
		log.Fatal(err)
	}
}
