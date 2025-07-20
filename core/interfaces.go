package core


// BaseNode defines the core interface for all nodes in the workflow
// This follows the three-phase execution model: Prep -> Exec -> Post
type BaseNode[State any , PrepResult any, ExecResults any] interface {
	// Prep generates the work items for the Exec phase
	Prep(state *State) []PrepResult

	// Exec performs the core logic on a single work item
	Exec(prepResult PrepResult) (ExecResults, error)

	// Post handles the results from all Exec calls and determines next action
	Post(state *State, prepRes []PrepResult, execResults ...ExecResults) Action

	// ExecFallback provides a default result if Exec fails after all retries
	ExecFallback(err error) ExecResults
}

// Workflow represents a unit of execution that can be connected to other workflows
// This interface is implemented by both Node and Flow to enable composition
type Workflow[State any] interface {
	// Run executes the workflow logic and returns an action for routing
	Run(state *State) Action

	// GetSuccessor returns the successor workflow for a given action
	GetSuccessor(action Action) Workflow[State]

	// AddSuccessor connects a successor workflow for a specific action
	AddSuccessor( successor Workflow[State], action ...Action,) Workflow[State]
}
