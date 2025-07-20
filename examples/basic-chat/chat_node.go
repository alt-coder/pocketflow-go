package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// ChatState represents the conversation state
type ChatState struct {
	Messages []llm.Message // Full conversation history using generic format
	Active   bool          // Whether chat is active
}

// PrepResult contains the conversation context for LLM call
type PrepResult struct {
	Messages  []llm.Message // Conversation history
}

// ExecResult contains the LLM response
type ExecResult struct {
	Response string // LLM response text
	Error    error  // Any error that occurred
}

// ChatNode implements BaseNode interface for conversational chat
type ChatNode struct {
	llmProvider llm.LLMProvider // LLM provider for generating responses
	config      *ChatConfig     // Configuration settings
	firstRun    bool            // Track if this is the first execution
}

// NewChatNode creates a new chat node with the specified LLM provider and configuration
func NewChatNode(provider llm.LLMProvider, config *ChatConfig) *ChatNode {
	return &ChatNode{
		llmProvider: provider,
		config:      config,
		firstRun:    true,
	}
}

// Prep implements the preparation phase of the three-phase execution model
// It displays welcome message, prompts for user input, and prepares conversation context
func (c *ChatNode) Prep(state *ChatState) []PrepResult {
	// Display welcome message on first run
	if c.firstRun {
		fmt.Println(c.config.WelcomeMsg)
		c.firstRun = false
	}

	// Prompt user for input
	fmt.Print("You: ")
	reader := bufio.NewReader(os.Stdin)
	userInput, err := reader.ReadString('\n')
	
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return []PrepResult{}
	}

	input := strings.TrimSpace(userInput)

	// Handle 'exit' command (case-insensitive)
	if strings.ToLower(input) == "exit" {
		state.Active = false
		return []PrepResult{} // Return empty to signal termination
	}

	// Handle empty input gracefully
	if input == "" {
		fmt.Println("Please enter a message or 'exit' to quit.")
		return c.Prep(state) // Recursively ask for input again
	}

	state.Messages = append(state.Messages, llm.Message{
		Role:    llm.RoleUser,
		Content: input,
	})

	// Return PrepResult with conversation context
	return []PrepResult{
		{
			Messages:  state.Messages,	
		},
	}
}

// Exec implements the execution phase - makes LLM API call
func (c *ChatNode) Exec(prepResult PrepResult) (ExecResult, error) {
	if len(prepResult.Messages) == 0 {
		return ExecResult{
			Response: "No conversation history provided.",
			Error:    nil, // No error, just a fallback response
		}, nil
	}
	// Make API call to LLM provider with conversation history
	response, err := c.llmProvider.CallLLM(context.Background(), prepResult.Messages)
	if err != nil {
		return ExecResult{
			Response: "",
			Error:    fmt.Errorf("LLM API call failed: %w", err),
		}, err
	}

	return ExecResult{
		Response: response.Content,
		Error:    nil,
	}, nil
}

// Post implements the post-processing phase
// It processes the LLM response, updates conversation history, and determines next action
func (c *ChatNode) Post(state *ChatState, prepResults []PrepResult, execResults ...ExecResult) core.Action {
	// Handle case where no prep results (exit command)
	if len(prepResults) == 0 {
		fmt.Println("Goodbye!")
		return core.ActionSuccess // Terminate the conversation
	}

	// Get the first (and only) prep and exec results
	execResult := execResults[0]

	// Handle execution errors
	if execResult.Error != nil {
		fmt.Printf("Error: %v\n", execResult.Error)
		// Continue conversation even after errors
		return core.ActionContinue
	}

	// Display the assistant's response
	fmt.Printf("Assistant: %s\n\n", execResult.Response)

	// Add assistant message
	assistantMessage := llm.Message{
		Role:    llm.RoleAssistant,
		Content: execResult.Response,
	}
	
	state.Messages = append(state.Messages, assistantMessage)

	// Trim conversation history if it exceeds maximum length
	if len(state.Messages) > c.config.MaxHistory {
		state.Messages = state.Messages[len(state.Messages)-c.config.MaxHistory:]
	}
	// Continue the conversation loop
	return core.ActionContinue
}

// ExecFallback provides a fallback response when Exec fails after all retries
func (c *ChatNode) ExecFallback(err error) ExecResult {
	return ExecResult{
		Response: "I'm sorry, I encountered an error and couldn't process your request. Please try again.",
		Error:    nil, // Don't propagate the error to continue conversation
	}
}