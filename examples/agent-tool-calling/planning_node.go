package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"log"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/structured"

)

// PlanningNode handles LLM decision making and tool call planning
type PlanningNode struct {
	llmProvider llm.LLMProvider // Will be llm.LLMProvider from ../../llm package
	config      *PlanningConfig
}

// NewPlanningNode creates a new planning node
func NewPlanningNode(llmProvider llm.LLMProvider, config *PlanningConfig) *PlanningNode {
	if config == nil {
		config = &PlanningConfig{
			SystemPrompt: `You are a helpful assistant with access to tools. Analyze the conversation and respond appropriately.

You must respond with structured YAML in the exact format specified below. Do not include any other text.`,
			MaxToolCalls: 3,
			Temperature:  0.7,
			AllowedTools: []string{}, // Empty means all tools allowed
		}
	}

	return &PlanningNode{
		llmProvider: llmProvider,
		config:      config,
	}
}

// Prep prepares the messages and context for LLM planning
func (n *PlanningNode) Prep(state *AgentState) []PlanningContext {
	
	availableTools := state.AvailableTools
	if len(state.CleanedMessages) == 1 {
		message := state.CleanedMessages[0]
		message.Content = n.buildSystemPromptWithTools(availableTools, state.SummarizedHistory)+"\n\n" + message.Content
		state.CleanedMessages[0] = message
		state.ActualMessages[0] = message
	}
	// Build system prompt with available tools information
	

	// Create messages for LLM call
	llmMessages := []llm.Message{}

	// Add conversation history messages
	llmMessages = append(llmMessages, state.CleanedMessages...)

	context := PlanningContext{
		Messages:       llmMessages,
		AvailableTools: availableTools,
	}

	return []PlanningContext{context}
}



// Exec calls planning LLM with the prepared messages
func (n *PlanningNode) Exec(plancontext PlanningContext) (llm.Message, error) {
	// Try to call the real LLM provider with prepared messages
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call LLM provider with messages prepared in Prep
	response, err := n.llmProvider.CallLLM(ctx, plancontext.Messages)

	return response, err
}

// Post processes LLM response, creates assistant message, and determines next action
func (n *PlanningNode) Post(state *AgentState, prepResults []PlanningContext, execResults ...llm.Message) core.Action {
	if len(execResults) == 0 {
		return core.Action(ActionFailure)
	}

	execResult := execResults[0]
	if strings.HasPrefix(execResult.Content, "I apologize") {
		log.Printf("LLM returned apology, treating as failure: %s", execResult.Content)
		return core.Action(ActionFailure)
	}

	// Parse the YAML response
	result, err := n.parseYAMLResponse(execResult.Content)

	if err != nil && state.RetryCount < 3 {
		state.RetryCount++
		log.Printf("Error parsing response: %v, retrying (%d/3)", err, state.RetryCount)
		state.AddMessage( llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("I apologize, but I encountered an error processing your response: %v.\n Please try again.", err),
		})
		return core.ActionRetry
	} else if err != nil {
		log.Printf("Failed to parse response after retries: %v", err)
		return core.Action(ActionFailure)
	}

	if result.Response == "" && len(result.LLMToolCalls) == 0 {
		return core.Action(ActionFailure)
	}
	execResult.ToolCalls = result.LLMToolCalls

	// Add the assistant message to state
	state.AddMessage(execResult)

	// Display the response to user
	fmt.Printf("\nAssistant: %s\n", result.Response)

	// If there are tool calls that require approval
	if len(execResult.ToolCalls) > 0{
		state.PendingToolCalls = execResult.ToolCalls
		fmt.Printf("Tools to be used: ")
		for i, tool := range execResult.ToolCalls {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(tool.ToolName)
		}
		fmt.Println()
		return core.Action(ActionRequestApproval)
	}

	// No tool calls, just continue to next user input
	return core.Action(ActionContinue)
}

// ExecFallback provides a safe error response
func (n *PlanningNode) ExecFallback(err error) llm.Message {
	return llm.Message{
		Content:         fmt.Sprintf("I apologize, but I'm having trouble processing your request right now Err %v. Could you please try again?", err),
	}
}

