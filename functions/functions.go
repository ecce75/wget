package funcs

import (
	"bufio"
	"fmt"
	"github.com/mxk/go-flowrate/flowrate"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"os"
	ppath "path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var downloadedURLs map[string]bool

func DownloadFile(url, filename string, path, rateLimit *string, isMirroring bool) error {
	if url == "" {
		return nil
	}
	fullPath := filepath.Join(*path, filename)
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check status code
	fmt.Println("sending request, awaiting response... ")
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println("status 200 OK")

	// Create a file
	out, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer out.Close()

	reader := resp.Body
	// Apply rate limiting if specified
	if *rateLimit != "" {
		limit, err := ParseRateLimit(*rateLimit)
		if err != nil {
			return fmt.Errorf("invalid rate limit: %v", err)
		}
		// burst size set to limit for simplicity
		reader = flowrate.NewReader(resp.Body, limit)
	}

	// Content size
	if resp.ContentLength == -1 {
		fmt.Println("content size: unknown")

		// Copy the response body to the file without progress bar
		written, err := io.Copy(out, reader)
		if err != nil {
			return err
		}

		fmt.Printf("Downloaded %d bytes\n", written)
	} else {
		fmt.Printf("content size: %d [~%.2fMB]\n", resp.ContentLength, float64(resp.ContentLength)/1024/1024)

		// Create a progress bar
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			"downloading",
		)

		// Create a multi writer to write to both the file and the progress bar
		multiWriter := io.MultiWriter(out, bar)

		// Copy the response body to the multiWriter
		_, err = io.Copy(multiWriter, reader)
		if err != nil {
			return err
		}
	}

	// Additional processing for mirroring
	if isMirroring {
		if downloadedURLs == nil {
			downloadedURLs = make(map[string]bool)
		}
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "text/css") {
			file, err := os.Open(fullPath)
			if err != nil {
				return err
			}
			defer file.Close()

			urls, err := ExtractURLs(file, url)
			if err != nil {
				return err
			}

			// Download each URL found in the HTML/CSS file
			for _, u := range urls {
				fmt.Println(u)
				if _, exists := downloadedURLs[url]; !exists {
					downloadedURLs[u] = true
					DownloadFile(u, ppath.Base(u), path, rateLimit, true)
				}
			}
		}
	}
	// File path
	fmt.Printf("saving file to: %s\n", fullPath)
	fmt.Printf("Downloaded [%s]\n", url)
	return nil
}

func DownloadFileInBackground(url, filename string, path, rateLimit *string, wg *sync.WaitGroup) {
	defer wg.Done()
	// Open log file
	logFile, err := os.OpenFile("wget-log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer logFile.Close()
	// Redirect output to log file
	os.Stdout = logFile
	os.Stderr = logFile

	// Start time
	startTime := time.Now()
	fmt.Printf("start at %s\n", startTime.Format("2006-01-02 15:04:05"))

	if err := DownloadFile(url, filename, path, rateLimit, false); err != nil {
		fmt.Fprintf(logFile, "Error downloading file: %v\n", err)
	}
}

func DownloadFromInput(inputFile string, path, rateLimit *string) {
	urls, err := ReadURLsFromFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading URLs from file: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			// Assuming the filename is derived from the URL
			filename := filepath.Base(url)
			fmt.Println(url, filename, *path, *rateLimit)
			if err := DownloadFile(url, filename, path, rateLimit, false); err != nil {
				fmt.Printf("Error downloading file from %s: %v\n", url, err)
			} else {
				fmt.Printf("finished %s\n", filename)
			}
		}(url)
	}
	wg.Wait()
	fmt.Printf("Download finished: %v\n", urls)
}

// ParseRateLimit parses a string like "200k" or "2M" into bytes per second.
func ParseRateLimit(limitStr string) (int64, error) {
	var limit int64
	var unit string

	_, err := fmt.Sscanf(limitStr, "%d%s", &limit, &unit)
	if err != nil {
		return 0, err
	}

	unit = strings.ToLower(unit)
	switch unit {
	case "k":
		limit *= 1024
	case "m":
		limit *= 1024 * 1024
	}

	return limit, nil
}

func ReadURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

func ExtractURLs(htmlContent io.Reader, baseURLString string) ([]string, error) {
	var urls []string
	doc, err := html.Parse(htmlContent)
	if err != nil {
		return nil, err
	}
	baseURL, err := url.Parse(baseURLString)
	if err != nil {
		return nil, err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				// Check for URLs in href and src attributes
				if a.Key == "href" || a.Key == "src" {
					addURL(a.Val, baseURL, &urls)
				}

				// Extract URLs from inline CSS in style attributes
				if a.Key == "style" {
					extractURLsFromCSS(a.Val, baseURL, &urls)
				}
			}
		}

		// Extract URLs from <style> tags
		if n.Type == html.ElementNode && n.Data == "style" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					extractURLsFromCSS(c.Data, baseURL, &urls)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return urls, nil
}

func addURL(rawurl string, baseURL *url.URL, urls *[]string) {
	resolvedURL, err := url.Parse(rawurl)
	if err != nil {
		return
	}
	resolvedURL = baseURL.ResolveReference(resolvedURL)
	*urls = append(*urls, resolvedURL.String())
}

func extractURLsFromCSS(css string, baseURL *url.URL, urls *[]string) {
	re := regexp.MustCompile(`url\(\s*(?:'([^']*)'|"([^"]*)"|([^'"\s][^)]*[^'"\s])|([^'"\s]))\s*\)`)
	matches := re.FindAllStringSubmatch(css, -1)
	for _, match := range matches {
		addURL(match[1], baseURL, urls)
	}
}

func GetDomainName(siteURL string) (string, error) {
	parsedURL, err := url.Parse(siteURL)
	if err != nil {
		return "", err
	}
	// Use Hostname() to extract the domain name
	return parsedURL.Hostname(), nil
}
