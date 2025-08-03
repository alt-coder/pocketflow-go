package main

import (

	"github.com/alt-coder/pocketflow-go/llm"
)

// AgentState represents the enhanced state for multi-step tool calling with approval
type AgentState struct {
	// Two separate message histories
	CleanedMessages       []llm.Message          `json:"cleaned_messages"`        // Cleaned conversation history for planner
}

func (s AgentState) GetConversation(_ string) *[]llm.Message {
	return &s.CleanedMessages
}

type StateInterface interface {
	GetConversation(key string) *[]llm.Message
	AddMessage(msg llm.Message)
}

// NewAgentState creates a new agent state with default values
func NewAgentState() *AgentState {
	return &AgentState{
		CleanedMessages:       make([]llm.Message, 0),

	}
}

// AddMessage adds a message to both actual and cleaned conversation histories
func (s *AgentState) AddMessage(msg llm.Message) {
	s.CleanedMessages = append(s.CleanedMessages, msg)
}

