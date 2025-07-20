package core

// Flow represents a workflow subgraph that implements Workflow interface
type Flow[State any] struct {
	startNode  Workflow[State]
	successors map[Action]Workflow[State]
}

// NewFlow creates a new flow with the given initial state
func NewFlow[State any](startNode Workflow[State]) *Flow[State] {

	return &Flow[State]{
		startNode:  startNode,
		successors: make(map[Action]Workflow[State]),
	}
}

// Run implements the Workflow interface - executes the flow and returns an action
func (f *Flow[State]) Run(state *State) Action {
	// Create working state copy to avoid modifying input state

	currentWorkflow := f.startNode
	if currentWorkflow == nil {
		return ActionFailure
	}
	var finalAction Action = ActionSuccess

	// Execute workflows in sequence following action-based transitions
	for currentWorkflow != nil {
		// Run the current workflow with working state
		action := currentWorkflow.Run(state)
		finalAction = action

		// Use GetSuccessor method for proper action-based routing
		nextWorkflow := currentWorkflow.GetSuccessor(action)

		// If no successor found in current workflow, check flow-level successors
		if nextWorkflow == nil {
			nextWorkflow = f.GetSuccessor(action)
		}

		// Update current workflow for next iteration
		currentWorkflow = nextWorkflow
	}
	return finalAction
}

// GetSuccessor implements the Workflow interface - returns the successor workflow for a given action
func (f *Flow[State]) GetSuccessor(action Action) Workflow[State] {
	return f.successors[action]
}

// AddSuccessor implements the Workflow interface - connects a successor workflow for a specific action
func (f *Flow[State]) AddSuccessor(successor Workflow[State], action ...Action) Workflow[State] {
	if f.successors == nil {
		f.successors = make(map[Action]Workflow[State])
	}
	if successor == nil {
		return successor
	}
	if len(action) == 0 {
		action = append(action, ActionSuccess)
	}
	f.successors[action[0]] = successor
	return successor
}
