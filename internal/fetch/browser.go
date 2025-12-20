// Package fetch - browser.go provides headless browser rendering for SPA sites.
package fetch

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// MinContentLength is the minimum extracted text length to consider HTTP fetch successful.
// If content is shorter, we should fall back to browser rendering.
const MinContentLength = 500

// ShouldUseBrowser returns true if the extracted text is too short,
// indicating the page is likely a JavaScript-rendered SPA.
func ShouldUseBrowser(extractedText string) bool {
	return len(strings.TrimSpace(extractedText)) < MinContentLength
}

// WithBrowser renders a page in a headless browser and returns the rendered HTML.
// This is useful for JavaScript-heavy pages that don't render content on initial load.
// Requires Chrome/Chromium to be installed on the system.
func WithBrowser(ctx context.Context, url string, timeout time.Duration, verbose bool) (string, error) {
	if verbose {
		log.Printf("[BROWSER] Starting headless browser for: %s", url)
	}

	// Create browser context with timeout
	allocCtx, cancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, timeout)
	defer cancel()

	var html string

	// Navigate, wait for page to be ready, then extract HTML
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		// Wait for the page to load - use a combination of strategies
		chromedp.WaitReady("body"),
		// Additional wait for JavaScript to render content
		chromedp.Sleep(3*time.Second),
		// Try to dismiss common cookie banners
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Click common "Accept" buttons - don't fail if not found
			_ = chromedp.Click(`button[id*="accept"], button[class*="accept"], button:contains("OK"), button:contains("Accept")`, chromedp.NodeVisible).Do(ctx)
			return nil
		}),
		chromedp.Sleep(1*time.Second),
		// Extract the full HTML
		chromedp.OuterHTML("html", &html),
	)

	if err != nil {
		return "", fmt.Errorf("browser rendering failed: %w", err)
	}

	if verbose {
		log.Printf("[BROWSER] Rendered HTML: %d bytes", len(html))
	}

	return html, nil
}

// BrowserSimple is a simplified version that uses default timeout.
func BrowserSimple(ctx context.Context, url string, verbose bool) (string, error) {
	return WithBrowser(ctx, url, 30*time.Second, verbose)
}
