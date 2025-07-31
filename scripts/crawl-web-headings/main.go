package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/chengchuu/gurl"
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
	// Failed URLs
	failedURLs := make(map[string]bool)
	// Ignored URLs
	ignoredURLs := make(map[string]bool)
	crawledCount := 0
	failedCount := 0

	// Colly
	// Create a new Colly Collector
	c := colly.NewCollector(
		colly.AllowedDomains(*allowedDomain), // Limit to the allowed domain
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
				// log.Println("Ignore title:", thatTitle)
				return
			}
		}
		// Ignore the title including "条评论"
		// if len(thatTitle) > 6 && strings.Contains(thatTitle, "条评论") {
		// 	fmt.Println("Ignore title:", thatTitle)
		// 	return
		// }
		log.Println("Title found:", thatTitle)
	})

	// Handle URLs found on the page
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		URL := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(URL)
		baseURL, err := gurl.GetBaseURL(absoluteURL)
		if err != nil {
			log.Println("Error getting base URL:", err)
			return
		}
		// Visit URL found on page
		if visitedURLs[baseURL] {
			return
		}
		if !strings.Contains(URL, *allowedDomain) {
			// fmt.Println("Ignore link:", URL)
			// fmt.Print(">")
			ignoredURLs[baseURL] = true
			return
		}
		// Handle the URL end wiht .html
		// if len(URL) < 5 || URL[len(URL)-5:] != ".html" {
		// 	fmt.Println("Ignore link:", URL)
		// 	return
		// }
		// log.Println("Next page found:", baseURL)
		fmt.Print(".")
		visitedURLs[baseURL] = true
		c.Visit(e.Request.AbsoluteURL(baseURL))
	})

	// c.OnScraped(func(r *colly.Response) {
	// 	fmt.Print(".") // Print a dot for progress
	// })

	// Handle errors during the request
	c.OnError(func(r *colly.Response, err error) {
		errURL := r.Request.URL.String()
		failedURLs[errURL] = true
		log.Printf("Error occurred on URL %s: %v", errURL, err)
	})

	// Visit the first URL
	err := c.Visit(*firstURL)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for all requests to complete
	c.Wait()
	// Count the number of visited URLs
	crawledCount = len(visitedURLs)
	failedCount = len(failedURLs)
	fmt.Printf("\nCrawled %d URLs.\n", crawledCount)
	if failedCount > 0 {
		fmt.Printf("Failed to crawl %d URLs.\n", failedCount)
		for url := range failedURLs {
			fmt.Printf("Failed URL: %s\n", url)
		}
	} else {
		fmt.Println("No failed URLs.")
	}
	log.Println("All URLs have been crawled.")
}
