# How PocketFlowGo Works - Complete Guide

## Overview

PocketFlowGo is a minimalist workflow framework designed around a simple but powerful abstraction: **Graph-based execution with three-phase nodes**. It's built on the principle that complex LLM applications can be decomposed into simple, composable units that follow a consistent execution pattern.

## Core Philosophy

**"Focus on the essential graph abstraction, not framework bloat"**

Unlike heavyweight frameworks that add layers of vendor-specific wrappers and application-specific abstractions, PocketFlowGo provides just the essential building blocks:

1. **Nodes** - Units of work that follow a consistent three-phase pattern
2. **Flows** - Orchestrators that chain nodes together
3. **Actions** - Control flow decisions that determine execution paths
4. **State** - Shared data that flows through the workflow

## Architecture Deep Dive

### 1. Three-Phase Execution Model

Every node in PocketFlowGo follows the same execution pattern:

```
Prep → Exec → Post
  ↓      ↓      ↓
Data   Work   Decision
```

**Why this pattern?**
- **Separation of Concerns**: Data preparation, execution, and decision-making are clearly separated
- **Testability**: Each phase can be tested independently
- **Retry Logic**: Only the Exec phase is retried, not data preparation or decision-making
- **Concurrency**: Multiple Exec operations can run in parallel

#### Detailed Phase Responsibilities

##### Prep Phase - Data Preparation & Work Item Generation
**Primary Purpose**: Transform the current state into discrete work items for execution

**Detailed Responsibilities**:
1. **State Analysis**: Examine the current workflow state to understand what work needs to be done
2. **Work Item Generation**: Create a list of discrete tasks/items that can be processed independently
3. **Context Preparation**: Extract and format relevant context for each work item in the state.


**Key Characteristics**:
- **Deterministic**: Given the same state, should always produce the same work items
- **Fast**: Should be lightweight since it's not retried
- **Side-effect Free**: Should not modify external systems or state
- **Parallelizable**: Work items should be independent of each other

**Example from basic-chat**:
```go
// ChatNode: Prepare conversation context for LLM call
func (c *ChatNode) Prep(state *ChatState) []PrepResult {
    // Display welcome message on first run
    if c.firstRun {
        fmt.Println(c.config.WelcomeMsg)
        c.firstRun = false
    }

    // Prompt user for input
    fmt.Print("You: ")
    reader := bufio.NewReader(os.Stdin)
    userInput, err := reader.ReadString('\n')
    
    if err != nil {
        fmt.Printf("Error reading input: %v\n", err)
        return []PrepResult{}
    }

    input := strings.TrimSpace(userInput)

    // Handle 'exit' command (case-insensitive)
    if strings.ToLower(input) == "exit" {
        state.Active = false
        return []PrepResult{} // Return empty to signal termination
    }

    // Handle empty input gracefully
    if input == "" {
        fmt.Println("Please enter a message or 'exit' to quit.")
        return c.Prep(state) // Recursively ask for input again
    }

    // Add user message to conversation history
    state.Messages = append(state.Messages, llm.Message{
        Role:    llm.RoleUser,
        Content: input,
    })

    // Return PrepResult with conversation context
    return []PrepResult{
        {
            Messages: state.Messages,
        },
    }
}
```

**Key Prep Phase Patterns**:
1. **User Interaction**: Captures user input and handles special commands
2. **State Validation**: Checks for exit conditions and empty input
3. **Context Building**: Adds user input to conversation history
4. **Work Item Creation**: Returns conversation context as work item for LLM processing


##### Exec Phase - Core Work Execution
**Primary Purpose**: Perform the actual computational work on individual work items

**Detailed Responsibilities**:
1. **Core Logic Execution**: Perform the main business logic (LLM calls, API requests, computations)
2. **External System Interaction**: Make calls to external services, databases, APIs
**No Error Handling** should be written in this phase. All errors are retried by recalling exec internally by the framework.

**Key Characteristics**:
- **Idempotent**: Should be safe to retry multiple times
- **Stateless**: Should not depend on previous executions
- **Independent**: Each work item should be processable independently
- **Efficient**: Should be optimized for the specific type of work

