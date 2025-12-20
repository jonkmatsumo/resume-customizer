package crawling

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractLinks extracts all same-domain links from HTML content
func ExtractLinks(htmlContent string, baseURL string) ([]string, error) {
	// Parse base URL to get domain
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, &LinkExtractionError{
			Message: "failed to parse base URL",
			Cause:   err,
		}
	}

	// Validate base URL has required fields (similar to ingestion pattern)
	if base.Scheme == "" || base.Host == "" {
		return nil, &LinkExtractionError{
			Message: fmt.Sprintf("invalid base URL: %s (must have scheme and host)", baseURL),
			Cause:   nil,
		}
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, &LinkExtractionError{
			Message: "failed to parse HTML",
			Cause:   err,
		}
	}

	// Extract all links from homepage, nav, footer
	linkSet := make(map[string]bool)
	links := make([]string, 0)

	// Extract from all <a> tags
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Parse the link URL (could be relative or absolute)
		linkURL, err := url.Parse(href)
		if err != nil {
			// Skip malformed URLs
			return
		}

		// Resolve relative URLs
		absoluteURL := base.ResolveReference(linkURL)

		// Filter same-domain links only
		if absoluteURL.Host != base.Host {
			return
		}

		// Normalize URL (remove fragment, sort query params would be nice but not critical)
		absoluteURL.Fragment = ""
		urlString := absoluteURL.String()

		// Remove trailing slash for consistency (optional, but helps with deduplication)
		urlString = strings.TrimSuffix(urlString, "/")

		// Add to set if not already seen
		if !linkSet[urlString] {
			linkSet[urlString] = true
			links = append(links, urlString)
		}
	})

	return links, nil
}
