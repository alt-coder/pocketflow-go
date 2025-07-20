package core

import (
	"reflect"
	"strings"
	"testing"
)

// MockWorkflow is a test implementation of the Workflow interface
type MockWorkflow[T State] struct {
	name       string
	runAction  Action
	runCalled  bool
	successors map[Action]Workflow[T]
}

func NewMockWorkflow[T State](name string, runAction Action) *MockWorkflow[T] {
	return &MockWorkflow[T]{
		name:       name,
		runAction:  runAction,
		successors: make(map[Action]Workflow[T]),
	}
}

func (m *MockWorkflow[State]) Run(state *State) Action {
	m.runCalled = true
	// Mark that this workflow was executed in the state

	if state != nil {
		(*state)[m.name+"_executed"] = true
	}
	return m.runAction
}

func (m *MockWorkflow[State]) GetSuccessor(action Action) Workflow[State] {
	return m.successors[action]
}

func (m *MockWorkflow[State]) AddSuccessor(successor Workflow[State], action ...Action) Workflow[State] {
	if m.successors == nil {
		m.successors = make(map[Action]Workflow[State])
	}
	if len(action) == 0 {
		action = []Action{m.runAction}
	}
	m.successors[action[0]] = successor
	return successor
}

// TestWorkflowInterface_GetSuccessor tests that GetSuccessor method returns correct workflow for actions
func TestWorkflowInterface_GetSuccessor(t *testing.T) {
	tests := []struct {
		name           string
		setupWorkflow  func() Workflow[State]
		testAction     Action
		expectedResult bool // true if successor should exist, false if nil
		description    string
	}{
		{
			name: "Node GetSuccessor with existing action",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{postAction: ActionSuccess}
				node := createNode(baseNode, 1, 1)
				successor := NewMockWorkflow("successor", ActionSuccess)
				node.AddSuccessor(successor, ActionContinue)
				return node
			},
			testAction:     ActionContinue,
			expectedResult: true,
			description:    "Node should return successor for existing action",
		},
		{
			name: "Node GetSuccessor with non-existing action",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{postAction: ActionSuccess}
				node := createNode(baseNode, 1, 1)
				return node
			},
			testAction:     ActionFailure,
			expectedResult: false,
			description:    "Node should return nil for non-existing action",
		},
		{
			name: "Flow GetSuccessor with existing action",
			setupWorkflow: func() Workflow[State] {
				startNode := NewMockWorkflow("start", ActionSuccess)
				flow := NewFlow(startNode)
				successor := NewMockWorkflow("successor", ActionSuccess)
				flow.AddSuccessor(successor, ActionSuccess)
				return flow
			},
			testAction:     ActionSuccess,
			expectedResult: true,
			description:    "Flow should return successor for existing action",
		},
		{
			name: "Flow GetSuccessor with non-existing action",
			setupWorkflow: func() Workflow[State] {
				startNode := NewMockWorkflow("start", ActionSuccess)
				flow := NewFlow(startNode)
				return flow
			},
			testAction:     ActionFailure,
			expectedResult: false,
			description:    "Flow should return nil for non-existing action",
		},
		{
			name: "MockWorkflow GetSuccessor with multiple actions",
			setupWorkflow: func() Workflow[State] {
				mock := NewMockWorkflow("mock", ActionSuccess)
				successor1 := NewMockWorkflow("successor1", ActionSuccess)
				successor2 := NewMockWorkflow("successor2", ActionFailure)
				mock.AddSuccessor(successor1, ActionContinue)
				mock.AddSuccessor(successor2, ActionFailure)
				return mock
			},
			testAction:     ActionContinue,
			expectedResult: true,
			description:    "MockWorkflow should return correct successor for specific action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := tt.setupWorkflow()
			result := workflow.GetSuccessor(tt.testAction)

			if tt.expectedResult {
				if result == nil {
					t.Errorf("GetSuccessor() = nil, expected non-nil successor for action %v", tt.testAction)
				}
			} else {
				if result != nil {
					t.Errorf("GetSuccessor() = %v, expected nil for action %v", result, tt.testAction)
				}
			}
		})
	}
}

