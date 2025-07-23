package structured

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/prompt"
	yaml "gopkg.in/yaml.v3"
)

// Config holds common configuration for structured parsing nodes
type Config struct {
	MaxRetries int           // Maximum retry attempts
	Timeout    time.Duration // LLM call timeout
}

// DefaultConfig returns a default configuration for structured parsing
func DefaultConfig() *Config {
	return &Config{
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
}

// ValidateConfig validates the configuration parameters
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}

	return nil
}

// Parser provides common structured parsing functionality
type Parser struct {
	llmProvider llm.LLMProvider
	config      *Config
}

// NewParser creates a new structured parser with the specified LLM provider and configuration
func NewParser(provider llm.LLMProvider, config *Config) (*Parser, error) {
	if provider == nil {
		return nil, fmt.Errorf("llm provider cannot be nil")
	}

	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return &Parser{
		llmProvider: provider,
		config:      config,
	}, nil
}

// ParseResult contains the result of structured parsing
type ParseResult[T any] struct {
	Data  *T
	Error error
}

// ParseWithPrompt executes LLM parsing with a custom prompt and parses the response into type T
func ParseWithPrompt[T any](p *Parser, ctx context.Context, customPrompt string) (ParseResult[T], error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Create message for LLM
	message := llm.Message{
		Role:    llm.RoleUser,
		Content: customPrompt,
	}

	// Call LLM provider with constructed prompt
	response, err := p.llmProvider.CallLLM(timeoutCtx, []llm.Message{message})
	if err != nil {
		return ParseResult[T]{
			Data:  nil,
			Error: fmt.Errorf("LLM call failed: %w", err),
		}, err
	}

	// Parse the response into the target type
	return parseResponse[T](response.Content)
}

// ParseWithStructuredPrompt generates a structured prompt for type T and executes parsing
func ParseWithStructuredPrompt[T any](p *Parser, ctx context.Context, inputData string, additionalContext ...string) (ParseResult[T], error) {
	// Generate structured prompt for type T
	structuredPrompt := prompt.GenerateStructuredPrompt[T]()

	// Build the full prompt with input data and context
	var promptBuilder strings.Builder
	promptBuilder.WriteString("Analyze the following data and extract the requested information.\n\n")

	promptBuilder.WriteString("**Input Data:**\n```\n")
	promptBuilder.WriteString(inputData)
	promptBuilder.WriteString("\n```\n\n")

	// Add additional context if provided
	for i, context := range additionalContext {
		promptBuilder.WriteString(fmt.Sprintf("**Additional Context %d:**\n", i+1))
		promptBuilder.WriteString(context)
		promptBuilder.WriteString("\n\n")
	}

	promptBuilder.WriteString(structuredPrompt)

	return ParseWithPrompt[T](p, ctx, promptBuilder.String())
}

// parseResponse parses LLM response content into the target type T
func parseResponse[T any](responseContent string) (ParseResult[T], error) {
	var result T

	// Try YAML parsing first
	yamlContent := ExtractYAMLFromResponse(responseContent)
	if yamlContent != "" {
		err := yaml.Unmarshal([]byte(yamlContent), &result)
		if err == nil {
			return ParseResult[T]{
				Data:  &result,
				Error: nil,
			}, nil
		}
	}

	// Try JSON parsing as fallback
	jsonContent := ExtractJSONFromResponse(responseContent)
	if jsonContent != "" {
		err := json.Unmarshal([]byte(jsonContent), &result)
		if err == nil {
			return ParseResult[T]{
				Data:  &result,
				Error: nil,
			}, nil
		}
	}

	// If both parsing methods fail, return error
	err := fmt.Errorf("failed to parse response as YAML or JSON")
	return ParseResult[T]{
		Data:  nil,
		Error: err,
	}, err
}

