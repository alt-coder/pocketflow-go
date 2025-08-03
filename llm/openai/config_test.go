package openai

import (
	"os"
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "empty model",
			config: &Config{
				APIKey:      "test-key",
				Model:       "",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "invalid temperature - too low",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: -0.1,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "invalid temperature - too high",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 2.1,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  -1,
				BaseURL:     "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "negative rate limit",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
				RateLimit:   -1,
			},
			wantErr: true,
		},
		{
			name: "rate limit enabled but invalid interval",
			config: &Config{
				APIKey:            "test-key",
				Model:             "gpt-4",
				Temperature:       0.7,
				MaxRetries:        3,
				BaseURL:           "https://api.openai.com/v1",
				RateLimit:         10,
				RateLimitInterval: 0,
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
				MaxTokens:   -1,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p - too low",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
				TopP:        -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p - too high",
			config: &Config{
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxRetries:  3,
				BaseURL:     "https://api.openai.com/v1",
				TopP:        1.1,
			},
			wantErr: true,
		},
		{
			name: "invalid frequency penalty - too low",
			config: &Config{
				APIKey:           "test-key",
				Model:            "gpt-4",
				Temperature:      0.7,
				MaxRetries:       3,
				BaseURL:          "https://api.openai.com/v1",
				FrequencyPenalty: -2.1,
			},
			wantErr: true,
		},
		{
			name: "invalid frequency penalty - too high",
			config: &Config{
				APIKey:           "test-key",
				Model:            "gpt-4",
				Temperature:      0.7,
				MaxRetries:       3,
				BaseURL:          "https://api.openai.com/v1",
				FrequencyPenalty: 2.1,
			},
			wantErr: true,
		},
		{
			name: "invalid presence penalty - too low",
			config: &Config{
				APIKey:          "test-key",
				Model:           "gpt-4",
				Temperature:     0.7,
				MaxRetries:      3,
				BaseURL:         "https://api.openai.com/v1",
				PresencePenalty: -2.1,
			},
			wantErr: true,
		},
		{
			name: "invalid presence penalty - too high",
			config: &Config{
				APIKey:          "test-key",
				Model:           "gpt-4",
				Temperature:     0.7,
				MaxRetries:      3,
				BaseURL:         "https://api.openai.com/v1",
				PresencePenalty: 2.1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewConfigFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"OPENAI_TEMPERATURE",
		"OPENAI_MAX_RETRIES",
		"OPENAI_BASE_URL",
		"OPENAI_ORG_ID",
		"OPENAI_RATE_LIMIT",
		"OPENAI_RATE_LIMIT_INTERVAL_SECONDS",
		"OPENAI_MAX_TOKENS",
		"OPENAI_TOP_P",
		"OPENAI_FREQUENCY_PENALTY",
		"OPENAI_PRESENCE_PENALTY",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	// Restore environment after test
	defer func() {
		for env, value := range originalEnv {
			if value != "" {
				os.Setenv(env, value)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Test with minimal required environment
	os.Setenv("OPENAI_API_KEY", "test-key")

	config, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("NewConfigFromEnv() failed: %v", err)
	}

	// Check defaults
	if config.APIKey != "test-key" {
		t.Errorf("Expected APIKey 'test-key', got '%s'", config.APIKey)
	}

	if config.Model != "gpt-4o" {
		t.Errorf("Expected Model 'gpt-4o', got '%s'", config.Model)
	}

	if config.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", config.Temperature)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}

	if config.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected BaseURL 'https://api.openai.com/v1', got '%s'", config.BaseURL)
	}

	if config.RateLimit != 0 {
		t.Errorf("Expected RateLimit 0, got %d", config.RateLimit)
	}

	if config.RateLimitInterval != 60*time.Second {
		t.Errorf("Expected RateLimitInterval 60s, got %v", config.RateLimitInterval)
	}

	if config.MaxTokens != 0 {
		t.Errorf("Expected MaxTokens 0, got %d", config.MaxTokens)
	}

	if config.TopP != 1.0 {
		t.Errorf("Expected TopP 1.0, got %f", config.TopP)
	}

	if config.FrequencyPenalty != 0.0 {
		t.Errorf("Expected FrequencyPenalty 0.0, got %f", config.FrequencyPenalty)
	}

	if config.PresencePenalty != 0.0 {
		t.Errorf("Expected PresencePenalty 0.0, got %f", config.PresencePenalty)
	}
}

func TestNewConfigFromEnv_CustomValues(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"OPENAI_TEMPERATURE",
		"OPENAI_MAX_RETRIES",
		"OPENAI_BASE_URL",
		"OPENAI_ORG_ID",
		"OPENAI_RATE_LIMIT",
		"OPENAI_RATE_LIMIT_INTERVAL_SECONDS",
		"OPENAI_MAX_TOKENS",
		"OPENAI_TOP_P",
		"OPENAI_FREQUENCY_PENALTY",
		"OPENAI_PRESENCE_PENALTY",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	// Restore environment after test
	defer func() {
		for env, value := range originalEnv {
			if value != "" {
				os.Setenv(env, value)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Set custom environment variables
	os.Setenv("OPENAI_API_KEY", "custom-key")
	os.Setenv("OPENAI_MODEL", "gpt-3.5-turbo")
	os.Setenv("OPENAI_TEMPERATURE", "0.5")
	os.Setenv("OPENAI_MAX_RETRIES", "5")
	os.Setenv("OPENAI_BASE_URL", "https://custom.api.com/v1")
	os.Setenv("OPENAI_ORG_ID", "org-123")
	os.Setenv("OPENAI_RATE_LIMIT", "10")
	os.Setenv("OPENAI_RATE_LIMIT_INTERVAL_SECONDS", "30")
	os.Setenv("OPENAI_MAX_TOKENS", "1000")
	os.Setenv("OPENAI_TOP_P", "0.9")
	os.Setenv("OPENAI_FREQUENCY_PENALTY", "0.5")
	os.Setenv("OPENAI_PRESENCE_PENALTY", "0.3")

	config, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("NewConfigFromEnv() failed: %v", err)
	}

	// Check custom values
	if config.APIKey != "custom-key" {
		t.Errorf("Expected APIKey 'custom-key', got '%s'", config.APIKey)
	}

	if config.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected Model 'gpt-3.5-turbo', got '%s'", config.Model)
	}

	if config.Temperature != 0.5 {
		t.Errorf("Expected Temperature 0.5, got %f", config.Temperature)
	}

	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", config.MaxRetries)
	}

	if config.BaseURL != "https://custom.api.com/v1" {
		t.Errorf("Expected BaseURL 'https://custom.api.com/v1', got '%s'", config.BaseURL)
	}

	if config.OrgID != "org-123" {
		t.Errorf("Expected OrgID 'org-123', got '%s'", config.OrgID)
	}

	if config.RateLimit != 10 {
		t.Errorf("Expected RateLimit 10, got %d", config.RateLimit)
	}

	if config.RateLimitInterval != 30*time.Second {
		t.Errorf("Expected RateLimitInterval 30s, got %v", config.RateLimitInterval)
	}

	if config.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens 1000, got %d", config.MaxTokens)
	}

	if config.TopP != 0.9 {
		t.Errorf("Expected TopP 0.9, got %f", config.TopP)
	}

	if config.FrequencyPenalty != 0.5 {
		t.Errorf("Expected FrequencyPenalty 0.5, got %f", config.FrequencyPenalty)
	}

	if config.PresencePenalty != 0.3 {
		t.Errorf("Expected PresencePenalty 0.3, got %f", config.PresencePenalty)
	}
}

func TestNewConfigFromEnv_MissingAPIKey(t *testing.T) {
	// Save original environment
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")

	// Restore environment after test
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		}
	}()

	_, err := NewConfigFromEnv()
	if err == nil {
		t.Error("Expected error when OPENAI_API_KEY is not set")
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Test getEnvOrDefault
	os.Setenv("TEST_STRING", "test_value")
	if getEnvOrDefault("TEST_STRING", "default") != "test_value" {
		t.Error("getEnvOrDefault failed for existing env var")
	}
	if getEnvOrDefault("NON_EXISTENT", "default") != "default" {
		t.Error("getEnvOrDefault failed for non-existent env var")
	}
	os.Unsetenv("TEST_STRING")

	// Test getEnvFloatOrDefault
	os.Setenv("TEST_FLOAT", "1.5")
	if getEnvFloatOrDefault("TEST_FLOAT", 2.0) != 1.5 {
		t.Error("getEnvFloatOrDefault failed for existing env var")
	}
	if getEnvFloatOrDefault("NON_EXISTENT", 2.0) != 2.0 {
		t.Error("getEnvFloatOrDefault failed for non-existent env var")
	}
	os.Setenv("TEST_FLOAT", "invalid")
	if getEnvFloatOrDefault("TEST_FLOAT", 2.0) != 2.0 {
		t.Error("getEnvFloatOrDefault failed for invalid env var")
	}
	os.Unsetenv("TEST_FLOAT")

	// Test getEnvIntOrDefault
	os.Setenv("TEST_INT", "42")
	if getEnvIntOrDefault("TEST_INT", 10) != 42 {
		t.Error("getEnvIntOrDefault failed for existing env var")
	}
	if getEnvIntOrDefault("NON_EXISTENT", 10) != 10 {
		t.Error("getEnvIntOrDefault failed for non-existent env var")
	}
	os.Setenv("TEST_INT", "invalid")
	if getEnvIntOrDefault("TEST_INT", 10) != 10 {
		t.Error("getEnvIntOrDefault failed for invalid env var")
	}
	os.Unsetenv("TEST_INT")
}
