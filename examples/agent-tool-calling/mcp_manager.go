package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/client"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/alt-coder/pocketflow-go/llm"
)

// MCPToolManager manages MCP client connections and tool discovery
type MCPToolManager struct {
	clients    map[string]*client.Client            // MCP clients by server name
	transports map[string]transport.ClientTransport // Transport connections
	tools      map[string]ToolSchema                // Available tools
	mu         sync.RWMutex                         // Thread safety
	config     *MCPConfig                           // MCP configuration
}

// NewMCPToolManager creates a new MCP tool manager
func NewMCPToolManager(config *MCPConfig) (*MCPToolManager, error) {
	if config == nil {
		config = &MCPConfig{
			Servers: make(map[string]MCPServerConfig),
		}
	}

	return &MCPToolManager{
		clients:    make(map[string]*client.Client),
		transports: make(map[string]transport.ClientTransport),
		tools:      make(map[string]ToolSchema),
		config:     config,
	}, nil
}

// Initialize initializes all MCP server connections
func (m *MCPToolManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize each configured MCP server
	for serverName, serverConfig := range m.config.Servers {
		if err := m.initializeServer(ctx, serverName, serverConfig); err != nil {
			fmt.Printf("Warning: Failed to initialize MCP server '%s': %v\n", serverName, err)
			continue
		}
	}

	return nil
}

// initializeServer initializes a single MCP server connection
func (m *MCPToolManager) initializeServer(ctx context.Context, serverName string, config MCPServerConfig) error {
	// Create transport based on configuration
	var t transport.ClientTransport
	var err error

	// For now, we only support stdio transport like in the example
	if config.Command != "" {
		t, err = transport.NewStdioClientTransport(config.Command, config.Args)
		if err != nil {
			return fmt.Errorf("failed to create stdio transport: %w", err)
		}
	} else {
		return fmt.Errorf("no transport configuration found for server %s", serverName)
	}

	// Create MCP client
	cli, err := client.NewClient(t, client.WithClientInfo(&protocol.Implementation{
		Name:    "agent-tool-calling",
		Version: "1.0.0",
	}))
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Store client and transport
	m.clients[serverName] = cli
	m.transports[serverName] = t

	// Discover tools from this server
	if err := m.discoverTools(ctx, serverName, cli); err != nil {
		fmt.Printf("Warning: Failed to discover tools from server '%s': %v\n", serverName, err)
		// Don't return error, continue with other servers
	}

	return nil
}

// discoverTools discovers available tools from an MCP server
func (m *MCPToolManager) discoverTools(ctx context.Context, serverName string, cli *client.Client) error {
	// Create context with timeout
	toolCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// List available tools
	toolsResponse, err := cli.ListTools(toolCtx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert MCP tools to our ToolSchema format
	for _, tool := range toolsResponse.Tools {
		toolSchema := ToolSchema{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.InputSchema.Properties,
		}

		// Store tool with server prefix to avoid conflicts
		toolKey := fmt.Sprintf("%s.%s", serverName, tool.Name)
		m.tools[toolKey] = toolSchema

		// Also store without prefix for convenience (last server wins in case of conflicts)
		m.tools[tool.Name] = toolSchema
	}

	return nil
}

// GetAvailableTools returns all available tools
func (m *MCPToolManager) GetAvailableTools() []ToolSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]ToolSchema, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteTool executes a tool call
func (m *MCPToolManager) ExecuteTool(ctx context.Context, toolCall llm.ToolCalls) (llm.ToolResults, error) {
	m.mu.RLock()
	_, exists := m.tools[toolCall.ToolName]
	m.mu.RUnlock()

	if !exists {
		return llm.ToolResults{
			Id: toolCall.Id,
			Content:    "",
			IsError:    true,
			Error:      fmt.Sprintf("Tool '%s' not found", toolCall.ToolName),
		}, nil
	}

	// Try to execute with real MCP client first
	if len(m.clients) > 0 {
		result, err := m.executeRealTool(ctx, toolCall)
		if err == nil {
			return result, nil
		}
		fmt.Printf("Real tool execution failed, falling back to mock: %v\n", err)
	}

	return llm.ToolResults{
		Id: toolCall.Id,
		Content:    "",
		IsError:    true,
		Error:      fmt.Sprintf("Tool '%s' not found", toolCall.ToolName),
	}, nil
}

// executeRealTool executes a tool using the actual MCP client
func (m *MCPToolManager) executeRealTool(ctx context.Context, toolCall llm.ToolCalls) (llm.ToolResults, error) {
	// Find which client has this tool
	var targetClient *client.Client
	for serverName, cli := range m.clients {
		// Check if this server has the tool
		toolKey := fmt.Sprintf("%s.%s", serverName, toolCall.ToolName)
		if _, exists := m.tools[toolKey]; exists {
			targetClient = cli
			break
		}
	}

	// If no specific server found, try the first available client
	if targetClient == nil && len(m.clients) > 0 {
		for _, cli := range m.clients {
			targetClient = cli
			break
		}
	}

	if targetClient == nil {
		return llm.ToolResults{}, fmt.Errorf("no MCP client available for tool %s", toolCall.ToolName)
	}

	// Create tool request
	request := &protocol.CallToolRequest{
		Name:      toolCall.ToolName,
		Arguments: toolCall.ToolArgs,
	}

	// Execute tool with timeout
	toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := targetClient.CallTool(toolCtx, request)

	if err != nil {
		return llm.ToolResults{}, fmt.Errorf("MCP tool execution failed: %w", err)
	}
	
	var toolCallResult llm.ToolResults
	for _, contentItem := range result.Content {
		switch contentItem.GetType() {
		case "text": toolCallResult.Content += contentItem.(*protocol.TextContent).Text + "\n"
		case "image":
			toolCallResult.Media = contentItem.(*protocol.ImageContent).Data
			toolCallResult.MetaData.ContentType = contentItem.(*protocol.ImageContent).MimeType

		}
	}
	// Convert MCP result to our format
	content := ""
	for _, contentItem := range result.Content {
		if textContent, ok := contentItem.(*protocol.TextContent); ok {
			content += textContent.Text + "\n"
		}
	}

	return llm.ToolResults{
		Id: toolCall.Id,
		Content:    content,
		IsError:    result.IsError,
	}, nil
}

// Close closes all MCP connections and cleans up resources
func (m *MCPToolManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all MCP clients
	for serverName, cli := range m.clients {
		if err := cli.Close(); err != nil {
			fmt.Printf("Warning: Failed to close MCP client '%s': %v\n", serverName, err)
		}
	}

	// Clear all data
	m.clients = make(map[string]*client.Client)
	m.transports = make(map[string]transport.ClientTransport)
	m.tools = make(map[string]ToolSchema)

	return nil
}
