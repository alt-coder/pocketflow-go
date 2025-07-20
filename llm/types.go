package llm

import "context"

// Message represents a generic chat message that can be used across different LLM providers
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string // The actual message content
	Media []byte
	MimeType  string
	ToolCalls []ToolCalls
	ToolResults []string
}



type ToolCalls struct{
	Id string
	ToolName string
	ToolArgs map[string]any
}

// LLMProvider interface defines the contract that all LLM implementations must follow
type LLMProvider interface {
	// CallLLM sends messages to the LLM and returns the response
	CallLLM(ctx context.Context, messages []Message) (Message, error)

	// GetName returns the name/identifier of the LLM provider
	GetName() string

	// SetConfig allows dynamic configuration updates for the provider
	SetConfig(config map[string]any) error
}

// Config holds configuration settings for LLM providers
type Config struct {
	Provider    string         // Provider name (e.g., "gemini", "openai")
	Model       string         // Model name to use
	Temperature float32        // Response creativity (0.0 to 1.0)
	MaxRetries  int            // Maximum retry attempts for failed requests
	APIKey      string         // API key for authentication
	Extra       map[string]any // Provider-specific additional configuration
}

const (
	// RoleSystem is used for system-level messages
	RoleSystem = "system"
	// RoleUser is used for user messages
	RoleUser = "user"
	// RoleAssistant is used for assistant messages
	RoleAssistant = "assistant"
)