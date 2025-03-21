package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadFile downloads a file from the specified URL and saves it to the destination path
func DownloadFile(client *http.Client, url, destPath string, baseDomain string) error {
	// Check if the destination directory exists and create if necessary
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Send HTTP request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for failed response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed, HTTP code: %d", resp.StatusCode)
	}

	// Write file to disk
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("file writing error: %w", err)
	}

	return nil
}

// extractAccountAndContainer extracts account and container information from the URL
func extractAccountAndContainer(url string, baseDomain string) (string, string) {
	// URL format: https://account.blob.core.windows.net/container/blobname
	parts := strings.Split(url, ".")
	if len(parts) < 2 {
		return "unknown", "unknown"
	}

	account := strings.TrimPrefix(parts[0], "https://")

	// Url'nin sonundaki /container/blobname kısmını bul
	hostAndPath := strings.SplitN(url, "."+baseDomain+"/", 2)
	if len(hostAndPath) < 2 {
		return account, "unknown"
	}

	// container/blobname kısmını container ve blobname olarak ayır
	containerPath := strings.SplitN(hostAndPath[1], "/", 2)
	if len(containerPath) < 1 || containerPath[0] == "" {
		return account, "unknown"
	}

	return account, containerPath[0]
}

// DebugDownloadFile performs the download operation with debug output
func DebugDownloadFile(client *http.Client, url, destPath string, baseDomain string) error {
	account, container := extractAccountAndContainer(url, baseDomain)

	fmt.Printf("[DEBUG] Starting download [%s/%s]: %s -> %s\n", account, container, url, destPath)

	// Check if the destination directory exists and create if necessary
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("[DEBUG] Directory creation error [%s/%s]: %v\n", account, container, err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	out, err := os.Create(destPath)
	if err != nil {
		fmt.Printf("[DEBUG] File creation error [%s/%s]: %v\n", account, container, err)
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	fmt.Printf("[DEBUG] Sending HTTP GET request [%s/%s]: %s\n", account, container, url)

	// Send HTTP request
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("[DEBUG] HTTP request error [%s/%s]: %v\n", account, container, err)
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] HTTP response received [%s/%s]: %d %s\n", account, container, resp.StatusCode, resp.Status)

	// Check for failed response
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[DEBUG] Failed HTTP response [%s/%s]: %d\n", account, container, resp.StatusCode)
		return fmt.Errorf("download failed, HTTP code: %d", resp.StatusCode)
	}

	fmt.Printf("[DEBUG] Content length [%s/%s]: %d bytes\n", account, container, resp.ContentLength)
	fmt.Printf("[DEBUG] Content type [%s/%s]: %s\n", account, container, resp.Header.Get("Content-Type"))

	// Write file to disk
	bytesWritten, err := io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("[DEBUG] File writing error [%s/%s]: %v\n", account, container, err)
		return fmt.Errorf("file writing error: %w", err)
	}

	fmt.Printf("[DEBUG] Download completed [%s/%s]. %d bytes written\n", account, container, bytesWritten)

	return nil
}
