package tools

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

// MCPManager manages MCP client connections and tool discovery
type MCPManager struct {
	clients    map[string]*client.Client            // MCP clients by server name
	transports map[string]transport.ClientTransport // Transport connections
	tools      map[string]MCPToolSchema             // Available tools
	mu         sync.RWMutex                         // Thread safety
	config     *MCPConfig                           // MCP configuration
}

// MCPToolSchema represents an MCP tool schema
type MCPToolSchema struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Parameters  map[string]*protocol.Property `json:"parameters"`
	ServerName  string                        `json:"server_name"`
}

// MCPConfig represents MCP configuration
type MCPConfig struct {
	Servers map[string]MCPServerConfig `json:"servers"`
}

// MCPServerConfig represents configuration for a single MCP server
type MCPServerConfig struct {
	Command  string            `json:"command"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env"`
	Disabled bool              `json:"disabled"`
}

// NewMCPManager creates a new MCP manager
func NewMCPManager(config *MCPConfig) (*MCPManager) {
	if config == nil {
		config = &MCPConfig{
			Servers: make(map[string]MCPServerConfig),
		}
	}

	return &MCPManager{
		clients:    make(map[string]*client.Client),
		transports: make(map[string]transport.ClientTransport),
		tools:      make(map[string]MCPToolSchema),
		config:     config,
	}
}

// Initialize initializes all MCP server connections
func (m *MCPManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize each configured MCP server
	for serverName, serverConfig := range m.config.Servers {
		if serverConfig.Disabled {
			continue
		}

		if err := m.initializeServer(ctx, serverName, serverConfig); err != nil {
			fmt.Printf("Warning: Failed to initialize MCP server '%s': %v\n", serverName, err)
			continue
		}
	}

	return nil
}

// initializeServer initializes a single MCP server connection
func (m *MCPManager) initializeServer(ctx context.Context, serverName string, config MCPServerConfig) error {
	// Create transport based on configuration
	var t transport.ClientTransport
	var err error

	// For now, we only support stdio transport
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
		Name:    "pocketflow-tool-manager",
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
func (m *MCPManager) discoverTools(ctx context.Context, serverName string, cli *client.Client) error {
	// Create context with timeout
	toolCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// List available tools
	toolsResponse, err := cli.ListTools(toolCtx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert MCP tools to our schema format
	for _, tool := range toolsResponse.Tools {
		toolSchema := MCPToolSchema{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.InputSchema.Properties,
			ServerName:  serverName,
		}

		// Store tool with server prefix to avoid conflicts
		toolKey := fmt.Sprintf("%s.%s", serverName, tool.Name)
		m.tools[toolKey] = toolSchema

		// Also store without prefix for convenience (last server wins in case of conflicts)
		m.tools[tool.Name] = toolSchema
	}

	return nil
}

// GetAvailableTools returns all available MCP tools
func (m *MCPManager) GetAvailableTools() []MCPToolSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]MCPToolSchema, 0, len(m.tools))
	seen := make(map[string]bool)

	for _, tool := range m.tools {
		// Avoid duplicates (prefer non-prefixed names)
		if !seen[tool.Name] {
			tools = append(tools, tool)
			seen[tool.Name] = true
		}
	}

	return tools
}

// ExecuteTool executes an MCP tool call
func (m *MCPManager) ExecuteTool(ctx context.Context, toolCall llm.ToolCalls) (llm.ToolResults, error) {
	m.mu.RLock()
	tool, exists := m.tools[toolCall.ToolName]
	m.mu.RUnlock()

	if !exists {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("MCP tool '%s' not found", toolCall.ToolName),
		}, nil
	}

	// Find the client for this tool's server
	m.mu.RLock()
	targetClient, clientExists := m.clients[tool.ServerName]
	m.mu.RUnlock()

	if !clientExists {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("MCP client for server '%s' not available", tool.ServerName),
		}, nil
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
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("MCP tool execution failed: %v", err),
		}, nil
	}

	// Convert MCP result to our format
	var toolResult llm.ToolResults
	toolResult.Id = toolCall.Id
	toolResult.IsError = result.IsError

	// Process content items
	for _, contentItem := range result.Content {
		switch contentItem.GetType() {
		case "text":
			if textContent, ok := contentItem.(*protocol.TextContent); ok {
				toolResult.Content += textContent.Text + "\n"
			}
		case "image":
			if imageContent, ok := contentItem.(*protocol.ImageContent); ok {
				toolResult.Media = imageContent.Data
				toolResult.MetaData.ContentType = imageContent.MimeType
			}
		}
	}

	// Trim trailing newline
	if len(toolResult.Content) > 0 && toolResult.Content[len(toolResult.Content)-1] == '\n' {
		toolResult.Content = toolResult.Content[:len(toolResult.Content)-1]
	}

	return toolResult, nil
}

// HasTool checks if an MCP tool exists
func (m *MCPManager) HasTool(toolName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.tools[toolName]
	return exists
}

// AddServer adds a new MCP server configuration and initializes it
func (m *MCPManager) AddServer(ctx context.Context, serverName string, config MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to configuration
	m.config.Servers[serverName] = config

	// Initialize the server if not disabled
	if !config.Disabled {
		return m.initializeServer(ctx, serverName, config)
	}

	return nil
}

// RemoveServer removes an MCP server and closes its connection
func (m *MCPManager) RemoveServer(serverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close client if exists
	if cli, exists := m.clients[serverName]; exists {
		if err := cli.Close(); err != nil {
			fmt.Printf("Warning: Failed to close MCP client '%s': %v\n", serverName, err)
		}
		delete(m.clients, serverName)
	}

	// Remove transport
	delete(m.transports, serverName)

	// Remove tools from this server
	for toolKey, tool := range m.tools {
		if tool.ServerName == serverName {
			delete(m.tools, toolKey)
		}
	}

	// Remove from configuration
	delete(m.config.Servers, serverName)

	return nil
}

// Close closes all MCP connections and cleans up resources
func (m *MCPManager) Close() error {
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
	m.tools = make(map[string]MCPToolSchema)

	return nil
}
