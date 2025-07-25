package main

import (
	"fmt"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
)

// CreateAgentWorkflow creates the complete agent workflow with all nodes connected
func CreateAgentWorkflow(llmProvider llm.LLMProvider, toolManager *MCPToolManager, config *AgentWorkflowConfig) (*core.Flow[AgentState], error) {
	// Create all nodes
	userInputNode := core.NewNode(
		NewUserInputNode(config.UserInput),
		1, 1, // No retries needed for user input, single threaded
	)

	summarizerNode := core.NewNode(
		NewSummarizerNode(llmProvider, config.Summarizer),
		2, 1, // Retry summarization, single threaded
	)

	planningNode := core.NewNode(
		NewPlanningNode(llmProvider, config.Planning),
		3, 1, // Retry LLM calls, single threaded
	)

	approvalNode := core.NewNode(
		NewApprovalNode(config.Approval),
		1, 1, // No retries needed for user approval, single threaded
	)

	toolExecutionNode := core.NewNode(
		NewToolExecutionNode(toolManager, config.ToolExecution),
		2, 5, // Retry tool calls, parallel execution
	)

	// Connect the workflow graph based on the state machine
	userInputNode.AddSuccessor(summarizerNode, core.Action(ActionSummarize))
	userInputNode.AddSuccessor(planningNode, core.Action(ActionPlan))

	summarizerNode.AddSuccessor(planningNode, core.Action(ActionPlan))

	planningNode.AddSuccessor(approvalNode, core.Action(ActionRequestApproval))
	planningNode.AddSuccessor(userInputNode, core.Action(ActionContinue))

	approvalNode.AddSuccessor(toolExecutionNode, core.Action(ActionApprove))
	approvalNode.AddSuccessor(planningNode, core.Action(ActionReject))
	approvalNode.AddSuccessor(planningNode, core.Action(ActionAppend))

	toolExecutionNode.AddSuccessor(planningNode, ActionPlan)

	toolExecutionNode.AddSuccessor(userInputNode, core.Action(ActionContinue))

	// Create flow starting with user input
	return core.NewFlow(userInputNode), nil
}

// AgentWorkflow wraps the core flow with agent-specific functionality
type AgentWorkflow struct {
	flow  *core.Flow[AgentState]
	state *AgentState
}

// NewAgentWorkflow creates a new agent workflow instance
func NewAgentWorkflow(llmProvider llm.LLMProvider, toolManager *MCPToolManager, config *AgentWorkflowConfig) (*AgentWorkflow, error) {
	flow, err := CreateAgentWorkflow(llmProvider, toolManager, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	state := NewAgentState()
	state.AvailableTools = toolManager.GetAvailableTools()
	state.MaxHistory = config.Agent.MaxHistory

	return &AgentWorkflow{
		flow:  flow,
		state: state,
	}, nil
}

// Run executes the agent workflow
func (w *AgentWorkflow) Run(state *AgentState) error {
	// Run the workflow loop
	for state.Active {
		action := w.flow.Run(state)

		// Handle terminal actions
		switch action {
		case core.ActionSuccess:
			fmt.Println("Agent workflow completed successfully.")
			return nil
		case core.ActionFailure:
			fmt.Println("Agent workflow failed.")
			return fmt.Errorf("workflow failed")
		}

		// Continue the loop for other actions
	}

	return nil
}
