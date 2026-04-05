package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/chengchuu/go-gin-gee/pkg/logger"
	"github.com/chengchuu/gurl"
	"github.com/gocolly/colly/v2"
)

// Examples:
// go run scripts/crawl-web-naruto/main.go -allowedDomain="example.com" -firstURL="http://example.com/first-page.html"
func main() {
	allowedDomain := flag.String("allowedDomain", "", "Allowed Domain")
	firstURL := flag.String("firstURL", "", "First URL to visit")
	extraURLs := flag.String("extraURLs", "", "Extra URLs to visit, separated by commas")
	isFoundURLs := flag.Bool("isFoundURLs", true, "Whether to find all URLs on the page")
	isBlogDev := flag.Bool("isBlogDev", false, "Whether to develop")

	flag.Parse()
	logger.Printf("Allowed Domain: %s", *allowedDomain)
	logger.Printf("First URL: %s", *firstURL)
	logger.Printf("Extra URLs: %s", *extraURLs)
	logger.Printf("Find All URLs: %v", *isFoundURLs)

	// Article navigation and related articles
	ignoreTitles := []string{
		"文章导航",
		"相关文章",
		"条评论",
	}
	// Visited URLs
	visitedURLs := make(map[string]bool)
	// Failed URLs
	failedURLs := make(map[string]string)
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
			if strings.Contains(thatTitle, title) {
				return
			}
		}
		// logger.Println("Title found:", thatTitle)
	})

	// Find <b>Warning</b> and Panic
	c.OnHTML("b", func(e *colly.HTMLElement) {
		thatText := e.Text
		if strings.Contains(strings.ToLower(thatText), "warning") {
			logger.Fatal("Warning found on page %s: %s", e.Request.URL.String(), thatText)
		}
	})

	// Handle URLs found on the page
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if !*isFoundURLs {
			return
		}
		// Get the absolute URL
		URL := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(URL)
		baseURL, err := gurl.GetBaseURL(absoluteURL)
		if err != nil {
			logger.Println("Error getting base URL:", err)
			return
		}
		// Visit URL found on page
		if visitedURLs[baseURL] {
			return
		}
		if !strings.Contains(URL, *allowedDomain) {
			ignoredURLs[baseURL] = true
			return
		}
		// blog - begin
		if *isBlogDev {
			if strings.Contains(baseURL, "https") {
				baseURL, err = gurl.SetProtocol(baseURL, "http")
				if err != nil {
					logger.Println("Error setting protocol:", err)
					return
				}
			}
			fileType, err := gurl.GetURLFileType(baseURL)
			if err != nil {
				logger.Println("Error getting file type:", err)
				return
			}
			if fileType == "html" {
				baseURL = strings.TrimSuffix(baseURL, ".html")
			}
		}
		// blog - end
		// logger.Println("Next page found:", baseURL)
		// logger.Println("Running ...")
		fmt.Print(".")
		visitedURLs[baseURL] = true
		c.Visit(e.Request.AbsoluteURL(baseURL))
	})

	// Handle errors during the request
	c.OnError(func(r *colly.Response, err error) {
		errURL := r.Request.URL.String()
		failedURLs[errURL] = err.Error()
		logger.Printf("Error occurred on URL %s: %v", errURL, err)
	})

	// Visit the first URL
	if *firstURL != "" {
		visitedURLs[*firstURL] = true
		c.Visit(*firstURL)
		// err := c.Visit(*firstURL)
		// if err != nil {
		// 	log.Fatal(err)
		// }
	}

	// Visit extra URLs
	if *extraURLs != "" {
		urls := strings.Split(*extraURLs, ",")
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			fmt.Print(".")
			visitedURLs[url] = true
			c.Visit(url)
			// err := c.Visit(url)
			// if err != nil {
			// 	logger.Printf("Error visiting extra URL %s: %v", url, err)
			// }
		}
	}

	if *firstURL == "" && *extraURLs == "" {
		logger.Println("No URLs to visit. Please provide at least one URL using -firstURL or -extraURLs flag.")
		return
	}

	// Wait for all requests to complete
	c.Wait()
	// Count the number of visited URLs
	crawledCount = len(visitedURLs)
	failedCount = len(failedURLs)
	fmt.Printf("\nCrawled %d URLs.\n", crawledCount)
	fmt.Print("--------------------------------\n")
	if failedCount > 0 {
		fmt.Printf("Failed to crawl %d URLs.\n", failedCount)
		fmt.Print("--------------------------------\n")
		for url := range failedURLs {
			fmt.Printf("Failed URL: %s\n", url)
			fmt.Printf("Error: %s\n", failedURLs[url])
			fmt.Print("--------------------------------\n")
			// Keep It
			// fmt.Printf("Failed URL: %s.html\n", url)
		}
	} else {
		fmt.Println("No failed URLs.")
	}
	logger.Println("All URLs have been crawled.")
}
