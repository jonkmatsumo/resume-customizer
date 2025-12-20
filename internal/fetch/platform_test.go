package fetch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectPlatform_Greenhouse(t *testing.T) {
	tests := []struct {
		url      string
		expected Platform
	}{
		{"https://job-boards.greenhouse.io/doordashusa/jobs/7063751", PlatformGreenhouse},
		{"https://boards.greenhouse.io/company/jobs/123", PlatformGreenhouse},
		{"https://greenhouse.io/jobs/456", PlatformGreenhouse},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectPlatform(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectPlatform_Lever(t *testing.T) {
	tests := []struct {
		url      string
		expected Platform
	}{
		{"https://jobs.lever.co/company/job-id", PlatformLever},
		{"https://lever.co/jobs/123", PlatformLever},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectPlatform(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectPlatform_Workday(t *testing.T) {
	tests := []struct {
		url      string
		expected Platform
	}{
		{"https://company.wd5.myworkdayjobs.com/en-US/External", PlatformWorkday},
		{"https://workday.com/jobs", PlatformWorkday},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectPlatform(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectPlatform_Unknown(t *testing.T) {
	tests := []struct {
		url      string
		expected Platform
	}{
		{"https://example.com/jobs", PlatformUnknown},
		{"https://linkedin.com/jobs/123", PlatformUnknown},
		{"https://indeed.com/viewjob", PlatformUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectPlatform(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlatformContentSelectors_Greenhouse(t *testing.T) {
	selectors := PlatformContentSelectors(PlatformGreenhouse)
	assert.Contains(t, selectors, ".job__description.body")
	assert.Contains(t, selectors, ".job__description")
}

func TestPlatformContentSelectors_Unknown(t *testing.T) {
	selectors := PlatformContentSelectors(PlatformUnknown)
	// Should fallback to generic JobPostingSelectors
	assert.Contains(t, selectors, ".job-description")
	assert.Contains(t, selectors, "main")
}

func TestPlatformNoiseSelectors_Greenhouse(t *testing.T) {
	selectors := PlatformNoiseSelectors(PlatformGreenhouse)
	// Common selectors
	assert.Contains(t, selectors, "#application-form")
	assert.Contains(t, selectors, "form")
	// Greenhouse-specific
	assert.Contains(t, selectors, ".application--wrapper")
	assert.Contains(t, selectors, ".voluntary-self-id")
}

func TestPlatformNoiseSelectors_Unknown(t *testing.T) {
	selectors := PlatformNoiseSelectors(PlatformUnknown)
	// Should have common noise selectors
	assert.Contains(t, selectors, "form")
	assert.Contains(t, selectors, "#application-form")
	assert.Contains(t, selectors, ".cookie-banner")
}
