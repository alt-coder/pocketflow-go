package main

import (
	"time"

	"github.com/alt-coder/pocketflow-go/llm"
)

// AgentState represents the enhanced state for multi-step tool calling with approval
type AgentState struct {
	// Two separate message histories
	ActualMessages        []llm.Message          `json:"actual_messages"`         // Full actual conversation history
	CleanedMessages       []llm.Message          `json:"cleaned_messages"`        // Cleaned conversation history for planner
	SummarizedHistory     string                 `json:"summarized_history"`      // Compressed conversation history
	AvailableTools        []ToolSchema           `json:"available_tools"`         // Tools available to the agent
	PendingToolCalls      []llm.ToolCalls             `json:"pending_tool_calls"`      // Tool calls waiting for approval/execution
	ToolCallHistory       []ToolInteraction      `json:"tool_call_history"`       // History of all tool interactions
	ToolPermissions       map[string]Permission  `json:"tool_permissions"`        // Stored tool permissions
	ProcessedHistoryIndex int                    `json:"processed_history_index"` // Index up to which tool results have been cleaned up
	Context               map[string]interface{} `json:"context"`                 // Additional context data
	MaxHistory            int                    `json:"max_history"`             // Maximum conversation history
	SummarizationNeeded   bool                   `json:"summarization_needed"`    // Flag indicating if summarization is needed
	Active                bool                   `json:"active"`                  // Whether agent is active
	RetryCount 		  int                    `json:"retry_count"`             // Count of retries for tool calls
}

// NewAgentState creates a new agent state with default values
func NewAgentState() *AgentState {
	return &AgentState{
		ActualMessages:        make([]llm.Message, 0),
		CleanedMessages:       make([]llm.Message, 0),
		AvailableTools:        make([]ToolSchema, 0),
		PendingToolCalls:      make([]llm.ToolCalls, 0),
		ToolCallHistory:       make([]ToolInteraction, 0),
		ToolPermissions:       make(map[string]Permission),
		Context:               make(map[string]interface{}),
		MaxHistory:            20,
		SummarizationNeeded:   false,
		Active:                true,
		ProcessedHistoryIndex: 0,
	}
}

// AddMessage adds a message to both actual and cleaned conversation histories
func (s *AgentState) AddMessage(msg llm.Message) {
	s.ActualMessages = append(s.ActualMessages, msg)
	s.CleanedMessages = append(s.CleanedMessages, msg)
}

// AddToolInteraction adds a tool interaction to the history
func (s *AgentState) AddToolInteraction(call llm.ToolCalls, result ToolResult, taskID string) {
	interaction := ToolInteraction{
		ToolCall:   call,
		ToolResult: result,
		Timestamp:  time.Now(),
		TaskID:     taskID,
		Approved:   true, // If we're adding it, it was approved
	}
	s.ToolCallHistory = append(s.ToolCallHistory, interaction)
}

// ShouldSummarize checks if summarization is needed
func (s *AgentState) ShouldSummarize() bool {
	return len(s.ActualMessages) > s.MaxHistory || s.SummarizationNeeded
}

// UpdateMessageWithToolResults updates a message with tool execution results
func (s *AgentState) UpdateMessageWithToolResults(messageIndex int, results []llm.ToolResults) {
	if messageIndex >= 0 && messageIndex < len(s.ActualMessages) {
		s.ActualMessages[messageIndex].ToolResults = results
		// Also update cleaned messages
		if messageIndex < len(s.CleanedMessages) {
			s.CleanedMessages[messageIndex].ToolResults = results
		}
	}
}

// FindMessageWithToolCalls finds the most recent message with pending tool calls
func (s *AgentState) FindMessageWithToolCalls() (int, *llm.Message) {
	for i := len(s.ActualMessages) - 1; i >= 0; i-- {
		msg := &s.ActualMessages[i]
		if len(msg.ToolCalls) > 0 && len(msg.ToolResults) == 0 {
			return i, msg
		}
	}
	return -1, nil
}

// GetRecentToolInteractions returns the most recent tool interactions
func (s *AgentState) GetRecentToolInteractions(count int) []ToolInteraction {
	if count <= 0 || len(s.ToolCallHistory) == 0 {
		return []ToolInteraction{}
	}

	start := len(s.ToolCallHistory) - count
	if start < 0 {
		start = 0
	}

	result := make([]ToolInteraction, len(s.ToolCallHistory)-start)
	copy(result, s.ToolCallHistory[start:])
	return result
}

// ClearCompletedToolCycle clears pending tool calls after processing
func (s *AgentState) ClearCompletedToolCycle() {
	s.PendingToolCalls = make([]llm.ToolCalls, 0)
}

// HasPermission checks if tool has permission
func (s *AgentState) HasPermission(toolName string) bool {
	if perm, exists := s.ToolPermissions[toolName]; exists {
		return perm.Granted
	}
	return false
}

// GrantPermission grants tool permission
func (s *AgentState) GrantPermission(toolName string, alwaysAllow bool) {
	s.ToolPermissions[toolName] = Permission{
		ToolName:    toolName,
		Granted:     true,
		AlwaysAllow: alwaysAllow,
		GrantedAt:   time.Now(),
	}
}

// RevokePermission revokes tool permission
func (s *AgentState) RevokePermission(toolName string) {
	if perm, exists := s.ToolPermissions[toolName]; exists {
		perm.Granted = false
		s.ToolPermissions[toolName] = perm
	}
}

// TrimHistory trims the message history to the maximum allowed length
func (s *AgentState) TrimHistory() {
	if len(s.ActualMessages) > s.MaxHistory {
		// Keep the most recent messages
		s.ActualMessages = s.ActualMessages[len(s.ActualMessages)-s.MaxHistory:]
	}
	if len(s.CleanedMessages) > s.MaxHistory {
		// Keep the most recent messages
		s.CleanedMessages = s.CleanedMessages[len(s.CleanedMessages)-s.MaxHistory:]
	}
}

// GetMessagesForPlanning returns cleaned messages for the planning node
func (s *AgentState) GetMessagesForPlanning() []llm.Message {
	return s.CleanedMessages
}

// ReplaceCleanedMessage replaces a message in the cleaned history (used by cleanup node)
func (s *AgentState) ReplaceCleanedMessage(index int, newMessage llm.Message) {
	if index >= 0 && index < len(s.CleanedMessages) {
		s.CleanedMessages[index] = newMessage
	}
}