**Example from basic-chat**:
```go
// ChatNode: Execute LLM call with conversation history
func (c *ChatNode) Exec(prepResult PrepResult) (ExecResult, error) {
    if len(prepResult.Messages) == 0 {
        return ExecResult{
            Response: "No conversation history provided.",
            Error:    nil, // No error, just a fallback response
        }, nil
    }
    
    // Make API call to LLM provider with conversation history
    response, err := c.llmProvider.CallLLM(context.Background(), prepResult.Messages)
    if err != nil {
        return ExecResult{
            Response: "",
            Error:    fmt.Errorf("LLM API call failed: %w", err),
        }, err
    }

    return ExecResult{
        Response: response.Content,
        Error:    nil,
    }, nil
}
```

**Key Exec Phase Patterns**:
1. **Input Validation**: Check if required data is present before processing
2. **External API Call**: Make the actual LLM API call with conversation context
3. **Error Propagation**: Return errors for the framework to handle retries
4. **Result Structuring**: Package the response in a consistent format
5. **Stateless Operation**: No dependency on previous executions or external state

##### Post Phase - Result Processing & Flow Control
**Primary Purpose**: Process all execution results and determine the next workflow action

**Detailed Responsibilities**:
1. **Result Aggregation**: Combine results from all parallel executions
2. **State Updates**: Modify the workflow state based on execution results
3. **Success/Failure Analysis**: Determine if the overall operation succeeded
4. **Flow Control Decision**: Choose the next action based on results and business logic
5. **Data Transformation**: Convert execution results into state updates
6. **Error Consolidation**: Handle partial failures and determine recovery strategy
7. **Metrics Collection**: Record performance and success metrics
8. **Cleanup**: Perform any necessary cleanup operations

**Key Characteristics**:
- **Comprehensive**: Must handle all possible execution outcomes
- **Decisive**: Must always return a clear next action
- **State-Aware**: Should consider both results and current state
- **Business Logic**: Encodes the business rules for flow control
- **Atomic**: State updates should be consistent

**Example from basic-chat**:
```go
// ChatNode: Process LLM response and update conversation
func (c *ChatNode) Post(state *ChatState, prepResults []PrepResult, execResults ...ExecResult) core.Action {
    // Handle case where no prep results (exit command)
    if len(prepResults) == 0 {
        fmt.Println("Goodbye!")
        return core.ActionSuccess // Terminate the conversation
    }

    // Get the first (and only) prep and exec results
    execResult := execResults[0]

    // Handle execution errors
    if execResult.Error != nil {
        fmt.Printf("Error: %v\n", execResult.Error)
        // Continue conversation even after errors
        return core.ActionContinue
    }

    // Display the assistant's response
    fmt.Printf("Assistant: %s\n\n", execResult.Response)

    // Add assistant message to conversation history
    assistantMessage := llm.Message{
        Role:    llm.RoleAssistant,
        Content: execResult.Response,
    }
    
    state.Messages = append(state.Messages, assistantMessage)

    // Trim conversation history if it exceeds maximum length
    if len(state.Messages) > c.config.MaxHistory {
        state.Messages = state.Messages[len(state.Messages)-c.config.MaxHistory:]
    }
    
    // Continue the conversation loop
    return core.ActionContinue
}
```

**Key Post Phase Patterns**:
1. **Result Analysis**: Check execution results and handle different outcomes
2. **State Updates**: Add assistant response to conversation history
3. **User Feedback**: Display the response to the user
4. **Memory Management**: Trim conversation history to prevent unbounded growth
5. **Flow Control**: Return appropriate action to continue or terminate conversation
6. **Error Handling**: Handle errors gracefully while maintaining conversation flow

##### ExecFallback Phase - Error Recovery & Default Values
**Primary Purpose**: Provide sensible defaults when Exec phase fails after all retries

**Detailed Responsibilities**:
1. **Error Analysis**: Understand why the execution failed
2. **Default Value Generation**: Provide meaningful fallback results
3. **Graceful Degradation**: Maintain system functionality despite failures
4. **User Communication**: Prepare user-friendly error messages
5. **Recovery Strategy**: Determine if the failure is recoverable
6. **State Preservation**: Ensure system remains in a consistent state

