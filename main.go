package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	funcs "wget/functions"
)

func main() {
	// Define flags
	background := flag.Bool("B", false, "Download in background")
	output := flag.String("O", "", "Output file name")
	path := flag.String("P", "./", "Output directory path")
	rateLimit := flag.String("rate-limit", "", "Download speed limit")
	inputFile := flag.String("i", "", "File containing multiple download links")
	mirror := flag.Bool("mirror", false, "Mirror a website")
	reject := flag.String("R", "", "Reject specific file types")
	exclude := flag.String("X", "", "Exclude specific directories")

	// Parse command-line arguments
	flag.Parse()

	var url string
	// Check for URL as positional argument
	if len(flag.Args()) > 0 {
		url = flag.Args()[0]
	}

	// Validate that URL or other required arguments are provided
	if url == "" && !*mirror && *inputFile == "" {
		fmt.Println("Please provide a URL, input file, or use the --mirror flag")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Access parsed flag values
	fmt.Println("URL:", url)
	fmt.Println("Background:", *background)
	fmt.Println("Output:", *output)
	fmt.Println("Path:", *path)
	fmt.Println("Rate Limit:", *rateLimit)
	fmt.Println("Input File:", *inputFile)
	fmt.Println("Mirror:", *mirror)
	fmt.Println("Reject:", *reject)
	fmt.Println("Exclude:", *exclude)

	filename := *output
	if *output == "" {
		// Extract filename from URL
		filename = filepath.Base(url)
	}

	if *background {
		var wg sync.WaitGroup
		wg.Add(1)
		go funcs.DownloadFileInBackground(url, filename, path, rateLimit, &wg)
		fmt.Println("Output will be written to \"wget-log\".")
		wg.Wait() // Wait for the background task to complete
		return
	}

	if *inputFile != "" {
		funcs.DownloadFromInput(*inputFile, path, rateLimit)
	}
	// Start time
	startTime := time.Now()
	fmt.Printf("start at %s\n", startTime.Format("2006-01-02 15:04:05"))

	// Download with progress bar
	if err := funcs.DownloadFile(url, filename, path, rateLimit); err != nil {
		fmt.Printf("Error downloading file: %v\n", err)
	}

	// End time
	endTime := time.Now()
	fmt.Printf("finished at %s\n", endTime.Format("2006-01-02 15:04:05"))
}