// TestWorkflowInterface_AddSuccessor tests that AddSuccessor method properly connects workflows
func TestWorkflowInterface_AddSuccessor(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() Workflow[State]
		testActions   []Action
		description   string
	}{
		{
			name: "Node AddSuccessor single connection",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{postAction: ActionSuccess}
				return createNode(baseNode, 1, 1)
			},
			testActions: []Action{ActionContinue},
			description: "Node should properly add single successor",
		},
		{
			name: "Node AddSuccessor multiple connections",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{postAction: ActionSuccess}
				return createNode(baseNode, 1, 1)
			},
			testActions: []Action{ActionContinue, ActionFailure, ActionRetry},
			description: "Node should properly add multiple successors",
		},
		{
			name: "Flow AddSuccessor single connection",
			setupWorkflow: func() Workflow[State] {
				startNode := NewMockWorkflow("start", ActionSuccess)
				return NewFlow(startNode)
			},
			testActions: []Action{ActionSuccess},
			description: "Flow should properly add single successor",
		},
		{
			name: "Flow AddSuccessor multiple connections",
			setupWorkflow: func() Workflow[State] {
				startNode := NewMockWorkflow("start", ActionSuccess)
				return NewFlow(startNode)
			},
			testActions: []Action{ActionSuccess, ActionFailure, ActionContinue},
			description: "Flow should properly add multiple successors",
		},
		{
			name: "MockWorkflow AddSuccessor comprehensive test",
			setupWorkflow: func() Workflow[State] {
				return NewMockWorkflow("mock", ActionSuccess)
			},
			testActions: []Action{ActionContinue, ActionFailure, ActionRetry, ActionDefault},
			description: "MockWorkflow should properly add all types of successors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := tt.setupWorkflow()

			// Add successors for each test action
			successors := make(map[Action]Workflow[State])
			for _, action := range tt.testActions {
				successor := NewMockWorkflow(string(action)+"_successor", ActionSuccess)
				successors[action] = successor
				workflow.AddSuccessor(successor, action)
			}

			// Verify each successor was added correctly
			for action, expectedSuccessor := range successors {
				actualSuccessor := workflow.GetSuccessor(action)
				if actualSuccessor != expectedSuccessor {
					t.Errorf("AddSuccessor() failed for action %v: got %v, expected %v", action, actualSuccessor, expectedSuccessor)
				}
			}

			// Verify non-added actions return nil
			nonExistentAction := Action("non_existent")
			if workflow.GetSuccessor(nonExistentAction) != nil {
				t.Errorf("GetSuccessor() should return nil for non-existent action %v", nonExistentAction)
			}
		})
	}
}

