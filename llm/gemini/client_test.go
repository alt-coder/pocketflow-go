package gemini

import (
	"context"
	"os"
	"testing"

)

func TestNewConfigFromEnv(t *testing.T) {
	// Save original env vars
	originalAPIKey := os.Getenv("GOOGLE_API_KEY")
	originalModel := os.Getenv("CHAT_MODEL")
	originalTemp := os.Getenv("CHAT_TEMPERATURE")

	// Clean up after test
	defer func() {
		os.Setenv("GOOGLE_API_KEY", originalAPIKey)
		os.Setenv("CHAT_MODEL", originalModel)
		os.Setenv("CHAT_TEMPERATURE", originalTemp)
	}()

	// Test with missing API key
	os.Unsetenv("GOOGLE_API_KEY")
	_, err := NewConfigFromEnv()
	if err == nil {
		t.Error("Expected error when GOOGLE_API_KEY is missing")
	}

	// Test with valid configuration
	os.Setenv("GOOGLE_API_KEY", "test-api-key")
	os.Setenv("CHAT_MODEL", "gemini-pro")
	os.Setenv("CHAT_TEMPERATURE", "0.5")

	config, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got '%s'", config.APIKey)
	}

	if config.Model != "gemini-pro" {
		t.Errorf("Expected Model 'gemini-pro', got '%s'", config.Model)
	}

	if config.Temperature != 0.5 {
		t.Errorf("Expected Temperature 0.5, got %f", config.Temperature)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:      "test-key",
				Model:       "gemini-pro",
				Temperature: 0.7,
				MaxRetries:  3,
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model:       "gemini-pro",
				Temperature: 0.7,
				MaxRetries:  3,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: Config{
				APIKey:      "test-key",
				Model:       "gemini-pro",
				Temperature: 1.5,
				MaxRetries:  3,
			},
			wantErr: true,
		},
		{
			name: "negative retries",
			config: Config{
				APIKey:      "test-key",
				Model:       "gemini-pro",
				Temperature: 0.7,
				MaxRetries:  -1,
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

func TestGeminiClient_GetName(t *testing.T) {
	client := &GeminiClient{}
	if name := client.GetName(); name != "gemini" {
		t.Errorf("Expected name 'gemini', got '%s'", name)
	}
}

func TestGeminiClient_SetConfig(t *testing.T) {
	client := &GeminiClient{
		config: &Config{
			Model:       "gemini-pro",
			Temperature: 0.7,
		},
	}

	config := map[string]any{
		"model":       "gemini-2.0-flash",
		"temperature": float32(0.5),
		"apiKey":      "new-key",
		"maxRetries":  5,
	}

	err := client.SetConfig(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if client.config.Model != "gemini-2.0-flash" {
		t.Errorf("Expected model 'gemini-2.0-flash', got '%s'", client.config.Model)
	}

	if client.config.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", client.config.Temperature)
	}

	if client.config.APIKey != "new-key" {
		t.Errorf("Expected API key 'new-key', got '%s'", client.config.APIKey)
	}

	if client.config.MaxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", client.config.MaxRetries)
	}
}



func TestNewGeminiClient_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Test with nil config
	_, err := NewGeminiClient(ctx, nil)
	if err == nil {
		t.Error("Expected error with nil config")
	}

	// Test with invalid config
	invalidConfig := &Config{
		APIKey:      "", // Missing API key
		Model:       "gemini-pro",
		Temperature: 0.7,
	}

	_, err = NewGeminiClient(ctx, invalidConfig)
	if err == nil {
		t.Error("Expected error with invalid config")
	}
}
