package core

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type State  map[string]interface{}

// TestDataProcessor simulates a data processing node
type TestDataProcessor struct {
	name          string
	processDelay  time.Duration
	shouldError   bool
	errorOnInput  interface{}
	multiplier    int
	successAction Action
	failureAction Action
}

func (t *TestDataProcessor) Prep(state State) []interface{} {
	if data, exists := state["input_data"]; exists {
		if slice, ok := data.([]interface{}); ok {
			return slice
		}
		return []interface{}{data}
	}
	return []interface{}{"default_data"}
}

func (t *TestDataProcessor) Exec(prepResult interface{}) (interface{}, error) {
	if t.processDelay > 0 {
		time.Sleep(t.processDelay)
	}

	if t.shouldError || prepResult == t.errorOnInput {
		return nil, fmt.Errorf("processing error in %s", t.name)
	}

	// Process different types of data
	switch v := prepResult.(type) {
	case int:
		return v * t.multiplier, nil
	case string:
		return fmt.Sprintf("%s_processed_by_%s", v, t.name), nil
	case float64:
		return v * float64(t.multiplier), nil
	default:
		return fmt.Sprintf("processed_%v_by_%s", v, t.name), nil
	}
}

func (t *TestDataProcessor) Post(state State, prepResults []interface{}, execResults ...interface{}) Action {
	// Update state with results
	state[fmt.Sprintf("%s_results", t.name)] = execResults
	state[fmt.Sprintf("%s_count", t.name)] = len(execResults)

	// Check for errors in results
	for _, result := range execResults {
		if result == nil {
			return t.failureAction
		}
	}

	return t.successAction
}

func (t *TestDataProcessor) ExecFallback(err error) interface{} {
	return fmt.Sprintf("fallback_result_from_%s", t.name)
}

// TestValidator simulates a validation node
type TestValidator struct {
	name           string
	validationRule func(interface{}) bool
	successAction  Action
	failureAction  Action
}

func (t *TestValidator) Prep(state State) []interface{} {
	// Get results from previous processors
	var allResults []interface{}
	for key, value := range state {
		if key != "input_data" && key != "validation_results" {
			if results, ok := value.([]interface{}); ok {
				allResults = append(allResults, results...)
			}
		}
	}
	if len(allResults) == 0 {
		return []interface{}{"no_data_to_validate"}
	}
	return allResults
}

func (t *TestValidator) Exec(prepResult interface{}) (interface{}, error) {
	if t.validationRule != nil && !t.validationRule(prepResult) {
		return nil, errors.New("validation failed")
	}
	return map[string]interface{}{
		"validated": true,
		"data":      prepResult,
		"validator": t.name,
	}, nil
}

func (t *TestValidator) Post(state State, prepResults []interface{}, execResults ...interface{}) Action {
	state["validation_results"] = execResults

	// Check if all validations passed
	for _, result := range execResults {
		if result == nil {
			return t.failureAction
		}
		if validationResult, ok := result.(map[string]interface{}); ok {
			if validated, exists := validationResult["validated"]; !exists || validated != true {
				return t.failureAction
			}
		}
	}

	return t.successAction
}

func (t *TestValidator) ExecFallback(err error) interface{} {
	return map[string]interface{}{
		"validated": false,
		"error":     err.Error(),
		"validator": t.name,
	}
}

// TestAggregator simulates an aggregation node
type TestAggregator struct {
	name          string
	successAction Action
}

func (t *TestAggregator) Prep(state State) []interface{} {
	return []interface{}{"aggregate_all_data"}
}

func (t *TestAggregator) Exec(prepResult interface{}) (interface{}, error) {
	// This node doesn't process individual items, it aggregates state
	return "aggregation_complete", nil
}

