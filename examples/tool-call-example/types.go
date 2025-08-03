package main

import (
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
type Permission string

const (
	PermissionAllow     Permission = "allow"
	PermissionDeny      Permission = "deny"
	PermissionAlwaysAsk Permission = "always_ask"
)

// LLMResponse represents the structured YAML response from the planning LLM
type LLMResponse struct {
	Intent    string                   `yaml:"intent"`
	Response  string                   `yaml:"response"`
	ToolCalls []string                 `yaml:"tool_calls"`
	ToolArgs  []map[string]interface{} `yaml:"tool_args"`
}

// UserInput represents user input data
type UserInput struct {
	Text string `json:"text"`
}

// ParsedResult represents the result of planning
type ParsedResult struct {
	Response     string          `json:"response"`
	LLMToolCalls []llm.ToolCalls `json:"llm_tool_calls"` // LLM format for message
	Error        error           `json:"error,omitempty"`
}

// ChatContext represents context for planning
type ChatContext struct {
	Messages *[]llm.Message `json:"messages"` // Messages prepared for LLM call
}
