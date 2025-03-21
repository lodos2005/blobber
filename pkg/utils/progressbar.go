package utils

import (
	"fmt"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// ProgressBar represents a progress bar for download operations
type ProgressBar struct {
	bar *progressbar.ProgressBar
	mu  sync.Mutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int) *ProgressBar {
	bar := progressbar.NewOptions(
		total,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription("[cyan]Downloading...[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	return &ProgressBar{
		bar: bar,
	}
}

// Increment increments the progress bar by one
func (p *ProgressBar) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bar.Add(1)
}

// Clear clears the progress bar
func (p *ProgressBar) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Println()
	p.bar.Clear()
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bar.Finish()
} 