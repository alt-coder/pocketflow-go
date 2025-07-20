package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/llm/gemini"
)

func main() {
	// Command-line flag parsing
	var (
		model = flag.String("model", "gemini-2.0-flash", "Gemini model to use")
		temp  = flag.Float64("temperature", 0.7, "Response temperature (0.0-1.0)")
	)
	flag.Parse()

	// Load configuration from environment variables
	geminiConfig, err := gemini.NewConfigFromEnv()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		fmt.Println("\nSetup Instructions:")
		fmt.Println("1. Set your Google API key: export GOOGLE_API_KEY=your_api_key_here")
		fmt.Println("2. Optional: Set model: export CHAT_MODEL=gemini-2.0-flash")
		fmt.Println("3. Optional: Set temperature: export CHAT_TEMPERATURE=0.7")
		os.Exit(1)
	}

	// Override config with command-line flags if provided
	if *model != "gemini-2.0-flash" {
		geminiConfig.Model = *model
	}
	if *temp != 0.7 {
		geminiConfig.Temperature = float32(*temp)
	}

	// Create Gemini client
	ctx := context.Background()
	geminiClient, err := gemini.NewGeminiClient(ctx, geminiConfig)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Create chat configuration
	chatConfig := NewChatConfig(geminiConfig)

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
	fmt.Printf("Starting chat with %s (model: %s, temperature: %.1f)\n\n", 
		geminiClient.GetName(), geminiConfig.Model, geminiConfig.Temperature)
	
	finalAction := flow.Run(&initialState)
	
	// Log final action for debugging
	fmt.Printf("Chat ended with action: %v\n", finalAction)
}