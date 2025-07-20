package pocketflow

import (
	"sync"
)

type Action string

// BaseNode defines the logic for a single step in the workflow.
type BaseNode interface {
	// Prep generates the work items for the Exec phase.
	Prep(state map[string]any) []string
	// Exec performs the core logic on a single work item.
	Exec(prepResult string) (string, error)
	// Post handles the results from all Exec calls.
	Post(state map[string]any, prepRes []string, execResult ...string) Action
	// ExecFallback provides a default result if Exec fails.
	ExecFallback(err error) string
}

// Node represents a single node in the workflow graph.
type Node struct {
	node        BaseNode
	maxRetries  int
	successors  map[Action]*Node
	routines    int
	prepResults chan task
	execResults []string
}

// task is a piece of data to be processed by a worker.
type task struct {
	pos    int
	result string
}

// worker is a goroutine that executes tasks. It was previously named Exec, which was confusing.
func (n *Node) worker(wg *sync.WaitGroup) {
	defer wg.Done()
	for item := range n.prepResults {
		var execResult string
		var err error
		// Try once, then retry up to maxRetries times. Total attempts: maxRetries + 1.
		for i := 0; i < n.maxRetries+1; i++ {
			execResult, err = n.node.Exec(item.result)
			if err == nil {
				n.execResults[item.pos] = execResult
				break
			}
		}
		if err != nil {
			fallbackResult := n.node.ExecFallback(err)
			n.execResults[item.pos] = fallbackResult
		}
	}
}

// Run executes the logic of the current node.
func (n *Node) Run(state map[string]any) Action {
	prepRes := n.node.Prep(state)
	if len(prepRes) == 0 {
		// Nothing to execute, just call Post.
		return n.node.Post(state, prepRes)
	}

	wg := &sync.WaitGroup{}
	n.prepResults = make(chan task, len(prepRes))
	n.execResults = make([]string, len(prepRes))

	numWorkers := n.routines
	if numWorkers > len(prepRes) {
		// Don't spawn more workers than there are items.
		numWorkers = len(prepRes)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go n.worker(wg)
	}

	for i, item := range prepRes {
		n.prepResults <- task{pos: i, result: item}
	}
	close(n.prepResults)
	wg.Wait()

	return n.node.Post(state, prepRes, n.execResults...)
}

// Next defines the next node in the workflow for a given action.
// If no action is provided, it sets the default next node.
func (n *Node) Next(basenode BaseNode, action ...Action) *Node {
	if n.successors == nil {
		n.successors = make(map[Action]*Node)
	}

	node := &Node{
		node:       basenode,
		maxRetries: n.maxRetries,
		routines:   n.routines,
		successors: make(map[Action]*Node),
	}

	if len(action) > 0 {
		n.successors[action[0]] = node
	} else {
		n.successors["default"] = node
	}
	return node
}

// Flow represents a series of nodes to be executed.
type Flow struct {
	*Node
}

// Run starts the execution of the workflow.
func (f *Flow) Run(state map[string]any) Action {
	var action Action
	node := f.Node

	for node != nil {
		action = node.Run(state)
		if nextNode, exists := node.successors[action]; exists {
			node = nextNode
		} else if defaultNode, exists := node.successors["default"]; exists {
			// Use default path if specific action is not handled
			node = defaultNode
		} else {
			node = nil // End of the flow
		}
	}
	return action
}

func CreateNode(basenode BaseNode, maxRetries int, maxRoutines int) *Node {
	if maxRoutines < 1 {
		// If routines is 0 or negative, it would hang. Default to 1 worker.
		maxRoutines = 1
	}
	return &Node{
		node:       basenode,
		maxRetries: maxRetries,
		routines:   maxRoutines,
	}
}
func CreateFlow(node *Node) *Flow {
	return &Flow{
		Node: node,
	}
}
