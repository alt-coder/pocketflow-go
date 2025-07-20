package core

// Action represents the result of a node execution that determines flow control
type Action string

// Common actions
const (
	ActionContinue Action = "continue"
	ActionSuccess  Action = "success"
	ActionFailure  Action = "failure"
	ActionRetry    Action = "retry"
	ActionDefault  Action = "default"
)

