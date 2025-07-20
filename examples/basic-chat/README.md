# Basic Chat Example

This example demonstrates PocketFlow-Go's core concepts through a conversational terminal application that integrates with Google's Gemini AI models. It showcases the three-phase execution model (Prep → Exec → Post) with a self-looping chat node that maintains conversation history.

## Features

- **Three-Phase Execution**: Demonstrates PocketFlow-Go's Prep → Exec → Post pattern
- **Self-Looping Flow**: Continuous conversation using PocketFlow-Go's action-based routing
- **LLM Integration**: Uses Google's Gemini models via the GenAI library
- **Conversation History**: Maintains full conversation context with automatic history trimming
- **Error Handling**: Graceful error recovery with user-friendly messages
- **Configurable**: Environment variables and command-line options for customization

## Prerequisites

1. **Go 1.23+** installed on your system
2. **Google API Key** for Gemini access
3. **Internet connection** for API calls

## Setup

### 1. Get a Google API Key

1. Go to [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Copy the API key for use in the next step

### 2. Set Environment Variables

```bash
# Required: Set your Google API key
export GOOGLE_API_KEY="your_api_key_here"

# Optional: Customize the model (default: gemini-2.0-flash)
export CHAT_MODEL="gemini-2.0-flash"

# Optional: Set response creativity (default: 0.7, range: 0.0-1.0)
export CHAT_TEMPERATURE="0.7"

# Optional: Set maximum retry attempts (default: 3)
export CHAT_MAX_RETRIES="3"
```

### 3. Install Dependencies

```bash
cd examples/basic-chat
go mod tidy
```

## Usage

### Basic Usage

```bash
go run .
```

### With Command-Line Options

```bash
# Use a different model
go run . -model="gemini-1.5-pro"

# Adjust creativity/temperature
go run . -temperature=0.9

# Combine options
go run . -model="gemini-1.5-pro" -temperature=0.3
```

### Example Session

```
Starting chat with gemini (model: gemini-2.0-flash, temperature: 0.7)

Welcome to PocketFlow-Go Chat! Type 'exit' to quit.
You: Hello! How are you today?
Assistant: Hello! I'm doing well, thank you for asking. I'm here and ready to help you with any questions or tasks you might have. How are you doing today?

You: Can you explain what PocketFlow-Go is?
Assistant: PocketFlow-Go is a workflow orchestration framework written in Go that helps you build complex, multi-step processes using a node-based architecture. It follows a three-phase execution model:

1. **Prep Phase**: Prepares the work items and context
2. **Exec Phase**: Performs the core processing logic
3. **Post Phase**: Handles results and determines next actions

The framework supports features like retry logic, concurrent execution, error handling, and action-based routing between nodes. It's particularly useful for building data processing pipelines, automation workflows, and applications like this chat example that demonstrate conversational AI integration.

You: exit
Goodbye!
Chat ended with action: success
```

## Architecture

### Components

1. **ChatNode**: Implements the `BaseNode` interface with three-phase execution
2. **ChatState**: Maintains conversation history and active status
3. **PrepResult**: Contains conversation context for LLM calls
4. **ExecResult**: Contains LLM responses and error information
5. **ChatConfig**: Configuration settings for the chat application

### Flow Structure

```go
// Create ChatNode with LLM provider
chatNode := NewChatNode(geminiClient, chatConfig)

// Wrap in PocketFlow-Go Node with retry and concurrency settings
node := core.NewNode[ChatState, PrepResult, ExecResult](chatNode, 3, 1)

// Configure self-loop for continuous conversation
node.AddSuccessor(node, core.ActionContinue)

// Create and run the flow
flow := core.NewFlow[ChatState](node)
flow.Run(initialState)
```

### Three-Phase Execution

#### Prep Phase
- Displays welcome message on first run
- Prompts user for input
- Handles 'exit' command detection
- Prepares conversation context with message history
- Returns `PrepResult` with messages and user input

#### Exec Phase
- Makes API call to Google Gemini using conversation history
- Handles API errors gracefully with proper error wrapping
- Returns `ExecResult` with response or error information

#### Post Phase
- Processes and displays the LLM response
- Updates conversation history with user and assistant messages
- Determines next action (continue loop or terminate)
- Manages conversation state and history trimming

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GOOGLE_API_KEY` | Google API key for Gemini access | - | ✅ |
| `CHAT_MODEL` | Gemini model to use | `gemini-2.0-flash` | ❌ |
| `CHAT_TEMPERATURE` | Response creativity (0.0-1.0) | `0.7` | ❌ |
| `CHAT_MAX_RETRIES` | Maximum retry attempts | `3` | ❌ |

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-model` | Gemini model to use | `gemini-2.0-flash` |
| `-temperature` | Response temperature (0.0-1.0) | `0.7` |

### Application Settings

- **Max History**: 50 messages (automatically trims older messages)
- **Retry Logic**: 3 attempts with exponential backoff
- **Concurrency**: Single-threaded execution for simplicity

## Error Handling

### API Key Validation
- Checks for `GOOGLE_API_KEY` environment variable
- Provides clear setup instructions if missing
- Validates configuration before starting

### API Call Failures
- Implements retry logic through PocketFlow-Go's built-in mechanism
- Handles rate limiting and network errors gracefully
- Provides meaningful error messages to users

### Input Validation
- Handles empty user input gracefully
- Validates 'exit' command case-insensitively
- Maintains conversation state consistency

### Error Recovery
- Uses `ExecFallback` for graceful error recovery
- Continues conversation after API failures
- Provides user-friendly error messages without exposing sensitive details

## Testing

Run the test suite:

```bash
go test -v
```

The tests cover:
- ChatNode creation and configuration
- Three-phase execution model
- Error handling and fallback behavior
- Interface compliance verification

## Troubleshooting

### Common Issues

1. **"Configuration error: GOOGLE_API_KEY environment variable is required"**
   - Solution: Set your Google API key using `export GOOGLE_API_KEY="your_key"`

2. **"Failed to create GenAI client"**
   - Check your API key is valid
   - Ensure you have internet connectivity
   - Verify the API key has proper permissions

3. **"LLM API call failed"**
   - Check your internet connection
   - Verify your API key hasn't expired
   - Check if you've exceeded API rate limits

4. **High response times**
   - Try using a faster model like `gemini-2.0-flash`
   - Check your internet connection speed
   - Consider reducing conversation history length

### Debug Mode

For detailed logging, you can modify the code to add debug output or use Go's built-in logging:

```go
import "log"

// Add to main.go
log.Printf("Using model: %s, temperature: %.1f", geminiConfig.Model, geminiConfig.Temperature)
```

## Extending the Example

### Adding New LLM Providers

1. Implement the `llm.LLMProvider` interface
2. Create provider-specific configuration
3. Update the main.go to support the new provider

### Customizing Conversation Behavior

1. Modify `ChatConfig` to add new settings
2. Update the `Prep`, `Exec`, or `Post` methods in `ChatNode`
3. Add new command handling in the `Prep` phase

### Adding Persistence

1. Implement conversation history saving/loading
2. Add database or file storage integration
3. Modify `ChatState` to include persistence metadata

## Related Examples

- **LLM Package**: Generic LLM provider interface and implementations
- **Core Package**: PocketFlow-Go framework fundamentals
- **Integration Tests**: Advanced workflow patterns and testing strategies

## License

This example is part of the PocketFlow-Go project and follows the same license terms.