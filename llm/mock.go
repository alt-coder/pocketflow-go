package llm

import (
	"context"
	"fmt"
	"strings"
)

// MockProvider implements LLMProvider interface for testing purposes
// It provides configurable response patterns and error simulation capabilities
type MockProvider struct {
	name          string
	responses     []string
	responseIndex int
	simulateError bool
	errorMessage  string
	config        map[string]any
	patterns      map[string]string // Pattern-based responses
	callCount     int               // Track number of calls for testing
}

// NewMockProvider creates a new mock LLM provider with configurable responses
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name:          name,
		responses:     []string{"Mock response from " + name},
		responseIndex: 0,
		simulateError: false,
		config:        make(map[string]any),
		patterns:      make(map[string]string),
		callCount:     0,
	}
}

// CallLLM simulates an LLM call and returns configured responses or errors
func (m *MockProvider) CallLLM(ctx context.Context, messages []Message) (Message, error) {
	m.callCount++

	// Check for delayed error simulation
	if delayedError, ok := m.config["delayedError"].(bool); ok && delayedError {
		if callsBeforeError, ok := m.config["callsBeforeError"].(int); ok {
			if m.callCount >= callsBeforeError {
				errorMsg := "delayed simulated error"
				if msg, ok := m.config["delayedErrorMessage"].(string); ok && msg != "" {
					errorMsg = msg
				}
				return Message{}, fmt.Errorf(errorMsg)
			}
		}
	}

	// Simulate immediate error if configured
	if m.simulateError {
		if m.errorMessage != "" {
			return Message{}, fmt.Errorf(m.errorMessage)
		}
		return Message{}, fmt.Errorf("simulated API error from %s", m.name)
	}

	// Check for pattern-based responses first
	if len(m.patterns) > 0 && len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		if lastMessage.Role == "user" {
			userInput := strings.ToLower(lastMessage.Content)

			// Check for pattern matches
			for pattern, response := range m.patterns {
				if strings.Contains(userInput, strings.ToLower(pattern)) {
					return Message{
						Role:    RoleAssistant,
						Content: response,
					}, nil
				}
			}
		}
	}

	// Return configured response if no pattern match
	if len(m.responses) == 0 {
		return Message{
			Role:    RoleAssistant,
			Content: "Default mock response",
		}, nil
	}

	response := m.responses[m.responseIndex]

	// Cycle through responses for multiple calls
	m.responseIndex = (m.responseIndex + 1) % len(m.responses)

	// Add context from the last user message if available
	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		if lastMessage.Role == "user" {
			response = fmt.Sprintf("Mock response to: %s", lastMessage.Content)
		}
	}

	return Message{
		Role:    RoleAssistant,
		Content: response,
	}, nil
}

// GetName returns the mock provider name
func (m *MockProvider) GetName() string {
	return m.name
}

// SetConfig updates the mock provider configuration
func (m *MockProvider) SetConfig(config map[string]any) error {
	m.config = config
	return nil
}

// SetResponses configures the responses that the mock will return
func (m *MockProvider) SetResponses(responses []string) {
	m.responses = responses
	m.responseIndex = 0
}

// SetError configures the mock to simulate an error
func (m *MockProvider) SetError(shouldError bool, errorMessage string) {
	m.simulateError = shouldError
	m.errorMessage = errorMessage
}

// AddResponse adds a single response to the response list
func (m *MockProvider) AddResponse(response string) {
	m.responses = append(m.responses, response)
}

// SetResponsePattern configures responses based on input patterns
func (m *MockProvider) SetResponsePattern(patterns map[string]string) {
	// This allows setting up responses based on input keywords
	// For example: {"hello": "Hi there!", "bye": "Goodbye!"}
	m.patterns = patterns
}

// Reset resets the mock provider to initial state
func (m *MockProvider) Reset() {
	m.responseIndex = 0
	m.simulateError = false
	m.errorMessage = ""
	m.config = make(map[string]any)
	m.patterns = make(map[string]string)
	m.callCount = 0
}

// GetCallCount returns the number of times CallLLM has been called
func (m *MockProvider) GetCallCount() int {
	return m.callCount
}

// SetDelayedError configures the mock to simulate an error after a certain number of calls
func (m *MockProvider) SetDelayedError(callsBeforeError int, errorMessage string) {
	m.config["delayedError"] = true
	m.config["callsBeforeError"] = callsBeforeError
	m.config["delayedErrorMessage"] = errorMessage
}

// ClearError removes any error simulation
func (m *MockProvider) ClearError() {
	m.simulateError = false
	m.errorMessage = ""
	delete(m.config, "delayedError")
	delete(m.config, "callsBeforeError")
	delete(m.config, "delayedErrorMessage")
}

func (m *MockProvider) SetResponse(message Message) {
	m.responses = []string{message.Content}

}
