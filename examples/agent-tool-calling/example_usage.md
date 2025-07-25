# Agent Tool Calling Example Usage

This document shows example interactions with the agent tool calling system.

## Basic Usage

1. **Start the agent:**
   ```bash
   go run .
   ```

2. **Example conversation:**
   ```
   🤖 Agent with Tool Calling Capabilities
   =====================================
   Available tools: list_directory, read_file, write_file, web_search
   Type your requests below. The agent will ask for approval before using tools.

   You: list the files in the current directory
   
   Assistant: I'll list the files in the directory for you.
   Tools to be used: list_directory
   The assistant wants to use tools. Approve? (y/n/a for always): y
   Tools approved. Executing...

   --- Tool: list_directory ---
   Result: Files in .:
   - file1.txt
   - file2.go
   - directory1/
   - directory2/
   --- End Tool Result ---

   Tool execution completed. Processing results...

   You: read the contents of file1.txt
   
   Assistant: I'll read the file for you.
   Tools to be used: read_file
   The assistant wants to use tools. Approve? (y/n/a for always): y
   Tools approved. Executing...

   --- Tool: read_file ---
   Result: Contents of file1.txt:
   This is mock file content for demonstration purposes.
   --- End Tool Result ---

   Tool execution completed. Processing results...

   You: search the web for "golang best practices"
   
   Assistant: I'll search the web for information about that.
   Tools to be used: web_search
   The assistant wants to use tools. Approve? (y/n/a for always): a
   Always allowing tool: web_search
   Tools approved. Executing...

   --- Tool: web_search ---
   Result: Search results for 'golang best practices':
   1. Mock Result 1 - Example website
   2. Mock Result 2 - Another example
   3. Mock Result 3 - Third example
   --- End Tool Result ---

   Tool execution completed. Processing results...

   You: exit
   Goodbye!
   Agent session ended.
   ```

## Features Demonstrated

### 1. Tool Approval System
- User can approve individual tool calls with `y`
- User can reject tool calls with `n` or `r`
- User can always allow a tool with `a`
- User can provide additional context instead of approving/rejecting

### 2. Multi-step Tool Calling
The agent can make multiple tool calls in sequence to accomplish complex tasks.

### 3. Conversation Summarization
When the conversation gets too long (configurable threshold), the agent automatically summarizes older messages while preserving recent context.

### 4. Tool Result Cleanup
Successful tool results are cleaned up and replaced with summary messages to maintain context without cluttering the conversation.

## Configuration

### Environment Variables
- `GOOGLE_API_KEY`: API key for Gemini LLM (if using real LLM)
- `SEARCH_API_KEY`: API key for web search (if using real search)

### Config File (config.json)
See the included `config.json` for full configuration options including:
- Agent behavior settings
- MCP server configurations
- LLM provider settings
- Node-specific configurations

## Mock vs Real Implementation

This example uses mock implementations for:
- **LLM Provider**: Uses simple keyword-based responses instead of real LLM calls
- **MCP Tools**: Uses mock tool implementations instead of real MCP servers
- **Tool Results**: Returns predefined mock results

To use with real implementations:
1. Replace the mock LLM provider with actual Gemini client
2. Configure real MCP servers in the config
3. Set up proper API keys and authentication

## Error Handling

The system includes comprehensive error handling:
- Tool execution failures are retried with exponential backoff
- Permanent errors (like "file not found") are not retried
- LLM failures fall back to safe default responses
- Network timeouts are handled gracefully

## State Management

The agent maintains:
- Full conversation history
- Tool interaction history
- User permissions for tools
- Current task context
- Summarized conversation history

This allows for complex multi-turn interactions while maintaining context and performance.