**Key Characteristics**:
- **Always Succeeds**: Must never throw exceptions or return errors
- **Meaningful Defaults**: Should provide useful fallback values
- **Context-Aware**: Should consider the type of failure and context
- **User-Friendly**: Should help maintain good user experience
- **Logged**: Should record enough information for debugging

**Example from basic-chat**:
```go
// ChatNode: Provide fallback response when LLM fails after all retries
func (c *ChatNode) ExecFallback(err error) ExecResult {
    return ExecResult{
        Response: "I'm sorry, I encountered an error and couldn't process your request. Please try again.",
        Error:    nil, // Don't propagate the error to continue conversation
    }
}
```

**Key ExecFallback Patterns**:
1. **User-Friendly Messages**: Provides a helpful message instead of technical error details
2. **Error Suppression**: Returns nil error to prevent conversation termination
3. **Graceful Degradation**: Maintains conversation flow despite LLM failures
4. **Consistent Format**: Returns the same result type as successful execution

### 2. Type-Safe Generic Design

PocketFlowGo uses Go generics to provide compile-time type safety:

```go
type BaseNode[State any, PrepResult any, ExecResults any] interface {
    Prep(state *State) []PrepResult
    Exec(prepResult PrepResult) (ExecResults, error)
    Post(state *State, prepRes []PrepResult, execResults ...ExecResults) Action
    ExecFallback(err error) ExecResults
}
```

**Benefits:**
- **Compile-time Safety**: Type mismatches are caught at compile time
- **Clear Contracts**: Input/output types are explicit in the interface
- **IDE Support**: Full autocomplete and refactoring support
- **Performance**: No runtime type assertions needed

### 3. Workflow Composition

Both Nodes and Flows implement the same `Workflow` interface, enabling composition:

```go
type Workflow[State any] interface {
    Run(state *State) Action
    GetSuccessor(action Action) Workflow[State]
    AddSuccessor(successor Workflow[State], action ...Action) Workflow[State]
}
```

This means you can:
- Chain individual nodes together
- Compose flows from other flows
- Create complex workflows from simple building blocks
- Test components in isolation

## Execution Flow Explained

### 1. Node Execution Lifecycle

When a Node's `Run` method is called:

```go
func (n *Node[State, PrepResult, ExecResults]) Run(state *State) Action {
    // Phase 1: Preparation
    prepRes := n.node.Prep(state)
    if len(prepRes) == 0 {
        return n.node.Post(state, prepRes) // Nothing to execute
    }

    // Phase 2: Execution (with concurrency and retry)
    execResults := make([]ExecResults, len(prepRes))
    
    // Concurrent execution with worker pool
    for i, item := range prepRes {
        execResult, err := n.executeWithRetry(item)
        if err != nil {
            execResults[i] = n.node.ExecFallback(err) // Fallback on failure
        } else {
            execResults[i] = execResult
        }
    }

    // Phase 3: Post-processing and decision
    return n.node.Post(state, prepRes, execResults...)
}
```

**Key Features:**
- **Automatic Retry**: Failed Exec calls are retried up to `maxRetries`
- **Concurrency**: Multiple Exec calls run in parallel (configurable worker pool)
- **Fallback Handling**: `ExecFallback` provides default values on persistent failures
- **State Management**: State is passed through all phases

### 2. Flow Orchestration

A Flow executes a sequence of connected workflows:

```go
func (f *Flow[State]) Run(state *State) Action {
    currentWorkflow := f.startNode
    var finalAction Action = ActionSuccess

    // Execute workflows in sequence following action-based transitions
    for currentWorkflow != nil {
        action := currentWorkflow.Run(state)
        finalAction = action

        // Find next workflow based on the action
        nextWorkflow := currentWorkflow.GetSuccessor(action)
        if nextWorkflow == nil {
            nextWorkflow = f.GetSuccessor(action) // Check flow-level successors
        }

        currentWorkflow = nextWorkflow
    }
    return finalAction
}
```

**Flow Control:**
- Workflows are connected via Actions (strings like "success", "failure", "retry")
- Each workflow can have multiple successors for different actions
- Execution continues until no successor is found for the current action
- State is shared and modified throughout the entire flow

