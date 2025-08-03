# OpenAI LLM Client

This package provides an OpenAI client implementation for the PocketFlow LLM interface using the official `github.com/sashabaranov/go-openai` client library.

## Features

- **Official OpenAI Client**: Built on the reliable `github.com/sashabaranov/go-openai` library
- **Full OpenAI API Support**: Chat completions, tool calls, and media handling
- **Rate Limiting**: Built-in token bucket rate limiting
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Environment Configuration**: Easy setup using environment variables
- **Tool Support**: Full function calling capabilities
- **Media Support**: Image input support with base64 encoding
- **Comprehensive Testing**: Full test coverage

## Quick Start

### Using Environment Variables

```go
import (
    "context"
    "github.com/alt-coder/pocketflow-go/llm/openai"
)

// Create client from environment variables
client, err := openai.NewOpenAIClientFromEnv(context.Background())
if err != nil {
    log.Fatal(err)
}

// Use the client
messages := []llm.Message{
    {
        Role:    llm.RoleUser,
        Content: "Hello, how are you?",
    },
}

response, err := client.CallLLM(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Using Manual Configuration

```go
config := &openai.Config{
    APIKey:      "your-api-key",
    Model:       "gpt-4o",
    Temperature: 0.7,
    MaxRetries:  3,
    BaseURL:     "https://api.openai.com/v1",
}

client, err := openai.NewOpenAIClient(context.Background(), config)
if err != nil {
    log.Fatal(err)
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key (required) | - |
| `OPENAI_MODEL` | Model to use | `gpt-4o` |
| `OPENAI_TEMPERATURE` | Response creativity (0.0-2.0) | `0.7` |
| `OPENAI_MAX_RETRIES` | Maximum retry attempts | `3` |
| `OPENAI_BASE_URL` | API base URL | `https://api.openai.com/v1` |
| `OPENAI_ORG_ID` | Organization ID (optional) | - |
| `OPENAI_RATE_LIMIT` | Requests per minute (0=disabled) | `0` |
| `OPENAI_RATE_LIMIT_INTERVAL_SECONDS` | Rate limit window | `60` |
| `OPENAI_MAX_TOKENS` | Maximum response tokens (0=no limit) | `0` |
| `OPENAI_TOP_P` | Nucleus sampling parameter | `1.0` |
| `OPENAI_FREQUENCY_PENALTY` | Frequency penalty (-2.0 to 2.0) | `0.0` |
| `OPENAI_PRESENCE_PENALTY` | Presence penalty (-2.0 to 2.0) | `0.0` |

## Configuration

### Basic Configuration

```go
config := &openai.Config{
    APIKey:      "sk-...",
    Model:       "gpt-4o",
    Temperature: 0.7,
    MaxRetries:  3,
}
```

### Advanced Configuration

```go
config := &openai.Config{
    APIKey:           "sk-...",
    Model:            "gpt-4o",
    Temperature:      0.7,
    MaxRetries:       3,
    BaseURL:          "https://api.openai.com/v1",
    OrgID:            "org-...",
    MaxTokens:        1000,
    TopP:             0.9,
    FrequencyPenalty: 0.1,
    PresencePenalty:  0.1,
    
    // Rate limiting
    RateLimit:         60,                    // 60 requests per minute
    RateLimitInterval: time.Minute,           // 1 minute window
}
```

## Tool Calling

The client supports OpenAI's function calling feature:

```go
// The LLM can make tool calls
response, err := client.CallLLM(ctx, messages)
if err != nil {
    log.Fatal(err)
}

// Check for tool calls
for _, toolCall := range response.ToolCalls {
    fmt.Printf("Tool: %s, Args: %v\n", toolCall.ToolName, toolCall.ToolArgs)
    
    // Execute your tool and create a result message
    toolResult := llm.ToolResults{
        Id:      toolCall.Id,
        Content: "Tool execution result",
        IsError: false,
    }
    
    // Add tool result to conversation
    messages = append(messages, llm.Message{
        Role:        llm.RoleUser,
        ToolResults: []llm.ToolResults{toolResult},
    })
}
```

## Media Support

Send images to vision-capable models:

```go
// Read image file
imageData, err := os.ReadFile("image.jpg")
if err != nil {
    log.Fatal(err)
}

message := llm.Message{
    Role:     llm.RoleUser,
    Content:  "What's in this image?",
    Media:    imageData,
    MimeType: "image/jpeg",
}

response, err := client.CallLLM(ctx, []llm.Message{message})
```

## Rate Limiting

Enable rate limiting to respect API limits:

```go
config := &openai.Config{
    APIKey:            "sk-...",
    Model:             "gpt-4o",
    RateLimit:         60,           // 60 requests per minute
    RateLimitInterval: time.Minute,  // 1 minute window
}
```

## Error Handling

The client provides detailed error information:

```go
response, err := client.CallLLM(ctx, messages)
if err != nil {
    // Handle different error types
    if strings.Contains(err.Error(), "rate limit") {
        // Handle rate limit error
        time.Sleep(time.Minute)
        // Retry...
    } else if strings.Contains(err.Error(), "API error") {
        // Handle API error
        log.Printf("API error: %v", err)
    } else {
        // Handle other errors
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Supported Models

The client works with all OpenAI chat models:

- `gpt-4o` (default)
- `gpt-4o-mini`
- `gpt-4-turbo`
- `gpt-4`
- `gpt-3.5-turbo`

And vision models for image input:
- `gpt-4o`
- `gpt-4-turbo`
- `gpt-4-vision-preview`

## Testing

Run the tests:

```bash
go test ./llm/openai/...
```

The tests include:
- Configuration validation
- Message conversion
- API call simulation
- Error handling
- Tool call processing
- Rate limiting

## Custom Base URL

For OpenAI-compatible APIs (like Azure OpenAI):

```go
config := &openai.Config{
    APIKey:  "your-key",
    Model:   "gpt-4",
    BaseURL: "https://your-custom-endpoint.com/v1",
}
```

## Thread Safety

The client is thread-safe and can be used concurrently from multiple goroutines. Rate limiting is handled safely across concurrent requests.

## Resource Cleanup

Remember to close the client when done:

```go
defer client.Close()
```

This stops the rate limiter and cleans up resources.