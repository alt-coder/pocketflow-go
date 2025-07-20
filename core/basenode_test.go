package core

import (
	"errors"
	"reflect"
	"testing"
)

// TestBaseNode is a mock implementation of BaseNode for testing
type TestBaseNode struct {
	prepResults   []any
	execResults   map[any]any
	execErrors    map[any]error
	postAction    Action
	fallbackValue any
}

// Prep returns the configured prep results as []any
func (t *TestBaseNode) Prep(state *State) []any {
	return t.prepResults
}

// Exec processes any input and returns any output
func (t *TestBaseNode) Exec(prepResult any) (any, error) {
	if err, exists := t.execErrors[prepResult]; exists {
		return nil, err
	}
	if result, exists := t.execResults[prepResult]; exists {
		return result, nil
	}
	return prepResult, nil // Default: return input as output
}

// Post processes any parameters and returns an Action
func (t *TestBaseNode) Post(state *State, prepResults []any, execResults ...any) Action {
	return t.postAction
}

// ExecFallback returns any fallback value
func (t *TestBaseNode) ExecFallback(err error) any {
	return t.fallbackValue
}

// Test Prep method with []any return type
func TestBaseNode_Prep_InterfaceSliceReturn(t *testing.T) {
	tests := []struct {
		name        string
		prepResults []any
		state       *State
		expected    []any
	}{
		{
			name:        "Empty slice return",
			prepResults: []any{},
			state:       &State{"key": "value"},
			expected:    []any{},
		},
		{
			name:        "String values",
			prepResults: []any{"task1", "task2", "task3"},
			state:       &State{"key": "value"},
			expected:    []any{"task1", "task2", "task3"},
		},
		{
			name:        "Mixed types",
			prepResults: []any{42, "string", true, 3.14, map[string]string{"key": "value"}},
			state:       &State{"key": "value"},
			expected:    []any{42, "string", true, 3.14, map[string]string{"key": "value"}},
		},
		{
			name:        "Nil values",
			prepResults: []any{nil, "valid", nil},
			state:       &State{"key": "value"},
			expected:    []any{nil, "valid", nil},
		},
		{
			name:        "Complex structures",
			prepResults: []any{[]int{1, 2, 3}, map[string]any{"nested": "value"}},
			state:       &State{"key": "value"},
			expected:    []any{[]int{1, 2, 3}, map[string]any{"nested": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseNode := &TestBaseNode{
				prepResults: tt.prepResults,
			}

			result := baseNode.Prep(tt.state)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Prep() = %v, expected %v", result, tt.expected)
			}

			// Verify return type is []interface{}
			if reflect.TypeOf(result).String() != "[]interface {}" {
				t.Errorf("Prep() return type = %s, expected []interface {}", reflect.TypeOf(result))
			}
		})
	}
}

// Test Exec method with any input and output
func TestBaseNode_Exec_InterfaceInputOutput(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		execResults map[any]any
		execErrors  map[any]error
		expected    any
		expectError bool
	}{
		{
			name:        "String input/output",
			input:       "test_input",
			execResults: map[any]any{"test_input": "processed_output"},
			expected:    "processed_output",
			expectError: false,
		},
		{
			name:        "Integer input/output",
			input:       42,
			execResults: map[any]any{42: 84},
			expected:    84,
			expectError: false,
		},
		{
			name:        "Boolean input/output",
			input:       true,
			execResults: map[any]any{true: false},
			expected:    false,
			expectError: false,
		},
		{
			name:        "Nil input handling",
			input:       nil,
			execResults: map[any]any{nil: "nil_processed"},
			expected:    "nil_processed",
			expectError: false,
		},
		{
			name:        "Float input/output",
			input:       3.14,
			execResults: map[any]any{3.14: 6.28},
			expected:    6.28,
			expectError: false,
		},
		{
			name:        "Error case",
			input:       "error_input",
			execErrors:  map[any]error{"error_input": errors.New("processing failed")},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Default passthrough",
			input:       "unknown_input",
			execResults: map[any]any{},
			expected:    "unknown_input",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseNode := &TestBaseNode{
				execResults: tt.execResults,
				execErrors:  tt.execErrors,
			}

			result, err := baseNode.Exec(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Exec() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Exec() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("Exec() = %v, expected %v", result, tt.expected)
				}
			}
		})
	}
}

// Test Post method with any parameters
func TestBaseNode_Post_InterfaceParameters(t *testing.T) {
	tests := []struct {
		name           string
		state          State
		prepResults    []any
		execResults    []any
		postAction     Action
		expectedAction Action
	}{
		{
			name:           "Empty results",
			state:          State{"key": "value"},
			prepResults:    []any{},
			execResults:    []any{},
			postAction:     ActionSuccess,
			expectedAction: ActionSuccess,
		},
		{
			name:           "String results",
			state:          State{"status": "processing"},
			prepResults:    []any{"task1", "task2"},
			execResults:    []any{"result1", "result2"},
			postAction:     ActionContinue,
			expectedAction: ActionContinue,
		},
		{
			name:           "Mixed type results",
			state:          State{"count": 3},
			prepResults:    []any{1, "two", true},
			execResults:    []any{10, "twenty", false},
			postAction:     ActionFailure,
			expectedAction: ActionFailure,
		},
		{
			name:           "Nil values in results",
			state:          State{"allow_nil": true},
			prepResults:    []any{nil, "valid"},
			execResults:    []any{nil, "processed"},
			postAction:     ActionRetry,
			expectedAction: ActionRetry,
		},
		{
			name:           "Complex structures",
			state:          State{"complex": true},
			prepResults:    []any{map[string]int{"prep": 1}},
			execResults:    []any{map[string]int{"exec": 2}},
			postAction:     ActionDefault,
			expectedAction: ActionDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseNode := &TestBaseNode{
				postAction: tt.postAction,
			}

			result := baseNode.Post(&tt.state, tt.prepResults, tt.execResults...)

			if result != tt.expectedAction {
				t.Errorf("Post() = %v, expected %v", result, tt.expectedAction)
			}
		})
	}
}