### 3. Action-Based Routing

Actions control the flow of execution:

```go
const (
    ActionContinue Action = "continue"
    ActionSuccess  Action = "success"
    ActionFailure  Action = "failure"
    ActionRetry    Action = "retry"
    ActionDefault  Action = "default"
)
```

**Routing Example:**
```go
// Connect nodes based on actions
planningNode.AddSuccessor(executionNode, ActionExecute)
planningNode.AddSuccessor(responseNode, ActionRespond)
executionNode.AddSuccessor(responseNode, ActionComplete)
responseNode.AddSuccessor(planningNode, ActionContinue)
```

## Practical Examples

### 1. Basic Chat Application (from examples/basic-chat)

The basic-chat example demonstrates a complete conversational AI application:

```go
// ChatNode implements BaseNode interface for conversational chat
type ChatNode struct {
    llmProvider llm.LLMProvider // LLM provider for generating responses
    config      *ChatConfig     // Configuration settings
    firstRun    bool            // Track if this is the first execution
}

// ChatState represents the conversation state
type ChatState struct {
    Messages []llm.Message // Full conversation history using generic format
    Active   bool          // Whether chat is active
}

// PrepResult contains the conversation context for LLM call
type PrepResult struct {
    Messages []llm.Message // Conversation history
}

// ExecResult contains the LLM response
type ExecResult struct {
    Response string // LLM response text
    Error    error  // Any error that occurred
}

// Complete workflow setup from main.go
func main() {
    // Create Gemini client
    geminiClient, err := gemini.NewGeminiClient(ctx, geminiConfig)
    if err != nil {
        log.Fatalf("Failed to create Gemini client: %v", err)
    }

    // Create ChatNode
    chatNode := NewChatNode(geminiClient, chatConfig)

    // Create PocketFlow-Go Node wrapper with retry and concurrency settings
    node := core.NewNode(chatNode, 3, 1)

    // Configure self-loop for continuous conversation
    node.AddSuccessor(node, core.ActionContinue)

    // Create Flow
    flow := core.NewFlow(node)

    // Initialize conversation state
    initialState := ChatState{
        Messages: []llm.Message{}, // Start with empty conversation history
        Active:   true,            // Chat is active
    }

    // Execute the chat flow
    finalAction := flow.Run(&initialState)
}
```

### 2. Structured Data Parsing (from examples/structured-parsing)

The structured-parsing example shows how to extract structured data from unstructured text:

```go
// ResumeData represents the structured output from resume parsing
type ResumeData struct {
    Name         string       `yaml:"name" json:"name" description:"Full name of the candidate"`
    Email        string       `yaml:"email" json:"email" description:"Email address of the candidate"`
    Experience   []Experience `yaml:"experience" json:"experience" description:"List of work experience entries"`
    SkillIndexes []int        `yaml:"skill_indexes" json:"skill_indexes" description:"Array of skill indexes matching the target skills list"`
}

// ResumeParserNode implements BaseNode interface for resume parsing
type ResumeParserNode struct {
    *structured.StructuredNode[ResumeData]
    config *ResumeParserConfig
}

// Prep extracts resume file path and target skills from state
func (r *ResumeParserNode) Prep(state *ResumeParserState) []PrepResult {
    resumeFilePath := state.ResumeFilePath
    if resumeFilePath == "" {
        return []PrepResult{}
    }

    targetSkills := state.TargetSkills
    if len(targetSkills) == 0 {
        targetSkills = r.config.TargetSkills
    }

    return []PrepResult{
        {
            ResumeText:   resumeFilePath, // Pass file path, will be read in Exec
            TargetSkills: targetSkills,
        },
    }
}

// Exec parses the resume using the structured framework
func (r *ResumeParserNode) Exec(prepResult PrepResult) (ExecResult, error) {
    ctx := context.Background()

    // Create additional context with target skills information
    skillsContext := fmt.Sprintf("**Target Skills (use these indexes for skill_indexes field):**\n```\n%s```\n\nImportant: For the skill_indexes field, only include the numeric indexes (0, 1, 2, etc.) of skills found in the resume that match the Target Skills list above.",
        structured.FormatIndexedList(prepResult.TargetSkills))

    // Parse from file using the structured framework
    result, err := r.ParseFromFile(ctx, prepResult.ResumeText, skillsContext)
    if err != nil {
        return result, err
    }

    // Validate the result
    if err := r.ValidateResult(result); err != nil {
        return ExecResult{
            Data:  nil,
            Error: fmt.Errorf("validation failed: %w", err),
        }, err
    }

    return result, nil
}

// Post processes results and displays formatted output
func (r *ResumeParserNode) Post(state *ResumeParserState, prepRes []PrepResult, execResults ...ExecResult) core.Action {
    if state.Context == nil {
        state.Context = make(map[string]interface{})
    }

    for num, execResult := range execResults {
        if execResult.Data != nil {
            // Store result in state
            state.Context[fmt.Sprintf("%d", num)] = execResult.Data
            
            // Extract matched skills for display
            skills := []string{}
            for _, index := range execResult.Data.SkillIndexes {
                skills = append(skills, prepRes[num].TargetSkills[index])
            }
            r.displayFormattedOutput(execResult.Data, skills)
        }
    }

    if len(state.Context) == 0 {
        return core.ActionFailure
    }

    return core.ActionSuccess
}
```

