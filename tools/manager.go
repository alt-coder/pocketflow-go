package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/alt-coder/pocketflow-go/llm"
)

// ToolManager provides a unified interface for managing both local and MCP tools
type ToolManager struct {
	localTools map[string]LocalTool
	mcpManager *MCPManager
	mu         sync.RWMutex
}

// LocalTool represents a locally defined tool function
type LocalTool struct {
	Name        string
	Description string
	Parameters  map[string]Parameter
	Handler     interface{} // func(InputStruct) OutputStruct
	inputType   reflect.Type
	outputType  reflect.Type
}

// Parameter represents a tool parameter definition
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolHandler is the function signature for local tool handlers
// Deprecated: Use func(InputStruct) OutputStruct pattern instead
type ToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)

// ToolSchema represents the schema of an available tool
type ToolSchema struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]Parameter `json:"parameters"`
	Source      string               `json:"source"` // "local" or "mcp"
}

// NewToolManager creates a new tool manager
func NewToolManager() *ToolManager {
	return &ToolManager{
		localTools: make(map[string]LocalTool),
	}
}

// AddLocalTool adds a local tool to the manager using reflection
func (tm *ToolManager) AddLocalTool(name, description string, handler interface{}) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}

	// Validate handler signature using reflection
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be a function")
	}

	if handlerType.NumIn() != 1 {
		return fmt.Errorf("handler must accept exactly one input parameter")
	}

	if handlerType.NumOut() != 1 {
		return fmt.Errorf("handler must return exactly one output parameter")
	}

	inputType := handlerType.In(0)
	outputType := handlerType.Out(0)

	// Generate parameters schema from input struct
	parameters, err := tm.generateParametersFromStruct(inputType)
	if err != nil {
		return fmt.Errorf("failed to generate parameters schema: %v", err)
	}

	tool := LocalTool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Handler:     handler,
		inputType:   inputType,
		outputType:  outputType,
	}

	tm.localTools[name] = tool
	return nil
}

// AddLocalToolLegacy adds a local tool using the legacy handler signature
// Deprecated: Use AddLocalTool with func(InputStruct) OutputStruct pattern
func (tm *ToolManager) AddLocalToolLegacy(tool LocalTool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if tool.Handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}

	tm.localTools[tool.Name] = tool
	return nil
}

// SetMCPManager sets the MCP manager for handling MCP tools
func (tm *ToolManager) SetMCPManager(mcpManager *MCPManager) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.mcpManager = mcpManager
}

// GetAvailableTools returns all available tools (local + MCP)
func (tm *ToolManager) GetAvailableTools() []ToolSchema {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tools []ToolSchema

	// Add local tools
	for _, tool := range tm.localTools {
		tools = append(tools, ToolSchema{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
			Source:      "local",
		})
	}

	// Add MCP tools if manager is available
	if tm.mcpManager != nil {
		mcpTools := tm.mcpManager.GetAvailableTools()
		for _, tool := range mcpTools {
			// Convert MCP tool schema to our format
			params := make(map[string]Parameter)
			for name, prop := range tool.Parameters {
				params[name] = Parameter{
					Type:        string(prop.Type),
					Description: prop.Description,
					Enum:       prop.Enum,
				}
			}

			tools = append(tools, ToolSchema{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
				Source:      "mcp",
			})
		}
	}

	return tools
}

// ExecuteTool executes a tool call, routing to local or MCP handler
func (tm *ToolManager) ExecuteTool(ctx context.Context, toolCall llm.ToolCalls) (llm.ToolResults, error) {
	tm.mu.RLock()
	localTool, isLocal := tm.localTools[toolCall.ToolName]
	tm.mu.RUnlock()

	// Try local tool first
	if isLocal {
		return tm.executeLocalTool(ctx, localTool, toolCall)
	}

	// Try MCP tool if manager is available
	if tm.mcpManager != nil {
		return tm.mcpManager.ExecuteTool(ctx, toolCall)
	}

	// Tool not found
	return llm.ToolResults{
		Id:      toolCall.Id,
		Content: "",
		IsError: true,
		Error:   fmt.Sprintf("Tool '%s' not found", toolCall.ToolName),
	}, nil
}

// executeLocalTool executes a local tool
func (tm *ToolManager) executeLocalTool(ctx context.Context, tool LocalTool, toolCall llm.ToolCalls) (llm.ToolResults, error) {
	// Check if this is a legacy handler
	if legacyHandler, ok := tool.Handler.(ToolHandler); ok {
		return tm.executeLegacyTool(ctx, tool, toolCall, legacyHandler)
	}

	// Handle new struct-based handler
	return tm.executeStructTool(ctx, tool, toolCall)
}

