package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// SummarizerNode handles conversation history summarization
type SummarizerNode struct {
	llmProvider llm.LLMProvider // Real LLM provider interface
	config      *SummarizerConfig
}

// NewSummarizerNode creates a new summarizer node
func NewSummarizerNode(llmProvider llm.LLMProvider, config *SummarizerConfig) *SummarizerNode {
	if config == nil {
		config = &SummarizerConfig{
			PreserveRecentCount: 5,
			SummaryPrompt:       "Please summarize the following conversation history, preserving key context and recent tool interactions:",
			MaxSummaryLength:    500,
		}
	}

	return &SummarizerNode{
		llmProvider: llmProvider,
		config:      config,
	}
}

// Prep prepares conversation history for summarization
func (n *SummarizerNode) Prep(state *AgentState) []SummarizationContext {
	// Determine how many messages to summarize vs preserve
	totalMessages := len(state.ActualMessages)
	if totalMessages <= n.config.PreserveRecentCount {
		// Not enough messages to summarize
		return []SummarizationContext{}
	}

	// Messages to summarize (older ones)
	messagesToSummarize := state.ActualMessages[:totalMessages-n.config.PreserveRecentCount]

	// Get recent tool interactions to preserve context
	recentToolInteractions := state.GetRecentToolInteractions(3)

	return []SummarizationContext{{
		Messages:               messagesToSummarize,
		RecentToolInteractions: recentToolInteractions,
	}}
}

// Exec generates summary of conversation history
func (n *SummarizerNode) Exec(SummaryContext SummarizationContext) (SummarizationResult, error) {
	if len(SummaryContext.Messages) == 0 {
		return SummarizationResult{
			Summary: "",
			Error:   nil,
		}, nil
	}

	// Build the summarization prompt
	prompt := n.buildSummarizationPrompt(SummaryContext)

	// Create LLM messages
	messages := []llm.Message{
		{
			Role:    "system",
			Content: n.config.SummaryPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call the LLM provider
	ctx := context.Background()
	response, err := n.llmProvider.CallLLM(ctx, messages)
	if err != nil {
		// Fall back to mock summary on error
		fmt.Printf("LLM summarization failed, using fallback: %v\n", err)
		summary := n.generateMockSummary(SummaryContext)
		return SummarizationResult{
			Summary: summary,
			Error:   nil,
		}, nil
	}

	// Ensure summary doesn't exceed max length
	summary := response.Content
	if len(summary) > n.config.MaxSummaryLength {
		summary = summary[:n.config.MaxSummaryLength-3] + "..."
	}

	return SummarizationResult{
		Summary: summary,
		Error:   nil,
	}, nil
}

// buildSummarizationPrompt builds the prompt for LLM summarization
func (n *SummarizerNode) buildSummarizationPrompt(context SummarizationContext) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString("Please summarize the following conversation history while preserving key context:\n\n")

	// Add conversation messages
	promptBuilder.WriteString("**Conversation History:**\n")
	for i, msg := range context.Messages {
		promptBuilder.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, strings.Title(msg.Role), msg.Content))

		// Add tool calls if present
		if len(msg.ToolCalls) > 0 {
			promptBuilder.WriteString("   Tool calls: ")
			for j, toolCall := range msg.ToolCalls {
				if j > 0 {
					promptBuilder.WriteString(", ")
				}
				promptBuilder.WriteString(toolCall.ToolName)
			}
			promptBuilder.WriteString("\n")
		}

		// Add tool results if present
		if len(msg.ToolResults) > 0 {
			promptBuilder.WriteString("   Tool results: ")
			for j, result := range msg.ToolResults {
				if j > 0 {
					promptBuilder.WriteString(", ")
				}
				promptBuilder.WriteString(result.Content)
			}
			promptBuilder.WriteString("\n")
		}
	}

	// Add tool interaction context
	if len(context.RecentToolInteractions) > 0 {
		promptBuilder.WriteString("\n**Recent Tool Interactions:**\n")
		for i, interaction := range context.RecentToolInteractions {
			promptBuilder.WriteString(fmt.Sprintf("%d. Tool: %s\n", i+1, interaction.ToolCall.ToolName))
			if !interaction.ToolResult.IsError {
				promptBuilder.WriteString(fmt.Sprintf("   Result: %s\n", interaction.ToolResult.Content))
			} else {
				promptBuilder.WriteString(fmt.Sprintf("   Error: %s\n", interaction.ToolResult.Error))
			}
		}
	}

	promptBuilder.WriteString(fmt.Sprintf("\nPlease provide a concise summary (max %d characters) that preserves the key context and recent tool usage.", n.config.MaxSummaryLength))

	return promptBuilder.String()
}

// generateMockSummary creates a mock summary for demonstration
func (n *SummarizerNode) generateMockSummary(context SummarizationContext) string {
	messageCount := len(context.Messages)
	toolCount := len(context.RecentToolInteractions)

	summary := fmt.Sprintf("Previous conversation summary (%d messages): ", messageCount)

	// Add basic context about the conversation
	if messageCount > 0 {
		summary += "The user and assistant have been having a conversation. "
	}

	// Add tool interaction context
	if toolCount > 0 {
		summary += fmt.Sprintf("Recent tool interactions included %d tool calls. ", toolCount)

		// Mention specific tools used
		toolNames := make(map[string]bool)
		for _, interaction := range context.RecentToolInteractions {
			toolNames[interaction.ToolCall.ToolName] = true
		}

		if len(toolNames) > 0 {
			summary += "Tools used: "
			first := true
			for toolName := range toolNames {
				if !first {
					summary += ", "
				}
				summary += toolName
				first = false
			}
			summary += ". "
		}
	}

	summary += "The conversation continues below."

	// Ensure summary doesn't exceed max length
	if len(summary) > n.config.MaxSummaryLength {
		summary = summary[:n.config.MaxSummaryLength-3] + "..."
	}

	return summary
}

// Post replaces old messages with summary and preserves recent context
func (n *SummarizerNode) Post(state *AgentState, prepResults []SummarizationContext, execResults ...SummarizationResult) core.Action {
	if len(execResults) == 0 || len(prepResults) == 0 {
		// No summarization was performed, continue to planning
		return core.Action(ActionPlan)
	}

	result := execResults[0]
	if result.Error != nil {
		fmt.Printf("Summarization failed: %v\n", result.Error)
		// Continue anyway, just without summarization
		return core.Action(ActionPlan)
	}

	// Replace old messages with summary in both actual and cleaned histories
	totalMessages := len(state.ActualMessages)
	recentMessages := state.ActualMessages[totalMessages-n.config.PreserveRecentCount:]

	// Create new message list with summary + recent messages
	newMessages := make([]llm.Message, 0, 1+len(recentMessages))

	// Add summary as a system message
	if result.Summary != "" {
		summaryMessage := llm.Message{
			Role:    llm.RoleSystem,
			Content: result.Summary,
		}
		newMessages = append(newMessages, summaryMessage)
	}

	// Add recent messages
	newMessages = append(newMessages, recentMessages...)

	// Update both message histories
	state.ActualMessages = newMessages
	state.CleanedMessages = make([]llm.Message, len(newMessages))
	copy(state.CleanedMessages, newMessages)

	state.SummarizedHistory = result.Summary
	state.SummarizationNeeded = false

	fmt.Println("Conversation history summarized to maintain context.")

	return core.Action(ActionPlan)
}

// ExecFallback provides fallback behavior when summarization fails
func (n *SummarizerNode) ExecFallback(err error) SummarizationResult {
	return SummarizationResult{
		Summary: "Previous conversation context preserved.",
		Error:   err,
	}
}
