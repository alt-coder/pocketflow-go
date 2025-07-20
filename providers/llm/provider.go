package llm

import (
	"fmt"
)

// Provider represents a generic LLM provider interface
type Provider interface {
	GenerateResponse(prompt string) (string, error)
	GetName() string
	SetConfig(config map[string]interface{}) error
}

// MockProvider is a simple mock implementation for testing
type MockProvider struct {
	name      string
	responses []string
	callCount int
}

// NewMockProvider creates a new mock provider with predefined responses
func NewMockProvider(name string, responses []string) *MockProvider {
	return &MockProvider{
		name:      name,
		responses: responses,
		callCount: 0,
	}
}

// GenerateResponse returns the next predefined response
func (m *MockProvider) GenerateResponse(prompt string) (string, error) {
	if len(m.responses) == 0 {
		return "", fmt.Errorf("no responses configured")
	}
	
	response := m.responses[m.callCount%len(m.responses)]
	m.callCount++
	
	return response, nil
}

// GetName returns the provider name
func (m *MockProvider) GetName() string {
	return m.name
}

// SetConfig configures the mock provider (no-op for mock)
func (m *MockProvider) SetConfig(config map[string]interface{}) error {
	return nil
}

// GetCallCount returns the number of times GenerateResponse was called
func (m *MockProvider) GetCallCount() int {
	return m.callCount
}

// Reset resets the call counter
func (m *MockProvider) Reset() {
	m.callCount = 0
}

// OpenAIProvider represents an OpenAI API provider (placeholder implementation)
type OpenAIProvider struct {
	name   string
	apiKey string
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		name:   "openai",
		apiKey: apiKey,
		model:  model,
	}
}

// GenerateResponse calls the OpenAI API (placeholder - would need actual implementation)
func (o *OpenAIProvider) GenerateResponse(prompt string) (string, error) {
	// This is a placeholder implementation
	// In a real implementation, this would make an HTTP request to OpenAI's API
	return "", fmt.Errorf("OpenAI provider not implemented - use MockProvider for testing")
}

// GetName returns the provider name
func (o *OpenAIProvider) GetName() string {
	return o.name
}

// SetConfig configures the OpenAI provider
func (o *OpenAIProvider) SetConfig(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); ok {
		o.apiKey = apiKey
	}
	if model, ok := config["model"].(string); ok {
		o.model = model
	}
	return nil
}

// AnthropicProvider represents an Anthropic API provider (placeholder implementation)
type AnthropicProvider struct {
	name   string
	apiKey string
	model  string
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	return &AnthropicProvider{
		name:   "anthropic",
		apiKey: apiKey,
		model:  model,
	}
}

// GenerateResponse calls the Anthropic API (placeholder - would need actual implementation)
func (a *AnthropicProvider) GenerateResponse(prompt string) (string, error) {
	// This is a placeholder implementation
	// In a real implementation, this would make an HTTP request to Anthropic's API
	return "", fmt.Errorf("Anthropic provider not implemented - use MockProvider for testing")
}

// GetName returns the provider name
func (a *AnthropicProvider) GetName() string {
	return a.name
}

// SetConfig configures the Anthropic provider
func (a *AnthropicProvider) SetConfig(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); ok {
		a.apiKey = apiKey
	}
	if model, ok := config["model"].(string); ok {
		a.model = model
	}
	return nil
}

// ProviderFactory helps create providers by name
type ProviderFactory struct{}

// CreateProvider creates a provider instance by name
func (pf *ProviderFactory) CreateProvider(providerType string, config map[string]interface{}) (Provider, error) {
	switch providerType {
	case "mock":
		responses, ok := config["responses"].([]string)
		if !ok {
			responses = []string{`{"success": true, "message": "mock response"}`}
		}
		name, ok := config["name"].(string)
		if !ok {
			name = "mock-provider"
		}
		return NewMockProvider(name, responses), nil
		
	case "openai":
		apiKey, ok := config["api_key"].(string)
		if !ok {
			return nil, fmt.Errorf("api_key required for OpenAI provider")
		}
		model, ok := config["model"].(string)
		if !ok {
			model = "gpt-3.5-turbo"
		}
		return NewOpenAIProvider(apiKey, model), nil
		
	case "anthropic":
		apiKey, ok := config["api_key"].(string)
		if !ok {
			return nil, fmt.Errorf("api_key required for Anthropic provider")
		}
		model, ok := config["model"].(string)
		if !ok {
			model = "claude-3-sonnet-20240229"
		}
		return NewAnthropicProvider(apiKey, model), nil
		
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// GetAvailableProviders returns a list of available provider types
func (pf *ProviderFactory) GetAvailableProviders() []string {
	return []string{"mock", "openai", "anthropic"}
}