// TestWorkflowInterface_AddSuccessor_EdgeCases tests edge cases for AddSuccessor
func TestWorkflowInterface_AddSuccessor_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() Workflow[State]
		testAction    Action
		testSuccessor Workflow[State]
		shouldAdd     bool
		description   string
	}{
		{
			name: "Node AddSuccessor with nil successor",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{postAction: ActionSuccess}
				return createNode(baseNode, 1, 1)
			},
			testAction:    ActionContinue,
			testSuccessor: nil,
			shouldAdd:     false,
			description:   "Node should not add nil successor",
		},
		{
			name: "Flow AddSuccessor with nil successor",
			setupWorkflow: func() Workflow[State] {
				startNode := NewMockWorkflow("start", ActionSuccess)
				return NewFlow(startNode)
			},
			testAction:    ActionContinue,
			testSuccessor: nil,
			shouldAdd:     true, // Flow doesn't validate nil successors
			description:   "Flow allows nil successor (no validation)",
		},
		{
			name: "Flow AddSuccessor with valid parameters",
			setupWorkflow: func() Workflow[State] {
				startNode := NewMockWorkflow("start", ActionSuccess)
				return NewFlow(startNode)
			},
			testAction:    ActionSuccess,
			testSuccessor: NewMockWorkflow("successor", ActionSuccess),
			shouldAdd:     true,
			description:   "Flow should add successor with valid parameters",
		},
		{
			name: "MockWorkflow AddSuccessor overwrite existing",
			setupWorkflow: func() Workflow[State] {
				mock := NewMockWorkflow("mock", ActionSuccess)
				// Pre-add a successor
				oldSuccessor := NewMockWorkflow("old_successor", ActionFailure)
				mock.AddSuccessor(oldSuccessor, ActionContinue)
				return mock
			},
			testAction:    ActionContinue,
			testSuccessor: NewMockWorkflow("new_successor", ActionSuccess),
			shouldAdd:     true,
			description:   "MockWorkflow should overwrite existing successor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := tt.setupWorkflow()
			workflow.AddSuccessor(tt.testSuccessor, tt.testAction)

			result := workflow.GetSuccessor(tt.testAction)
			if tt.shouldAdd {
				if result != tt.testSuccessor {
					t.Errorf("AddSuccessor() should have added successor: got %v, expected %v", result, tt.testSuccessor)
				}
			} else {
				// For cases where we expect the successor NOT to be added
				if tt.testSuccessor == nil {
					// When testing nil successor, we expect GetSuccessor to return nil
					if result != nil {
						t.Errorf("AddSuccessor() should not have added nil successor, but GetSuccessor returned %v", result)
					}
				} else {
					// For other invalid cases, successor should not match
					if result == tt.testSuccessor {
						t.Errorf("AddSuccessor() should not have added successor with invalid parameters")
					}
				}
			}
		})
	}
}

// TestWorkflowInterface_Run tests that Run method maintains proper execution flow
func TestWorkflowInterface_Run(t *testing.T) {
	tests := []struct {
		name            string
		setupWorkflow   func() Workflow[State]
		inputState      State
		expectedAction  Action
		stateValidation func(State) bool
		description     string
	}{
		{
			name: "Node Run with successful execution",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{
					prepResults: []any{"task1", "task2"},
					execResults: map[any]any{
						"task1": "result1",
						"task2": "result2",
					},
					postAction: ActionSuccess,
				}
				return createNode(baseNode, 1, 2)
			},
			inputState:     State{"input": "test"},
			expectedAction: ActionSuccess,
			stateValidation: func(state State) bool {
				return state["input"] == "test" // State should be preserved
			},
			description: "Node should execute successfully and return correct action",
		},
		{
			name: "Node Run with empty prep results",
			setupWorkflow: func() Workflow[State] {
				baseNode := &TestBaseNode{
					prepResults: []any{}, // Empty prep results
					postAction:  ActionContinue,
				}
				return createNode(baseNode, 1, 1)
			},
			inputState:     State{"empty": "test"},
			expectedAction: ActionContinue,
			stateValidation: func(state State) bool {
				return state["empty"] == "test"
			},
			description: "Node should handle empty prep results and call Post directly",
		},
		{
			name: "Flow Run with single node",
			setupWorkflow: func() Workflow[State] {
				mockNode := NewMockWorkflow("single_node", ActionSuccess)
				return NewFlow(mockNode)
			},
			inputState:     State{"flow": "test"},
			expectedAction: ActionSuccess,
			stateValidation: func(state State) bool {
				return state["single_node_executed"] == true
			},
			description: "Flow should execute single node and return its action",
		},
		{
			name: "MockWorkflow Run basic execution",
			setupWorkflow: func() Workflow[State] {
				return NewMockWorkflow("mock", ActionRetry)
			},
			inputState:     State{"mock": "test"},
			expectedAction: ActionRetry,
			stateValidation: func(state State) bool {
				return state["mock_executed"] == true
			},
			description: "MockWorkflow should execute and mark state correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := tt.setupWorkflow()
			result := workflow.Run(&tt.inputState)

			if result != tt.expectedAction {
				t.Errorf("Run() = %v, expected %v", result, tt.expectedAction)
			}

			if tt.stateValidation != nil && !tt.stateValidation(tt.inputState) {
				t.Errorf("Run() state validation failed for test: %s", tt.description)
			}
		})
	}
}

