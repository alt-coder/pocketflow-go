package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/structured"
)

// ResumeData represents the structured output from resume parsing
type ResumeData struct {
	Name         string       `yaml:"name" json:"name" description:"Full name of the candidate"`
	Email        string       `yaml:"email" json:"email" description:"Email address of the candidate"`
	Experience   []Experience `yaml:"experience" json:"experience" description:"List of work experience entries"`
	SkillIndexes []int        `yaml:"skill_indexes" json:"skill_indexes" description:"Array of skill indexes matching the target skills list"`
}

// Experience represents work experience entry
type Experience struct {
	Title   string `yaml:"title" json:"title" description:"Job title or position held"`
	Company string `yaml:"company" json:"company" description:"Company or organization name"`
}

// ResumeParserConfig holds configuration specific to resume parsing
type ResumeParserConfig struct {
	*structured.StructuredConfig
	TargetSkills     []string                // Skills to match against resume
	ValidationConfig *ResumeValidationConfig // Resume-specific validation config
}

// ResumeParserState represents the shared state for resume parsing workflow
type ResumeParserState struct {
	ResumeFilePath string                 `json:"resume_file_path"`
	TargetSkills   []string               `json:"target_skills,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

// PrepResult contains data prepared for LLM execution
type PrepResult struct {
	ResumeText   string
	TargetSkills []string
}

// ExecResult contains the parsed structured data
type ExecResult = structured.ParseResult[ResumeData]

// ResumeParserNode implements BaseNode interface for resume parsing using structured package
type ResumeParserNode struct {
	*structured.StructuredNode[ResumeData]
	config *ResumeParserConfig
}

// NewResumeParserNode creates a new resume parser node with the specified LLM provider and configuration
func NewResumeParserNode(provider llm.LLMProvider, config *ResumeParserConfig) (*ResumeParserNode, error) {
	if provider == nil {
		return nil, fmt.Errorf("llm provider cannot be nil")
	}

	if config == nil {
		config = DefaultResumeParserConfig()
	}

	// Validate configuration parameters
	if len(config.TargetSkills) == 0 {
		return nil, fmt.Errorf("target skills cannot be empty")
	}

	// Create a resume-specific validator
	validator := NewResumeValidator(config.ValidationConfig)

	baseNode, err := structured.NewStructuredNode(provider, config.StructuredConfig, validator)
	if err != nil {
		return nil, fmt.Errorf("failed to create base node: %w", err)
	}

	return &ResumeParserNode{
		StructuredNode: baseNode,
		config:         config,
	}, nil
}

// Prep implements the preparation phase of the three-phase execution model
func (r *ResumeParserNode) Prep(state *ResumeParserState) []PrepResult {
	// Extract resume file path from state
	resumeFilePath := state.ResumeFilePath
	if resumeFilePath == "" {
		return []PrepResult{}
	}

	// Determine target skills - use from state if available, otherwise from config
	targetSkills := state.TargetSkills
	if len(targetSkills) == 0 {
		targetSkills = r.config.TargetSkills
	}

	// Return PrepResult - file reading will be handled in Exec
	return []PrepResult{
		{
			ResumeText:   resumeFilePath, // Pass file path, will be read in Exec
			TargetSkills: targetSkills,
		},
	}
}

// Exec implements the execution phase using the structured parsing framework
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

	// Additional validation for skill indexes
	if result.Data != nil && result.Data.SkillIndexes != nil {
		if err := ValidateIndexes(result.Data.SkillIndexes, len(prepResult.TargetSkills)-1, "skill_indexes"); err != nil {
			return ExecResult{
				Data:  nil,
				Error: err,
			}, err
		}
	}

	return result, nil
}

// Post implements the post-processing phase with state management and output
func (r *ResumeParserNode) Post(state *ResumeParserState, prepRes []PrepResult, execResults ...ExecResult) core.Action {
	// Use the base node's result handling

	if state.Context == nil {
		state.Context = make(map[string]interface{})
	}
	for num, execResult := range execResults {
		// Display results
		if execResult.Data != nil {
			// Store result in state
			state.Context[fmt.Sprintf("%d", num)] = execResult.Data
			fmt.Println(num)
			skills := []string{}
			for _, index := range execResult.Data.SkillIndexes {
				skills = append(skills, prepRes[num].TargetSkills[index])
			}
			r.displayFormattedOutput(execResult.Data, skills)
		}
	}
	if len(state.Context)==0 {
		return core.ActionFailure
	}

	return core.ActionSuccess
}

// displayFormattedOutput generates and prints formatted output showing extracted information
func (r *ResumeParserNode) displayFormattedOutput(data *ResumeData, foundSkills []string) {
	fmt.Println("\n=== Resume Parsing Results ===")

	// Display name
	fmt.Printf("Name: %s\n", data.Name)

	// Display email
	fmt.Printf("Email: %s\n", data.Email)

	// Display experience
	fmt.Println("\nExperience:")
	if len(data.Experience) == 0 {
		fmt.Println("  No experience found")
	} else {
		for i, exp := range data.Experience {
			fmt.Printf("  %d. %s at %s\n", i+1, exp.Title, exp.Company)
		}
	}

	// Display found skills
	fmt.Println("\nMatched Skills:")
	if len(foundSkills) == 0 {
		fmt.Println("  No matching skills found")
	} else {
		for i, skill := range foundSkills {
			fmt.Printf("  %d. %s\n", i+1, skill)
		}
	}

	fmt.Println("==============================")
}

// ExecFallback provides a default result if Exec fails after all retries
func (r *ResumeParserNode) ExecFallback(err error) ExecResult {
	return r.CreateFallbackResult(err)
}

// DefaultResumeParserConfig returns a default configuration for the resume parser node
func DefaultResumeParserConfig() *ResumeParserConfig {
	return &ResumeParserConfig{
		StructuredConfig: structured.DefaultBaseConfig(),
		TargetSkills: []string{
			"Team leadership & management",
			"CRM software",
			"Project management",
			"Public speaking",
			"Microsoft Office",
			"Python",
			"Data Analysis",
		},
		ValidationConfig: DefaultResumeValidationConfig(),
	}
}

// ValidateIndexes validates that indexes are within the valid range
// This is a common utility that many structured parsing nodes might need
func ValidateIndexes(indexes []int, maxIndex int, fieldName string) error {
	var validationErrors []string

	for i, index := range indexes {
		if index < 0 {
			validationErrors = append(validationErrors,
				fmt.Sprintf("%s[%d] contains negative value %d, must be >= 0", fieldName, i, index))
		}
		if index > maxIndex {
			validationErrors = append(validationErrors,
				fmt.Sprintf("%s[%d] contains value %d, must be <= %d", fieldName, i, index, maxIndex))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("index validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}