func (t *TestAggregator) Post(state State, prepResults []interface{}, execResults ...interface{}) Action {
	// Aggregate all processing results
	aggregatedData := make(map[string]interface{})
	totalCount := 0

	for key, value := range state {
		if key != "input_data" {
			aggregatedData[key] = value
			if key == "processor1_count" || key == "processor2_count" {
				if count, ok := value.(int); ok {
					totalCount += count
				}
			}
		}
	}

	aggregatedData["total_processed_count"] = totalCount
	state["final_aggregation"] = aggregatedData

	return t.successAction
}

func (t *TestAggregator) ExecFallback(err error) interface{} {
	return map[string]interface{}{
		"aggregation_failed": true,
		"error":              err.Error(),
	}
}

// Test complex workflows with multiple nodes and flows
func TestComplexWorkflowExecution(t *testing.T) {
	// Create test nodes
	processor1 := createNode(&TestDataProcessor{
		name:          "processor1",
		multiplier:    2,
		successAction: ActionContinue,
		failureAction: ActionFailure,
	}, 2, 3)

	processor2 := createNode(&TestDataProcessor{
		name:          "processor2",
		multiplier:    3,
		successAction: ActionContinue,
		failureAction: ActionFailure,
	}, 2, 2)

	validator := createNode(&TestValidator{
		name: "validator",
		validationRule: func(data interface{}) bool {
			// Validate that processed data is not nil and has expected format
			return data != nil
		},
		successAction: ActionContinue,
		failureAction: ActionRetry,
	}, 1, 1)

	aggregator := createNode(&TestAggregator{
		name:          "aggregator",
		successAction: ActionSuccess,
	}, 1, 1)

	// Connect nodes in sequence
	processor1.AddSuccessor( processor2, ActionContinue)
	processor2.AddSuccessor(validator, ActionContinue,)
	validator.AddSuccessor(aggregator, ActionContinue,)

	// Create flow with complex data
	flow := NewFlow(processor1)

	initialState := State{
		"input_data": []interface{}{1, 2, 3, "test", 4.5},
	}

	// Execute the complex workflow
	finalAction := flow.Run(initialState)

	// Verify final action
	if finalAction != ActionSuccess {
		t.Errorf("Complex workflow expected ActionSuccess, got %v", finalAction)
	}

	// Verify state propagation - now that Flow.Run propagates state changes back
	if _, exists := initialState["processor1_results"]; !exists {
		t.Error("State missing processor1_results")
	}
	if _, exists := initialState["processor2_results"]; !exists {
		t.Error("State missing processor2_results")
	}
	if _, exists := initialState["validation_results"]; !exists {
		t.Error("State missing validation_results")
	}
	if _, exists := initialState["final_aggregation"]; !exists {
		t.Error("State missing final_aggregation")
	}

	// Verify data types are preserved through the workflow
	if aggregation, ok := initialState["final_aggregation"].(map[string]interface{}); ok {
		if totalCount, exists := aggregation["total_processed_count"]; exists {
			if count, ok := totalCount.(int); !ok || count != 10 { // 5 items processed by 2 processors
				t.Errorf("Expected total_processed_count to be 10, got %v", totalCount)
			}
		} else {
			t.Error("Missing total_processed_count in aggregation")
		}
	} else {
		t.Error("final_aggregation is not the expected type")
	}
}

