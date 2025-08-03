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
	node := core.NewNode(chatNode, 3, 1)
	node.AddSuccessor(node, core.Action(ActionContinue))
	node.AddSuccessor(node, core.ActionRetry)
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
		llmProvider:        llmProvider,
		config:             config,
		AlwaysAllowedTools: make(map[string]struct{}), // Initialize here to prevent nil map
	}
}

// Prep prepares the messages and context for LLM planning
func (n *ChatNode[T]) Prep(state *T) []ChatContext {
	messages := (*state).GetConversation(n.key)

	// Handle first interaction or when user input is required
	if len(*messages) == 0 || n.isUserInputRequired {
		userInput := n.getUserInput(messages)
		if userInput != "" {
			message := llm.Message{
				Role:    llm.RoleUser,
				Content: userInput,
			}
			(*state).AddMessage(message)
			n.isUserInputRequired = false
		} else {
			// Return empty context if no input provided
			return []ChatContext{}
		}
	}

	// Refresh messages after potential addition
	messages = (*state).GetConversation(n.key)

	context := ChatContext{
		Messages: messages,
	}

	return []ChatContext{context}
}

// getUserInput handles user input collection with proper validation
func (n *ChatNode[T]) getUserInput(messages *[]llm.Message) string {
	// Show welcome message only on first interaction
	if len(*messages) == 0 {
		fmt.Println("How may I help you today?")
	}

	fmt.Print("You: ")
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	lastLineEmpty := false

	for {
		if !scanner.Scan() {
			// Handle EOF or error
			if err := scanner.Err(); err != nil {
				log.Printf("Error reading input: %v", err)
			}
			break
		}

		line := scanner.Text()

		// Two consecutive blank lines = end
		if line == "" && lastLineEmpty {
			break
		}

		lastLineEmpty = (line == "")
		lines = append(lines, line)
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}

// Exec calls planning LLM with the prepared messages
func (n *ChatNode[T]) Exec(chatcontext ChatContext) (llm.Message, error) {
	// Validate context
	if chatcontext.Messages == nil || len(*chatcontext.Messages) == 0 {
		return llm.Message{}, fmt.Errorf("no messages to process")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Prepare messages with system prompt
	messages := n.prepareMessagesWithSystemPrompt(*chatcontext.Messages)

	// Call LLM provider
	response, err := n.llmProvider.CallLLM(ctx, messages)
	if err != nil {
		return llm.Message{}, fmt.Errorf("LLM call failed: %w", err)
	}

	return response, nil
}

// prepareMessagesWithSystemPrompt adds system prompt to messages if needed
func (n *ChatNode[T]) prepareMessagesWithSystemPrompt(messages []llm.Message) []llm.Message {
	// Check if first message is system message
	if len(messages) > 0 && messages[0].Role == llm.RoleSystem {
		return messages
	}

	// Prepend system message
	systemPrompt := n.buildSystemPromptWithTools("")
	systemMessage := llm.Message{
		Role:    llm.RoleSystem,
		Content: systemPrompt,
	}

	return append([]llm.Message{systemMessage}, messages...)
}

// Post processes LLM response, creates assistant message, and determines next action
func (n *ChatNode[T]) Post(state *T, prepResults []ChatContext, execResults ...llm.Message) core.Action {
	if len(execResults) == 0 {
		log.Println("No execution results received")
		return core.Action(ActionFailure)
	}

	execResult := execResults[0]

	// Check for maximum retry limit
	if n.errorRetryCount >= 3 {
		log.Printf("Maximum retry limit reached. Last response: %s", execResult.Content)
		n.errorRetryCount = 0 // Reset for next interaction
		return core.Action(ActionFailure)
	}

	// Parse the YAML response
	result, err := n.parseYAMLResponse(execResult.Content)
	if err != nil {
		n.errorRetryCount++
		log.Printf("Error parsing response: %v, retrying (%d/3)", err, n.errorRetryCount)

		// Add the failed response and error message to conversation
		(*state).AddMessage(llm.Message{
			Role:    llm.RoleAssistant,
			Content: execResult.Content,
		})
		(*state).AddMessage(llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("I apologize, but I encountered an error processing your response: %v. Please try again with the correct YAML format.", err),
		})
		return core.ActionRetry
	}

	// Validate response content
	if result.Response == "" && len(result.LLMToolCalls) == 0 {
		log.Println("Empty response and no tool calls")
		return core.Action(ActionFailure)
	}

	// Set tool calls on the message
	execResult.ToolCalls = result.LLMToolCalls

	// Add the assistant message to state
	(*state).AddMessage(execResult)

	// Reset error retry count on success
	n.errorRetryCount = 0

	// Display the response to user
	if result.Response != "" {
		fmt.Printf("\nAssistant: %s\n", result.Response)
	}

	// Handle tool calls if present
	if len(execResult.ToolCalls) > 0 {
		return n.handleToolCalls(state, execResult.ToolCalls)
	}

	// No tool calls, require user input for next interaction
	n.isUserInputRequired = true
	return core.ActionSuccess
}

// handleToolCalls processes tool calls with permission checking and execution
func (n *ChatNode[T]) handleToolCalls(state *T, toolCalls []llm.ToolCalls) core.Action {
	approvedTools := toolCalls
	var action core.Action = core.ActionSuccess

	// Check permissions if not set to always allow
	if n.toolUse != PermissionAllow {
		approvedTools, action = n.AskToolPermission(*state, toolCalls)
		if action == core.ActionFailure || action == core.ActionContinue {
			return action
		}
	}

	// Execute approved tools
	results := make([]llm.ToolResults, 0, len(approvedTools))
	responseContent := ""

	for _, tool := range approvedTools {
		result, err := n.toolManager.ExecuteTool(context.Background(), tool)
		if err != nil {
			log.Printf("Error executing tool %s: %v", tool.ToolName, err)
			result = llm.ToolResults{
				Id:      tool.Id,
				Content: result.Content,
				IsError: true,
				Error:   fmt.Sprintf("Tool execution failed: %v", err),
			}
		}
		results = append(results, result)

		// Build response content
		responseContent += fmt.Sprintf("## Tool %s result:\n%s\n", tool.ToolName, result.Content)
		if result.IsError {
			responseContent += fmt.Sprintf("Error: %s\n", result.Error)
		}
	}

	// Create tool results message
	if len(results) > 0 {
		responseMessage := llm.Message{
			Role:        llm.RoleUser,
			Content:     responseContent,
			ToolCalls:   approvedTools,
			ToolResults: results,
		}

		// Handle media from tool results
		for _, result := range results {
			if len(result.Media) > 0 {
				responseMessage.Media = result.Media
				responseMessage.MimeType = result.MetaData.ContentType
				break // Only handle first media result
			}
		}

		(*state).AddMessage(responseMessage)
	}

	return core.ActionSuccess
}

// AskToolPermission handles tool permission requests with improved input validation
func (n *ChatNode[T]) AskToolPermission(state T, availableTools []llm.ToolCalls) ([]llm.ToolCalls, core.Action) {
	if len(availableTools) == 0 {
		return []llm.ToolCalls{}, core.ActionSuccess
	}

	results := make([]llm.ToolCalls, 0, len(availableTools))
	scanner := bufio.NewScanner(os.Stdin)

	for _, tool := range availableTools {
		// Skip if already always allowed
		if _, ok := n.AlwaysAllowedTools[tool.ToolName]; ok {
			results = append(results, tool)
			continue
		}

		fmt.Printf("\nTool '%s' requires permission.\n", tool.ToolName)
		fmt.Printf("Arguments: %v\n", tool.ToolArgs)

		for {
			fmt.Print("Allow? [y=yes, n=no, a=always allow]: ")

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					log.Printf("Error reading input: %v", err)
				}
				return []llm.ToolCalls{}, core.ActionFailure
			}

			response := strings.TrimSpace(strings.ToLower(scanner.Text()))

			switch response {
			case "y", "yes":
				results = append(results, tool)
				goto nextTool
			case "n", "no":
				goto nextTool
			case "a", "always":
				n.AlwaysAllowedTools[tool.ToolName] = struct{}{}
				results = append(results, tool)
				goto nextTool
			case "":
				fmt.Println("Please enter a valid response (y/n/a)")
				continue
			default:
				// Treat other input as user message
				message := llm.Message{
					Role:    llm.RoleUser,
					Content: response,
				}
				state.AddMessage(message)
				return []llm.ToolCalls{}, core.Action(ActionContinue)
			}
		}
	nextTool:
	}

	return results, core.ActionSuccess
}

