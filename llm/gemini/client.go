package gemini

import (
	"context"
	"fmt"

	"github.com/alt-coder/pocketflow-go/llm"
	"google.golang.org/genai"
)

// GeminiClient implements LLMProvider interface for Google's Gemini models
type GeminiClient struct {
	genaiClient *genai.Client
	config      *Config
}

// CallLLM implements the generic interface, converting messages internally
func (c *GeminiClient) CallLLM(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	result := llm.Message{}
	if len(messages) == 0 {
		return result, fmt.Errorf("no messages to send")
	}

	// Convert messages to Gemini format
	genaiMessages, err := c.convertToGenaiMessages(messages)
	if err != nil {
		return result, fmt.Errorf("failed to convert messages: %w", err)
	}

	respone, err := c.genaiClient.Models.GenerateContent(ctx, c.config.Model, genaiMessages, nil)

	if err != nil {
		return llm.Message{}, fmt.Errorf("failed to generate content: %w", err)
	}

	for _, functionCall := range respone.FunctionCalls() {
		result.ToolCalls = append(result.ToolCalls, llm.ToolCalls{
			Id:       functionCall.ID,
			ToolName: functionCall.Name,
			ToolArgs: functionCall.Args,
		})
	}
	result.Role = "assistant"
	result.Content = respone.Text()
	return result, nil

}

// convertToGenaiMessages converts generic messages to Gemini format
func (c *GeminiClient) convertToGenaiMessages(messages []llm.Message) ([]*genai.Content, error) {
	var genaiMessages []*genai.Content

	for _, msg := range messages {
		content := &genai.Content{
			Role: getRole(msg.Role),
			Parts: []*genai.Part{
				{
					Text: msg.Content,
				},
			},
		}
		if len(msg.Media)> 0 {
			content.Parts = append(content.Parts, &genai.Part{
				InlineData : &genai.Blob{
					MIMEType: msg.MimeType,
					Data: msg.Media,
				},
			})
		}

		genaiMessages = append(genaiMessages, content)
	}

	return genaiMessages, nil
}

func getRole (role string) string {
	switch role {
	case "user":
		return genai.RoleUser
	case "assistant":
		return genai.RoleModel
	default:
		return genai.RoleUser
	}
}

// GetName returns the provider name
func (c *GeminiClient) GetName() string {
	return "gemini"
}

// SetConfig updates the client configuration
func (c *GeminiClient) SetConfig(config map[string]any) error {
	if c.config == nil {
		c.config = &Config{}
	}

	// Update configuration fields if provided
	if model, ok := config["model"].(string); ok {
		c.config.Model = model
	}
	if temp, ok := config["temperature"].(float32); ok {
		c.config.Temperature = temp
	}
	if apiKey, ok := config["apiKey"].(string); ok {
		c.config.APIKey = apiKey
	}
	if maxRetries, ok := config["maxRetries"].(int); ok {
		c.config.MaxRetries = maxRetries
	}

	return nil
}

// NewGeminiClient creates a new Gemini client with the provided configuration
func NewGeminiClient(ctx context.Context, config *Config) (*GeminiClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create the GenAI client with the specified backend
	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: config.Backend,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	return &GeminiClient{
		genaiClient: genaiClient,
		config:      config,
	}, nil
}

// NewGeminiClientFromEnv creates a new Gemini client using environment variables
func NewGeminiClientFromEnv(ctx context.Context) (*GeminiClient, error) {
	config, err := NewConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from environment: %w", err)
	}

	return NewGeminiClient(ctx, config)
}
