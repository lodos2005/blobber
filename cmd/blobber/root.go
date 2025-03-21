package blobber

import (
	"bufio"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// Blob structures for XML parsing
type BlobError struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

type BlobProperties struct {
	CreationTime      string `xml:"Creation-Time"`
	LastModified      string `xml:"Last-Modified"`
	Etag              string `xml:"Etag"`
	ContentLength     int64  `xml:"Content-Length"`
	ContentType       string `xml:"Content-Type"`
	ContentEncoding   string `xml:"Content-Encoding"`
	ContentLanguage   string `xml:"Content-Language"`
	ContentCRC64      string `xml:"Content-CRC64"`
	ContentMD5        string `xml:"Content-MD5"`
	CacheControl      string `xml:"Cache-Control"`
	ContentDisposition string `xml:"Content-Disposition"`
	BlobType          string `xml:"BlobType"`
	AccessTier        string `xml:"AccessTier"`
}

type Blob struct {
	XMLName    xml.Name      `xml:"Blob"`
	Name       string        `xml:"Name"`
	Properties BlobProperties `xml:"Properties"`
}

type Blobs struct {
	XMLName xml.Name `xml:"Blobs"`
	Blob    []Blob   `xml:"Blob"`
}

type EnumerationResults struct {
	XMLName       xml.Name `xml:"EnumerationResults"`
	ServiceEndpoint string   `xml:"ServiceEndpoint,attr"`
	ContainerName string   `xml:"ContainerName,attr"`
	Blobs         Blobs    `xml:"Blobs"`
	NextMarker    string   `xml:"NextMarker"`
}

// Command line flags
var (
	accounts            string
	containers          string
	isDownload          bool
	outputPath          string
	skipSSL             bool
	maxGoroutines       int
	maxParallelDownload int
	debug               bool
	listBlobs           bool
	limit               int
	baseDomain          string
	totalCount          bool
	foundContainers     int // Erişilebilir container sayacı
	foundContainerLock  sync.Mutex // Sayaç için mutex
	
	// Global progress bar
	mainProgressBar     *progressbar.ProgressBar
)

// Global HTTP client
var client *http.Client

// BarPrintf, progressbar'ı bozmadan renkli çıktı yazdırmak için yardımcı fonksiyon
func BarPrintf(bar *progressbar.ProgressBar, c *color.Color, format string, a ...interface{}) {
	coloredText := c.Sprintf(format, a...)
	progressbar.Bprintf(bar, "%s\n", coloredText)
}

// BarPrintln, progressbar'ı bozmadan renkli çıktı yazdırmak için yardımcı fonksiyon
func BarPrintln(bar *progressbar.ProgressBar, c *color.Color, a ...interface{}) {
	args := make([]interface{}, len(a))
	for i, arg := range a {
		switch v := arg.(type) {
		case string:
			args[i] = c.Sprint(v)
		default:
			args[i] = v
		}
	}
	progressbar.Bprintln(bar, args...)
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "blobber",
	Short: "Blobber checks for publicly accessible Azure Blob Storage containers",
	Long: `Blobber is a tool to check if Azure Blob Storage containers are publicly accessible.
It can list and download files from publicly accessible containers.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check for incompatible flags - output sadece list ile birlikte kullanılamaz
		if outputPath != "" && listBlobs {
			red := color.New(color.FgRed)
			fmt.Println(red.Sprintf("Error: --output cannot be used with --list parameter"))
			return
		}

		// If output is specified or download is not requested, set default limit to 99999
		if !isDownload && outputPath != ""  && limit == 10 {
			limit = 99999
		}

		// Initialize HTTP client
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSL},
		}
		client = &http.Client{
			Transport: tr,
			Timeout:   time.Second * 30,
		}

		// Process accounts
		accountList := processInput(accounts)
		if len(accountList) == 0 {
			red := color.New(color.FgRed)
			fmt.Println(red.Sprintf("No accounts provided. Use --accounts parameter."))
			fmt.Println()
			cmd.Help()
			return
		}

		// Process containers
		containerList := processInput(containers)
		if len(containerList) == 0 {
			red := color.New(color.FgRed)
			fmt.Println(red.Sprintf("No containers provided. Use --containers parameter."))
			return
		}

		// Calculate total number of checks to perform
		totalChecks := len(accountList) * len(containerList)
		
		cyan := color.New(color.FgCyan)
		fmt.Println(cyan.Sprintf("Starting scan of %d account(s) × %d container(s) = %d total combinations", 
			len(accountList), len(containerList), totalChecks))

		// Create a main progress bar for overall progress
		mainProgressBar = progressbar.NewOptions(totalChecks,
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionSetDescription(fmt.Sprintf("Checking %d account(s) x %d container(s)", len(accountList), len(containerList))),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[magenta]=[reset]",
				SaucerHead:    "[magenta]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))

		// Create semaphore for limiting goroutines
		sem := make(chan struct{}, maxGoroutines)
		var wg sync.WaitGroup
		var countLock sync.Mutex
		var checkedCount int

		// Check all combinations
		for _, account := range accountList {
			// Check if the domain exists using DNS lookup
			if !domainExists(account + "." + baseDomain) {
				if debug {
					yellow := color.New(color.FgYellow)
					BarPrintf(mainProgressBar, yellow, "[DEBUG] Domain %s.%s does not exist", account, baseDomain)
				}
				
				// Update progress bar for skipped domains
				countLock.Lock()
				mainProgressBar.Add(len(containerList))
				checkedCount += len(containerList)
				countLock.Unlock()
				
				continue
			}

			for _, container := range containerList {
				wg.Add(1)
				sem <- struct{}{} // Acquire semaphore
				go func(acc, cont string) {
					defer wg.Done()
					defer func() { 
						<-sem 
						
						// Update progress bar after checking each container
						countLock.Lock()
						mainProgressBar.Add(1)
						checkedCount++
						countLock.Unlock()
					}() // Release semaphore

					checkContainer(acc, cont)
				}(account, container)
			}
		}

		wg.Wait()
		fmt.Println() // Add a newline after progress bar
		
		// Sonuç mesajını göster
		yellow := color.New(color.FgYellow)
		if foundContainers > 0 {
			fmt.Println(yellow.Sprintf("Scan completed. Found %d publicly accessible container(s).", foundContainers))
		} else {
			fmt.Println(yellow.Sprintf("Scan completed. No publicly accessible containers found. Use --debug for more details."))
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().StringVarP(&accounts, "accounts", "a", "", "Account names (comma-separated) or path to a file containing account names")
	RootCmd.Flags().StringVarP(&containers, "containers", "c", "", "Container names (comma-separated) or path to a file containing container names")
	RootCmd.Flags().BoolVarP(&isDownload, "download", "d", false, "Download files from publicly accessible containers")
	RootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for lists or downloads")
	RootCmd.Flags().BoolVarP(&skipSSL, "skipSSL", "s", true, "Skip SSL verification")
	RootCmd.Flags().IntVarP(&maxGoroutines, "maxGoroutines", "g", 500, "Maximum number of concurrent goroutines")
	RootCmd.Flags().IntVarP(&maxParallelDownload, "maxParallelDownload", "p", 10, "Maximum number of parallel downloads")
	RootCmd.Flags().BoolVarP(&debug, "debug", "v", false, "Enable debug output")
	RootCmd.Flags().BoolVarP(&listBlobs, "list", "l", false, "List all found blob URLs")
	RootCmd.Flags().IntVarP(&limit, "limit", "L", 10, "Limit the number of results")
	RootCmd.Flags().StringVarP(&baseDomain, "baseDomain", "b", "blob.core.windows.net", "Base domain for Azure Blob Storage")
	RootCmd.Flags().BoolVarP(&totalCount, "total", "t", false, "Count total number of blobs traversing all NextMarkers")
}

// processInput processes the input (comma-separated string or file path)
func processInput(input string) []string {
	var result []string

	// Check if input is empty
	if input == "" {
		return result
	}

	// Check if input is a file path
	if _, err := os.Stat(input); err == nil {
		file, err := os.Open(input)
		if err != nil {
			red := color.New(color.FgRed)
			fmt.Println(red.Sprintf("Error opening file: %v", err))
			return result
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				result = append(result, line)
			}
		}

		if err := scanner.Err(); err != nil {
			red := color.New(color.FgRed)
			fmt.Println(red.Sprintf("Error reading file: %v", err))
		}
	} else {
		// Input is a comma-separated string
		for _, item := range strings.Split(input, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
	}

	return result
}

// domainExists checks if a domain exists using DNS lookup
func domainExists(domain string) bool {
	_, err := net.LookupHost(domain)
	return err == nil
}

// checkContainer checks if a container is publicly accessible
func checkContainer(account, container string) {
	baseURL := fmt.Sprintf("https://%s.%s/%s", account, baseDomain, container)
	listURL := fmt.Sprintf("%s?restype=container&comp=list", baseURL)

	if debug {
		cyan := color.New(color.FgCyan)
		BarPrintf(mainProgressBar, cyan, "[DEBUG] Checking: %s", listURL)
	}

	// Send HTTP request
	resp, err := client.Get(listURL)
	if err != nil {
		if debug {
			red := color.New(color.FgRed)
			BarPrintf(mainProgressBar, red, "[DEBUG] Error: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug {
			red := color.New(color.FgRed)
			BarPrintf(mainProgressBar, red, "[DEBUG] Error reading response: %v", err)
		}
		return
	}

	// Parse XML response
	var blobError BlobError
	if err := xml.Unmarshal(body, &blobError); err == nil && blobError.Code != "" {
		switch blobError.Code {
		case "NoAuthenticationInformation":
			if debug {
				yellow := color.New(color.FgYellow)
				BarPrintf(mainProgressBar, yellow, "[DEBUG] %s/%s: No authentication information", account, container)
			}
			return
		case "PublicAccessNotPermitted":
			yellow := color.New(color.FgYellow)
			BarPrintf(mainProgressBar, yellow, "[INFO] %s/%s: Public access not permitted", account, container)
			return
		case "ResourceNotFound":
			// Resource not found, might be accessible
			break
		default:
			if debug {
				yellow := color.New(color.FgYellow)
				BarPrintf(mainProgressBar, yellow, "[DEBUG] %s/%s: %s - %s", account, container, blobError.Code, blobError.Message)
			}
			return
		}
	}

	// Parse as EnumerationResults
	var results EnumerationResults
	if err := xml.Unmarshal(body, &results); err != nil || len(results.Blobs.Blob) == 0 {
		if debug {
			yellow := color.New(color.FgYellow)
			BarPrintf(mainProgressBar, yellow, "[DEBUG] %s/%s: Not accessible or no blobs found", account, container)
		}
		return
	}

	// Container is accessible and has blobs
	if totalCount && results.NextMarker != "" {
		// Başlangıçtaki blob sayısını alıyoruz
		totalBlobCount := len(results.Blobs.Blob)
		nextMarker := results.NextMarker
		
		// Total 10000 ile başlayalım çünkü toplam sayıyı bilmiyoruz
		countBar := progressbar.NewOptions(10000,
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionSetDescription(fmt.Sprintf("Counting blobs in %s/%s", account, container)),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionOnCompletion(func() { fmt.Println() }),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[cyan]=[reset]",
				SaucerHead:    "[cyan]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
		
		// İlk sayfadaki blob sayısını progress bar'a ekleyelim
		countBar.Add(totalBlobCount)
		
		// NextMarker ile tüm blob'ları sayıyoruz
		for nextMarker != "" {
			nextURL := fmt.Sprintf("%s&marker=%s", listURL, url.QueryEscape(nextMarker))
			
			if debug {
				cyan := color.New(color.FgCyan)
				BarPrintf(countBar, cyan, "[DEBUG] Counting blobs with next marker: %s", nextURL)
			}
			
			resp, err := client.Get(nextURL)
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(countBar, red, "[DEBUG] Error fetching next marker for count: %v", err)
				}
				break
			}
			
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(countBar, red, "[DEBUG] Error reading next marker response for count: %v", err)
				}
				break
			}
			
			var nextResults EnumerationResults
			if err := xml.Unmarshal(body, &nextResults); err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(countBar, red, "[DEBUG] Error parsing next marker response for count: %v", err)
				}
				break
			}
			
			blobCount := len(nextResults.Blobs.Blob)
			totalBlobCount += blobCount
			countBar.Add(blobCount)
			
			// Progress bar'ın maksimum değerini gerekirse güncelle
			if totalBlobCount > 9000 {
				countBar.ChangeMax(totalBlobCount + 5000)
			}
			
			nextMarker = nextResults.NextMarker
			
			if debug {
				cyan := color.New(color.FgCyan)
				BarPrintf(countBar, cyan, "[DEBUG] Total blobs counted so far: %d", totalBlobCount)
			}
		}
		
		// Progress bar'ı bozmadan renkli mesajımızı gösterelim
		green := color.New(color.FgGreen)
		BarPrintf(countBar, green, "[FOUND] %s/%s is publicly accessible with %d blobs (total)", account, container, totalBlobCount)
		
		// Erişilebilir container sayacını artır
		foundContainerLock.Lock()
		foundContainers++
		foundContainerLock.Unlock()
	} else if len(results.Blobs.Blob) >= 5000 {
		// Ana ilerleme çubuğu üzerinde renkli mesajımızı gösterelim
		green := color.New(color.FgGreen)
		BarPrintf(mainProgressBar, green, "[FOUND] %s/%s is publicly accessible with more than 5000 blobs", account, container)
		
		// Erişilebilir container sayacını artır
		foundContainerLock.Lock()
		foundContainers++
		foundContainerLock.Unlock()
	} else {
		// Ana ilerleme çubuğu üzerinde renkli mesajımızı gösterelim
		green := color.New(color.FgGreen)
		BarPrintf(mainProgressBar, green, "[FOUND] %s/%s is publicly accessible with %d blobs", account, container, len(results.Blobs.Blob))
		
		// Erişilebilir container sayacını artır
		foundContainerLock.Lock()
		foundContainers++
		foundContainerLock.Unlock()
	}

	// If limit is greater than 5000 and NextMarker is present, get additional blobs
	allBlobs := results.Blobs.Blob
	nextMarker := results.NextMarker

	if limit > 5000 && nextMarker != "" {
		// Progress bar oluştur
		listBar := progressbar.NewOptions(limit,
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionSetDescription(fmt.Sprintf("Fetching blobs from %s/%s", account, container)),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionOnCompletion(func() { fmt.Println() }),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[blue]=[reset]",
				SaucerHead:    "[blue]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
		
		// İlk sayfadaki blob sayısını progress bar'a ekle
		listBar.Add(len(allBlobs))
		
		for nextMarker != "" && len(allBlobs) < limit {
			nextURL := fmt.Sprintf("%s&marker=%s", listURL, url.QueryEscape(nextMarker))
			
			if debug {
				cyan := color.New(color.FgCyan)
				BarPrintf(listBar, cyan, "[DEBUG] Fetching next marker: %s", nextURL)
			}
			
			resp, err := client.Get(nextURL)
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(listBar, red, "[DEBUG] Error fetching next marker: %v", err)
				}
				break
			}
			
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(listBar, red, "[DEBUG] Error reading next marker response: %v", err)
				}
				break
			}
			
			var nextResults EnumerationResults
			if err := xml.Unmarshal(body, &nextResults); err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(listBar, red, "[DEBUG] Error parsing next marker response: %v", err)
				}
				break
			}
			
			blobCount := len(nextResults.Blobs.Blob)
			allBlobs = append(allBlobs, nextResults.Blobs.Blob...)
			listBar.Add(blobCount)
			
			nextMarker = nextResults.NextMarker
			
			if debug {
				cyan := color.New(color.FgCyan)
				BarPrintf(listBar, cyan, "[DEBUG] Total blobs found so far: %d", len(allBlobs))
			}
		}
		
		fmt.Println() // Add a newline after progress bar
	}
	
	// Limit the number of blobs if necessary
	if limit > 0 && len(allBlobs) > limit {
		allBlobs = allBlobs[:limit]
	}

	// Process blobs according to the requested action
	 if isDownload {
		downloadBlobs(account, container, allBlobs)
	} else if outputPath != "" {
		saveBlobList(account, container, allBlobs)
	} else if listBlobs {
		listBlobURLs(account, container, allBlobs)
	} else {
		// Just print the count, already done above
	}
}

// listBlobURLs prints URLs of blobs to console
func listBlobURLs(account, container string, blobs []Blob) {
	// Progress bar oluştur
	listURLBar := progressbar.NewOptions(len(blobs),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription(fmt.Sprintf("Listing %d URLs from %s/%s", len(blobs), account, container)),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[blue]=[reset]",
			SaucerHead:    "[blue]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	
	blue := color.New(color.FgBlue)
	for _, blob := range blobs {
		blobURL := fmt.Sprintf("https://%s.%s/%s/%s", account, baseDomain, container, blob.Name)
		BarPrintf(listURLBar, blue, "%s", blobURL)
		listURLBar.Add(1)
	}
}

// saveBlobList saves the list of blob URLs to a file
func saveBlobList(account, container string, blobs []Blob) {
	outputFile := outputPath
	if outputFile == "" {
		outputFile = fmt.Sprintf("%s_%s_blobs.txt", account, container)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		red := color.New(color.FgRed)
		fmt.Println(red.Sprintf("Error creating output file: %v", err))
		return
	}
	defer file.Close()

	// When writing to a file, assume the limit should be large (99999)
	saveLimit := 99999
	if limit > saveLimit {
		saveLimit = limit
	}

	// Limit the number of blobs if necessary
	blobsToSave := blobs
	if len(blobsToSave) > saveLimit {
		blobsToSave = blobsToSave[:saveLimit]
	}

	// Progress bar oluştur
	saveBar := progressbar.NewOptions(len(blobsToSave),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription(fmt.Sprintf("Saving %d URLs to %s", len(blobsToSave), outputFile)),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for _, blob := range blobsToSave {
		blobURL := fmt.Sprintf("https://%s.%s/%s/%s", account, baseDomain, container, blob.Name)
		fmt.Fprintln(file, blobURL)
		saveBar.Add(1)
	}

	// Progress bar'ı bozmadan renkli mesajımızı gösterelim
	green := color.New(color.FgGreen)
	BarPrintf(saveBar, green, "Saved %d blob URLs to %s", len(blobsToSave), outputFile)
}

// downloadBlobs downloads all blobs from a container
func downloadBlobs(account, container string, blobs []Blob) {
	// Create output directory
	outputDir := filepath.Join(outputPath, account, container)
	

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		red := color.New(color.FgRed)
		fmt.Println(red.Sprintf("Error creating output directory: %v", err))
		return
	}

	// Create progress bar
	bar := progressbar.NewOptions(len(blobs),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading %d files", len(blobs))),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Create semaphore for limiting parallel downloads
	sem := make(chan struct{}, maxParallelDownload)
	var wg sync.WaitGroup

	for i, blob := range blobs {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore
		go func(i int, blob Blob) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			blobURL := fmt.Sprintf("https://%s.%s/%s/%s", account, baseDomain, container, blob.Name)
			
			// Create directories for the blob path if needed
			filename := filepath.Join(outputDir, blob.Name)
			err := os.MkdirAll(filepath.Dir(filename), 0755)
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(bar, red, "[DEBUG] Error creating directory for %s: %v", filename, err)
				}
				bar.Add(1)
				return
			}

			// Download the blob
			resp, err := client.Get(blobURL)
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(bar, red, "[DEBUG] Error downloading %s: %v", blobURL, err)
				}
				bar.Add(1)
				return
			}
			defer resp.Body.Close()

			// Create the file
			out, err := os.Create(filename)
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(bar, red, "[DEBUG] Error creating file %s: %v", filename, err)
				}
				bar.Add(1)
				return
			}
			defer out.Close()

			// Write to file and update progress
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				if debug {
					red := color.New(color.FgRed)
					BarPrintf(bar, red, "[DEBUG] Error writing to file %s: %v", filename, err)
				}
			}

			bar.Add(1)
		}(i, blob)
	}

	wg.Wait()
	
	// Progress bar'ı bozmadan renkli mesajımızı gösterelim
	green := color.New(color.FgGreen)
	BarPrintf(bar, green, "Downloaded %d files to %s", len(blobs), outputDir)
} 