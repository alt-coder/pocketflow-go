package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// ApprovalNode handles user approval for tool execution
type ApprovalNode struct {
	config *ApprovalConfig
}

// NewApprovalNode creates a new approval node
func NewApprovalNode(config *ApprovalConfig) *ApprovalNode {
	if config == nil {
		config = &ApprovalConfig{
			ApprovalPrompt:   "The assistant wants to use tools. Approve? (y/n/a for always): ",
			RejectionMessage: "I understand you don't want me to use those tools. Let me help you in another way.",
		}
	}

	return &ApprovalNode{
		config: config,
	}
}

// Prep parses user input for approval decision
func (n *ApprovalNode) Prep(state *AgentState) []ApprovalResult {
	if len(state.PendingToolCalls) == 0 {
		return []ApprovalResult{} // No tools to approve
	}
	result := make([]ApprovalResult, 0)

	for _, toolCall := range state.PendingToolCalls {
		if !state.HasPermission(toolCall.ToolName) {
			fmt.Printf("Tool %s requires approval\n", toolCall.ToolName)
			// Display approval prompt
			fmt.Print(n.config.ApprovalPrompt)
			scanner := bufio.NewScanner(os.Stdin)

			var input string
			if scanner.Scan() {
				input = strings.TrimSpace(scanner.Text())
			}

			if scanner.Err() != nil {
				fmt.Printf("Error reading input: %v\n", scanner.Err())
				continue
			}
			switch input {
			case "y", "yes":
				result = append(result, ApprovalResult{
					Action:       "approve",
					AlwaysAllow:  false,
					AppendedText: "",
				})
			case "a", "always":
				result = append(result, ApprovalResult{
					Action:       "approve",
					AlwaysAllow:  true,
					AppendedText: "",
				})
			case "r", "reject", "n", "no":
				result = append(result, ApprovalResult{
					Action:       "reject",
					AlwaysAllow:  false,
					AppendedText: "",
				})
			default:
				// Any other input is treated as additional context
				return []ApprovalResult{{
					Action:       "append",
					AlwaysAllow:  false,
					AppendedText: input,
				}}
			}

		}else{
			result = append(result, ApprovalResult{
			Action:       "approve",
			AlwaysAllow:  false,
			AppendedText: "",
		})
		}
		
	}

	return result
}

// Exec processes approval input (no external calls needed)
func (n *ApprovalNode) Exec(input ApprovalResult) (ApprovalResult, error) {
	return input, nil
}

// Post handles approval result and updates state accordingly
func (n *ApprovalNode) Post(state *AgentState, prepResults []ApprovalResult, execResults ...ApprovalResult) core.Action {
	rejectionMessage := llm.Message{
		Role:    llm.RoleAssistant,
		Content: n.config.RejectionMessage,
	}
	if len(execResults) == 0 || len(prepResults) == 0 {
		state.AddMessage(rejectionMessage)
		return core.Action(ActionReject) // Default to rejection if no results
	}
	rejectMarker := make([]int, len(execResults))
	for i, result := range execResults {

		switch result.Action {
		case "approve":
			// Store permissions if always allow
			if result.AlwaysAllow {
				for _, toolCall := range state.PendingToolCalls {
					state.GrantPermission(toolCall.ToolName, true)
				}
				fmt.Println("Tools approved and permission granted for future use.")
			} else {
				fmt.Println("Tools approved for this use.")
			}
			return core.Action(ActionApprove)

		case "reject":
			// // Add rejection message to conversation

			// Clear pending tool calls
			rejectMarker[i] = 1

			fmt.Printf("Assistant: %s\n", n.config.RejectionMessage)

		case "append":
			// Add the additional input to conversation
			if result.AppendedText != "" {
				userMessage := llm.Message{
					Role:    llm.RoleUser,
					Content: result.AppendedText,
				}
				state.AddMessage(userMessage)
			}
			return core.Action(ActionAppend)

		default:
			return core.Action(ActionReject)
		}
	}
	newToolList := make([]llm.ToolCalls, 0, len(state.PendingToolCalls))
	for i := range state.PendingToolCalls {
		if rejectMarker[i] == 0 {
			newToolList = append(newToolList, state.PendingToolCalls[i])
		}
	}
	state.PendingToolCalls = newToolList

	return core.Action(ActionReject)
}

// ExecFallback handles execution failures (N/A for approval node)
func (n *ApprovalNode) ExecFallback(err error) ApprovalResult {
	return ApprovalResult{
		Action:       "reject",
		AlwaysAllow:  false,
		AppendedText: "",
	}
}