// TestWorkflowInterface_ActionBasedRouting tests action-based routing between connected workflows
func TestWorkflowInterface_ActionBasedRouting(t *testing.T) {
	tests := []struct {
		name               string
		setupWorkflow      func() Workflow[State]
		expectedExecutions []string
		finalAction        Action
		description        string
	}{
		{
			name: "Simple Node to Node routing",
			setupWorkflow: func() Workflow[State] {
				// Create first mock node that returns ActionContinue
				node1 := NewMockWorkflow("node1", ActionContinue)

				// Create second mock node that returns ActionSuccess
				node2 := NewMockWorkflow("node2", ActionSuccess)

				// Connect nodes
				node1.AddSuccessor(node2, ActionContinue)

				// Create flow starting with node1
				return NewFlow(node1)
			},
			expectedExecutions: []string{"node1_executed", "node2_executed"},
			finalAction:        ActionSuccess,
			description:        "Flow should route from node1 to node2 based on ActionContinue",
		},
		{
			name: "Node to Flow routing",
			setupWorkflow: func() Workflow[State] {
				// Create node that routes to a subflow
				baseNode := &TestBaseNode{
					prepResults: []any{"main_task"},
					execResults: map[any]any{"main_task": "main_result"},
					postAction:  ActionContinue,
				}
				mainNode := createNode(baseNode, 1, 1)

				// Create subflow
				subflowNode := NewMockWorkflow("subflow_node", ActionSuccess)
				subflow := NewFlow(subflowNode)

				// Connect main node to subflow
				mainNode.AddSuccessor(subflow, ActionContinue)

				return NewFlow(mainNode)
			},
			expectedExecutions: []string{"subflow_node_executed"},
			finalAction:        ActionSuccess,
			description:        "Flow should route from node to subflow based on action",
		},
		{
			name: "Flow to Flow routing",
			setupWorkflow: func() Workflow[State] {
				// Create first subflow
				subflow1Node := NewMockWorkflow("subflow1_node", ActionContinue)
				subflow1 := NewFlow(subflow1Node)

				// Create second subflow
				subflow2Node := NewMockWorkflow("subflow2_node", ActionSuccess)
				subflow2 := NewFlow(subflow2Node)

				// Connect subflows
				subflow1.AddSuccessor(subflow2, ActionContinue)

				return subflow1
			},
			expectedExecutions: []string{"subflow1_node_executed", "subflow2_node_executed"},
			finalAction:        ActionSuccess,
			description:        "Flow should route from subflow1 to subflow2 based on action",
		},
		{
			name: "Complex multi-path routing",
			setupWorkflow: func() Workflow[State] {
				// Create router node that can go to different paths
				routerNode := NewMockWorkflow("router", ActionContinue)

				// Create success path
				successNode := NewMockWorkflow("success_path", ActionSuccess)

				// Create failure path (not used in this test)
				failureNode := NewMockWorkflow("failure_path", ActionFailure)

				// Connect router to both paths
				routerNode.AddSuccessor(successNode, ActionContinue)
				routerNode.AddSuccessor(failureNode, ActionFailure)

				return NewFlow(routerNode)
			},
			expectedExecutions: []string{"router_executed", "success_path_executed"},
			finalAction:        ActionSuccess,
			description:        "Flow should route to correct path based on router action",
		},
		{
			name: "Chain routing with multiple hops",
			setupWorkflow: func() Workflow[State] {
				// Create chain: node1 -> node2 -> node3
				node1 := NewMockWorkflow("node1", ActionContinue)
				node2 := NewMockWorkflow("node2", ActionContinue)
				node3 := NewMockWorkflow("node3", ActionSuccess)

				// Connect chain
				node1.AddSuccessor(node2, ActionContinue)
				node2.AddSuccessor(node3, ActionContinue)

				return NewFlow(node1)
			},
			expectedExecutions: []string{"node1_executed", "node2_executed", "node3_executed"},
			finalAction:        ActionSuccess,
			description:        "Flow should route through entire chain based on actions",
		},
		{
			name: "No successor routing (terminates)",
			setupWorkflow: func() Workflow[State] {
				// Create node with no successors
				terminalNode := NewMockWorkflow("terminal", ActionSuccess)
				return NewFlow(terminalNode)
			},
			expectedExecutions: []string{"terminal_executed"},
			finalAction:        ActionSuccess,
			description:        "Flow should terminate when no successor exists for action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := tt.setupWorkflow()
			state := State{}

			result := workflow.Run(&state)

			// Verify final action
			if result != tt.finalAction {
				t.Errorf("Run() = %v, expected %v", result, tt.finalAction)
			}

			// Verify expected executions occurred
			for _, expectedExecution := range tt.expectedExecutions {
				if executed, exists := state[expectedExecution]; !exists || executed != true {
					t.Errorf("Expected execution %s did not occur or was not marked in state", expectedExecution)
				}
			}

			// Verify no unexpected executions occurred (for complex routing tests)
			if tt.name == "Complex multi-path routing" {
				if executed, exists := state["failure_path_executed"]; exists && executed == true {
					t.Error("Failure path should not have been executed")
				}
			}
		})
	}
}