// ExtractYAMLFromResponse extracts YAML content from LLM response using string parsing
func ExtractYAMLFromResponse(response string) string {
	// Look for YAML code blocks first (```yaml ... ```)
	yamlBlockStart := "```yaml"
	yamlBlockEnd := "```"

	startIndex := strings.Index(response, yamlBlockStart)
	if startIndex != -1 {
		// Found YAML code block, extract content between markers
		startIndex += len(yamlBlockStart)
		endIndex := strings.Index(response[startIndex:], yamlBlockEnd)
		if endIndex != -1 {
			yamlContent := strings.TrimSpace(response[startIndex : startIndex+endIndex])
			return yamlContent
		}
	}

	// Look for generic code blocks (``` ... ```)
	codeBlockStart := "```"
	startIndex = strings.Index(response, codeBlockStart)
	if startIndex != -1 {
		startIndex += len(codeBlockStart)
		// Skip any language identifier on the same line
		if newlineIndex := strings.Index(response[startIndex:], "\n"); newlineIndex != -1 {
			startIndex += newlineIndex + 1
		}

		endIndex := strings.Index(response[startIndex:], yamlBlockEnd)
		if endIndex != -1 {
			yamlContent := strings.TrimSpace(response[startIndex : startIndex+endIndex])
			return yamlContent
		}
	}

	// If no code blocks found, try to extract YAML-like content
	lines := strings.Split(response, "\n")
	var yamlLines []string
	inYAML := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Start collecting when we see YAML-like content (key: value patterns)
		if strings.Contains(trimmedLine, ":") && !strings.HasPrefix(trimmedLine, "http") {
			inYAML = true
		}

		if inYAML {
			// Stop if we hit a line that doesn't look like YAML
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") &&
				!strings.Contains(trimmedLine, ":") && !strings.HasPrefix(trimmedLine, "-") &&
				!strings.HasPrefix(trimmedLine, " ") {
				break
			}
			yamlLines = append(yamlLines, line)
		}
	}

	if len(yamlLines) > 0 {
		return strings.Join(yamlLines, "\n")
	}

	// Return empty string if no YAML content found
	return ""
}

// ExtractJSONFromResponse extracts JSON content from LLM response
func ExtractJSONFromResponse(response string) string {
	// Look for JSON code blocks first (```json ... ```)
	jsonBlockStart := "```json"
	jsonBlockEnd := "```"

	startIndex := strings.Index(response, jsonBlockStart)
	if startIndex != -1 {
		startIndex += len(jsonBlockStart)
		endIndex := strings.Index(response[startIndex:], jsonBlockEnd)
		if endIndex != -1 {
			jsonContent := strings.TrimSpace(response[startIndex : startIndex+endIndex])
			return jsonContent
		}
	}

	// Look for generic code blocks that might contain JSON
	codeBlockStart := "```"
	startIndex = strings.Index(response, codeBlockStart)
	if startIndex != -1 {
		startIndex += len(codeBlockStart)
		// Skip any language identifier on the same line
		if newlineIndex := strings.Index(response[startIndex:], "\n"); newlineIndex != -1 {
			startIndex += newlineIndex + 1
		}

		endIndex := strings.Index(response[startIndex:], jsonBlockEnd)
		if endIndex != -1 {
			content := strings.TrimSpace(response[startIndex : startIndex+endIndex])
			// Check if it looks like JSON (starts with { or [)
			if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
				return content
			}
		}
	}

	// Try to find JSON objects in the response
	lines := strings.Split(response, "\n")
	var jsonLines []string
	inJSON := false
	braceCount := 0

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "{") {
			inJSON = true
			braceCount = 0
		}

		if inJSON {
			jsonLines = append(jsonLines, line)

			// Count braces to find the end of JSON object
			for _, char := range trimmedLine {
				switch char {
				case '{':
					braceCount++
				case '}':
					braceCount--
				}
			}

			// If braces are balanced, we've found the complete JSON
			if braceCount == 0 && len(jsonLines) > 0 {
				break
			}
		}
	}

	if len(jsonLines) > 0 {
		return strings.Join(jsonLines, "\n")
	}

	return ""
}
