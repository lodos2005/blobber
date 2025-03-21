package azure

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fatih/color"
)

// Scanner scans Azure Blob Storage (simplified)
type Scanner struct {
	client *http.Client
	config Config
}

// NewScanner creates a new Scanner object (simplified)
func NewScanner(config Config) *Scanner {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 30,
	}

	return &Scanner{
		client: client,
		config: config,
	}
}

// CheckAccess checks access to an account and container
func (s *Scanner) CheckAccess(account, container string) AccessResult {
	return s.checkAccess(account, container)
}

// checkAccess checks accessibility for a specific account and container (simplified)
func (s *Scanner) checkAccess(account, container string) AccessResult {
	url := fmt.Sprintf("https://%s.%s/%s?restype=container", account, s.config.BaseDomain, container)
	
	result := AccessResult{
		Account:   account,
		Container: container,
		URL:       url,
	}

	if s.config.Debug {
		fmt.Printf(color.CyanString("[DEBUG] Sending request [%s/%s]: %s\n"), account, container, url)
	}

	resp, err := s.client.Get(url)
	if err != nil {
		if s.config.Debug {
			fmt.Printf(color.RedString("[DEBUG] Error [%s/%s]: %v\n"), account, container, err)
		}
		result.ErrorCode = "RequestFailed"
		return result
	}
	defer resp.Body.Close()

	if s.config.Debug {
		fmt.Printf(color.CyanString("[DEBUG] Response received [%s/%s]: HTTP %d\n"), account, container, resp.StatusCode)
	}

	// Successful response (HTTP 200) is directly accepted as public access
	if resp.StatusCode == http.StatusOK {
		if s.config.Debug {
			fmt.Printf(color.GreenString("[DEBUG] HTTP 200 received [%s/%s], public access available\n"), account, container)
		}
		result.IsPublic = true
		return result
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if s.config.Debug {
			fmt.Printf(color.RedString("[DEBUG] Error reading body [%s/%s]: %v\n"), account, container, err)
		}
		result.ErrorCode = "ReadFailed"
		return result
	}

	// Analyze XML response
	var errorResp ErrorResponse
	err = xml.Unmarshal(body, &errorResp)
	if err != nil {
		// If XML can't be parsed or response is empty, it might be publicly accessible
		if s.config.Debug {
			fmt.Printf(color.GreenString("[DEBUG] XML couldn't be parsed [%s/%s], might be publicly accessible: %v\n"), account, container, err)
			if len(body) > 0 {
				fmt.Printf(color.GreenString("[DEBUG] Body [%s/%s]: %s\n"), account, container, string(body))
			} else {
				fmt.Printf(color.GreenString("[DEBUG] Body [%s/%s]: <empty>\n"), account, container)
			}
		}
		result.IsPublic = true
		return result
	}

	if s.config.Debug {
		fmt.Printf(color.CyanString("[DEBUG] XML error code [%s/%s]: %s\n"), account, container, errorResp.Code)
	}

	if  errorResp.Code == "" {
		// ResourceNotFound error or empty error code might also indicate public access
		if s.config.Debug {
			fmt.Printf(color.GreenString("[DEBUG] ResourceNotFound/Empty code received [%s/%s], public access available\n"), account, container)
		}
		result.IsPublic = true
	} else {
		result.ErrorCode = errorResp.Code
	}

	return result
}

// ListBlobs lists blobs in an account/container combination
func (s *Scanner) ListBlobs(account, container string) []string {
	url := fmt.Sprintf("https://%s.%s/%s?restype=container&comp=list", 
		account, s.config.BaseDomain, container)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if s.config.Debug {
			fmt.Printf("[DEBUG] Error creating request: %v\n", err)
		}
		return []string{}
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		if s.config.Debug {
			fmt.Printf("[DEBUG] Error getting blob list: %v\n", err)
		}
		return []string{}
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		if s.config.Debug {
			fmt.Printf("[DEBUG] Error response code: %d\n", resp.StatusCode)
		}
		return []string{}
	}
	
	// Read and parse the XML response
	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		if s.config.Debug {
			fmt.Printf("[DEBUG] Error reading response body: %v\n", err)
		}
		return []string{}
	}
	
	var results EnumerationResults
	if err := xml.Unmarshal(xmlData, &results); err != nil {
		if s.config.Debug {
			fmt.Printf("[DEBUG] Error parsing XML: %v\n", err)
		}
		return []string{}
	}
	
	// Extract blob URLs
	var blobURLs []string
	for _, blob := range results.BlobList.Blobs {
		blobURL := fmt.Sprintf("https://%s.%s/%s/%s", 
			account, s.config.BaseDomain, container, blob.Name)
		blobURLs = append(blobURLs, blobURL)
	}
	
	return blobURLs
}

// listBlobs lists blobs in a specific account and container (simplest version)
func (s *Scanner) listBlobs(account, container string) []string {
	return []string{} // For this example, we return an empty array
} 