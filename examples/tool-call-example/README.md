# Tool Call Example with OpenAI

This example demonstrates how to use PocketFlow-Go with OpenAI's LLM provider and MCP (Model Context Protocol) tools.

## Features

- **OpenAI Integration**: Uses the official OpenAI Go client for reliable API communication
- **Tool Calling**: Supports OpenAI's function calling capabilities
- **MCP Tools**: Integrates with Model Context Protocol servers for extended functionality
- **Permission System**: Asks for user approval before executing tools
- **Rate Limiting**: Built-in rate limiting to respect API limits
- **Flexible Configuration**: Supports both file-based and environment-based configuration

## Setup

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Set Environment Variables

```bash
export OPENAI_API_KEY="sk-your-openai-api-key-here"

# Optional: Use custom OpenAI-compatible endpoint
export OPENAI_BASE_URL="https://api.openai.com/v1"
```

### 3. Configure MCP Tools (Optional)

The example includes Google Sheets MCP server configuration. You can modify `config.json` to add other MCP servers:

```json
{
  "mcp": {
    "servers": {
      "filesystem": {
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "~/workspace"]
      },
      "web-search": {
        "command": "uvx",
        "args": ["mcp-server-web-search"],
        "env": {
          "SEARCH_API_KEY": "${SEARCH_API_KEY}"
        }
      }
    }
  }
}
```

## Running the Example

```bash
go run .
```

## Configuration

### Option 1: Configuration File (config.json)

```json
{
  "agent": {
    "max_tool_calls": 5,
    "max_history": 20,
    "system_prompt": "You are a helpful assistant with access to various tools. Use tools when necessary to help the user accomplish their tasks."
  },
  "mcp": {
    "servers": {
      "google-sheets": {
        "command": "uvx",
        "args": ["mcp-google-sheets@latest"],
        "env": {
          "DRIVE_FOLDER_ID": "your-google-drive-folder-id"
        }
      }
    }
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4o",
    "api_key": "${OPENAI_API_KEY}",
    "base_url": "https://api.openai.com/v1",
    "temperature": 0.2
  }
}
```

### Option 2: Environment Variables

If no `config.json` is found, the application will use environment variables:

```bash
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # Optional: custom endpoint

# Optional: Use Gemini instead
export GOOGLE_API_KEY="your-gemini-api-key"
```

## Supported LLM Providers

### OpenAI (Default)
- **Models**: gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-3.5-turbo
- **Features**: Tool calling, vision (with gpt-4o), streaming
- **Environment**: `OPENAI_API_KEY`

### Gemini (Alternative)
- **Models**: gemini-2.0-flash, gemini-2.5-pro
- **Features**: Tool calling, multimodal
- **Environment**: `GOOGLE_API_KEY`

## Usage Example

1. **Start the application**:
   ```bash
   go run .
   ```

2. **Interact with the agent**:
   ```
   You: Can you help me create a spreadsheet with sales data?
   
   Assistant: I'll help you create a spreadsheet with sales data. Let me use the Google Sheets tool to create this for you.
   
   Tool 'sheets_create' requires permission.
   Arguments: map[title:Sales Data Spreadsheet]
   Allow? [y=yes, n=no, a=always allow]: y
   
   ## Tool sheets_create result:
   Created spreadsheet "Sales Data Spreadsheet" with ID: 1abc...xyz
   
   Assistant: I've successfully created a new Google Sheets spreadsheet called "Sales Data Spreadsheet". The spreadsheet is now ready for you to add your sales data...
   ```

3. **Tool Permission System**:
   - `y` - Allow this specific tool call
   - `n` - Deny this tool call
   - `a` - Always allow this tool (no future prompts)

## Available MCP Tools

The example can be configured with various MCP servers:

### Google Sheets
```bash
uvx mcp-google-sheets@latest
```

### Filesystem
```bash
npx -y @modelcontextprotocol/server-filesystem ~/workspace
```

### Web Search
```bash
uvx mcp-server-web-search
```

### SQLite
```bash
uvx mcp-server-sqlite --db-path ./database.db
```

## Code Structure

- `main.go` - Main application entry point
- `config.go` - Configuration loading and management
- `chat_node.go` - Core chat logic with tool calling
- `state.go` - Conversation state management
- `types.go` - Type definitions

## Error Handling

The application includes comprehensive error handling:

- **API Errors**: Automatic retries with exponential backoff
- **Rate Limiting**: Built-in token bucket rate limiting
- **Tool Failures**: Graceful handling of tool execution errors
- **Configuration Errors**: Clear error messages for setup issues

## Customization

### Adding Custom Tools

You can add local tools in addition to MCP tools:

```go
// In main.go
toolManager.AddLocalTool("custom_tool", "Description", func(input CustomInput) CustomOutput {
    // Your tool logic here
    return CustomOutput{Result: "success"}
})
```

### Modifying System Prompt

Update the system prompt in `config.json`:

```json
{
  "agent": {
    "system_prompt": "You are a specialized assistant for data analysis tasks..."
  }
}
```

### Changing Models

Switch between different models:

```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",  // Faster, cheaper option
    "temperature": 0.1       // More deterministic responses
  }
}
```

### Custom Base URLs

Use OpenAI-compatible APIs or custom endpoints:

#### Azure OpenAI
```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "${AZURE_OPENAI_API_KEY}",
    "base_url": "https://your-resource.openai.azure.com/openai/deployments/your-deployment-name",
    "temperature": 0.7
  }
}
```

#### Local LLM (e.g., Ollama, LM Studio)
```json
{
  "llm": {
    "provider": "openai",
    "model": "llama2",
    "api_key": "not-needed",
    "base_url": "http://localhost:1234/v1",
    "temperature": 0.7
  }
}
```

#### Other OpenAI-compatible APIs
```json
{
  "llm": {
    "provider": "openai",
    "model": "mixtral-8x7b-32768",
    "api_key": "${GROQ_API_KEY}",
    "base_url": "https://api.groq.com/openai/v1",
    "temperature": 0.7
  }
}
```

## Troubleshooting

### Common Issues

1. **API Key Not Set**:
   ```
   Error: OPENAI_API_KEY environment variable is required
   ```
   Solution: Set your OpenAI API key in environment variables

2. **MCP Server Not Found**:
   ```
   Error: Failed to start MCP server
   ```
   Solution: Install the MCP server using `uvx` or `npx`

3. **Rate Limiting**:
   ```
   Error: Rate limit exceeded
   ```
   Solution: The application includes built-in rate limiting, but you may need to adjust the limits in configuration

### Debug Mode

Enable verbose logging by setting:
```bash
export DEBUG=true
```

## Contributing

Feel free to extend this example with additional MCP servers, custom tools, or enhanced features. The modular design makes it easy to add new capabilities.