// Test state propagation through workflow chains
func TestStatePropagationThroughWorkflowChains(t *testing.T) {
	// Create nodes that modify state
	stateModifier1 := createNode(&TestDataProcessor{
		name:          "modifier1",
		multiplier:    1,
		successAction: ActionContinue,
		failureAction: ActionFailure,
	}, 1, 1)

	stateModifier2 := createNode(&TestDataProcessor{
		name:          "modifier2",
		multiplier:    1,
		successAction: ActionSuccess,
		failureAction: ActionFailure,
	}, 1, 1)

	// Connect nodes directly (not through subflow for this test)
	stateModifier1.AddSuccessor( stateModifier2, ActionContinue)

	// Create main flow
	mainFlow := NewFlow(stateModifier1)

	initialState := State{
		"input_data":    []interface{}{"state_test"},
		"initial_value": "original",
	}

	// Execute workflow
	finalAction := mainFlow.Run(initialState)

	// Verify final action
	if finalAction != ActionSuccess {
		t.Errorf("Expected ActionSuccess, got %v", finalAction)
	}

	// Verify state propagation - now that Flow.Run properly propagates state changes
	if _, exists := initialState["modifier1_results"]; !exists {
		t.Error("State missing modifier1_results - state not propagated from main flow")
	}
	if _, exists := initialState["modifier2_results"]; !exists {
		t.Error("State missing modifier2_results - state not propagated through workflow chain")
	}

	// Verify original state is preserved alongside new state
	if initialState["initial_value"] != "original" {
		t.Error("Original state value was modified unexpectedly")
	}

	// Verify both processors saw the same input data
	modifier1Results := initialState["modifier1_results"].([]interface{})
	modifier2Results := initialState["modifier2_results"].([]interface{})

	if len(modifier1Results) != 1 || len(modifier2Results) != 1 {
		t.Error("State propagation failed - processors didn't receive expected input")
	}
}

// Test error handling and fallback behavior with interface{} types
func TestErrorHandlingAndFallbackBehavior(t *testing.T) {
	// Create node that will error on specific input but continue processing
	errorProneProcessor := createNode(&TestDataProcessor{
		name:          "error_processor",
		multiplier:    2,
		shouldError:   false,
		errorOnInput:  "error_trigger",
		successAction: ActionContinue, // Continue even with some errors
		failureAction: ActionContinue, // Continue to validator even on failure
	}, 2, 2) // 2 retries, 2 workers

	fallbackValidator := createNode(&TestValidator{
		name: "fallback_validator",
		validationRule: func(data interface{}) bool {
			// Accept fallback results and processed data
			if str, ok := data.(string); ok {
				return str == "fallback_result_from_error_processor" ||
					str == "normal_data_processed_by_error_processor" ||
					str == "more_data_processed_by_error_processor"
			}
			return true
		},
		successAction: ActionSuccess,
		failureAction: ActionFailure,
	}, 1, 1)

	// Connect nodes - processor continues to validator regardless of individual item failures
	errorProneProcessor.AddSuccessor( fallbackValidator, ActionContinue,)

	// Create flow
	flow := NewFlow(errorProneProcessor)

	// Test with mixed data including error trigger
	initialState := State{
		"input_data": []interface{}{"normal_data", "error_trigger", "more_data"},
	}

	finalAction := flow.Run(initialState)

	// Should succeed because fallback handles the error
	if finalAction != ActionSuccess {
		t.Errorf("Expected ActionSuccess with fallback handling, got %v", finalAction)
	}

	// Verify fallback results are present in the state
	if results, exists := initialState["error_processor_results"]; exists {
		if resultSlice, ok := results.([]interface{}); ok {
			foundFallback := false
			for _, result := range resultSlice {
				if str, ok := result.(string); ok && str == "fallback_result_from_error_processor" {
					foundFallback = true
					break
				}
			}
			if !foundFallback {
				t.Error("Fallback result not found in processor results")
			}
		}
	} else {
		t.Error("Error processor results not found in state")
	}

	// Verify validation accepted fallback results
	if validationResults, exists := initialState["validation_results"]; exists {
		if resultSlice, ok := validationResults.([]interface{}); ok {
			for _, result := range resultSlice {
				if validationMap, ok := result.(map[string]interface{}); ok {
					if validated, exists := validationMap["validated"]; !exists || validated != true {
						t.Error("Validation should have accepted fallback results")
					}
				}
			}
		}
	}
}