### 3. Invoice Processing Example

Another structured parsing example for invoice data extraction:

```go
// InvoiceData represents the structured output from invoice parsing
type InvoiceData struct {
    InvoiceNumber string        `yaml:"invoice_number" json:"invoice_number" description:"Invoice number or ID"`
    Date          string        `yaml:"date" json:"date" description:"Invoice date"`
    VendorName    string        `yaml:"vendor_name" json:"vendor_name" description:"Name of the vendor or company"`
    TotalAmount   float64       `yaml:"total_amount" json:"total_amount" description:"Total amount due"`
    LineItems     []InvoiceItem `yaml:"line_items" json:"line_items" description:"List of invoice line items"`
}

// InvoiceParserNode demonstrates how easy it is to create new structured parsing nodes
type InvoiceParserNode struct {
    *structured.StructuredNode[InvoiceData]
    config *InvoiceParserConfig
}

// Exec parses the invoice using the structured framework
func (i *InvoiceParserNode) Exec(filePath string) (structured.ParseResult[InvoiceData], error) {
    ctx := context.Background()

    // Create additional context about currency expectations
    currencyContext := fmt.Sprintf("Expected currency: %s. Convert all amounts to this currency if needed.", i.config.Currency)

    // Parse from file using the structured framework - that's it!
    return i.ParseFromFile(ctx, filePath, currencyContext)
}

// Post handles the results and stores them in state
func (i *InvoiceParserNode) Post(state *InvoiceParserState, prepRes []string, execResults ...structured.ParseResult[InvoiceData]) core.Action {
    if state.Context == nil {
        state.Context = make(map[string]interface{})
    }
    
    for num, execResult := range execResults {
        if execResult.Data != nil {
            // Store result in state
            state.Context[fmt.Sprintf("%d", num)] = execResult.Data
            i.displayInvoiceResults(execResult.Data)
        }
    }
    
    if len(state.Context) == 0 {
        return core.ActionFailure
    }

    return core.ActionSuccess
}
```

### 4. Complete Application Setup Pattern

Both examples follow a similar main function pattern:

```go
func main() {
    // 1. Parse command-line flags
    var (
        model = flag.String("model", "gemini-2.0-flash", "Gemini model to use")
        temp  = flag.Float64("temperature", 0.7, "Response temperature (0.0-1.0)")
    )
    flag.Parse()

    // 2. Load configuration from environment
    geminiConfig, err := gemini.NewConfigFromEnv()
    if err != nil {
        // Handle configuration errors with helpful messages
        fmt.Printf("Configuration error: %v\n", err)
        os.Exit(1)
    }

    // 3. Create LLM client
    ctx := context.Background()
    geminiClient, err := gemini.NewGeminiClient(ctx, geminiConfig)
    if err != nil {
        log.Fatalf("Failed to create Gemini client: %v", err)
    }

    // 4. Create application-specific node
    appNode := NewApplicationNode(geminiClient, config)

    // 5. Wrap in PocketFlow Node with retry/concurrency settings
    node := core.NewNode(appNode, 3, 1) // 3 retries, 1 worker

    // 6. Set up workflow connections (varies by application)
    node.AddSuccessor(node, core.ActionContinue) // Self-loop for chat
    // OR
    // No successors for single-shot processing

    // 7. Create flow and execute
    flow := core.NewFlow(node)
    initialState := ApplicationState{...}
    finalAction := flow.Run(&initialState)
}
```

