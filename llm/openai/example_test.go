package openai_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/llm/openai"
)

func ExampleNewOpenAIClientFromEnv() {
	// Set required environment variable
	os.Setenv("OPENAI_API_KEY", "sk-your-api-key-here")

	// Create client from environment - uses official go-openai client internally
	client, err := openai.NewOpenAIClientFromEnv(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a simple conversation
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: "Hello! Can you help me with Go programming?",
		},
	}

	// Call the LLM - benefits from official client's reliability and features
	response, err := client.CallLLM(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Assistant: %s\n", response.Content)
}

func ExampleNewOpenAIClient() {
	// Manual configuration
	config := &openai.Config{
		APIKey:      "sk-your-api-key-here",
		Model:       "gpt-4o",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	client, err := openai.NewOpenAIClient(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Multi-turn conversation
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are a helpful programming assistant.",
		},
		{
			Role:    llm.RoleUser,
			Content: "What's the difference between a slice and an array in Go?",
		},
	}

	response, err := client.CallLLM(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Assistant: %s\n", response.Content)
}

func ExampleOpenAIClient_CallLLM_withToolCalls() {
	// This example shows how tool calls would be handled
	// Note: This is a conceptual example - actual tool calls would come from the LLM

	config := &openai.Config{
		APIKey:      "sk-your-api-key-here",
		Model:       "gpt-4o",
		Temperature: 0.7,
	}

	client, err := openai.NewOpenAIClient(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: "What's the weather like in New York?",
		},
	}

	response, err := client.CallLLM(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the LLM wants to make tool calls
	if len(response.ToolCalls) > 0 {
		for _, toolCall := range response.ToolCalls {
			fmt.Printf("LLM wants to call tool: %s with args: %v\n",
				toolCall.ToolName, toolCall.ToolArgs)

			// Here you would execute the actual tool and create a result
			toolResult := llm.ToolResults{
				Id:      toolCall.Id,
				Content: "Weather in New York: 72°F, sunny",
				IsError: false,
			}

			// Add tool result to conversation
			messages = append(messages, llm.Message{
				Role:        llm.RoleUser,
				ToolResults: []llm.ToolResults{toolResult},
			})

			// Continue conversation with tool results
			finalResponse, err := client.CallLLM(context.Background(), messages)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Final response: %s\n", finalResponse.Content)
		}
	} else {
		fmt.Printf("Assistant: %s\n", response.Content)
	}
}

func ExampleOpenAIClient_CallLLM_withImage() {
	config := &openai.Config{
		APIKey: "sk-your-api-key-here",
		Model:  "gpt-4o", // Vision-capable model
	}

	client, err := openai.NewOpenAIClient(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Read an image file (this is just example data)
	imageData := []byte("base64-encoded-image-data-here")

	message := llm.Message{
		Role:     llm.RoleUser,
		Content:  "What do you see in this image?",
		Media:    imageData,
		MimeType: "image/jpeg",
	}

	response, err := client.CallLLM(context.Background(), []llm.Message{message})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Vision analysis: %s\n", response.Content)
}

func ExampleConfig_Validate() {
	// Valid configuration
	config := &openai.Config{
		APIKey:      "sk-valid-key",
		Model:       "gpt-4o",
		Temperature: 0.7,
		MaxRetries:  3,
		BaseURL:     "https://api.openai.com/v1",
	}

	if err := config.Validate(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
	} else {
		fmt.Println("Configuration is valid")
	}

	// Invalid configuration
	invalidConfig := &openai.Config{
		APIKey:      "", // Missing API key
		Model:       "gpt-4o",
		Temperature: 3.0, // Invalid temperature
	}

	if err := invalidConfig.Validate(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
	}

	// Output:
	// Configuration is valid
	// Configuration error: OPENAI_API_KEY environment variable is required. Please set it with your OpenAI API key
}
