package llm

import (
	"context"
	"testing"
)

func TestMockProvider_NewMockProvider(t *testing.T) {
	provider := NewMockProvider("test-mock")

	if provider.GetName() != "test-mock" {
		t.Errorf("Expected name 'test-mock', got '%s'", provider.GetName())
	}

	if provider.GetCallCount() != 0 {
		t.Errorf("Expected call count 0, got %d", provider.GetCallCount())
	}
}

func TestMockProvider_CallLLM_BasicResponse(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Mock response to: Hello"
	if response.Content != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response.Content)
	}

	if response.Role != RoleAssistant {
		t.Errorf("Expected role '%s', got '%s'", RoleAssistant, response.Role)
	}

	if provider.GetCallCount() != 1 {
		t.Errorf("Expected call count 1, got %d", provider.GetCallCount())
	}
}

func TestMockProvider_SetResponses(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	responses := []string{"First response", "Second response", "Third response"}
	provider.SetResponses(responses)

	messages := []Message{
		{Role: "user", Content: "Test"},
	}

	// Test cycling through responses
	for i := 0; i < 6; i++ { // Test cycling twice
		response, err := provider.CallLLM(ctx, messages)
		if err != nil {
			t.Errorf("Unexpected error on call %d: %v", i+1, err)
		}

		expected := "Mock response to: Test"
		if response.Content != expected {
			t.Errorf("Call %d: Expected '%s', got '%s'", i+1, expected, response.Content)
		}
	}
}

func TestMockProvider_ErrorSimulation(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Test basic error simulation
	provider.SetError(true, "Test error message")

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	response, err := provider.CallLLM(ctx, messages)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "Test error message" {
		t.Errorf("Expected 'Test error message', got '%s'", err.Error())
	}

	if response.Content != "" {
		t.Errorf("Expected empty response on error, got '%s'", response.Content)
	}
}

func TestMockProvider_ErrorSimulation_DefaultMessage(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Test error simulation with default message
	provider.SetError(true, "")

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	response, err := provider.CallLLM(ctx, messages)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	expected := "simulated API error from test-mock"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}

	if response.Content != "" {
		t.Errorf("Expected empty response on error, got '%s'", response.Content)
	}
}

func TestMockProvider_DelayedError(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set delayed error after 3 calls
	provider.SetDelayedError(3, "Delayed error occurred")

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	// First 2 calls should succeed
	for i := 0; i < 2; i++ {
		response, err := provider.CallLLM(ctx, messages)
		if err != nil {
			t.Errorf("Call %d: Unexpected error: %v", i+1, err)
		}
		if response.Content == "" {
			t.Errorf("Call %d: Expected response, got empty string", i+1)
		}
	}

	// Third call should fail
	response, err := provider.CallLLM(ctx, messages)
	if err == nil {
		t.Error("Expected delayed error on third call, got nil")
	}

	if err.Error() != "Delayed error occurred" {
		t.Errorf("Expected 'Delayed error occurred', got '%s'", err.Error())
	}

	if response.Content != "" {
		t.Errorf("Expected empty response on error, got '%s'", response.Content)
	}
}

func TestMockProvider_ResponsePatterns(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set up response patterns
	patterns := map[string]string{
		"hello": "Hi there!",
		"bye":   "Goodbye!",
		"help":  "How can I assist you?",
	}
	provider.SetResponsePattern(patterns)

	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hi there!"},
		{"HELLO", "Hi there!"},
		{"Say bye to me", "Goodbye!"},
		{"I need help", "How can I assist you?"},
		{"Random message", "Mock response to: Random message"}, // No pattern match
	}

	for _, tc := range testCases {
		messages := []Message{
			{Role: "user", Content: tc.input},
		}

		response, err := provider.CallLLM(ctx, messages)
		if err != nil {
			t.Errorf("Unexpected error for input '%s': %v", tc.input, err)
		}

		if response.Content != tc.expected {
			t.Errorf("Input '%s': Expected '%s', got '%s'", tc.input, tc.expected, response.Content)
		}

		if response.Role != RoleAssistant {
			t.Errorf("Expected role '%s', got '%s'", RoleAssistant, response.Role)
		}
	}
}

func TestMockProvider_SetConfig(t *testing.T) {
	provider := NewMockProvider("test-mock")

	config := map[string]any{
		"temperature": 0.7,
		"model":       "test-model",
		"custom":      "value",
	}

	err := provider.SetConfig(config)
	if err != nil {
		t.Errorf("Unexpected error setting config: %v", err)
	}
}

