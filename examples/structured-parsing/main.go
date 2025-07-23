package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm/gemini"
)

func main() {
	// Command-line flag parsing
	var (
		model      = flag.String("model", "gemini-2.0-flash", "Gemini model to use")
		temp       = flag.Float64("temperature", 0.3, "Response temperature (0.0-1.0)")
		resumeFile = flag.String("resume", "examples/structured-parsing/data.txt", "Path to resume file")
	)
	flag.Parse()

	// Load configuration from environment variables
	geminiConfig, err := gemini.NewConfigFromEnv()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		fmt.Println("\nSetup Instructions:")
		fmt.Println("1. Set your Google API key: export GOOGLE_API_KEY=your_api_key_here")
		fmt.Println("2. Optional: Set model: export GEMINI_MODEL=gemini-2.0-flash")
		fmt.Println("3. Optional: Set temperature: export GEMINI_TEMPERATURE=0.3")
		os.Exit(1)
	}

	// Override config with command-line flags if provided
	if *model != "gemini-2.0-flash" {
		geminiConfig.Model = *model
	}
	if *temp != 0.3 {
		geminiConfig.Temperature = float32(*temp)
	}

	// Create Gemini client
	ctx := context.Background()
	geminiClient, err := gemini.NewGeminiClient(ctx, geminiConfig)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Create resume parser configuration with target skills matching Python example
	config := &ResumeParserConfig{
		TargetSkills: []string{
			"Team leadership & management",
			"CRM software",
			"Project management",
			"Public speaking",
			"Microsoft Office",
			"Python",
			"Data Analysis",
		},
	}

	// Create ResumeParserNode
	resumeParserNode, err := NewResumeParserNode(geminiClient, config)
	if err != nil {
		log.Fatalf("Failed to create resume parser node: %v", err)
	}

	// Create PocketFlow-Go Node wrapper with retry and concurrency settings
	node := core.NewNode(resumeParserNode, 3, 1)

	// Create Flow
	flow := core.NewFlow(node)

	// Initialize resume parsing state
	initialState := ResumeParserState{
		ResumeFilePath: *resumeFile,
		TargetSkills:   config.TargetSkills,
		Context:        make(map[string]interface{}),
	}

	// Execute the resume parsing flow
	fmt.Printf("Starting resume parsing with %s (model: %s, temperature: %.1f)\n",
		geminiClient.GetName(), geminiConfig.Model, geminiConfig.Temperature)
	fmt.Printf("Resume file: %s\n\n", *resumeFile)

	finalAction := flow.Run(&initialState)

	// Log final action and display results
	fmt.Printf("\nResume parsing completed with action: %v\n", finalAction)

	// Display final state information
	if finalAction == core.ActionSuccess {
		if structuredData, exists := initialState.Context["structured_data"]; exists {
			fmt.Println("\nFinal structured data stored in state:")
			if resumeData, ok := structuredData.(*ResumeData); ok {
				fmt.Printf("  Name: %s\n", resumeData.Name)
				fmt.Printf("  Email: %s\n", resumeData.Email)
				fmt.Printf("  Experience count: %d\n", len(resumeData.Experience))
				fmt.Printf("  Skill matches: %d\n", len(resumeData.SkillIndexes))
			}
		}

		if foundSkills, exists := initialState.Context["found_skills"]; exists {
			if skills, ok := foundSkills.([]string); ok && len(skills) > 0 {
				fmt.Println("\nMatched skills:")
				for i, skill := range skills {
					fmt.Printf("  %d. %s\n", i+1, skill)
				}
			}
		}
	}
}
