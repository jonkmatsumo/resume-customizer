// Package crawling provides functionality to crawl company websites and build text corpora for brand voice analysis.
package crawling

import "fmt"

// CrawlError represents a general crawling failure
type CrawlError struct {
	Message string
	Cause   error
}

func (e *CrawlError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("crawl error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("crawl error: %s", e.Message)
}

func (e *CrawlError) Unwrap() error {
	return e.Cause
}

// LinkExtractionError represents a failure in extracting links from HTML
type LinkExtractionError struct {
	Message string
	Cause   error
}

func (e *LinkExtractionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("link extraction error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("link extraction error: %s", e.Message)
}

func (e *LinkExtractionError) Unwrap() error {
	return e.Cause
}

// ClassificationError represents a failure in LLM link classification
type ClassificationError struct {
	Message string
	Cause   error
}

func (e *ClassificationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("classification error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("classification error: %s", e.Message)
}

func (e *ClassificationError) Unwrap() error {
	return e.Cause
}

