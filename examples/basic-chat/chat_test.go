package main

import (
	"context"
	"testing"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/llm/gemini"
)

// MockLLMProvider for testing
type MockLLMProvider struct {
	responses []string
	callCount int
}

func (m *MockLLMProvider) CallLLM(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	result := llm.Message{}
	if m.callCount < len(m.responses) {
		result.Content = m.responses[m.callCount]
		m.callCount++
		return result, nil
	}
	result.Content= "Mock response"
	return result, nil
}

func (m *MockLLMProvider) GetName() string {
	return "mock"
}

func (m *MockLLMProvider) SetConfig(config map[string]any) error {
	return nil
}

func TestNewChatNode(t *testing.T) {
	// Create mock provider
	mockProvider := &MockLLMProvider{
		responses: []string{"Hello! How can I help you?"},
	}

	// Create config
	geminiConfig := &gemini.Config{
		APIKey:      "test-key",
		Model:       "gemini-2.0-flash",
		Temperature: 0.7,
		MaxRetries:  3,
	}
	chatConfig := NewChatConfig(geminiConfig)

	// Create ChatNode
	chatNode := NewChatNode(mockProvider, chatConfig)

	// Verify ChatNode was created correctly
	if chatNode == nil {
		t.Fatal("ChatNode should not be nil")
	}
	if chatNode.llmProvider != mockProvider {
		t.Error("ChatNode should have the correct LLM provider")
	}
	if chatNode.config != chatConfig {
		t.Error("ChatNode should have the correct config")
	}
	if !chatNode.firstRun {
		t.Error("ChatNode should be marked as first run")
	}
}

func TestChatNode_ExecFallback(t *testing.T) {
	// Create mock provider
	mockProvider := &MockLLMProvider{}
	
	// Create config
	geminiConfig := &gemini.Config{
		APIKey:      "test-key",
		Model:       "gemini-2.0-flash",
		Temperature: 0.7,
		MaxRetries:  3,
	}
	chatConfig := NewChatConfig(geminiConfig)

	// Create ChatNode
	chatNode := NewChatNode(mockProvider, chatConfig)

	// Test ExecFallback
	result := chatNode.ExecFallback(nil)

	// Verify fallback response
	expectedResponse := "I'm sorry, I encountered an error and couldn't process your request. Please try again."
	if result.Response != expectedResponse {
		t.Errorf("Expected fallback response: %s, got: %s", expectedResponse, result.Response)
	}
	if result.Error != nil {
		t.Error("ExecFallback should not return an error to continue conversation")
	}
}

func TestChatNode_Exec(t *testing.T) {
	// Create mock provider with predefined response
	mockProvider := &MockLLMProvider{
		responses: []string{"Hello! How can I help you today?"},
	}

	// Create config
	geminiConfig := &gemini.Config{
		APIKey:      "test-key",
		Model:       "gemini-2.0-flash",
		Temperature: 0.7,
		MaxRetries:  3,
	}
	chatConfig := NewChatConfig(geminiConfig)

	// Create ChatNode
	chatNode := NewChatNode(mockProvider, chatConfig)

	// Create test PrepResult
	prepResult := PrepResult{
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	// Test Exec
	result, err := chatNode.Exec(prepResult)

	// Verify execution result
	if err != nil {
		t.Errorf("Exec should not return error: %v", err)
	}
	if result.Response != "Hello! How can I help you today?" {
		t.Errorf("Expected response: 'Hello! How can I help you today?', got: %s", result.Response)
	}
	if result.Error != nil {
		t.Error("ExecResult should not contain error on successful execution")
	}
}

func TestChatConfig(t *testing.T) {
	// Create Gemini config
	geminiConfig := &gemini.Config{
		APIKey:      "test-key",
		Model:       "gemini-2.0-flash",
		Temperature: 0.7,
		MaxRetries:  3,
	}

	// Create ChatConfig
	chatConfig := NewChatConfig(geminiConfig)

	// Verify ChatConfig
	if chatConfig.LLMConfig != geminiConfig {
		t.Error("ChatConfig should have the correct LLM config")
	}
	if chatConfig.MaxHistory != 50 {
		t.Errorf("Expected MaxHistory: 50, got: %d", chatConfig.MaxHistory)
	}
	if chatConfig.WelcomeMsg == "" {
		t.Error("ChatConfig should have a welcome message")
	}
}

func TestChatNode_BaseNodeInterface(t *testing.T) {
	// Verify that ChatNode implements BaseNode interface
	mockProvider := &MockLLMProvider{}
	geminiConfig := &gemini.Config{
		APIKey:      "test-key",
		Model:       "gemini-2.0-flash",
		Temperature: 0.7,
		MaxRetries:  3,
	}
	chatConfig := NewChatConfig(geminiConfig)
	chatNode := NewChatNode(mockProvider, chatConfig)

	// This should compile if ChatNode implements BaseNode interface correctly
	var _ core.BaseNode[ChatState, PrepResult, ExecResult] = chatNode
}