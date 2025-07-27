package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/structured"
	"github.com/alt-coder/pocketflow-go/tools"
)

// ChatNode handles LLM decision making and tool call planning
type ChatNode[T StateInterface] struct {
	llmProvider         llm.LLMProvider
	config              *AgentConfig
	tools               []tools.ToolSchema
	key                 string
	toolUse             Permission
	AlwaysAllowedTools  map[string]struct{}
	errorRetryCount     int
	toolManager         *tools.ToolManager
	isUserInputRequired bool
}

type ChatNodeOptions[T StateInterface] func(n *ChatNode[T])

func WithToolUse[T StateInterface](toolUse Permission) ChatNodeOptions[T] {
	return func(n *ChatNode[T]) {
		n.toolUse = toolUse
	}
}

func NewToolUsageFlow[T StateInterface](manager *tools.ToolManager, llmProvider llm.LLMProvider,
	config *AgentConfig, state T, options ...ChatNodeOptions[T]) core.Workflow[T] {
	tools := manager.GetAvailableTools()
	chatNode := NewChatNode[T](llmProvider, config)
	chatNode.tools = tools
	chatNode.toolManager = manager
	chatNode.AlwaysAllowedTools = make(map[string]struct{})
	for _, option := range options {
		option(chatNode)
	}
	chatNode.key = "chat"
	chatNode.isUserInputRequired = false
	node := core.NewNode(chatNode, 3, 1)
	node.AddSuccessor(node, core.Action(ActionContinue))
	return core.NewFlow(node)
}

// NewChatNode creates a new planning node
func NewChatNode[T StateInterface](llmProvider llm.LLMProvider, config *AgentConfig) *ChatNode[T] {
	if config == nil {
		config = &AgentConfig{
			SystemPrompt: `You are a helpful assistant with access to tools. Analyze the conversation and respond appropriately.

You must respond with structured YAML in the exact format specified below. Do not include any other text.`,
		}
	}

	return &ChatNode[T]{
		llmProvider: llmProvider,
		config:      config,
	}
}

// Prep prepares the messages and context for LLM planning
func (n *ChatNode[T]) Prep(state *T) []ChatContext {
	messages := (*state).GetConversation(n.key)
	message := llm.Message{
		Role: llm.RoleUser,
	}
	if len(*messages) < 1 {
		message.Content = n.buildSystemPromptWithTools( "") + "\n\n"
		n.isUserInputRequired = true
		fmt.Println("How may I help you today? \n ")
	}
	if n.isUserInputRequired {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("You: ")
		var input string
		
		if scanner.Scan() {
			input = strings.TrimSpace(scanner.Text())
		}

		if scanner.Err() != nil {
			fmt.Printf("Error reading input: %v\n", scanner.Err())
		}
		message.Content = message.Content + fmt.Sprintf("\n##User: %s\n", input)
		(*state).AddMessage(message)
		n.isUserInputRequired = false
	}

	// Build system prompt with available tools information

	context := ChatContext{
		Messages: (*state).GetConversation(n.key),
	}

	return []ChatContext{context}
}

// Exec calls planning LLM with the prepared messages
func (n *ChatNode[T]) Exec(chatcontext ChatContext) (llm.Message, error) {
	// Try to call the real LLM provider with prepared messages
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call LLM provider with messages prepared in Prep
	response, err := n.llmProvider.CallLLM(ctx, *chatcontext.Messages)

	return response, err
}