// buildSystemPromptWithTools creates a system prompt that includes available tools and instructions
func (n *PlanningNode) buildSystemPromptWithTools(availableTools []ToolSchema, summarizedHistory string) string {
	var promptBuilder strings.Builder

	// Start with base system prompt
	promptBuilder.WriteString(n.config.SystemPrompt)
	promptBuilder.WriteString("\n\n")

	// Add conversation context if available
	if summarizedHistory != "" {
		promptBuilder.WriteString("## Previous Conversation Summary:\n")
		promptBuilder.WriteString(summarizedHistory)
		promptBuilder.WriteString("\n\n")
	}

	// Add available tools
	if len(availableTools) > 0 {
		promptBuilder.WriteString("## Available Tools:\n")
		for _, tool := range availableTools {
			promptBuilder.WriteString(fmt.Sprintf("- **%s**: %s\n", tool.Name, tool.Description))
			if len(tool.Parameters) > 0 {
				promptBuilder.WriteString("  Parameters:\n")
				n.formatToolParameters(&promptBuilder, tool.Parameters, "    ")
			}
		}
		promptBuilder.WriteString("\n")
	}

	// Add strict YAML format instructions
	promptBuilder.WriteString("## Response Format:\n")
	promptBuilder.WriteString("Respond with EXACTLY this YAML structure (no additional text):\n\n")
	promptBuilder.WriteString("```yaml\n")
	promptBuilder.WriteString("intent: \"Brief description of what you're trying to accomplish\"\n")
	promptBuilder.WriteString("response: \"Your response to the user\"\n")
	promptBuilder.WriteString("tool_calls:\n")
	promptBuilder.WriteString("  - \"tool_name_1\"\n")
	promptBuilder.WriteString("  - \"tool_name_2\"\n")
	promptBuilder.WriteString("tool_args:\n")
	promptBuilder.WriteString("  - arg1: \"value1\"\n")
	promptBuilder.WriteString("    arg2: \"value2\"\n")
	promptBuilder.WriteString("  - arg1: \"value3\"\n")
	promptBuilder.WriteString("```\n\n")
	promptBuilder.WriteString("If no tools are needed, use empty arrays:\n")
	promptBuilder.WriteString("tool_calls: []\n")
	promptBuilder.WriteString("tool_args: []\n\n")
	promptBuilder.WriteString("Analyze the conversation and respond with the structured YAML format.")

	return promptBuilder.String()
}

// formatToolParameters recursively formats tool parameters from protocol.Property
func (n *PlanningNode) formatToolParameters(builder *strings.Builder, params map[string]*protocol.Property, indent string) {
	for paramName, property := range params {
		if property == nil {
			continue
		}

		// Format parameter with type and description
		builder.WriteString(fmt.Sprintf("%s- %s (%s)", indent, paramName, property.Type))

		if property.Description != "" {
			builder.WriteString(fmt.Sprintf(": %s", property.Description))
		}

		// Add required indicator
		// Note: Required is typically at the parent level, but we'll check if this param is in any required list

		// Add enum values if present
		if len(property.Enum) > 0 {
			builder.WriteString(fmt.Sprintf(" [options: %s]", strings.Join(property.Enum, ", ")))
		}

		builder.WriteString("\n")

		// Handle nested properties for object types
		if property.Type == "object" && len(property.Properties) > 0 {
			n.formatToolParameters(builder, property.Properties, indent+"  ")
		}

		// Handle array item types
		if property.Type == "array" && property.Items != nil {
			builder.WriteString(fmt.Sprintf("%s  Items: %s", indent, property.Items.Type))
			if property.Items.Description != "" {
				builder.WriteString(fmt.Sprintf(" - %s", property.Items.Description))
			}
			builder.WriteString("\n")

			// Handle nested properties in array items
			if property.Items.Type == "object" && len(property.Items.Properties) > 0 {
				n.formatToolParameters(builder, property.Items.Properties, indent+"    ")
			}
		}
	}
}

// parseYAMLResponse parses the strict YAML response from LLM
func (n *PlanningNode) parseYAMLResponse(responseContent string) (PlanningResult, error) {

	parsedResp, err := structured.ParseResponse[PlanningResponse](responseContent)
	if err != nil {
		return PlanningResult{}, fmt.Errorf("failed to parse YAML response: %w", err)
	}
	planningResp := parsedResp.Data

	llmToolCalls := make([]llm.ToolCalls, 0)

	for i, toolName := range planningResp.ToolCalls {
		if i >= len(planningResp.ToolArgs) {
			break // Safety check
		}

		callID := fmt.Sprintf("call_%d_%d", time.Now().Unix(), i+1)

		// LLM format
		llmToolCall := llm.ToolCalls{
			Id:       callID,
			ToolName: toolName,
			ToolArgs: planningResp.ToolArgs[i],
		}
		llmToolCalls = append(llmToolCalls, llmToolCall)
	}

	return PlanningResult{
		Response:         planningResp.Response,
		LLMToolCalls:     llmToolCalls,

	}, nil
}
