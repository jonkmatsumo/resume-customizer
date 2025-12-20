// Package fetch - platform.go provides platform detection and platform-specific selectors.
package fetch

import (
	"net/url"
	"strings"
)

// Platform represents a known job board platform.
type Platform string

const (
	// PlatformGreenhouse is the Greenhouse ATS platform
	PlatformGreenhouse Platform = "greenhouse"
	// PlatformLever is the Lever ATS platform
	PlatformLever Platform = "lever"
	// PlatformWorkday is the Workday ATS platform
	PlatformWorkday Platform = "workday"
	// PlatformUnknown is an unrecognized platform
	PlatformUnknown Platform = "unknown"
)

// DetectPlatform identifies the job board platform from a URL.
func DetectPlatform(urlStr string) Platform {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return PlatformUnknown
	}

	host := strings.ToLower(parsed.Host)

	// Greenhouse patterns
	if strings.Contains(host, "greenhouse.io") ||
		strings.Contains(host, "boards.greenhouse.io") {
		return PlatformGreenhouse
	}

	// Lever patterns
	if strings.Contains(host, "lever.co") ||
		strings.Contains(host, "jobs.lever.co") {
		return PlatformLever
	}

	// Workday patterns
	if strings.Contains(host, "workday.com") ||
		strings.Contains(host, "myworkdayjobs.com") {
		return PlatformWorkday
	}

	return PlatformUnknown
}

// PlatformContentSelectors returns content selectors optimized for a specific platform.
func PlatformContentSelectors(platform Platform) []string {
	switch platform {
	case PlatformGreenhouse:
		return []string{
			".job__description.body",    // Primary Greenhouse selector
			".job__description",         // Fallback
			".job-description__content", // Alternative
			"#content",                  // Generic fallback
			".job-post-container",       // Container level
		}
	case PlatformLever:
		return []string{
			".posting-page",
			".section-wrapper.page-full-width",
			".posting-description",
			".content",
		}
	case PlatformWorkday:
		return []string{
			"[data-automation-id='jobDescription']",
			".WDXK",
			".gwt-HTML",
			".job-description",
		}
	default:
		return JobPostingSelectors()
	}
}

// PlatformNoiseSelectors returns noise exclusion selectors for a specific platform.
func PlatformNoiseSelectors(platform Platform) []string {
	// Common noise selectors for all platforms
	common := []string{
		// Application forms
		"form",
		"#application-form",
		".application-form",
		".application--container",
		".apply-button-container",
		"[data-testid='application-form']",

		// EEO and legal
		".voluntary-disclosure",
		".eeo-statement",
		".eeo-section",
		"[data-testid='eeo']",
		".legal-disclosure",
		".self-identification",

		// Social and share buttons
		".social-share",
		".share-buttons",
		".social-links",

		// Cookie and GDPR
		".cookie-banner",
		".cookie-consent",
		".gdpr-notice",

		// Generic navigation already handled in fetch.go
	}

	// Platform-specific noise selectors
	switch platform {
	case PlatformGreenhouse:
		return append(common,
			".application--wrapper",
			".voluntary-self-id",
			".voluntary-self-id-wrapper",
			"#usa_self_id_section",
			".post-apply",
		)
	case PlatformLever:
		return append(common,
			".apply-section",
			".lever-application-form",
			".posting-apply",
		)
	case PlatformWorkday:
		return append(common,
			"[data-automation-id='applyButton']",
			".application-section",
			".WDAF",
		)
	default:
		return common
	}
}
