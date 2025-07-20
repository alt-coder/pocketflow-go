package main

import (
	"github.com/alt-coder/pocketflow-go/llm/gemini"
)

// ChatConfig holds configuration settings for the chat application
type ChatConfig struct {
	LLMConfig   *gemini.Config // Gemini-specific configuration
	MaxHistory  int            // Maximum conversation history length
	WelcomeMsg  string         // Welcome message to display
}

// NewChatConfig creates a new chat configuration with sensible defaults
func NewChatConfig(llmConfig *gemini.Config) *ChatConfig {
	return &ChatConfig{
		LLMConfig:  llmConfig,
		MaxHistory: 50, // Keep last 50 messages to prevent unbounded growth
		WelcomeMsg: "Welcome to PocketFlow-Go Chat! Type 'exit' to quit.",
	}
}