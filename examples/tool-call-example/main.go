package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/llm/gemini"
	"github.com/alt-coder/pocketflow-go/llm/openai"
	"github.com/alt-coder/pocketflow-go/tools"
)

func main() {
	// Load configuration
	config, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Printf("Using %s provider with model %s\n", config.LLM.Provider, config.LLM.Model)
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

	// Initialize LLM provider based on configuration
	llmProvider, err := createLLMProvider(ctx, config.LLM)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}
	defer closeLLMProvider(llmProvider)
	agentState := &AgentState{}
	workflow := NewToolUsageFlow(toolManager, llmProvider, nil, agentState)
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

// createLLMProvider creates the appropriate LLM provider based on configuration
func createLLMProvider(ctx context.Context, config *LLMConfig) (llm.LLMProvider, error) {
	switch strings.ToLower(config.Provider) {
	case "openai":
		// Use custom base URL if provided, otherwise default to OpenAI
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}

		openaiConfig := &openai.Config{
			APIKey:            config.APIKey,
			Model:             config.Model,
			Temperature:       config.Temperature,
			MaxRetries:        3,
			BaseURL:           baseURL,
			RateLimit:         30, // 60 requests per minute
			RateLimitInterval: time.Minute,
		}
		return openai.NewOpenAIClient(ctx, openaiConfig)

	case "gemini":
		geminiConfig := &gemini.Config{
			APIKey:            config.APIKey,
			Model:             config.Model,
			Temperature:       config.Temperature,
			MaxRetries:        3,
			RateLimit:         1,
			RateLimitInterval: 8 * time.Second,
		}
		return gemini.NewGeminiClient(ctx, geminiConfig)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s. Supported providers: openai, gemini", config.Provider)
	}
}

// closeLLMProvider safely closes the LLM provider if it supports closing
func closeLLMProvider(provider llm.LLMProvider) {
	// Check if the provider has a Close method and call it
	if closer, ok := provider.(interface{ Close() }); ok {
		closer.Close()
	}
}

// Example configuration file (config.json):
/*
{
  "agent": {
    "max_tool_calls": 5,
    "max_history": 20,
    "system_prompt": "You are a helpful assistant with access to various tools. Use tools when necessary to help the user accomplish their tasks."
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
    "provider": "openai",
    "model": "gpt-4o",
    "api_key": "${OPENAI_API_KEY}",
    "base_url": "https://api.openai.com/v1",
    "temperature": 0.2
  }
}

Environment Variables:
- OPENAI_API_KEY: Your OpenAI API key (required)
- OPENAI_BASE_URL: Custom OpenAI-compatible API endpoint (optional)
- SEARCH_API_KEY: Your search API key (if using web search MCP server)

Examples of custom base URLs:
- Azure OpenAI: "https://your-resource.openai.azure.com/openai/deployments/your-deployment"
- Local LLM: "http://localhost:1234/v1"
- Other OpenAI-compatible APIs: "https://api.groq.com/openai/v1"
*/
