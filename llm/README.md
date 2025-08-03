# LLM Utilities

Generic LLM provider interfaces and implementations for PocketFlow-Go applications.

## Overview

This package provides a unified interface for working with different Large Language Model providers, making it easy to switch between providers or test with mock implementations.

## Components

### Core Interface (`types.go`)
- `LLMProvider` interface for all LLM implementations
- Generic `Message` struct for cross-provider compatibility
- `Config` struct for provider configuration

### Implementations

#### Gemini Provider (`gemini/`)
- Google Gemini AI integration using the official GenAI library
- Automatic message format conversion
- Environment-based configuration
- Error handling and retry logic

#### OpenAI Provider (`openai/`)
- OpenAI API integration using the official `go-openai` client
- Tool calling and function execution
- Vision model support for image inputs
- Rate limiting with token bucket algorithm
- Comprehensive error handling and retries

#### Mock Provider (`mock.go`)
- Testing-focused implementation
- Configurable response patterns
- Error simulation capabilities
- No external dependencies

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/alt-coder/pocketflow-go/llm"
    "github.com/alt-coder/pocketflow-go/llm/openai"
)

// Create an OpenAI client
client, err := openai.NewOpenAIClientFromEnv(context.Background())
if err != nil {
    log.Fatal(err)
}

// Use the generic interface
messages := []llm.Message{
    {Role: llm.RoleUser, Content: "Hello, how are you?"},
}

response, err := client.CallLLM(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Response:", response.Content)
```

### Configuration

#### Environment Variables

**OpenAI:**
```bash
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_MODEL="gpt-4o"
export OPENAI_TEMPERATURE="0.7"
export OPENAI_MAX_RETRIES="3"
```

**Gemini:**
```bash
export GOOGLE_API_KEY="your-api-key"
export CHAT_MODEL="gemini-2.0-flash"
export CHAT_TEMPERATURE="0.7"
export CHAT_MAX_RETRIES="3"
```

#### Programmatic Configuration

**OpenAI:**
```go
config := &openai.Config{
    APIKey:      "sk-your-api-key",
    Model:       "gpt-4o",
    Temperature: 0.7,
    MaxRetries:  3,
}
client, err := openai.NewOpenAIClient(ctx, config)
```

**Gemini:**
```go
config := &gemini.Config{
    APIKey:      "your-api-key",
    Model:       "gemini-2.0-flash",
    Temperature: 0.7,
    MaxRetries:  3,
}
client, err := gemini.NewGeminiClient(ctx, config)
```

## Testing

### Using Mock Provider

```go
import "github.com/alt-coder/pocketflow-go/examples/llm"

// Create mock provider
mockProvider := llm.NewMockProvider()
mockProvider.SetResponse("Hello! I'm a mock AI assistant.")

// Use like any other provider
response, err := mockProvider.CallLLM(ctx, messages)
```

### Error Simulation

```go
mockProvider := llm.NewMockProvider()
mockProvider.SimulateError(errors.New("API rate limit exceeded"))

// Next call will return the simulated error
_, err := mockProvider.CallLLM(ctx, messages)
// err will be "API rate limit exceeded"
```

## Adding New Providers

To add support for a new LLM provider:

1. Create a new package (e.g., `openai/`)
2. Implement the `LLMProvider` interface
3. Handle message format conversion if needed
4. Add provider-specific configuration
5. Include comprehensive tests

Example structure:
```
llm/
├── types.go              # Core interfaces
├── mock.go               # Mock implementation
├── gemini/
│   ├── client.go         # Gemini implementation
│   ├── config.go         # Gemini configuration
│   └── client_test.go    # Tests
├── openai/
│   ├── client.go         # OpenAI implementation
│   ├── config.go         # OpenAI configuration
│   └── client_test.go    # Tests
└── newprovider/          # Template for new providers
    ├── client.go
    ├── config.go
    └── client_test.go
```

## Message Format

All providers use a common message format:

```go
type Message struct {
    Role        string        // "user", "assistant", "system"
    Content     string        // Message content
    Media       []byte        // Optional media content (images, etc.)
    MimeType    string        // MIME type for media
    ToolCalls   []ToolCalls   // Tool/function calls made by LLM
    ToolResults []ToolResults // Results from tool executions
}
```

Providers handle internal format conversion automatically.

## Error Handling

The package includes comprehensive error handling:

- Network errors with retry logic
- API key validation
- Rate limiting detection
- Graceful degradation options

## Dependencies

- `google.golang.org/genai` - For Gemini integration
- `github.com/sashabaranov/go-openai` - Official OpenAI Go client
- `github.com/stretchr/testify` - For testing utilities (optional)

## Examples

See `examples/basic-chat/` for a complete implementation using these utilities.