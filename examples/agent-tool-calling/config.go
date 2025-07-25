package main

import (
	"encoding/json"
	"os"
	"time"
)

// AgentWorkflowConfig represents the complete configuration for the agent workflow
type AgentWorkflowConfig struct {
	Agent             *AgentConfig         `json:"agent"`
	MCP               *MCPConfig           `json:"mcp"`
	LLM               *LLMConfig           `json:"llm"`
	UserInput         *UserInputConfig     `json:"user_input"`
	Summarizer        *SummarizerConfig    `json:"summarizer"`
	Planning          *PlanningConfig      `json:"planning"`
	Approval          *ApprovalConfig      `json:"approval"`
	ToolExecution     *ToolExecutionConfig `json:"tool_execution"`
	ToolResultCleanup *CleanupConfig       `json:"tool_result_cleanup"`
}

// AgentConfig represents the main agent configuration
type AgentConfig struct {
	MaxToolCalls int           `json:"max_tool_calls"` // Maximum tool calls per turn
	ToolTimeout  time.Duration `json:"tool_timeout"`   // Timeout for tool execution
	MaxHistory   int           `json:"max_history"`    // Maximum conversation history
	SystemPrompt string        `json:"system_prompt"`  // System prompt for the agent
	AllowedTools []string      `json:"allowed_tools"`  // Whitelist of allowed tools (empty = all)
	Temperature  float32       `json:"temperature"`    // LLM temperature
}

// MCPConfig represents MCP server configuration
type MCPConfig struct {
	Servers map[string]MCPServerConfig `json:"servers"`
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
	Temperature float32 `json:"temperature"`
}

// UserInputConfig represents configuration for user input handling
type UserInputConfig struct {
	Prompt                 string   `json:"prompt"`                  // Input prompt to display
	ExitCommands           []string `json:"exit_commands"`           // Commands that trigger exit
	SummarizationThreshold int      `json:"summarization_threshold"` // Message count that triggers summarization
}

// SummarizerConfig represents configuration for conversation summarization
type SummarizerConfig struct {
	PreserveRecentCount int    `json:"preserve_recent_count"` // Number of recent messages to preserve
	SummaryPrompt       string `json:"summary_prompt"`        // Prompt for summarization
	MaxSummaryLength    int    `json:"max_summary_length"`    // Maximum length of summary
}

// PlanningConfig represents configuration for the planning node
type PlanningConfig struct {
	SystemPrompt string   `json:"system_prompt"`
	MaxToolCalls int      `json:"max_tool_calls"`
	Temperature  float32  `json:"temperature"`
	AllowedTools []string `json:"allowed_tools"` // Whitelist of allowed tools
}

// ApprovalConfig represents configuration for tool approval
type ApprovalConfig struct {
	ApprovalPrompt   string `json:"approval_prompt"`   // Prompt to display for approval
	RejectionMessage string `json:"rejection_message"` // Message to add when tools are rejected
}

// ToolExecutionConfig represents configuration for tool execution
type ToolExecutionConfig struct {
	Timeout        time.Duration `json:"timeout"`
	MaxConcurrency int           `json:"max_concurrency"`
	RetryAttempts  int           `json:"retry_attempts"`
	FailureMode    string        `json:"failure_mode"`   // "continue", "abort", "retry"
	MaxToolSteps   int           `json:"max_tool_steps"` // Maximum tool calling iterations per task
}