## Testing Strategies

### 1. Unit Testing Nodes

```go
func TestLLMNode(t *testing.T) {
    // Create mock provider
    mockProvider := llm.NewMockProvider("test", []string{"Hello, World!"})
    
    // Create node
    node := NewLLMNode(mockProvider, "Say hello")
    
    // Test each phase
    state := &ChatState{}
    
    // Test Prep
    prepResults := node.Prep(state)
    assert.Len(t, prepResults, 1)
    
    // Test Exec
    execResult, err := node.Exec(prepResults[0])
    assert.NoError(t, err)
    assert.Equal(t, "Hello, World!", execResult)
    
    // Test Post
    action := node.Post(state, prepResults, execResult)
    assert.Equal(t, core.ActionSuccess, action)
}
```

### 2. Integration Testing Flows

```go
func TestChatWorkflow(t *testing.T) {
    mockProvider := llm.NewMockProvider("test", []string{"Response 1", "Response 2"})
    flow := CreateChatWorkflow(mockProvider)
    
    state := &ChatState{Messages: []llm.Message{}}
    
    // Execute flow
    action := flow.Run(state)
    
    // Verify results
    assert.Equal(t, core.ActionSuccess, action)
    assert.Len(t, state.Messages, 2) // User + Assistant message
}
```

### 3. Mock Providers

```go
// Built-in mock provider for testing
mockProvider := llm.NewMockProvider("test-llm", []string{
    "First response",
    "Second response",
    "Third response",
})

// Custom mock with error simulation
type ErrorMockProvider struct {
    shouldError bool
}

func (e *ErrorMockProvider) CallLLM(ctx context.Context, messages []llm.Message) (llm.Message, error) {
    if e.shouldError {
        return llm.Message{}, errors.New("simulated error")
    }
    return llm.Message{Content: "success"}, nil
}
```

## Performance Characteristics

### 1. Concurrency Model

- **Worker Pool**: Each node can configure the number of concurrent workers
- **Goroutine Management**: Framework handles goroutine lifecycle automatically
- **Backpressure**: Channel-based communication prevents memory issues
- **Resource Control**: Configurable limits prevent resource exhaustion

### 2. Memory Management

- **State Sharing**: Single state object shared across workflow (no copying)
- **Streaming**: Large datasets can be processed in chunks via Prep phase
- **Garbage Collection**: Go's GC handles cleanup automatically
- **Resource Cleanup**: Defer statements ensure proper resource cleanup

### 3. Scalability

- **Horizontal**: Multiple workflow instances can run independently
- **Vertical**: Individual nodes can scale with worker pools
- **Stateless**: Nodes are stateless (state is external), enabling easy scaling
- **Composable**: Complex workflows built from simple, reusable components

## Key Advantages

1. **Simplicity**: Minimal API surface area, easy to understand and use
2. **Type Safety**: Compile-time guarantees prevent runtime errors
3. **Testability**: Each component can be tested in isolation
4. **Composability**: Build complex workflows from simple building blocks
5. **Concurrency**: Built-in support for parallel execution
6. **Resilience**: Automatic retry logic and graceful error handling
7. **Flexibility**: No vendor lock-in, works with any LLM provider
8. **Performance**: Efficient execution with minimal overhead

## When to Use PocketFlowGo

**Perfect for:**
- LLM-powered applications with complex logic
- Multi-step AI workflows
- Applications requiring high reliability and error handling
- Systems that need to integrate multiple AI services
- Prototyping and experimentation with AI workflows

**Not ideal for:**
- Simple, single-step LLM calls (use direct API calls instead)
- Applications requiring real-time streaming responses
- Workflows that don't benefit from the three-phase model
