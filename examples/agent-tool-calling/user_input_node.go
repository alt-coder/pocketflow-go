package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// UserInputNode handles user input and determines next action
type UserInputNode struct {
	config *UserInputConfig
}

// NewUserInputNode creates a new user input node
func NewUserInputNode(config *UserInputConfig) *UserInputNode {
	if config == nil {
		config = &UserInputConfig{
			Prompt:                 "You: ",
			ExitCommands:           []string{"exit", "quit", "bye"},
			SummarizationThreshold: 15,
		}
	}

	return &UserInputNode{
		config: config,
	}
}

// Prep displays prompt and captures user input
func (n *UserInputNode) Prep(state *AgentState) []UserInput {
	if !state.Active {
		return []UserInput{} // No input needed if not active
	}

	// Display prompt and get user input
	fmt.Print(n.config.Prompt)
	scanner := bufio.NewScanner(os.Stdin)

	var input string
	if scanner.Scan() {
		input = strings.TrimSpace(scanner.Text())
	}

	if scanner.Err() != nil {
		fmt.Printf("Error reading input: %v\n", scanner.Err())
		return []UserInput{}
	}

	return []UserInput{{Text: input}}
}

// Exec processes and validates user input
func (n *UserInputNode) Exec(input UserInput) (UserInputResult, error) {
	// Check for exit commands
	lowerInput := strings.ToLower(input.Text)
	for _, exitCmd := range n.config.ExitCommands {
		if lowerInput == strings.ToLower(exitCmd) {
			return UserInputResult{
				ProcessedInput: input.Text,
				ShouldExit:     true,
				NeedsSummary:   false,
			}, nil
		}
	}

	// Check if input is empty
	if strings.TrimSpace(input.Text) == "" {
		return UserInputResult{
			ProcessedInput: "",
			ShouldExit:     false,
			NeedsSummary:   false,
		}, fmt.Errorf("empty input")
	}

	return UserInputResult{
		ProcessedInput: input.Text,
		ShouldExit:     false,
		NeedsSummary:   false, // Will be determined in Post based on message count
	}, nil
}

// Post updates state with user message and determines next action
func (n *UserInputNode) Post(state *AgentState, prepResults []UserInput, execResults ...UserInputResult) core.Action {
	// Handle case where no prep results (not active)
	if len(prepResults) == 0 {
		state.Active = false
		return core.ActionSuccess
	}

	// Handle execution errors or empty results
	if len(execResults) == 0 {
		fmt.Println("Please enter a valid command.")
		return core.Action(ActionContinue) // Stay in user input loop
	}

	result := execResults[0]

	// Handle exit request
	if result.ShouldExit {
		state.Active = false
		fmt.Println("Goodbye!")
		return core.ActionSuccess
	}

	// Handle empty input
	if result.ProcessedInput == "" {
		return core.Action(ActionContinue) // Stay in user input loop
	}

	// Add user message to conversation
	userMessage := llm.Message{
		Role:    llm.RoleUser,
		Content: result.ProcessedInput,
	}
	state.AddMessage(userMessage)

	// Clear previous tool cycle data for new turn
	state.ClearCompletedToolCycle()

	// Check if summarization is needed
	if len(state.ActualMessages) > n.config.SummarizationThreshold {
		state.SummarizationNeeded = true
		return core.Action(ActionSummarize)
	}

	// Proceed to planning
	return core.Action(ActionPlan)
}

// ExecFallback handles execution failures
func (n *UserInputNode) ExecFallback(err error) UserInputResult {
	return UserInputResult{
		ProcessedInput: "",
		ShouldExit:     false,
		NeedsSummary:   false,
	}
}