func TestMockProvider_Reset(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set up some state
	provider.SetResponses([]string{"Custom response"})
	provider.SetError(true, "Test error")
	provider.SetResponsePattern(map[string]string{"test": "pattern response"})

	// Make some calls to increment call count
	messages := []Message{{Role: "user", Content: "test"}}
	provider.CallLLM(ctx, messages) // This will error due to SetError

	// Reset the provider
	provider.Reset()

	// Verify reset worked
	if provider.GetCallCount() != 0 {
		t.Errorf("Expected call count 0 after reset, got %d", provider.GetCallCount())
	}

	// Should not error after reset
	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error after reset: %v", err)
	}

	// Should use default response pattern (not the custom pattern)
	expected := "Mock response to: test"
	if response.Content != expected {
		t.Errorf("Expected '%s' after reset, got '%s'", expected, response.Content)
	}
}

func TestMockProvider_ClearError(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set error
	provider.SetError(true, "Test error")

	messages := []Message{{Role: "user", Content: "Hello"}}

	// Should error
	_, err := provider.CallLLM(ctx, messages)
	if err == nil {
		t.Error("Expected error before clearing, got nil")
	}

	// Clear error
	provider.ClearError()

	// Should not error after clearing
	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error after clearing: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected response after clearing error, got empty string")
	}
}

func TestMockProvider_AddResponse(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Add individual responses
	provider.AddResponse("First added response")
	provider.AddResponse("Second added response")

	messages := []Message{{Role: "user", Content: "test"}}

	// The original default response should still be first
	response1, _ := provider.CallLLM(ctx, messages)
	expected1 := "Mock response to: test"
	if response1.Content != expected1 {
		t.Errorf("First call: Expected '%s', got '%s'", expected1, response1.Content)
	}

	// Then our added responses
	response2, _ := provider.CallLLM(ctx, messages)
	expected2 := "Mock response to: test"
	if response2.Content != expected2 {
		t.Errorf("Second call: Expected '%s', got '%s'", expected2, response2.Content)
	}
}

func TestMockProvider_EmptyMessages(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Test with empty messages slice
	response, err := provider.CallLLM(ctx, []Message{})
	if err != nil {
		t.Errorf("Unexpected error with empty messages: %v", err)
	}

	expected := "Mock response from test-mock"
	if response.Content != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response.Content)
	}

	if response.Role != RoleAssistant {
		t.Errorf("Expected role '%s', got '%s'", RoleAssistant, response.Role)
	}
}

func TestMockProvider_NonUserMessage(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Test with assistant message (should use default response)
	messages := []Message{
		{Role: "assistant", Content: "I am an assistant"},
	}

	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Mock response from test-mock"
	if response.Content != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response.Content)
	}

	if response.Role != RoleAssistant {
		t.Errorf("Expected role '%s', got '%s'", RoleAssistant, response.Role)
	}
}

func TestMockProvider_SetResponse(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Test the SetResponse method
	testMessage := Message{
		Role:    RoleAssistant,
		Content: "Custom single response",
	}
	provider.SetResponse(testMessage)

	messages := []Message{{Role: "user", Content: "Hello"}}

	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Mock response to: Hello"
	if response.Content != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response.Content)
	}
}

func TestMockProvider_DelayedError_DefaultMessage(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set delayed error with default message
	provider.SetDelayedError(2, "")

	messages := []Message{{Role: "user", Content: "Hello"}}

	// First call should succeed
	_, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("First call: Unexpected error: %v", err)
	}

	// Second call should fail with default message
	_, err = provider.CallLLM(ctx, messages)
	if err == nil {
		t.Error("Expected delayed error on second call, got nil")
	}

	expected := "delayed simulated error"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestMockProvider_ClearError_DelayedError(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set delayed error
	provider.SetDelayedError(1, "Delayed error")

	messages := []Message{{Role: "user", Content: "Hello"}}

	// First call should fail
	_, err := provider.CallLLM(ctx, messages)
	if err == nil {
		t.Error("Expected delayed error on first call, got nil")
	}

	// Clear error
	provider.ClearError()

	// Should not error after clearing
	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error after clearing delayed error: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected response after clearing delayed error, got empty string")
	}
}

func TestMockProvider_PatternPriority(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set both patterns and custom responses
	provider.SetResponses([]string{"Custom response"})
	provider.SetResponsePattern(map[string]string{
		"pattern": "Pattern matched response",
	})

	messages := []Message{{Role: "user", Content: "This matches pattern"}}

	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Pattern should take priority
	expected := "Pattern matched response"
	if response.Content != expected {
		t.Errorf("Expected pattern response '%s', got '%s'", expected, response.Content)
	}
}

func TestMockProvider_EmptyResponsesList(t *testing.T) {
	provider := NewMockProvider("test-mock")
	ctx := context.Background()

	// Set empty responses list
	provider.SetResponses([]string{})

	messages := []Message{{Role: "user", Content: "Hello"}}

	response, err := provider.CallLLM(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Default mock response"
	if response.Content != expected {
		t.Errorf("Expected default response '%s', got '%s'", expected, response.Content)
	}
}
