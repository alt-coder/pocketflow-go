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
    "github.com/alt-coder/pocketflow-go/examples/llm"
    "github.com/alt-coder/pocketflow-go/examples/llm/gemini"
)

// Create a Gemini client
config := gemini.NewConfigFromEnv()
client, err := gemini.NewGeminiClient(context.Background(), config)
if err != nil {
    log.Fatal(err)
}

// Use the generic interface
messages := []llm.Message{
    {Role: "user", Content: "Hello, how are you?"},
}

response, err := client.CallLLM(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Response:", response)
```

### Configuration

#### Environment Variables
```bash
export GOOGLE_API_KEY="your-api-key"
export CHAT_MODEL="gemini-2.0-flash"
export CHAT_TEMPERATURE="0.7"
export CHAT_MAX_RETRIES="3"
```

#### Programmatic Configuration
```go
config := &llm.Config{
    Provider:    "gemini",
    Model:       "gemini-2.0-flash",
    Temperature: 0.7,
    APIKey:      "your-api-key",
}
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
└── openai/               # New provider
    ├── client.go
    ├── config.go
    └── client_test.go
```

## Message Format

All providers use a common message format:

```go
type Message struct {
    Role    string // "user", "assistant", "system"
    Content string // Message content
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
- `github.com/stretchr/testify` - For testing utilities

## Examples

See `examples/basic-chat/` for a complete implementation using these utilities.