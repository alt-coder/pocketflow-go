package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm/gemini"
	"github.com/alt-coder/pocketflow-go/tools"
)

func main() {
	// Load configuration
	config, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Printf(" Using Gemini : %s with model %s\n", config.LLM.Provider, config.LLM.Model)
	// Initialize MCP tool manager
	mcpManager := tools.NewMCPManager(config.MCP)
	if err != nil {
		log.Fatalf("Failed to create MCP tool manager: %v", err)
	}
	defer mcpManager.Close()

	// Initialize MCP connections
	ctx := context.Background()
	if err := mcpManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize MCP tool manager: %v", err)
	}
	toolManager := tools.NewToolManager()
	toolManager.SetMCPManager(mcpManager)


	// Initialize real Gemini LLM provider
	geminiConfig := &gemini.Config{
		APIKey:            config.LLM.APIKey,
		Model:             config.LLM.Model,
		Temperature:       config.LLM.Temperature,
		MaxRetries:        3,
		RateLimit:         1,
		RateLimitInterval: 8 * time.Second,
	}

	llmProvider, err := gemini.NewGeminiClient(ctx, geminiConfig)
	if err != nil {
		log.Fatalf("Failed to create Gemini LLM client: %v", err)
	}
	agentState := &AgentState{}
	workflow := NewToolUsageFlow(toolManager,llmProvider, nil, agentState)
	workflow.AddSuccessor(workflow, core.ActionSuccess)

	// Display welcome message
	fmt.Println("🤖 Agent with Tool Calling Capabilities")
	fmt.Println("=====================================")
	fmt.Printf("Available tools: ")
	for i, tool := range toolManager.GetAvailableTools() {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(tool.Name)
	}
	fmt.Println()
	fmt.Println("Type your requests below. The agent will ask for approval before using tools.")
	fmt.Println()

	workflow.Run(&agentState)

	fmt.Println("Agent session ended.")
}

// loadConfiguration loads the agent configuration from file or environment
func loadConfiguration() (*AgentWorkflowConfig, error) {
	// Try to load from config file first
	if _, err := os.Stat("config.json"); err == nil {
		config, err := LoadConfig("config.json")
		if err != nil {
			return nil, fmt.Errorf("failed to load config.json: %w", err)
		}
		return config, nil
	}

	// Fall back to environment variables
	fmt.Println("No config.json found, using environment variables and defaults.")
	config := LoadConfigFromEnv()

	// Validate required configuration
	if config.LLM.APIKey == "" {
		fmt.Println("Warning: No LLM API key configured. Using mock responses.")
	}

	return config, nil
}

// Example configuration file (config.json):
/*
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
*/
