package openai

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds OpenAI-specific configuration settings
type Config struct {
	APIKey      string  // OpenAI API key
	Model       string  // Default: "gpt-4o"
	Temperature float32 // Default: 0.7
	MaxRetries  int     // Default: 3
	BaseURL     string  // Default: "https://api.openai.com/v1"
	OrgID       string  // Optional organization ID

	// Rate limiting configuration (optional)
	RateLimit         int           // Requests per minute, 0 = disabled (default)
	RateLimitInterval time.Duration // Rate limit window, default: 1 minute

	// Advanced settings
	MaxTokens        int     // Maximum tokens in response, 0 = no limit (default)
	TopP             float32 // Nucleus sampling parameter, default: 1.0
	FrequencyPenalty float32 // Frequency penalty, default: 0.0
	PresencePenalty  float32 // Presence penalty, default: 0.0
}

// NewConfigFromEnv creates config from environment variables with sensible defaults
func NewConfigFromEnv() (*Config, error) {
	config := &Config{
		APIKey:            getEnvOrDefault("OPENAI_API_KEY", ""),
		Model:             getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
		Temperature:       getEnvFloatOrDefault("OPENAI_TEMPERATURE", 0.7),
		MaxRetries:        getEnvIntOrDefault("OPENAI_MAX_RETRIES", 3),
		BaseURL:           getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OrgID:             getEnvOrDefault("OPENAI_ORG_ID", ""),
		RateLimit:         getEnvIntOrDefault("OPENAI_RATE_LIMIT", 0),
		RateLimitInterval: time.Duration(getEnvIntOrDefault("OPENAI_RATE_LIMIT_INTERVAL_SECONDS", 60)) * time.Second,
		MaxTokens:         getEnvIntOrDefault("OPENAI_MAX_TOKENS", 0),
		TopP:              getEnvFloatOrDefault("OPENAI_TOP_P", 1.0),
		FrequencyPenalty:  getEnvFloatOrDefault("OPENAI_FREQUENCY_PENALTY", 0.0),
		PresencePenalty:   getEnvFloatOrDefault("OPENAI_PRESENCE_PENALTY", 0.0),
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if the configuration is valid and complete
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable is required. Please set it with your OpenAI API key")
	}

	if c.Model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", c.Temperature)
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("maxRetries cannot be negative, got %d", c.MaxRetries)
	}

	if c.RateLimit < 0 {
		return fmt.Errorf("rateLimit cannot be negative, got %d", c.RateLimit)
	}

	if c.RateLimit > 0 && c.RateLimitInterval <= 0 {
		return fmt.Errorf("rateLimitInterval must be positive when rate limiting is enabled, got %v", c.RateLimitInterval)
	}

	if c.MaxTokens < 0 {
		return fmt.Errorf("maxTokens cannot be negative, got %d", c.MaxTokens)
	}

	if c.TopP < 0.0 || c.TopP > 1.0 {
		return fmt.Errorf("topP must be between 0.0 and 1.0, got %f", c.TopP)
	}

	if c.FrequencyPenalty < -2.0 || c.FrequencyPenalty > 2.0 {
		return fmt.Errorf("frequencyPenalty must be between -2.0 and 2.0, got %f", c.FrequencyPenalty)
	}

	if c.PresencePenalty < -2.0 || c.PresencePenalty > 2.0 {
		return fmt.Errorf("presencePenalty must be between -2.0 and 2.0, got %f", c.PresencePenalty)
	}

	return nil
}

// getEnvOrDefault returns the environment variable value or a default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvFloatOrDefault returns the environment variable as float32 or default if not set/invalid
func getEnvFloatOrDefault(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(parsed)
		}
	}
	return defaultValue
}

// getEnvIntOrDefault returns the environment variable as int or default if not set/invalid
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