// CleanupConfig represents configuration for tool result cleanup
type CleanupConfig struct {
	SuccessMessage       string `json:"success_message"`       // Message to replace successful tool calls
	ClassificationPrompt string `json:"classification_prompt"` // Prompt for LLM to classify tool results
	SkipCurrentTurn      bool   `json:"skip_current_turn"`     // Whether to skip cleaning current turn's tools
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
	if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
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
	if config.Agent.ToolTimeout == 0 {
		config.Agent.ToolTimeout = 30 * time.Second
	}
	if config.Agent.MaxHistory == 0 {
		config.Agent.MaxHistory = 20
	}
	if config.Agent.SystemPrompt == "" {
		config.Agent.SystemPrompt = "You are a helpful assistant with access to various tools. Use tools when necessary to help the user accomplish their tasks."
	}
	if config.Agent.Temperature == 0 {
		config.Agent.Temperature = 0.7
	}

	if config.MCP == nil {
		config.MCP = &MCPConfig{
			Servers: make(map[string]MCPServerConfig),
		}
	}

	if config.LLM == nil {
		config.LLM = &LLMConfig{}
	}
	if config.LLM.Provider == "" {
		config.LLM.Provider = "gemini"
	}
	if config.LLM.Model == "" {
		config.LLM.Model = "gemini-2.0-flash"
	}
	if config.LLM.Temperature == 0 {
		config.LLM.Temperature = 0.7
	}

	if config.UserInput == nil {
		config.UserInput = &UserInputConfig{}
	}
	if config.UserInput.Prompt == "" {
		config.UserInput.Prompt = "You: "
	}
	if len(config.UserInput.ExitCommands) == 0 {
		config.UserInput.ExitCommands = []string{"exit", "quit", "bye"}
	}
	if config.UserInput.SummarizationThreshold == 0 {
		config.UserInput.SummarizationThreshold = 15
	}

	if config.Summarizer == nil {
		config.Summarizer = &SummarizerConfig{}
	}
	if config.Summarizer.PreserveRecentCount == 0 {
		config.Summarizer.PreserveRecentCount = 5
	}
	if config.Summarizer.SummaryPrompt == "" {
		config.Summarizer.SummaryPrompt = "Please summarize the following conversation history, preserving key context and recent tool interactions:"
	}
	if config.Summarizer.MaxSummaryLength == 0 {
		config.Summarizer.MaxSummaryLength = 500
	}

	if config.Planning == nil {
		config.Planning = &PlanningConfig{}
	}
	if config.Planning.SystemPrompt == "" {
		config.Planning.SystemPrompt = `You are a helpful assistant with access to tools. Analyze the user's request and respond with structured YAML.

Format your response as:
intent: "Brief description of what you're trying to accomplish"
response: "Your response to the user"
tool_calls: ["tool1", "tool2"]  # List of tools to call, empty array if none
tool_args: [{"arg1": "value1"}, {"arg2": "value2"}]  # Arguments for each tool call

If you don't need tools, use empty arrays for tool_calls and tool_args.`
	}
	if config.Planning.MaxToolCalls == 0 {
		config.Planning.MaxToolCalls = 3
	}
	if config.Planning.Temperature == 0 {
		config.Planning.Temperature = 0.7
	}

	if config.Approval == nil {
		config.Approval = &ApprovalConfig{}
	}
	if config.Approval.ApprovalPrompt == "" {
		config.Approval.ApprovalPrompt = "The assistant wants to use tools. Approve? (y/n/a for always): "
	}
	if config.Approval.RejectionMessage == "" {
		config.Approval.RejectionMessage = "I understand you don't want me to use those tools. Let me help you in another way."
	}

	if config.ToolExecution == nil {
		config.ToolExecution = &ToolExecutionConfig{}
	}
	if config.ToolExecution.Timeout == 0 {
		config.ToolExecution.Timeout = 30 * time.Second
	}
	if config.ToolExecution.MaxConcurrency == 0 {
		config.ToolExecution.MaxConcurrency = 3
	}
	if config.ToolExecution.RetryAttempts == 0 {
		config.ToolExecution.RetryAttempts = 2
	}
	if config.ToolExecution.FailureMode == "" {
		config.ToolExecution.FailureMode = "continue"
	}
	if config.ToolExecution.MaxToolSteps == 0 {
		config.ToolExecution.MaxToolSteps = 10
	}

	if config.ToolResultCleanup == nil {
		config.ToolResultCleanup = &CleanupConfig{}
	}
	if config.ToolResultCleanup.SuccessMessage == "" {
		config.ToolResultCleanup.SuccessMessage = "Tool was successfully executed"
	}
	if config.ToolResultCleanup.ClassificationPrompt == "" {
		config.ToolResultCleanup.ClassificationPrompt = "Classify this tool result as success or error. Respond with 'SUCCESS' or 'ERROR':"
	}
}
