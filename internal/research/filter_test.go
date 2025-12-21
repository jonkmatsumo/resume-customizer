package research

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDomainFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"Simple URL", "https://doordash.com", "doordash.com"},
		{"URL with www", "https://www.doordash.com", "doordash.com"},
		{"URL with path", "https://careers.doordash.com/jobs", "careers.doordash.com"},
		{"URL without scheme", "doordash.com", "doordash.com"},
		{"Empty URL", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDomainFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFromCompanyDomain(t *testing.T) {
	companyDomains := []string{"doordash.com", "careersatdoordash.com"}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Main domain", "https://doordash.com/about", true},
		{"Subdomain", "https://careers.doordash.com/jobs", true},
		{"Related domain", "https://careersatdoordash.com/values", true},
		{"Third-party", "https://greenhouse.io/jobs", false},
		{"Government", "https://usa.gov/info", false},
		{"Different company", "https://uber.com/about", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFromCompanyDomain(tt.url, companyDomains)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterToCompanyDomains(t *testing.T) {
	companyDomains := []string{"doordash.com", "careersatdoordash.com"}

	urls := []string{
		"https://doordash.com/about",
		"https://greenhouse.io/jobs",
		"https://careersatdoordash.com/values",
		"https://usa.gov/info",
		"https://about.doordash.com/team",
	}

	filtered := FilterToCompanyDomains(urls, companyDomains)

	assert.Len(t, filtered, 3)
	assert.Contains(t, filtered, "https://doordash.com/about")
	assert.Contains(t, filtered, "https://careersatdoordash.com/values")
	assert.Contains(t, filtered, "https://about.doordash.com/team")
	assert.NotContains(t, filtered, "https://greenhouse.io/jobs")
	assert.NotContains(t, filtered, "https://usa.gov/info")
}

func TestAssignPathPriority(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		expectedMinimum float64
	}{
		{"Mission and values", "https://careers.doordash.com/mission-and-values", 0.9},
		{"Values page", "https://company.com/values", 0.9},
		{"Culture page", "https://company.com/culture", 0.8},
		{"About page", "https://company.com/about", 0.8},
		{"Engineering blog", "https://company.com/engineering", 0.8},
		{"Product page", "https://doordash.com/p/alcohol-delivery", 0.0},
		{"Promotional", "https://doordash.com/catering-near-me", 0.0},
		{"Generic page", "https://doordash.com/page", 0.4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := AssignPathPriority(tt.url)
			assert.GreaterOrEqual(t, priority, tt.expectedMinimum,
				"URL %s should have priority >= %.2f, got %.2f", tt.url, tt.expectedMinimum, priority)
		})
	}
}

func TestIsThirdParty(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Greenhouse", "https://job-boards.greenhouse.io/doordash", true},
		{"Lever", "https://jobs.lever.co/company", true},
		{"LinkedIn", "https://linkedin.com/company/doordash", true},
		{"USA Gov", "https://go.usa.gov/abc", true},
		{"Getcovey", "https://getcovey.com/product", true},
		{"Medium", "https://medium.com/@doordash", true},
		{"Company domain", "https://doordash.com/about", false},
		{"Careers subdomain", "https://careers.doordash.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsThirdParty(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractUniqueDomains(t *testing.T) {
	urls := []string{
		"https://doordash.com/about",
		"https://doordash.com/careers",
		"https://careers.doordash.com/jobs",
		"https://greenhouse.io/jobs",
		"https://usa.gov/info",
	}

	domains := extractUniqueDomains(urls)

	assert.Len(t, domains, 4) // doordash.com, careers.doordash.com, greenhouse.io, usa.gov
	assert.Contains(t, domains, "doordash.com")
	assert.Contains(t, domains, "careers.doordash.com")
	assert.Contains(t, domains, "greenhouse.io")
	assert.Contains(t, domains, "usa.gov")
}
