# Agent with Tool Calling Capabilities

This example demonstrates an agent with tool calling capabilities using the PocketFlowGo framework and Model Context Protocol (MCP) for tool management.

## Features

- **Multi-step tool calling**: Agent can make multiple tool calls in sequence
- **Tool approval system**: User can approve/reject tool executions
- **Conversation summarization**: Automatic history compression when context gets too long
- **Tool result cleanup**: Intelligent cleanup of tool results to maintain context
- **MCP integration**: Uses Model Context Protocol for tool server communication

## Architecture

The agent follows a finite state machine with these nodes:
- **UserInputNode**: Captures user input and determines next action
- **SummarizerNode**: Compresses conversation history when needed
- **PlanningNode**: LLM decides what to do and which tools to use
- **ApprovalNode**: Handles user approval for tool execution
- **ToolExecutionNode**: Executes approved tools via MCP
- **ToolResultCleanupNode**: Cleans up tool results to maintain context

## State Machine Flow

```
[*] --> UserInputNode
UserInputNode --> SummarizerNode : ActionSummarize (Context overflow)
UserInputNode --> PlanningNode : ActionPlan (Valid input)
UserInputNode --> [*] : ActionExit (User request)

SummarizerNode --> PlanningNode : ActionPlan (Success)
SummarizerNode --> [*] : ActionFailure (Critical error)

PlanningNode --> ApprovalNode : ActionRequestApproval (Tool calls detected)
PlanningNode --> UserInputNode : ActionContinue (Final response)
PlanningNode --> [*] : ActionFailure (Exhausted retries)

ApprovalNode --> ToolExecutionNode : ActionApprove ("y" or "a")
ApprovalNode --> PlanningNode : ActionReject ("r")
ApprovalNode --> PlanningNode : ActionAppend (Other input)

ToolExecutionNode --> ToolResultCleanupNode : ActionCleanup (Tools executed)
ToolExecutionNode --> ToolResultCleanupNode : ActionCleanup (Permanent error in tool result)

ToolResultCleanupNode --> PlanningNode : ActionContinue (Cleanup complete)
```

## Usage

### Basic Usage

```bash
go run .
```

### With Configuration File

Create a `config.json` file:

```json
{
  "agent": {
    "max_tool_calls": 5,
    "tool_timeout": "30s",
    "max_history": 20,
    "system_prompt": "You are a helpful assistant with access to various tools...",
    "temperature": 0.7
  },
  "mcp": {
    "servers": {
      "filesystem": {
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "~/workspace"],
        "timeout": "10s"
      },
      "web_search": {
        "command": "uvx",
        "args": ["mcp-server-web-search"],
        "env": {
          "SEARCH_API_KEY": "${SEARCH_API_KEY}"
        }
      }
    }
  },
  "llm": {
    "provider": "gemini",
    "model": "gemini-2.0-flash",
    "api_key": "${GOOGLE_API_KEY}",
    "temperature": 0.7
  }
}
```

### Environment Variables

Set the following environment variables:
- `GOOGLE_API_KEY`: Your Google API key for Gemini
- `SEARCH_API_KEY`: Your search API key (if using web search tools)

## Tool Approval Commands

When the agent wants to use tools, you can respond with:
- `y` or `yes`: Approve tool execution for this time
- `a` or `always`: Approve and grant permanent permission for these tools
- `r`, `reject`, `n`, or `no`: Reject tool execution
- Any other text: Add additional context to the conversation

## Example Interactions

### File Operations
```
You: List the files in my current directory
Assistant: I'll list the files in the directory for you.
Tools to be used: list_directory
The assistant wants to use tools. Approve? (y/n/a for always): y
Tools approved for this use.
Executing tool: list_directory
Tool list_directory result: [file1.txt, file2.py, folder1/]
```

### Web Search
```
You: Search for information about machine learning
Assistant: I'll search the web for information about machine learning.
Tools to be used: web_search
The assistant wants to use tools. Approve? (y/n/a for always): y
Tools approved for this use.
Executing tool: web_search
Tool web_search result: Found 5 results about machine learning...
```

## Configuration Options

### Agent Configuration
- `max_tool_calls`: Maximum tool calls per turn (default: 5)
- `tool_timeout`: Timeout for tool execution (default: 30s)
- `max_history`: Maximum conversation history length (default: 20)
- `system_prompt`: System prompt for the agent
- `temperature`: LLM temperature (default: 0.7)

### MCP Configuration
- `servers`: Map of MCP server configurations
- Each server needs `command`, `args`, and optional `env` variables

### LLM Configuration
- `provider`: LLM provider (currently supports "gemini")
- `model`: Model name (default: "gemini-2.0-flash")
- `api_key`: API key for the LLM provider
- `temperature`: LLM temperature

## Dependencies

- Go 1.21+
- Google Gemini API access
- MCP servers (optional, for tool functionality)

## MCP Server Setup

To use tools, you'll need MCP servers. Some popular ones:

### Filesystem Server
```bash
npx -y @modelcontextprotocol/server-filesystem ~/workspace
```

### Web Search Server
```bash
uvx mcp-server-web-search
```

### Git Server
```bash
uvx mcp-server-git
```

## Troubleshooting

### No Tools Available
- Check that MCP servers are properly configured and running
- Verify the command paths and arguments in your configuration
- Check server logs for connection issues

### LLM Errors
- Verify your API key is set correctly
- Check your API quota and rate limits
- Ensure you have access to the specified model

### Tool Execution Failures
- Check tool parameters are valid
- Verify MCP server is responding
- Check timeout settings if tools are slow