// executeLegacyTool executes a legacy tool handler
func (tm *ToolManager) executeLegacyTool(ctx context.Context, tool LocalTool, toolCall llm.ToolCalls, handler ToolHandler) (llm.ToolResults, error) {
	// Validate required parameters
	if err := tm.validateParameters(tool, toolCall.ToolArgs); err != nil {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("Parameter validation failed: %v", err),
		}, nil
	}

	// Execute the tool handler
	result, err := handler(ctx, toolCall.ToolArgs)
	if err != nil {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("Tool execution failed: %v", err),
		}, nil
	}

	return llm.ToolResults{
		Id:      toolCall.Id,
		Content: result,
		IsError: false,
	}, nil
}

// executeStructTool executes a struct-based tool handler
func (tm *ToolManager) executeStructTool(ctx context.Context, tool LocalTool, toolCall llm.ToolCalls) (llm.ToolResults, error) {
	// Create input struct instance
	inputValue := reflect.New(tool.inputType).Elem()

	// Populate struct fields from tool arguments
	if err := tm.populateStructFromArgs(inputValue, toolCall.ToolArgs); err != nil {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("Failed to populate input struct: %v", err),
		}, nil
	}

	// Call the handler function
	handlerValue := reflect.ValueOf(tool.Handler)
	results := handlerValue.Call([]reflect.Value{inputValue})

	// Get the result
	resultValue := results[0]

	// Convert result to JSON string
	resultBytes, err := json.Marshal(resultValue.Interface())
	if err != nil {
		return llm.ToolResults{
			Id:      toolCall.Id,
			Content: "",
			IsError: true,
			Error:   fmt.Sprintf("Failed to marshal result: %v", err),
		}, nil
	}

	return llm.ToolResults{
		Id:      toolCall.Id,
		Content: string(resultBytes),
		IsError: false,
	}, nil
}

// populateStructFromArgs populates a struct from tool arguments
func (tm *ToolManager) populateStructFromArgs(structValue reflect.Value, args map[string]interface{}) error {
	structType := structValue.Type()

	// Track which required fields we've seen
	requiredFields := make(map[string]bool)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name (use json tag if available)
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Skip fields marked as json:"-"
		if fieldName == "-" {
			continue
		}

		// Check if field is required
		isRequired := field.Type.Kind() != reflect.Ptr && field.Tag.Get("default") == ""
		if isRequired {
			requiredFields[fieldName] = false
		}

		// Get value from arguments
		argValue, exists := args[fieldName]
		if !exists {
			// Check if field has default value
			if defaultTag := field.Tag.Get("default"); defaultTag != "" {
				if err := tm.setFieldFromString(fieldValue, defaultTag); err != nil {
					return fmt.Errorf("failed to set default value for %s: %v", fieldName, err)
				}
			} else if isRequired {
				return fmt.Errorf("required parameter '%s' is missing", fieldName)
			}
			continue
		}

		// Mark required field as provided
		if isRequired {
			requiredFields[fieldName] = true
		}

		// Validate enum values if specified
		if enumTag := field.Tag.Get("enum"); enumTag != "" {
			enumValues := strings.Split(enumTag, ",")
			argStr := fmt.Sprintf("%v", argValue)
			validEnum := false
			for _, enumValue := range enumValues {
				if strings.TrimSpace(enumValue) == argStr {
					validEnum = true
					break
				}
			}
			if !validEnum {
				return fmt.Errorf("parameter '%s' value '%v' is not in allowed enum values: %v", fieldName, argValue, enumValues)
			}
		}

		// Set field value
		if err := tm.setFieldValue(fieldValue, argValue); err != nil {
			return fmt.Errorf("failed to set field %s: %v", fieldName, err)
		}
	}

	// Check that all required fields were provided
	for fieldName, provided := range requiredFields {
		if !provided {
			return fmt.Errorf("required parameter '%s' is missing", fieldName)
		}
	}

	return nil
}

// setFieldValue sets a struct field value from an interface{}
func (tm *ToolManager) setFieldValue(fieldValue reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	valueReflect := reflect.ValueOf(value)
	fieldType := fieldValue.Type()

	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldType.Elem()))
		}
		return tm.setFieldValue(fieldValue.Elem(), value)
	}

	// Direct assignment if types match
	if valueReflect.Type().AssignableTo(fieldType) {
		fieldValue.Set(valueReflect)
		return nil
	}

	// Type conversion
	if valueReflect.Type().ConvertibleTo(fieldType) {
		fieldValue.Set(valueReflect.Convert(fieldType))
		return nil
	}

	return fmt.Errorf("cannot assign %T to %s", value, fieldType)
}

