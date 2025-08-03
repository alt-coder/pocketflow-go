package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/sashabaranov/go-openai"
)

// OpenAIClient implements LLMProvider interface for OpenAI's models
type OpenAIClient struct {
	client *openai.Client
	config *Config

	// Rate limiting
	rateLimiter *time.Ticker
	tokens      chan struct{}
}

// CallLLM implements the generic interface, converting messages internally
func (c *OpenAIClient) CallLLM(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	result := llm.Message{}
	if len(messages) == 0 {
		return result, fmt.Errorf("no messages to send")
	}

	// Apply rate limiting if enabled
	if c.tokens != nil {
		select {
		case <-c.tokens:
			// Token acquired, proceed with request
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}

	// Convert messages to OpenAI format
	openaiMessages, err := c.convertToOpenAIMessages(messages)
	if err != nil {
		return result, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Create request
	request := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: openaiMessages,
	}

	// Add optional parameters
	if c.config.Temperature != 0.7 { // Only set if different from default
		request.Temperature = c.config.Temperature
	}
	if c.config.MaxTokens > 0 {
		request.MaxTokens = c.config.MaxTokens
	}
	if c.config.TopP != 1.0 {
		request.TopP = c.config.TopP
	}
	if c.config.FrequencyPenalty != 0.0 {
		request.FrequencyPenalty = c.config.FrequencyPenalty
	}
	if c.config.PresencePenalty != 0.0 {
		request.PresencePenalty = c.config.PresencePenalty
	}

	// Make API call with retries
	var response openai.ChatCompletionResponse
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		response, lastErr = c.client.CreateChatCompletion(ctx, request)
		if lastErr == nil {
			break
		}

		if attempt < c.config.MaxRetries {
			// Wait before retry with exponential backoff
			waitTime := time.Duration(attempt+1) * time.Second
			select {
			case <-time.After(waitTime):
				continue
			case <-ctx.Done():
				return result, ctx.Err()
			}
		}
	}

	if lastErr != nil {
		return result, fmt.Errorf("failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}

	if len(response.Choices) == 0 {
		return result, fmt.Errorf("no choices returned from OpenAI API")
	}

	// Convert response back to generic format
	choice := response.Choices[0]
	result.Role = llm.RoleAssistant
	result.Content = choice.Message.Content

	// Handle tool calls
	for _, toolCall := range choice.Message.ToolCalls {
		if toolCall.Type == openai.ToolTypeFunction {
			// Parse function arguments
			var args map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return result, fmt.Errorf("failed to parse tool arguments: %w", err)
			}

			result.ToolCalls = append(result.ToolCalls, llm.ToolCalls{
				Id:       toolCall.ID,
				ToolName: toolCall.Function.Name,
				ToolArgs: args,
			})
		}
	}

	return result, nil
}

// convertToOpenAIMessages converts generic messages to OpenAI format
func (c *OpenAIClient) convertToOpenAIMessages(messages []llm.Message) ([]openai.ChatCompletionMessage, error) {
	var openaiMessages []openai.ChatCompletionMessage

	for _, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role: msg.Role,
		}

		// Handle content with media
		if len(msg.Media) > 0 {
			// Multi-part content with image
			parts := []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: msg.Content,
				},
			}

			// Add image part
			imageURL := fmt.Sprintf("data:%s;base64,%s", msg.MimeType, base64.StdEncoding.EncodeToString(msg.Media))
			parts = append(parts, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL:    imageURL,
					Detail: openai.ImageURLDetailAuto,
				},
			})

			openaiMsg.MultiContent = parts
		} else {
			// Simple text content
			openaiMsg.Content = msg.Content
		}

		// Handle tool calls
		for _, toolCall := range msg.ToolCalls {
			args, err := json.Marshal(toolCall.ToolArgs)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool arguments: %w", err)
			}

			openaiMsg.ToolCalls = append(openaiMsg.ToolCalls, openai.ToolCall{
				ID:   toolCall.Id,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      toolCall.ToolName,
					Arguments: string(args),
				},
			})
		}

		// Handle tool results
		for _, toolResult := range msg.ToolResults {
			// Add tool result as a separate message
			toolResultMsg := openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    toolResult.Content,
				ToolCallID: toolResult.Id,
			}
			openaiMessages = append(openaiMessages, toolResultMsg)
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	return openaiMessages, nil
}

