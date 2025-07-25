package main

import (
	"fmt"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// We'll use the LLMProvider interface from ../../llm package
// and llm.Message type instead of defining our own

// Action definitions for the agent workflow
const (
	// Flow control actions
	ActionPlan            core.Action = "plan"             // UserInput -> Planning
	ActionSummarize       core.Action = "summarize"        // UserInput -> Summarizer (context overflow)
	ActionRequestApproval core.Action = "request_approval" // Planning -> ApprovalNode (tool calls detected)
	ActionApprove         core.Action = "approve"          // ApprovalNode -> ToolExecution ("y" or "a")
	ActionReject          core.Action = "reject"           // ApprovalNode -> Planning ("r")
	ActionAppend          core.Action = "append"           // ApprovalNode -> Planning (other input)
	ActionCleanup         core.Action = "cleanup"          // ToolExecution -> ToolResultCleanupNode (tools executed)
	ActionContinue        core.Action = "continue"         // Planning -> UserInput (final response) OR ToolResultCleanupNode -> Planning (cleanup complete)

	// Terminal actions
	ActionExit    = "exit"    // User requested exit
	ActionFailure = "failure" // Unrecoverable error

	// Retry actions
	ActionRetry = "retry" // Retry current operation
)

// ToolCall represents a tool call made by the LLM (matches llm.ToolCalls structure)
type ToolCall struct {
	ID   string                 `json:"id"`        // Unique identifier for the tool call
	Name string                 `json:"tool_name"` // Tool name (matches llm.ToolCalls.ToolName)
	Args map[string]interface{} `json:"tool_args"` // Tool arguments (matches llm.ToolCalls.ToolArgs)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"` // References the tool call ID
	Content    string `json:"content"`      // Tool execution result
	IsError    bool   `json:"is_error"`     // Whether the result is an error
	Error      string `json:"error"`        // Error message if IsError is true
}

// ToolSchema represents the schema of an available tool
type ToolSchema struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Parameters  map[string]*protocol.Property `json:"parameters"`
}

// ToolInteraction represents a complete tool interaction
type ToolInteraction struct {
	ToolCall   llm.ToolCalls `json:"tool_call"`   // The tool call made
	ToolResult ToolResult    `json:"tool_result"` // The result received
	Timestamp  time.Time     `json:"timestamp"`   // When the interaction occurred
	TaskID     string        `json:"task_id"`     // ID linking related tool calls in a task
	Approved   bool          `json:"approved"`    // Whether the tool call was approved
}

// Permission represents tool execution permission
type Permission struct {
	ToolName    string    `json:"tool_name"`    // Name of the tool
	Granted     bool      `json:"granted"`      // Whether permission is granted
	AlwaysAllow bool      `json:"always_allow"` // Whether to always allow this tool
	GrantedAt   time.Time `json:"granted_at"`   // When permission was granted
}

// PlanningResponse represents the structured YAML response from the planning LLM
type PlanningResponse struct {
	Intent    string                   `yaml:"intent"`
	Response  string                   `yaml:"response"`
	ToolCalls []string                 `yaml:"tool_calls"`
	ToolArgs  []map[string]interface{} `yaml:"tool_args"`
}

// Validate ensures the YAML response is properly structured
func (pr *PlanningResponse) Validate() error {
	if len(pr.ToolCalls) != len(pr.ToolArgs) {
		return fmt.Errorf("tool_calls and tool_args must have the same length")
	}
	return nil
}

// UserInput represents user input data
type UserInput struct {
	Text string `json:"text"`
}

// UserInputResult represents the result of processing user input
type UserInputResult struct {
	ProcessedInput string `json:"processed_input"`
	ShouldExit     bool   `json:"should_exit"`
	NeedsSummary   bool   `json:"needs_summary"`
}

// SummarizationContext represents context for summarization
type SummarizationContext struct {
	Messages               []llm.Message     `json:"messages"`
	RecentToolInteractions []ToolInteraction `json:"recent_tool_interactions"`
}

// SummarizationResult represents the result of summarization
type SummarizationResult struct {
	Summary string `json:"summary"`
	Error   error  `json:"error,omitempty"`
}

// PlanningContext represents context for planning
type PlanningContext struct {
	Messages       []llm.Message `json:"messages"`        // Messages prepared for LLM call
	AvailableTools []ToolSchema  `json:"available_tools"` // Available tools for reference
}

// PlanningResult represents the result of planning
type PlanningResult struct {
	Response     string          `json:"response"`
	LLMToolCalls []llm.ToolCalls `json:"llm_tool_calls"` // LLM format for message
	Error        error           `json:"error,omitempty"`
}

// ApprovalInput represents input for approval processing
type ApprovalInput struct {
	UserInput    string        `json:"user_input"`
	PendingTools llm.ToolCalls `json:"pending_tools"` // Index of this input in the pending tools
}

// ApprovalResult represents the result of approval processing
type ApprovalResult struct {
	Action       string `json:"action"`        // "approve", "reject", "append"
	AlwaysAllow  bool   `json:"always_allow"`  // Whether to always allow these tools
	AppendedText string `json:"appended_text"` // Additional text if action is "append"
}

// CleanupItem represents an item to be cleaned up
type CleanupItem struct {
	MessageIndex  int             `json:"message_index"`   // Index in Messages array
	ToolCall      ToolCall        `json:"tool_call"`       // The tool call to evaluate
	ToolResult    llm.ToolResults `json:"tool_result"`     // The tool result to evaluate
	IsCurrentTurn bool            `json:"is_current_turn"` // Whether this is from current turn
	Message       string          `json:"message"`         // Message to replace tool result with
	PrevMessage   string          `json:"prev_message"`    // Previous message before cleanup
}

// CleanupResult represents the result of cleanup evaluation
type CleanupResult struct {
	ShouldClean bool   `json:"should_clean"` // Whether to clean this tool interaction
	IsSuccess   bool   `json:"is_success"`   // Whether the tool execution was successful
	ErrorReason string `json:"error_reason"` // Reason if it's an error (for preservation)
}
