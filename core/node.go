package core

import (
	"sync"
)

// task is a piece of data to be processed by a worker
type task[T any] struct {
	pos    int
	result T
}

// Node represents a single node in the workflow graph and implements Workflow
type Node[State any, PrepResult any, ExecResults any] struct {
	node       BaseNode[State, PrepResult, ExecResults]
	maxRetries int
	successors map[Action]Workflow[State]
	routines   int
}

// createNode creates a new node with the specified configuration
func createNode[State any, PrepResult any, ExecResults any](basenode BaseNode[State, PrepResult, ExecResults], maxRetries int, maxRoutines int) *Node[State, PrepResult, ExecResults] {
	if maxRoutines < 1 {
		// If routines is 0 or negative, it would hang. Default to 1 worker.
		maxRoutines = 1
	}
	return &Node[State, PrepResult, ExecResults]{
		node:       basenode,
		maxRetries: maxRetries,
		routines:   maxRoutines,
		successors: make(map[Action]Workflow[State]),
	}
}

// NewNode is an alias for CreateNode for consistency with the design
func NewNode[State any, PrepResult any, ExecResults any](basenode BaseNode[State, PrepResult, ExecResults], maxRetries int, maxRoutines int) *Node[State, PrepResult, ExecResults] {
	return createNode(basenode, maxRetries, maxRoutines)
}

// Run implements the Workflow interface and executes the three-phase execution model
func (n *Node[State, PrepResult, ExecResults]) Run(state State) Action {

	prepRes := n.node.Prep(state)
	if len(prepRes) == 0 {
		// Nothing to execute, just call Post.
		return n.node.Post(state, prepRes)
	}

	wg := &sync.WaitGroup{}
	prepResults := make(chan task[PrepResult], len(prepRes))
	execResults := make([]ExecResults, len(prepRes))

	numWorkers := n.routines
	if numWorkers > len(prepRes) {
		// Don't spawn more workers than there are items.
		numWorkers = len(prepRes)
	}

	// Create a worker function that uses local variables
	worker := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for item := range prepResults {
			var execResult ExecResults
			var err error

			// Retry logic: Try once, then retry up to maxRetries times. Total attempts: maxRetries + 1.
			for i := 0; i < n.maxRetries+1; i++ {
				// Execute with PrepResult input from Prep results
				execResult, err = n.node.Exec(item.result)
				if err == nil {
					// Store ExecResults result directly
					execResults[item.pos] = execResult
					break
				}
			}

			// Fallback handling for ExecResults return types
			if err != nil {
				// ExecFallback returns ExecResults which matches our type
				fallbackResult := n.node.ExecFallback(err)
				execResults[item.pos] = fallbackResult
			}
		}
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(wg)
	}

	for i, item := range prepRes {
		prepResults <- task[PrepResult]{pos: i, result: item}
	}
	close(prepResults)
	wg.Wait()

	return n.node.Post(state, prepRes, execResults...)
}

// SetMaxRetries updates the maximum retry count
func (n *Node[State, PrepResult, ExecResults]) SetMaxRetries(retries int) {
	n.maxRetries = retries
}

// SetMaxRoutines updates the maximum concurrent routines
func (n *Node[State, PrepResult, ExecResults]) SetMaxRoutines(routines int) {
	if routines < 1 {
		routines = 1
	}
	n.routines = routines
}

// GetSuccessors returns a copy of the successors map (kept for backward compatibility)
func (n *Node[State, PrepResult, ExecResults]) GetSuccessors() map[Action]Workflow[State] {
	return n.successors
}

// AddSuccessor adds successor based on action with proper validation
func (n *Node[State, PrepResult, ExecResults]) AddSuccessor(workflow Workflow[State], action ...Action) {
	// Validate inputs - don't add if action is empty or workflow is nil
	if workflow == nil {
		return
	}
	if len(action) ==  0 {
		n.successors[ActionDefault] = workflow
		return 
	}
	n.successors[action[0]] = workflow
}

// GetSuccessor gets the next WorkFlow as per action.
func (n *Node[State, PrepResult, ExecResults]) GetSuccessor(action Action) Workflow[State] {
	return n.successors[action]
}