// Test ExecFallback with any return type
func TestBaseNode_ExecFallback_InterfaceReturn(t *testing.T) {
	tests := []struct {
		name          string
		inputError    error
		fallbackValue any
		expected      any
	}{
		{
			name:          "String fallback",
			inputError:    errors.New("execution failed"),
			fallbackValue: "fallback_string",
			expected:      "fallback_string",
		},
		{
			name:          "Integer fallback",
			inputError:    errors.New("number processing failed"),
			fallbackValue: -1,
			expected:      -1,
		},
		{
			name:          "Boolean fallback",
			inputError:    errors.New("boolean operation failed"),
			fallbackValue: false,
			expected:      false,
		},
		{
			name:          "Nil fallback",
			inputError:    errors.New("nil handling failed"),
			fallbackValue: nil,
			expected:      nil,
		},
		{
			name:          "Complex struct fallback",
			inputError:    errors.New("struct processing failed"),
			fallbackValue: map[string]any{"error": true, "message": "fallback"},
			expected:      map[string]any{"error": true, "message": "fallback"},
		},
		{
			name:          "Slice fallback",
			inputError:    errors.New("slice processing failed"),
			fallbackValue: []string{"fallback", "values"},
			expected:      []string{"fallback", "values"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseNode := &TestBaseNode{
				fallbackValue: tt.fallbackValue,
			}

			result := baseNode.ExecFallback(tt.inputError)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExecFallback() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Integration test: Test complete BaseNode interface workflow with any types
func TestBaseNode_IntegrationWorkflow(t *testing.T) {
	// Setup test data with various any types
	prepData := []any{
		"string_task",
		42,
		true,
		3.14,
	}

	execResults := map[any]any{
		"string_task": "processed_string",
		42:            84,
		true:          false,
		3.14:          6.28,
	}

	baseNode := &TestBaseNode{
		prepResults:   prepData,
		execResults:   execResults,
		postAction:    ActionSuccess,
		fallbackValue: "fallback_result",
	}

	state := State{"test": "integration"}

	// Test Prep phase
	prepResult := baseNode.Prep(&state)
	if !reflect.DeepEqual(prepResult, prepData) {
		t.Errorf("Integration Prep() = %v, expected %v", prepResult, prepData)
	}

	// Test Exec phase for each prep result
	var execResultsSlice []any
	for _, prepItem := range prepResult {
		execResult, err := baseNode.Exec(prepItem)
		if err != nil {
			t.Errorf("Integration Exec() unexpected error: %v", err)
		}
		execResultsSlice = append(execResultsSlice, execResult)
	}

	// Verify exec results match expected
	expectedExecResults := []any{
		"processed_string",
		84,
		false,
		6.28,
	}

	if !reflect.DeepEqual(execResultsSlice, expectedExecResults) {
		t.Errorf("Integration Exec results = %v, expected %v", execResultsSlice, expectedExecResults)
	}

	// Test Post phase
	postResult := baseNode.Post(&state, prepResult, execResultsSlice...)
	if postResult != ActionSuccess {
		t.Errorf("Integration Post() = %v, expected %v", postResult, ActionSuccess)
	}
}

// Test BaseNode interface compliance
func TestBaseNode_InterfaceCompliance(t *testing.T) {
	var _ BaseNode[State, any, any] = (*TestBaseNode)(nil)

	// Verify that TestBaseNode implements all BaseNode methods
	baseNode := &TestBaseNode{}

	// Test method signatures match interface requirements
	prepResult := baseNode.Prep(&State{})
	if reflect.TypeOf(prepResult).String() != "[]interface {}" {
		t.Errorf("Prep method signature incorrect, got %s", reflect.TypeOf(prepResult))
	}

	execResult, _ := baseNode.Exec("test")
	if reflect.TypeOf(execResult) == nil {
		// execResult can be any type, so we just verify it's not restricted
		t.Log("Exec method correctly returns any")
	}

	postResult := baseNode.Post(&State{}, []any{}, "test")
	if reflect.TypeOf(postResult).String() != "core.Action" {
		t.Errorf("Post method signature incorrect, got %s", reflect.TypeOf(postResult))
	}

	fallbackResult := baseNode.ExecFallback(errors.New("test"))
	if reflect.TypeOf(fallbackResult) == nil {
		// fallbackResult can be any type, so we just verify it's not restricted
		t.Log("ExecFallback method correctly returns any")
	}
}