// Post processes LLM response, creates assistant message, and determines next action
func (n *ChatNode[T]) Post(state *T, prepResults []ChatContext, execResults ...llm.Message) core.Action {
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

	if err != nil && n.errorRetryCount < 3 {
		n.errorRetryCount++
		log.Printf("Error parsing response: %v, retrying (%d/3)", err, n.errorRetryCount)
		(*state).AddMessage(llm.Message{
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
	(*state).AddMessage(execResult)

	// Display the response to user
	fmt.Printf("\nAssistant: %s\n", result.Response)

	// If there are tool calls that require approval
	if len(execResult.ToolCalls) > 0 {
		tools := execResult.ToolCalls
		results := []llm.ToolResults{}
		var action core.Action
		if n.toolUse != PermissionAllow {
			tools, action = n.AskToolPermission(*state, execResult.ToolCalls)
			switch action {
			case core.ActionFailure, core.ActionContinue:
				return action
			}
		}
		for _, tool := range tools {
			// Check if the tool is allowed
			result, _ := n.toolManager.ExecuteTool(context.Background(), tool)
			results = append(results, result)
		}

		responseMessage := llm.Message{
			Role:        llm.RoleUser,
			ToolCalls:   tools,
			ToolResults: results,
		}

		for i, results := range results {
			responseMessage.Content += fmt.Sprintf("## Tool %s result:\n %s\n", tools[i].ToolName, results.Content)
			if len(results.Media) > 0 {
				responseMessage.Media = results.Media
				responseMessage.MimeType = results.MetaData.ContentType
			}
		}
		(*state).AddMessage(responseMessage)
	} else {
		n.isUserInputRequired = true
	}

	return core.ActionSuccess

}

// No tool calls, just continue to next user input

func (n *ChatNode[T]) AskToolPermission(state T, availableTools []llm.ToolCalls) ([]llm.ToolCalls, core.Action) {
	results := []llm.ToolCalls{}
	for _, tool := range availableTools {
		if _, ok := n.AlwaysAllowedTools[tool.ToolName]; ok {
			results = append(results, tool)
			continue
		}
		fmt.Printf("Tool %s requires permission: \n", tool.ToolName)
		fmt.Printf("Allow? [y = yes ; n = no ; a = allways allow]: ")
		var response string
		fmt.Scanln(&response)
		if response == "y" {
			results = append(results, tool)
		} else if response == "a" {
			n.AlwaysAllowedTools[tool.ToolName] = struct{}{}
			results = append(results, tool)
		} else if response == "n" {
			continue
		} else {
			message := llm.Message{
				Role:    llm.RoleUser,
				Content: response,
			}
			state.AddMessage(message)
			return []llm.ToolCalls{}, ActionContinue
		}
	}
	return results, core.ActionSuccess
}

// ExecFallback provides a safe error response
func (n *ChatNode[T]) ExecFallback(err error) llm.Message {
	return llm.Message{
		Content: fmt.Sprintf("I apologize, but I'm having trouble processing your request right now Err %v. Could you please try again?", err),
	}
}

// buildSystemPromptWithTools creates a system prompt that includes available tools and instructions
func (n *ChatNode[T]) buildSystemPromptWithTools(summarizedHistory string) string {
	var promptBuilder strings.Builder
	var availableTools []tools.ToolSchema
	if n.toolManager != nil {
		availableTools = n.toolManager.GetAvailableTools()
	}
	// Start with base system prompt
	promptBuilder.WriteString(n.config.SystemPrompt)
	// promptBuilder.WriteString("\n\n")

	if summarizedHistory != "" {
		promptBuilder.WriteString("## Previous Conversation Summary:\n")
		promptBuilder.WriteString(summarizedHistory)
		promptBuilder.WriteString("\n\n")
	}
	if len(availableTools) > 0 {
		promptBuilder.WriteString("You can use the following tools to assist the user")
		promptBuilder.WriteString("\n\n")
		promptBuilder.WriteString("")
		promptBuilder.WriteString("## Available Tools:\n")
		for _, tool := range availableTools {
			promptBuilder.WriteString(fmt.Sprintf("\n- **%s**: %s\n", tool.Name, tool.Description))
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

// formatToolParameters recursively formats tool parameters from tools.Parameter
func (n *ChatNode[T]) formatToolParameters(builder *strings.Builder, params map[string]tools.Parameter, indent string) {
	for paramName, param := range params {
		// Format parameter with type and description
		builder.WriteString(fmt.Sprintf("%s- %s (%s)", indent, paramName, param.Type))

		if param.Description != "" {
			builder.WriteString(fmt.Sprintf(": %s", param.Description))
		}

		// Add required indicator
		if param.Required {
			builder.WriteString(" [required]")
		}

		// Add enum values if present
		if len(param.Enum) > 0 {
			builder.WriteString(fmt.Sprintf(" [options: %s]", strings.Join(param.Enum, ", ")))
		}

		// Add default value if present
		if param.Default != nil {
			builder.WriteString(fmt.Sprintf(" [default: %v]", param.Default))
		}

		builder.WriteString("\n")
	}
}

// parseYAMLResponse parses the strict YAML response from LLM
func (n *ChatNode[T]) parseYAMLResponse(responseContent string) (PlanningResult, error) {

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
		Response:     planningResp.Response,
		LLMToolCalls: llmToolCalls,
	}, nil
}