// ExecFallback provides a safe error response
func (n *ChatNode[T]) ExecFallback(err error) llm.Message {
	return llm.Message{
		Role:    llm.RoleAssistant,
		Content: fmt.Sprintf("I apologize, but I'm having trouble processing your request right now. Error: %v. Could you please try again?", err),
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
	promptBuilder.WriteString("\n\n")

	if summarizedHistory != "" {
		promptBuilder.WriteString("## Previous Conversation Summary:\n")
		promptBuilder.WriteString(summarizedHistory)
		promptBuilder.WriteString("\n\n")
	}

	if len(availableTools) > 0 {
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
	promptBuilder.WriteString("IMPORTANT: Use sequential tool calls only when there's a dependency between them. For independent operations, use a single tool call with all required arguments.\n")
	promptBuilder.WriteString("Analyze the following request and respond with the structured YAML format that must be parseable.\n")

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

// parseYAMLResponse parses the strict YAML response from LLM with better error handling
func (n *ChatNode[T]) parseYAMLResponse(responseContent string) (ParsedResult, error) {
	parsedResp, err := structured.ParseResponse[LLMResponse](responseContent)
	if err != nil {
		return ParsedResult{}, fmt.Errorf("failed to parse YAML response: %w", err)
	}

	response := parsedResp.Data
	if response == nil {
		return ParsedResult{}, fmt.Errorf("parsed response data is nil")
	}

	llmToolCalls := make([]llm.ToolCalls, 0, len(response.ToolCalls))

	for i, toolName := range response.ToolCalls {
		if i >= len(response.ToolArgs) {
			log.Printf("Warning: tool_calls and tool_args length mismatch at index %d", i)
			break
		}

		callID := fmt.Sprintf("call_%d_%d", time.Now().Unix(), i+1)

		llmToolCall := llm.ToolCalls{
			Id:       callID,
			ToolName: toolName,
			ToolArgs: response.ToolArgs[i],
		}
		llmToolCalls = append(llmToolCalls, llmToolCall)
	}

	return ParsedResult{
		Response:     response.Response,
		LLMToolCalls: llmToolCalls,
	}, nil
}
