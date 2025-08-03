# Tools Package

This package provides a unified tool management system for the PocketFlow framework, supporting both local function calls and MCP (Model Context Protocol) tool integration.

## Overview

The tools package offers:

- **Unified Tool Manager**: Single interface for managing both local and MCP tools
- **Reflection-Based Tools**: Automatically generate tool schemas from any Go function
- **MCP Integration**: Connect to MCP servers for external tool capabilities
- **Automatic Type Conversion**: Convert JSON parameters to Go types automatically
- **Parameter Validation**: Automatic validation of tool parameters and types
- **Schema Generation**: Generate tool schemas suitable for LLM prompts
- **Thread Safety**: Safe for concurrent use across multiple goroutines

## Quick Start

```go
import "github.com/alt-coder/pocketflow-go/tools"

// Create tool manager
tm := tools.NewToolManager()

// Add common tools (file ops, calculator, etc.)
tools.SetupCommonTools(tm)

// Add custom function as tool using reflection
func Greet(name string) string {
    return fmt.Sprintf("Hello %s!", name)
}

tm.AddFunction("greet", "Generate a greeting", Greet)

// Execute tool
result, err := tm.ExecuteTool(ctx, llm.ToolCalls{
    Id: "1", 
    ToolName: "greet", 
    ToolArgs: map[string]interface{}{"param1": "World"},
})
```

## Components

### ToolManager
The main interface for managing and executing tools. Handles routing between local and MCP tools.

### MCPManager  
Manages connections to MCP servers and discovers available tools.

### LocalTool
Represents a locally defined tool with custom handler function.

### Parameter Validation
Automatic validation of tool parameters including type checking, required fields, and enum values.

## Examples

See `examples/tool-manager/` for a complete working example demonstrating:
- Local tool creation
- MCP server integration  
- Tool execution
- Schema generation for LLMs

## API Reference

### ToolManager Methods

- `NewToolManager()` - Create a new tool manager
- `AddFunction(name, description, fn)` - Add any Go function as a tool
- `AddFunctionWithParams(name, description, fn, paramConfig)` - Add function with custom parameter names
- `AddLocalTool(tool LocalTool)` - Add a local tool (deprecated - use AddFunction)
- `SetMCPManager(mcpManager *MCPManager)` - Set MCP manager for external tools
- `GetAvailableTools()` - Get all available tools (local + MCP)
- `ExecuteTool(ctx, toolCall)` - Execute a tool call
- `HasTool(toolName)` - Check if a tool exists
- `RemoveLocalTool(toolName)` - Remove a local tool
- `Close()` - Clean up resources

### MCPManager Methods

- `NewMCPManager(config *MCPConfig)` - Create MCP manager
- `Initialize(ctx)` - Initialize MCP server connections
- `AddServer(ctx, serverName, config)` - Add new MCP server
- `RemoveServer(serverName)` - Remove MCP server
- `GetAvailableTools()` - Get MCP tools
- `ExecuteTool(ctx, toolCall)` - Execute MCP tool
- `Close()` - Close all connections

### Helper Functions

- `SetupCommonTools(tm *ToolManager)` - Add built-in tools
- `GenerateToolSchemaForLLM(tools []ToolSchema)` - Generate LLM-friendly schema
- `CreateFileWriteTool()` - File writing tool
- `CreateFileReadTool()` - File reading tool  
- `CreateExecuteCommandTool()` - Command execution tool
- `CreateGetTimeTool()` - Time/date tool
- `CreateCalculatorTool()` - Basic calculator tool