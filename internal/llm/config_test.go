package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, ProviderGemini, config.Provider)
	assert.Equal(t, "gemini-2.5-flash-lite", config.GetModel(TierLite))
	assert.Equal(t, "gemini-2.5-flash", config.GetModel(TierStandard))
	assert.Equal(t, "gemini-2.5-pro", config.GetModel(TierAdvanced))
}

func TestGetModel_Fallback(t *testing.T) {
	config := &Config{
		Provider: ProviderGemini,
		Models: map[ModelTier]string{
			TierLite: "fallback-model",
		},
	}

	// Unknown tier should fallback to TierStandard, then TierLite
	assert.Equal(t, "fallback-model", config.GetModel("unknown"))
}

func TestGetModel_EmptyConfig(t *testing.T) {
	config := &Config{
		Provider: ProviderGemini,
		Models:   map[ModelTier]string{},
	}

	// Empty config should return empty string
	assert.Equal(t, "", config.GetModel(TierAdvanced))
}

func TestWithModel(t *testing.T) {
	config := DefaultConfig()
	newConfig := config.WithModel(TierAdvanced, "custom-model")

	// Original should be unchanged
	assert.Equal(t, "gemini-2.5-pro", config.GetModel(TierAdvanced))

	// New config should have custom model
	assert.Equal(t, "custom-model", newConfig.GetModel(TierAdvanced))

	// Other tiers should be copied
	assert.Equal(t, "gemini-2.5-flash-lite", newConfig.GetModel(TierLite))
}

func TestModelTierConstants(t *testing.T) {
	assert.Equal(t, ModelTier("lite"), TierLite)
	assert.Equal(t, ModelTier("standard"), TierStandard)
	assert.Equal(t, ModelTier("advanced"), TierAdvanced)
}

func TestProviderConstants(t *testing.T) {
	assert.Equal(t, Provider("gemini"), ProviderGemini)
	assert.Equal(t, Provider("openai"), ProviderOpenAI)
	assert.Equal(t, Provider("anthropic"), ProviderAnthropic)
}
