package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Client is an abstraction over LLM providers
type Client interface {
	// GenerateContent generates text content using the specified model tier
	GenerateContent(ctx context.Context, prompt string, tier ModelTier) (string, error)
	// GenerateJSON generates JSON content using the specified model tier
	GenerateJSON(ctx context.Context, prompt string, tier ModelTier) (string, error)
	// GetModel returns the underlying provider model for a tier (for direct access if needed)
	GetModel(tier ModelTier) string
	// Close releases any resources held by the client
	Close() error
}

// NewClient creates a new LLM client based on configuration
func NewClient(ctx context.Context, config *Config, apiKey string) (Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Provider {
	case ProviderGemini:
		return NewGeminiClient(ctx, config, apiKey)
	// case ProviderOpenAI:
	//     return NewOpenAIClient(ctx, config, apiKey)
	// case ProviderAnthropic:
	//     return NewClaudeClient(ctx, config, apiKey)
	default:
		return NewGeminiClient(ctx, config, apiKey)
	}
}

// GeminiClient implements Client for Google Gemini
type GeminiClient struct {
	client *genai.Client
	config *Config
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(ctx context.Context, config *Config, apiKey string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiClient{
		client: client,
		config: config,
	}, nil
}

// GenerateContent generates text content using the specified model tier with fallback support
func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string, tier ModelTier) (string, error) {
	tiers := c.getFallbackTiers(tier)

	var lastErr error
	for _, t := range tiers {
		modelName := c.config.GetModel(t)
		if modelName == "" {
			// If this is a synthesized tier for fallback (like "safety"), GetModel might return empty
			if t == "safety" {
				modelName = "gemini-2.0-flash"
			} else {
				continue
			}
		}

		res, err := c.tryGenerate(ctx, prompt, modelName, false)
		if err == nil {
			return res, nil
		}
		lastErr = err
		// Log fallback if verbose? We don't have easy access to logger here, but we can assume it's okay for now.
	}

	return "", fmt.Errorf("all model tiers failed, last error: %w", lastErr)
}

// GenerateJSON generates JSON content using the specified model tier with fallback support
func (c *GeminiClient) GenerateJSON(ctx context.Context, prompt string, tier ModelTier) (string, error) {
	tiers := c.getFallbackTiers(tier)

	var lastErr error
	for _, t := range tiers {
		modelName := c.config.GetModel(t)
		if modelName == "" {
			if t == "safety" {
				modelName = "gemini-2.0-flash"
			} else {
				continue
			}
		}

		res, err := c.tryGenerate(ctx, prompt, modelName, true)
		if err == nil {
			return cleanJSONBlock(res), nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("all model tiers failed (JSON), last error: %w", lastErr)
}

func (c *GeminiClient) getFallbackTiers(tier ModelTier) []ModelTier {
	switch tier {
	case TierAdvanced:
		return []ModelTier{TierAdvanced, TierStandard, TierLite, "safety"}
	case TierStandard:
		return []ModelTier{TierStandard, TierLite, "safety"}
	case TierLite:
		return []ModelTier{TierLite, "safety"}
	default:
		return []ModelTier{tier, TierLite, "safety"}
	}
}

func (c *GeminiClient) tryGenerate(ctx context.Context, prompt string, modelName string, isJSON bool) (string, error) {
	model := c.client.GenerativeModel(modelName)
	model.SetTemperature(0.1)
	if isJSON {
		model.ResponseMIMEType = "application/json"
	}

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	return extractTextFromResponse(resp)
}

// GetModel returns the model name for a tier
func (c *GeminiClient) GetModel(tier ModelTier) string {
	return c.config.GetModel(tier)
}

// Close releases resources held by the client
func (c *GeminiClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// extractTextFromResponse extracts text from Gemini API response
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	var parts []string
	for _, part := range candidate.Content.Parts {
		if text, ok := part.(genai.Text); ok {
			parts = append(parts, string(text))
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no text parts in response")
	}

	return strings.Join(parts, ""), nil
}

// cleanJSONBlock removes markdown code block wrappers from JSON
func cleanJSONBlock(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