// GetName returns the provider name
func (c *OpenAIClient) GetName() string {
	return "openai"
}

// SetConfig updates the client configuration
func (c *OpenAIClient) SetConfig(config map[string]any) error {
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
		// Recreate client with new API key
		clientConfig := openai.DefaultConfig(apiKey)
		if c.config.BaseURL != "" {
			clientConfig.BaseURL = c.config.BaseURL
		}
		if c.config.OrgID != "" {
			clientConfig.OrgID = c.config.OrgID
		}
		c.client = openai.NewClientWithConfig(clientConfig)
	}
	if maxRetries, ok := config["maxRetries"].(int); ok {
		c.config.MaxRetries = maxRetries
	}
	if baseURL, ok := config["baseURL"].(string); ok {
		c.config.BaseURL = baseURL
		// Recreate client with new base URL
		clientConfig := openai.DefaultConfig(c.config.APIKey)
		clientConfig.BaseURL = baseURL
		if c.config.OrgID != "" {
			clientConfig.OrgID = c.config.OrgID
		}
		c.client = openai.NewClientWithConfig(clientConfig)
	}
	if orgID, ok := config["orgID"].(string); ok {
		c.config.OrgID = orgID
		// Recreate client with new org ID
		clientConfig := openai.DefaultConfig(c.config.APIKey)
		if c.config.BaseURL != "" {
			clientConfig.BaseURL = c.config.BaseURL
		}
		clientConfig.OrgID = orgID
		c.client = openai.NewClientWithConfig(clientConfig)
	}
	if rateLimit, ok := config["rateLimit"].(int); ok {
		c.config.RateLimit = rateLimit
	}
	if rateLimitInterval, ok := config["rateLimitInterval"].(time.Duration); ok {
		c.config.RateLimitInterval = rateLimitInterval
	}
	if maxTokens, ok := config["maxTokens"].(int); ok {
		c.config.MaxTokens = maxTokens
	}
	if topP, ok := config["topP"].(float32); ok {
		c.config.TopP = topP
	}
	if freqPenalty, ok := config["frequencyPenalty"].(float32); ok {
		c.config.FrequencyPenalty = freqPenalty
	}
	if presPenalty, ok := config["presencePenalty"].(float32); ok {
		c.config.PresencePenalty = presPenalty
	}

	return nil
}

// NewOpenAIClient creates a new OpenAI client with the provided configuration
func NewOpenAIClient(ctx context.Context, config *Config) (*OpenAIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create OpenAI client configuration
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	if config.OrgID != "" {
		clientConfig.OrgID = config.OrgID
	}

	// Create the OpenAI client
	openaiClient := openai.NewClientWithConfig(clientConfig)

	client := &OpenAIClient{
		client: openaiClient,
		config: config,
	}

	// Initialize rate limiter only if rate limiting is enabled
	if config.RateLimit > 0 {
		tokens := make(chan struct{}, config.RateLimit)
		rateLimiter := time.NewTicker(config.RateLimitInterval / time.Duration(config.RateLimit))

		// Fill initial tokens
		for i := 0; i < config.RateLimit; i++ {
			tokens <- struct{}{}
		}

		client.rateLimiter = rateLimiter
		client.tokens = tokens

		// Start token refill goroutine
		go client.refillTokens()
	}

	return client, nil
}

// NewOpenAIClientFromEnv creates a new OpenAI client using environment variables
func NewOpenAIClientFromEnv(ctx context.Context) (*OpenAIClient, error) {
	config, err := NewConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from environment: %w", err)
	}

	return NewOpenAIClient(ctx, config)
}

// refillTokens runs in a goroutine to refill the token bucket at the configured rate
func (c *OpenAIClient) refillTokens() {
	for range c.rateLimiter.C {
		select {
		case c.tokens <- struct{}{}:
			// Token added successfully
		default:
			// Token bucket is full, skip
		}
	}
}

// Close stops the rate limiter and cleans up resources
func (c *OpenAIClient) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}