// Test concurrent execution with interface{} data types
func TestConcurrentExecutionWithInterfaceTypes(t *testing.T) {
	// Create processor with delay to test concurrency
	concurrentProcessor := createNode(&TestDataProcessor{
		name:          "concurrent_processor",
		processDelay:  50 * time.Millisecond, // Small delay to test concurrency
		multiplier:    1,
		successAction: ActionSuccess,
		failureAction: ActionFailure,
	}, 1, 5) // 5 concurrent workers

	// Create large dataset with mixed types
	largeDataset := make([]interface{}, 20)
	for i := 0; i < 20; i++ {
		switch i % 4 {
		case 0:
			largeDataset[i] = i
		case 1:
			largeDataset[i] = fmt.Sprintf("string_%d", i)
		case 2:
			largeDataset[i] = float64(i) * 1.5
		case 3:
			largeDataset[i] = i%2 == 0
		}
	}

	flow := NewFlow(concurrentProcessor)
	initialState := State{
		"input_data": largeDataset,
	}

	// Measure execution time
	startTime := time.Now()
	finalAction := flow.Run(initialState)
	executionTime := time.Since(startTime)

	// Verify successful execution
	if finalAction != ActionSuccess {
		t.Errorf("Expected ActionSuccess, got %v", finalAction)
	}

	// Verify all data was processed and state was updated
	if results, exists := initialState["concurrent_processor_results"]; exists {
		if resultSlice, ok := results.([]interface{}); ok {
			if len(resultSlice) != 20 {
				t.Errorf("Expected 20 results, got %d", len(resultSlice))
			}

			// Verify different data types were processed correctly
			typeCount := make(map[string]int)
			for _, result := range resultSlice {
				switch result.(type) {
				case int:
					typeCount["int"]++
				case string:
					typeCount["string"]++
				case float64:
					typeCount["float64"]++
				default:
					typeCount["other"]++
				}
			}

			// Should have processed different types
			if len(typeCount) < 3 {
				t.Errorf("Expected multiple data types in results, got %v", typeCount)
			}
		}
	} else {
		t.Error("Concurrent processor results not found")
	}

	// Verify concurrency improved performance (should be faster than sequential)
	// With 5 workers and 50ms delay per item, 20 items should take roughly 200ms instead of 1000ms
	maxExpectedTime := 500 * time.Millisecond // Allow some buffer
	if executionTime > maxExpectedTime {
		t.Errorf("Execution took too long (%v), concurrency may not be working", executionTime)
	}
}

// Test nested flows with complex state management
func TestNestedFlowsWithComplexStateManagement(t *testing.T) {
	// Create inner flow components
	innerProcessor1 := createNode(&TestDataProcessor{
		name:          "inner_proc1",
		multiplier:    2,
		successAction: ActionContinue,
		failureAction: ActionFailure,
	}, 1, 2)

	innerProcessor2 := createNode(&TestDataProcessor{
		name:          "inner_proc2",
		multiplier:    3,
		successAction: ActionSuccess,
		failureAction: ActionFailure,
	}, 1, 2)

	// Connect inner processors
	innerProcessor1.AddSuccessor( innerProcessor2, ActionContinue,)

	// Create inner flow
	innerFlow := NewFlow(innerProcessor1)

	// Create outer flow components
	outerProcessor := createNode(&TestDataProcessor{
		name:          "outer_proc",
		multiplier:    1,
		successAction: ActionContinue,
		failureAction: ActionFailure,
	}, 1, 1)

	// Connect outer processor to inner flow
	outerProcessor.AddSuccessor( innerFlow, ActionContinue)

	// Create main flow that includes the outer processor and inner flow
	mainFlow := NewFlow(outerProcessor)

	// Test with complex nested state
	initialState := State{
		"input_data":   []interface{}{1, 2, 3},
		"outer_config": "main_flow",
		"nested_level": 0,
	}

	finalAction := mainFlow.Run(initialState)

	// Verify successful execution
	if finalAction != ActionSuccess {
		t.Errorf("Expected ActionSuccess, got %v", finalAction)
	}

	// Verify all processors executed and modified state
	expectedKeys := []string{
		"outer_proc_results",
		"inner_proc1_results",
		"inner_proc2_results",
	}

	for _, key := range expectedKeys {
		if _, exists := initialState[key]; !exists {
			t.Errorf("Missing expected state key: %s", key)
		}
	}

	// Verify state consistency across nested flows
	if config, exists := initialState["outer_config"]; !exists || config != "main_flow" {
		t.Error("Outer config was lost or modified during nested execution")
	}

	// Verify nested flow processed the data correctly
	outerResults := initialState["outer_proc_results"].([]interface{})
	innerResults1 := initialState["inner_proc1_results"].([]interface{})
	innerResults2 := initialState["inner_proc2_results"].([]interface{})

	if len(outerResults) != 3 || len(innerResults1) != 3 || len(innerResults2) != 3 {
		t.Error("Nested flow state propagation failed - processors didn't receive expected input")
	}
}

