package ingestion

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jonathan/resume-customizer/internal/fetch"
)

var (
	// ErrInvalidURL is returned when URL is malformed
	ErrInvalidURL = fmt.Errorf("invalid URL")
	// ErrHTTPRequestFailed is returned when HTTP request fails
	ErrHTTPRequestFailed = fmt.Errorf("HTTP request failed")
	// ErrContentExtractionFailed is returned when content extraction fails
	ErrContentExtractionFailed = fmt.Errorf("content extraction failed")
)

// IngestFromURL fetches content from a URL, extracts text, cleans it, and returns cleaned text with metadata.
// It uses platform detection to apply platform-specific selectors for better content extraction.
// If apiKey is provided, it uses LLM to extract structured job requirements.
// If useBrowser is true, falls back to headless browser for SPA sites with insufficient content.
// If verbose is true, logs detailed information about the extraction process.
func IngestFromURL(ctx context.Context, urlStr string, apiKey string, useBrowser bool, verbose bool) (string, *Metadata, error) {
	// Detect platform for platform-specific selectors
	platform := fetch.DetectPlatform(urlStr)
	if verbose {
		log.Printf("[VERBOSE] URL: %s", urlStr)
		log.Printf("[VERBOSE] Detected platform: %s", platform)
	}

	// Fetch HTML using the generic fetch package
	result, err := fetch.URL(ctx, urlStr, nil)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrHTTPRequestFailed, err)
	}
	if verbose {
		log.Printf("[VERBOSE] Fetched HTML: %d bytes", len(result.HTML))
	}

	// Get platform-specific selectors
	contentSelectors := fetch.PlatformContentSelectors(platform)
	noiseSelectors := fetch.PlatformNoiseSelectors(platform)
	if verbose {
		log.Printf("[VERBOSE] Content selectors: %v", contentSelectors)
		log.Printf("[VERBOSE] Noise selectors count: %d", len(noiseSelectors))
	}

	// Extract text from HTML using platform-specific selectors and noise removal
	textContent, err := fetch.ExtractMainText(result.HTML, contentSelectors, noiseSelectors...)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrContentExtractionFailed, err)
	}
	if verbose {
		log.Printf("[VERBOSE] Extracted text: %d chars", len(textContent))
	}

	// Check if we should use browser fallback for SPA sites
	if useBrowser && fetch.ShouldUseBrowser(textContent) {
		if verbose {
			log.Printf("[VERBOSE] Content too short (%d chars < %d), falling back to browser rendering...",
				len(textContent), fetch.MinContentLength)
		}

		// Fetch with headless browser
		browserHTML, browserErr := fetch.BrowserSimple(ctx, urlStr, verbose)
		if browserErr != nil {
			if verbose {
				log.Printf("[VERBOSE] Browser rendering failed: %v, using HTTP content", browserErr)
			}
			// Continue with HTTP content if browser fails
		} else {
			// Re-extract from browser-rendered HTML
			textContent, err = fetch.ExtractMainText(browserHTML, contentSelectors, noiseSelectors...)
			if err != nil {
				if verbose {
					log.Printf("[VERBOSE] Browser content extraction failed: %v", err)
				}
			} else if verbose {
				log.Printf("[VERBOSE] Browser extracted text: %d chars", len(textContent))
			}
		}
	}

	// Clean text
	cleanedText := CleanText(textContent)
	if verbose {
		log.Printf("[VERBOSE] Cleaned text: %d chars", len(cleanedText))
	}

	// Extract links for research seeds
	_, links, _ := CleanHTML(result.HTML)

	// Generate metadata
	metadata := NewMetadata(cleanedText, urlStr)
	metadata.Platform = string(platform)
	metadata.ExtractedLinks = links

	// If API key is provided, use LLM to extract structured content
	if apiKey != "" {
		if verbose {
			log.Printf("[VERBOSE] Calling LLM for structured extraction...")
		}
		extracted, err := ExtractWithLLM(ctx, cleanedText, apiKey)
		if err == nil {
			if verbose {
				log.Printf("[VERBOSE] LLM extraction successful")
				log.Printf("[VERBOSE] Team context: %d chars", len(extracted.TeamContext))
				log.Printf("[VERBOSE] Requirements: %d items", len(extracted.Requirements))
				log.Printf("[VERBOSE] Responsibilities: %d items", len(extracted.Responsibilities))
				log.Printf("[VERBOSE] Nice to have: %d items", len(extracted.NiceToHave))
			}
			// Format extracted content with team context
			cleanedText = FormatExtractedContent(extracted)
			metadata.AdminInfo = extracted.AdminInfo
			metadata.Company = extracted.Company
			metadata.AboutCompany = extracted.AboutCompany
		} else {
			if verbose {
				log.Printf("[VERBOSE] LLM extraction failed: %v, using cleaned text", err)
			}
		}
	}

	return cleanedText, metadata, nil
}

// FormatExtractedContent formats the structured extraction as readable text.
func FormatExtractedContent(extracted *ExtractedContent) string {
	var sb strings.Builder

	// Team context first (important for brand voice)
	if extracted.TeamContext != "" {
		sb.WriteString("Team Context:\n")
		sb.WriteString(extracted.TeamContext)
		sb.WriteString("\n\n")
	}

	// Requirements
	if len(extracted.Requirements) > 0 {
		sb.WriteString("Requirements:\n")
		for _, req := range extracted.Requirements {
			sb.WriteString("- " + req + "\n")
		}
		sb.WriteString("\n")
	}

	// Responsibilities
	if len(extracted.Responsibilities) > 0 {
		sb.WriteString("Responsibilities:\n")
		for _, resp := range extracted.Responsibilities {
			sb.WriteString("- " + resp + "\n")
		}
		sb.WriteString("\n")
	}

	// Nice to have
	if len(extracted.NiceToHave) > 0 {
		sb.WriteString("Nice to Have:\n")
		for _, nth := range extracted.NiceToHave {
			sb.WriteString("- " + nth + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
