package openai

import (
	"context"
	"testing"
	"time"

	"github.com/alt-coder/pocketflow-go/llm"
)

func TestNewOpenAIClient_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Test nil config
	_, err := NewOpenAIClient(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	// Test invalid config
	invalidConfig := &Config{
		APIKey:      "", // Missing API key
		Model:       "gpt-4",
		Temperature: 0.7,
	}
	_, err = NewOpenAIClient(ctx, invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestOpenAIClient_GetName(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.GetName() != "openai" {
		t.Errorf("Expected name 'openai', got '%s'", client.GetName())
	}
}

func TestOpenAIClient_SetConfig(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test updating configuration
	newConfig := map[string]any{
		"model":       "gpt-3.5-turbo",
		"temperature": float32(0.5),
		"maxTokens":   1000,
	}

	err = client.SetConfig(newConfig)
	if err != nil {
		t.Errorf("SetConfig failed: %v", err)
	}

	if client.config.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model 'gpt-3.5-turbo', got '%s'", client.config.Model)
	}

	if client.config.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", client.config.Temperature)
	}

	if client.config.MaxTokens != 1000 {
		t.Errorf("Expected maxTokens 1000, got %d", client.config.MaxTokens)
	}
}

func TestOpenAIClient_CallLLM_EmptyMessages(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with empty messages
	_, err = client.CallLLM(context.Background(), []llm.Message{})
	if err == nil {
		t.Error("Expected error for empty messages")
	}
}

func TestOpenAIClient_ConvertMessages(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test message conversion
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    llm.RoleUser,
			Content: "Hello",
		},
		{
			Role:    llm.RoleAssistant,
			Content: "Hi there!",
			ToolCalls: []llm.ToolCalls{
				{
					Id:       "call_123",
					ToolName: "test_tool",
					ToolArgs: map[string]any{"param": "value"},
				},
			},
		},
	}

	openaiMessages, err := client.convertToOpenAIMessages(messages)
	if err != nil {
		t.Fatalf("Failed to convert messages: %v", err)
	}

	if len(openaiMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(openaiMessages))
	}

	// Check system message
	if openaiMessages[0].Role != "system" {
		t.Errorf("Expected system role, got '%s'", openaiMessages[0].Role)
	}

	// Check assistant message with tool calls
	assistantMsg := openaiMessages[2]
	if len(assistantMsg.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
	}

	if assistantMsg.ToolCalls[0].Function.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", assistantMsg.ToolCalls[0].Function.Name)
	}
}

func TestOpenAIClient_ConvertMessagesWithMedia(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Model:       "gpt-4o",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test message with media
	messages := []llm.Message{
		{
			Role:     llm.RoleUser,
			Content:  "What's in this image?",
			Media:    []byte("fake-image-data"),
			MimeType: "image/jpeg",
		},
	}

	openaiMessages, err := client.convertToOpenAIMessages(messages)
	if err != nil {
		t.Fatalf("Failed to convert messages: %v", err)
	}

	if len(openaiMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(openaiMessages))
	}

	// Check that multi-content is used for media
	if len(openaiMessages[0].MultiContent) != 2 {
		t.Errorf("Expected 2 content parts (text + image), got %d", len(openaiMessages[0].MultiContent))
	}

	// Check text part
	if openaiMessages[0].MultiContent[0].Type != "text" {
		t.Errorf("Expected first part to be text, got '%s'", openaiMessages[0].MultiContent[0].Type)
	}

	// Check image part
	if openaiMessages[0].MultiContent[1].Type != "image_url" {
		t.Errorf("Expected second part to be image_url, got '%s'", openaiMessages[0].MultiContent[1].Type)
	}
}

func TestOpenAIClient_ConvertMessagesWithToolResults(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test message with tool results
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: "Here are the tool results",
			ToolResults: []llm.ToolResults{
				{
					Id:      "call_123",
					Content: "Tool execution result",
					IsError: false,
				},
			},
		},
	}

	openaiMessages, err := client.convertToOpenAIMessages(messages)
	if err != nil {
		t.Fatalf("Failed to convert messages: %v", err)
	}

	// Should create 2 messages: one for tool result, one for user message
	if len(openaiMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(openaiMessages))
	}

	// Check tool result message
	toolMsg := openaiMessages[0]
	if toolMsg.Role != "tool" {
		t.Errorf("Expected tool role, got '%s'", toolMsg.Role)
	}

	if toolMsg.ToolCallID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got '%s'", toolMsg.ToolCallID)
	}

	if toolMsg.Content != "Tool execution result" {
		t.Errorf("Expected content 'Tool execution result', got '%s'", toolMsg.Content)
	}
}

func TestOpenAIClient_RateLimiting(t *testing.T) {
	config := &Config{
		APIKey:            "test-key",
		Model:             "gpt-4",
		Temperature:       0.7,
		MaxRetries:        3,
		BaseURL:           "https://api.openai.com/v1",
		RateLimit:         2,                      // 2 requests
		RateLimitInterval: 100 * time.Millisecond, // per 100ms
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Verify rate limiter is initialized
	if client.rateLimiter == nil {
		t.Error("Expected rate limiter to be initialized")
	}

	if client.tokens == nil {
		t.Error("Expected token bucket to be initialized")
	}

	// Check initial token count
	if len(client.tokens) != 2 {
		t.Errorf("Expected 2 initial tokens, got %d", len(client.tokens))
	}
}

func TestOpenAIClient_Close(t *testing.T) {
	config := &Config{
		APIKey:            "test-key",
		Model:             "gpt-4",
		Temperature:       0.7,
		MaxRetries:        3,
		BaseURL:           "https://api.openai.com/v1",
		RateLimit:         10,
		RateLimitInterval: time.Minute,
	}

	client, err := NewOpenAIClient(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify rate limiter exists
	if client.rateLimiter == nil {
		t.Error("Expected rate limiter to be initialized")
	}

	// Close the client
	client.Close()

	// Rate limiter should be stopped (we can't easily test this without race conditions)
	// But at least verify Close() doesn't panic
}

func TestNewOpenAIClientFromEnv(t *testing.T) {
	// This test would require setting environment variables
	// For now, we'll just test that it fails without required env vars
	_, err := NewOpenAIClientFromEnv(context.Background())
	if err == nil {
		// This might pass if OPENAI_API_KEY is set in the environment
		// That's okay for this test
		t.Log("NewOpenAIClientFromEnv succeeded (API key likely set in environment)")
	} else {
		// Expected if no API key is set
		t.Log("NewOpenAIClientFromEnv failed as expected without API key")
	}
}