// TestWorkflowInterface_ActionBasedRouting_EdgeCases tests edge cases in action-based routing
func TestWorkflowInterface_ActionBasedRouting_EdgeCases(t *testing.T) {
	t.Run("Action mismatch routing", func(t *testing.T) {
		// Test case where node returns action that has no successor
		node1 := NewMockWorkflow("node1", ActionFailure) // Returns ActionFailure
		node2 := NewMockWorkflow("node2", ActionSuccess)

		// Connect for ActionContinue, but node1 returns ActionFailure
		node1.AddSuccessor(node2, ActionContinue)

		workflow := NewFlow(node1)
		state := State{"mismatch_test": true}

		result := workflow.Run(&state)

		// Should terminate with ActionFailure since no successor exists for that action
		if result != ActionFailure {
			t.Errorf("Expected ActionFailure for action mismatch, got %v", result)
		}

		// Verify only node1 executed
		if state["node1_executed"] != true {
			t.Error("Node1 should have executed")
		}
		if state["node2_executed"] == true {
			t.Error("Node2 should not have executed due to action mismatch")
		}
	})
}

// TestWorkflowInterface_Compliance verifies that Node and Flow implement Workflow interface correctly
func TestWorkflowInterface_Compliance(t *testing.T) {
	// Verify Node implements Workflow
	var _ Workflow[State] = (*Node[State, any, any])(nil)

	// Verify Flow implements Workflow
	var _ Workflow[State] = (*Flow[State])(nil)

	// Verify MockWorkflow implements Workflow
	var _ Workflow[State] = (*MockWorkflow[State])(nil)

	// Test that all implementations have the required methods
	baseNode := &TestBaseNode{postAction: ActionSuccess}
	node := NewNode[State, any, any](baseNode, 1, 1)

	// Test Node method signatures - check that methods exist and have correct basic structure
	runType := reflect.TypeOf(node.Run).String()
	if !strings.Contains(runType, "func(") || !strings.Contains(runType, "State)") || !strings.Contains(runType, "Action") {
		t.Errorf("Node.Run method signature incorrect: %s", runType)
	}

	getSuccessorType := reflect.TypeOf(node.GetSuccessor).String()
	if !strings.Contains(getSuccessorType, "func(") || !strings.Contains(getSuccessorType, "Action)") || !strings.Contains(getSuccessorType, "Workflow") {
		t.Errorf("Node.GetSuccessor method signature incorrect: %s", getSuccessorType)
	}

	// AddSuccessor signature is variadic with new interface
	addSuccessorType := reflect.TypeOf(node.AddSuccessor).String()
	if !strings.Contains(addSuccessorType, "func(") || !strings.Contains(addSuccessorType, "Workflow") || !strings.Contains(addSuccessorType, "...") {
		t.Errorf("Node.AddSuccessor method signature incorrect: %s", addSuccessorType)
	}

	// Test Flow method signatures - check that methods exist and have correct basic structure
	flow := NewFlow(node)
	flowRunType := reflect.TypeOf(flow.Run).String()
	if !strings.Contains(flowRunType, "func(") || !strings.Contains(flowRunType, "State)") || !strings.Contains(flowRunType, "Action") {
		t.Errorf("Flow.Run method signature incorrect: %s", flowRunType)
	}

	flowGetSuccessorType := reflect.TypeOf(flow.GetSuccessor).String()
	if !strings.Contains(flowGetSuccessorType, "func(") || !strings.Contains(flowGetSuccessorType, "Action)") || !strings.Contains(flowGetSuccessorType, "Workflow") {
		t.Errorf("Flow.GetSuccessor method signature incorrect: %s", flowGetSuccessorType)
	}

	// AddSuccessor signature is variadic with new interface
	flowAddSuccessorType := reflect.TypeOf(flow.AddSuccessor).String()
	if !strings.Contains(flowAddSuccessorType, "func(") || !strings.Contains(flowAddSuccessorType, "Workflow") || !strings.Contains(flowAddSuccessorType, "...") {
		t.Errorf("Flow.AddSuccessor method signature incorrect: %s", flowAddSuccessorType)
	}

	// Test MockWorkflow method signatures
	mock := NewMockWorkflow[State]("test", ActionSuccess)
	mockRunType := reflect.TypeOf(mock.Run).String()
	if !strings.Contains(mockRunType, "func(") || !strings.Contains(mockRunType, "State)") || !strings.Contains(mockRunType, "Action") {
		t.Errorf("MockWorkflow.Run method signature incorrect: %s", mockRunType)
	}

	mockGetSuccessorType := reflect.TypeOf(mock.GetSuccessor).String()
	if !strings.Contains(mockGetSuccessorType, "func(") || !strings.Contains(mockGetSuccessorType, "Action)") || !strings.Contains(mockGetSuccessorType, "Workflow") {
		t.Errorf("MockWorkflow.GetSuccessor method signature incorrect: %s", mockGetSuccessorType)
	}

	mockAddSuccessorType := reflect.TypeOf(mock.AddSuccessor).String()
	if !strings.Contains(mockAddSuccessorType, "func(") || !strings.Contains(mockAddSuccessorType, "Workflow") || !strings.Contains(mockAddSuccessorType, "...") {
		t.Errorf("MockWorkflow.AddSuccessor method signature incorrect: %s", mockAddSuccessorType)
	}
}

