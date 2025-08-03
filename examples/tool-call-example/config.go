package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/alt-coder/pocketflow-go/tools"
)

// AgentWorkflowConfig represents the complete configuration for the agent workflow
type AgentWorkflowConfig struct {
	Agent *AgentConfig     `json:"agent"`
	MCP   *tools.MCPConfig `json:"mcp"`
	LLM   *LLMConfig       `json:"llm"`
}

// AgentConfig represents the main agent configuration
type AgentConfig struct {
	MaxToolCalls int    `json:"max_tool_calls"` // Maximum tool calls per turn
	MaxHistory   int    `json:"max_history"`    // Maximum conversation history
	SystemPrompt string `json:"system_prompt"`  // System prompt for the agent
}

// MCPServerConfig represents configuration for a single MCP server
type MCPServerConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
}

// LLMConfig represents LLM provider configuration
type LLMConfig struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url,omitempty"` // Custom base URL for OpenAI-compatible APIs
	Temperature float32 `json:"temperature"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*AgentWorkflowConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config AgentWorkflowConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults
	applyDefaults(&config)
	return &config, nil
}

// LoadConfigFromEnv loads configuration from environment variables with defaults
func LoadConfigFromEnv() *AgentWorkflowConfig {
	config := &AgentWorkflowConfig{}
	applyDefaults(config)

	// Override with environment variables if present
	// Check for OpenAI API key first, then Gemini
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
		config.LLM.Provider = "openai"
		config.LLM.Model = "gpt-4o"

		// Set custom base URL if provided
		if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
			config.LLM.BaseURL = baseURL
		}
	} else if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
		config.LLM.Provider = "gemini"
		config.LLM.Model = "gemini-2.0-flash"
	}

	return config
}

// applyDefaults applies default values to the configuration
func applyDefaults(config *AgentWorkflowConfig) {
	if config.Agent == nil {
		config.Agent = &AgentConfig{}
	}
	if config.Agent.MaxToolCalls == 0 {
		config.Agent.MaxToolCalls = 5
	}
	if config.Agent.MaxHistory == 0 {
		config.Agent.MaxHistory = 20
	}
	if config.Agent.SystemPrompt == "" {
		config.Agent.SystemPrompt = "You are a helpful assistant with who might have access to various tools. Use tools(if available) when necessary to help the user accomplish their tasks."
	}

	if config.MCP == nil {
		config.MCP = &tools.MCPConfig{
			Servers: make(map[string]tools.MCPServerConfig),
		}
	}

	if config.LLM == nil {
		config.LLM = &LLMConfig{}
	}
	if config.LLM.Provider == "" {
		config.LLM.Provider = "openai" // Default to OpenAI
	}
	if config.LLM.Model == "" {
		// Set default model based on provider
		switch config.LLM.Provider {
		case "openai":
			config.LLM.Model = "gpt-4o"
		case "gemini":
			config.LLM.Model = "gemini-2.0-flash"
		default:
			config.LLM.Model = "gpt-4o" // Default fallback
		}
	}
	if config.LLM.Temperature == 0 {
		config.LLM.Temperature = 0.7
	}

}