// setFieldFromString sets a field value from a string (for default values)
func (tm *ToolManager) setFieldFromString(fieldValue reflect.Value, value string) error {
	fieldType := fieldValue.Type()

	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldType.Elem()))
		}
		return tm.setFieldFromString(fieldValue.Elem(), value)
	}

	switch fieldType.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
	case reflect.Bool:
		if value == "true" {
			fieldValue.SetBool(true)
		} else if value == "false" {
			fieldValue.SetBool(false)
		} else {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
	default:
		// For other types, try JSON unmarshaling
		var result interface{}
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return tm.setFieldValue(fieldValue, result)
		}
		return fmt.Errorf("unsupported default value type for %s", fieldType.Kind())
	}

	return nil
}

// validateParameters validates tool parameters against the schema
func (tm *ToolManager) validateParameters(tool LocalTool, args map[string]interface{}) error {
	// Skip validation for struct-based tools (validation happens during struct population)
	if tool.inputType != nil {
		return nil
	}

	// Legacy validation for old-style tools
	// Check required parameters
	for paramName, param := range tool.Parameters {
		if param.Required {
			if _, exists := args[paramName]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", paramName)
			}
		}
	}

	// Validate parameter types (basic validation)
	for paramName, value := range args {
		param, exists := tool.Parameters[paramName]
		if !exists {
			return fmt.Errorf("unknown parameter '%s'", paramName)
		}

		if err := tm.validateParameterType(param, value); err != nil {
			return fmt.Errorf("parameter '%s': %v", paramName, err)
		}
	}

	return nil
}

// validateParameterType validates a single parameter type
func (tm *ToolManager) validateParameterType(param Parameter, value interface{}) error {
	switch param.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	// Validate enum values if specified
	if len(param.Enum) > 0 {
		valueStr := fmt.Sprintf("%v", value)
		for _, enumValue := range param.Enum {
			if enumValue == valueStr {
				return nil
			}
		}
		return fmt.Errorf("value '%v' is not in allowed enum values: %v", value, param.Enum)
	}

	return nil
}

// RemoveLocalTool removes a local tool from the manager
func (tm *ToolManager) RemoveLocalTool(toolName string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.localTools[toolName]; !exists {
		return fmt.Errorf("tool '%s' not found", toolName)
	}

	delete(tm.localTools, toolName)
	return nil
}

// HasTool checks if a tool exists (local or MCP)
func (tm *ToolManager) HasTool(toolName string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Check local tools
	if _, exists := tm.localTools[toolName]; exists {
		return true
	}

	// Check MCP tools
	if tm.mcpManager != nil {
		return tm.mcpManager.HasTool(toolName)
	}

	return false
}

// generateParametersFromStruct generates parameter schema from a struct type using reflection
func (tm *ToolManager) generateParametersFromStruct(structType reflect.Type) (map[string]Parameter, error) {
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input type must be a struct, got %s", structType.Kind())
	}

	parameters := make(map[string]Parameter)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name (use json tag if available, otherwise field name)
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}else if yamltag := field.Tag.Get("yaml"); yamltag != "" {
			if parts := strings.Split(yamltag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Skip fields marked as json:"-"
		if fieldName == "-" {
			continue
		}

		// Get description from tag
		description := field.Tag.Get("description")
		if description == "" {
			description = fmt.Sprintf("Parameter %s", fieldName)
		}

		// Determine if field is required (not a pointer and no default tag)
		required := field.Type.Kind() != reflect.Ptr && field.Tag.Get("default") == ""

		// Get enum values if specified
		var enumValues []string
		if enumTag := field.Tag.Get("enum"); enumTag != "" {
			enumValues = strings.Split(enumTag, ",")
		}

		// Get default value if specified
		var defaultValue interface{}
		if defaultTag := field.Tag.Get("default"); defaultTag != "" {
			defaultValue = defaultTag
		}

		// Convert Go type to JSON schema type
		jsonType, err := tm.goTypeToJSONType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("unsupported field type for %s: %v", fieldName, err)
		}

		parameters[fieldName] = Parameter{
			Type:        jsonType,
			Description: description,
			Required:    required,
			Enum:        enumValues,
			Default:     defaultValue,
		}
	}

	return parameters, nil
}

// goTypeToJSONType converts Go types to JSON schema types
func (tm *ToolManager) goTypeToJSONType(goType reflect.Type) (string, error) {
	// Handle pointers
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	switch goType.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Slice, reflect.Array:
		return "array", nil
	case reflect.Map, reflect.Struct:
		return "object", nil
	default:
		return "", fmt.Errorf("unsupported type: %s", goType.Kind())
	}
}

// Close closes the tool manager and cleans up resources
func (tm *ToolManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Clear local tools
	tm.localTools = make(map[string]LocalTool)

	// Close MCP manager if available
	if tm.mcpManager != nil {
		return tm.mcpManager.Close()
	}

	return nil
}
