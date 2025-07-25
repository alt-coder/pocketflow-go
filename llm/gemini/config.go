package gemini

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"google.golang.org/genai"
)

// Config holds Gemini-specific configuration settings
type Config struct {
	APIKey      string        // Google API key
	Model       string        // Default: "gemini-2.0-flash"
	Temperature float32       // Default: 0.7
	MaxRetries  int           // Default: 3
	Backend     genai.Backend // Default: genai.BackendGeminiAPI

	// Rate limiting configuration (optional)
	RateLimit         int           // Requests per minute, 0 = disabled (default)
	RateLimitInterval time.Duration // Rate limit window, default: 1 minute
}

// NewConfigFromEnv creates config from environment variables with sensible defaults
func NewConfigFromEnv() (*Config, error) {
	config := &Config{
		APIKey:            getEnvOrDefault("GOOGLE_API_KEY", ""),
		Model:             getEnvOrDefault("CHAT_MODEL", "gemini-2.0-flash"),
		Temperature:       getEnvFloatOrDefault("CHAT_TEMPERATURE", 0.7),
		MaxRetries:        getEnvIntOrDefault("CHAT_MAX_RETRIES", 3),
		Backend:           genai.BackendGeminiAPI,
		RateLimit:         getEnvIntOrDefault("GEMINI_RATE_LIMIT", 0),
		RateLimitInterval: time.Duration(getEnvIntOrDefault("GEMINI_RATE_LIMIT_INTERVAL_SECONDS", 60)) * time.Second,
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
		return fmt.Errorf("GOOGLE_API_KEY environment variable is required. Please set it with your Google API key")
	}

	if c.Model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if c.Temperature < 0.0 || c.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0, got %f", c.Temperature)
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
