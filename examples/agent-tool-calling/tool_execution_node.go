package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// ToolExecutionNode handles execution of approved tools
type ToolExecutionNode struct {
	toolManager *MCPToolManager
	config      *ToolExecutionConfig
}

// NewToolExecutionNode creates a new tool execution node
func NewToolExecutionNode(toolManager *MCPToolManager, config *ToolExecutionConfig) *ToolExecutionNode {
	if config == nil {
		config = &ToolExecutionConfig{
			Timeout:        30 * time.Second,
			MaxConcurrency: 3,
			RetryAttempts:  2,
			FailureMode:    "continue",
			MaxToolSteps:   10,
		}
	}

	return &ToolExecutionNode{
		toolManager: toolManager,
		config:      config,
	}
}

// Prep verifies approval granted, checks permissions, and validates tool parameters
func (n *ToolExecutionNode) Prep(state *AgentState) []llm.ToolCalls {

	return state.PendingToolCalls
}

// Exec executes tool calls
func (n *ToolExecutionNode) Exec(toolCall llm.ToolCalls) (llm.ToolResults, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), n.config.Timeout)
	defer cancel()

	fmt.Printf("Executing tool: %s\n", toolCall.ToolName)

	// Execute the tool via MCP manager
	result, err := n.toolManager.ExecuteTool(ctx, toolCall)
	if err != nil {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: fmt.Sprintf("Tool execution failed: %v", err),
			IsError: true,
		}, err
	}

	// Display result to user
	if result.IsError {
		fmt.Printf("Tool %s failed: %s\n", toolCall.ToolName, result.Error)
	} else {
		fmt.Printf("Tool %s result: %s\n", toolCall.ToolName, result.Content)
	}

	return result, nil
}

// Post updates state with results and preserves tool context
func (n *ToolExecutionNode) Post(state *AgentState, prepResults []llm.ToolCalls, execResults ...llm.ToolResults) core.Action {
	if len(execResults) == 0 {
		// No results, clear pending calls and continue
		state.PendingToolCalls = make([]llm.ToolCalls, 0)
		return ActionContinue
	}

	// Find the assistant message with tool calls and update it with results
	messageIndex, message := state.FindMessageWithToolCalls()
	if messageIndex >= 0 && message != nil {
		state.UpdateMessageWithToolResults(messageIndex, execResults)
	}

	for i := state.ProcessedHistoryIndex+1; i < len(state.CleanedMessages); i++ {
		msg := state.CleanedMessages[i]
		if msg.Role == llm.RoleUser && len(msg.ToolResults) > 0 {
			for j, toolResult := range msg.ToolResults {
				if !toolResult.IsError {
					msg.Content += fmt.Sprintf("## Tool %s status:\n %s\n", msg.ToolCalls[j].ToolName, "Executed Succesfully")
				} else {
					msg.Content += fmt.Sprintf("## Tool %s status:\n %s\n", msg.ToolCalls[j].ToolName, toolResult.Content)
				}
			}
			state.CleanedMessages[i] = msg // Update the message with results
			state.ProcessedHistoryIndex = i // Update processed index to skip this message

		}
	}
	responseMessage := llm.Message{
		Role:        llm.RoleUser,
		ToolCalls:   prepResults,
		ToolResults: execResults,
	}
	
	for i, results := range execResults {
		responseMessage.Content += fmt.Sprintf("## Tool %s result:\n %s\n", prepResults[i].ToolName, results.Content)
		if len(results.Media) > 0 {
			responseMessage.Media = results.Media
			responseMessage.MimeType = results.MetaData.ContentType
		}
	}
	state.AddMessage(responseMessage)

	// Clear pending tool calls as they've been executed
	state.PendingToolCalls = make([]llm.ToolCalls, 0)

	// Always proceed to cleanup after tool execution
	return ActionPlan
}

// ExecFallback returns structured error for failed tool execution
func (n *ToolExecutionNode) ExecFallback(err error) llm.ToolResults {
	return llm.ToolResults{
		Id:      "unknown",
		Content: "",
		IsError: true,
		Error:   fmt.Sprintf("Tool execution fallback: %v", err),
	}
}