// TestWorkflowInterface_StateManagement tests state management across workflow executions
func TestWorkflowInterface_StateManagement(t *testing.T) {
	// Create a workflow that modifies state
	baseNode := &TestBaseNode{
		prepResults: []any{"state_task"},
		execResults: map[any]any{"state_task": "state_result"},
		postAction:  ActionContinue,
	}
	node1 := createNode(baseNode, 1, 1)

	// Create second node that also modifies state
	mockNode := NewMockWorkflow("state_modifier", ActionSuccess)
	node1.AddSuccessor(mockNode, ActionContinue)

	flow := NewFlow(node1)

	// Test state preservation and modification
	initialState := State{
		"initial_value": "preserved",
		"counter":       0,
	}

	result := flow.Run(&initialState)

	// Verify final action
	if result != ActionSuccess {
		t.Errorf("Expected ActionSuccess, got %v", result)
	}

	// Verify initial state is preserved
	if initialState["initial_value"] != "preserved" {
		t.Error("Initial state value was not preserved")
	}

	// Verify state was modified by workflow execution
	if initialState["state_modifier_executed"] != true {
		t.Error("State was not modified by workflow execution")
	}

	// Verify state is shared across workflow components
	if initialState["counter"] != 0 {
		t.Error("State should be shared and consistent across workflow components")
	}
}