// Test workflow execution with action-based routing
func TestActionBasedRouting(t *testing.T) {
	// Create a simple processor that always succeeds and routes to continue
	routingProcessor := createNode(&TestDataProcessor{
		name:          "routing_processor",
		multiplier:    1,
		successAction: ActionContinue,
		failureAction: ActionFailure,
	}, 1, 1)

	// Create success path processor
	successProcessor := createNode(&TestDataProcessor{
		name:          "success_processor",
		multiplier:    10,
		successAction: ActionSuccess,
		failureAction: ActionFailure,
	}, 1, 1)

	// Connect routing processor to success processor
	routingProcessor.AddSuccessor( successProcessor, ActionContinue)

	// Test the routing
	flow := NewFlow(routingProcessor)
	state := State{
		"input_data": []interface{}{10},
	}

	finalAction := flow.Run(state)
	if finalAction != ActionSuccess {
		t.Errorf("Expected ActionSuccess, got %v", finalAction)
	}

	// Verify both processors were executed in sequence
	if _, exists := state["routing_processor_results"]; !exists {
		t.Error("Routing processor was not executed")
	}
	if _, exists := state["success_processor_results"]; !exists {
		t.Error("Success processor was not executed")
	}

	// Verify the data flowed through both processors
	routingResults := state["routing_processor_results"].([]interface{})
	successResults := state["success_processor_results"].([]interface{})

	if len(routingResults) != 1 || len(successResults) != 1 {
		t.Error("Action-based routing failed - processors didn't receive expected input")
	}
}

// Test concurrent workflow execution safety
func TestConcurrentWorkflowExecutionSafety(t *testing.T) {
	// Create shared processor
	sharedProcessor := createNode(&TestDataProcessor{
		name:          "shared_processor",
		processDelay:  10 * time.Millisecond,
		multiplier:    1,
		successAction: ActionSuccess,
		failureAction: ActionFailure,
	}, 1, 3)

	// Create multiple flows using the same processor
	numFlows := 10
	var wg sync.WaitGroup
	results := make([]Action, numFlows)
	states := make([]State, numFlows)

	// Execute multiple flows concurrently
	for i := 0; i < numFlows; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			flow := NewFlow(sharedProcessor)
			state := State{
				"input_data": []interface{}{fmt.Sprintf("data_%d", index)},
				"flow_id":    index,
			}

			action := flow.Run(state)
			results[index] = action
			states[index] = state
		}(i)
	}

	wg.Wait()

	// Verify all flows completed successfully
	for i, action := range results {
		if action != ActionSuccess {
			t.Errorf("Flow %d failed with action %v", i, action)
		}
	}

	// Verify each flow maintained its own state
	for i, state := range states {
		if flowID, exists := state["flow_id"]; !exists || flowID != i {
			t.Errorf("Flow %d lost its state identity", i)
		}

		if results, exists := state["shared_processor_results"]; exists {
			if resultSlice, ok := results.([]interface{}); ok && len(resultSlice) > 0 {
				expectedData := fmt.Sprintf("data_%d_processed_by_shared_processor", i)
				if resultSlice[0] != expectedData {
					t.Errorf("Flow %d got wrong result: %v", i, resultSlice[0])
				}
			}
		} else {
			t.Errorf("Flow %d missing processor results", i)
		}
	}
}
