// Package llm provides centralized LLM configuration and client abstractions.
// This package enables easy switching between model tiers and future multi-provider support.
package llm

// ModelTier represents the complexity/capability level of a model
type ModelTier string

const (
	// TierLite is for simple tasks: classification, extraction, basic summarization
	TierLite ModelTier = "lite"
	// TierStandard is for moderate reasoning: parsing, structured output
	TierStandard ModelTier = "standard"
	// TierAdvanced is for complex reasoning: rewriting, repair, planning
	TierAdvanced ModelTier = "advanced"
)

// Provider represents an LLM provider
type Provider string

// Provider constants define supported LLM providers
const (
	// ProviderGemini is the Google Gemini provider
	ProviderGemini Provider = "gemini"
	// ProviderOpenAI is the OpenAI provider (future)
	ProviderOpenAI Provider = "openai"
	// ProviderAnthropic is the Anthropic/Claude provider (future)
	ProviderAnthropic Provider = "anthropic"
)

// Config holds the model configuration for the application
type Config struct {
	Provider Provider
	Models   map[ModelTier]string
}

// DefaultConfig returns the default configuration (currently Gemini)
func DefaultConfig() *Config {
	return DefaultGeminiConfig()
}

// DefaultGeminiConfig returns the default Gemini configuration
func DefaultGeminiConfig() *Config {
	return &Config{
		Provider: ProviderGemini,
		Models: map[ModelTier]string{
			TierLite:     "gemini-2.5-flash-lite",
			TierStandard: "gemini-2.5-flash",
			TierAdvanced: "gemini-2.5-pro",
		},
	}
}

// GetModel returns the model name for a given tier
func (c *Config) GetModel(tier ModelTier) string {
	if model, ok := c.Models[tier]; ok {
		return model
	}
	// Fallback chain: try standard, then lite
	if model, ok := c.Models[TierStandard]; ok {
		return model
	}
	if model, ok := c.Models[TierLite]; ok {
		return model
	}
	return "" // No model configured
}

// WithModel returns a new Config with a specific model for a tier
func (c *Config) WithModel(tier ModelTier, model string) *Config {
	newConfig := &Config{
		Provider: c.Provider,
		Models:   make(map[ModelTier]string),
	}
	for k, v := range c.Models {
		newConfig.Models[k] = v
	}
	newConfig.Models[tier] = model
	return newConfig
}
