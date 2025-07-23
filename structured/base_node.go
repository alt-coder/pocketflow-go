package structured

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alt-coder/pocketflow-go/llm"
)

// StructuredConfig holds common configuration options
type StructuredConfig struct {
	*Config
}

// DefaultBaseConfig returns a default base configuration
func DefaultBaseConfig() *StructuredConfig {
	return &StructuredConfig{
		Config: DefaultConfig(),
	}
}

// StructuredNode provides common functionality for structured parsing nodes
type StructuredNode[T any] struct {
	parser    *Parser
	validator ValidatorInterface[T]
	config    *StructuredConfig
}

// NewStructuredNode creates a new base node with the specified LLM provider, configuration, and validator
func NewStructuredNode[T any](provider llm.LLMProvider, config *StructuredConfig, validator ValidatorInterface[T]) (*StructuredNode[T], error) {
	if config == nil {
		config = DefaultBaseConfig()
	}

	parser, err := NewParser(provider, config.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	// Use provided validator or create default no-op validator if nil
	if validator == nil {
		validator = NewNoOpValidator[T]()
	}

	return &StructuredNode[T]{
		parser:    parser,
		validator: validator,
		config:    config,
	}, nil
}

// ParseFromFile reads a file and parses its content into the specified type
func (b *StructuredNode[T]) ParseFromFile(ctx context.Context, filePath string, additionalContext ...string) (ParseResult[T], error) {
	// Read file content
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return ParseResult[T]{
			Data:  nil,
			Error: fmt.Errorf("failed to read file %s: %w", filePath, err),
		}, err
	}

	// Convert to string and validate content
	fileContent := strings.TrimSpace(string(fileBytes))
	if fileContent == "" {
		err := fmt.Errorf("file %s is empty", filePath)
		return ParseResult[T]{
			Data:  nil,
			Error: err,
		}, err
	}

	// Parse using structured prompt
	return ParseWithStructuredPrompt[T](b.parser, ctx, fileContent, additionalContext...)
}

// ParseFromText parses text content into the specified type
func (b *StructuredNode[T]) ParseFromText(ctx context.Context, textContent string, additionalContext ...string) (ParseResult[T], error) {
	if strings.TrimSpace(textContent) == "" {
		err := fmt.Errorf("text content is empty")
		return ParseResult[T]{
			Data:  nil,
			Error: err,
		}, err
	}

	return ParseWithStructuredPrompt[T](b.parser, ctx, textContent, additionalContext...)
}

// ParseWithCustomPrompt parses using a custom prompt
func (b *StructuredNode[T]) ParseWithCustomPrompt(ctx context.Context, customPrompt string) (ParseResult[T], error) {
	return ParseWithPrompt[T](b.parser, ctx, customPrompt)
}

// ValidateResult validates the parsed result using the configured validator
func (b *StructuredNode[T]) ValidateResult(result ParseResult[T]) error {
	if result.Error != nil {
		return result.Error
	}

	if result.Data == nil {
		return fmt.Errorf("parsed data is nil")
	}

	return b.validator.Validate(result.Data)
}

// CreateFallbackResult creates a fallback result with default values
func (b *StructuredNode[T]) CreateFallbackResult(err error) ParseResult[T] {
	fmt.Printf("Creating fallback result due to error: %v\n", err)

	var zero T
	return ParseResult[T]{
		Data:  &zero,
		Error: err, 
	}
}

// FormatIndexedList creates a formatted string of items with their indexes
func FormatIndexedList(items []string) string {
	var builder strings.Builder
	for i, item := range items {
		builder.WriteString(fmt.Sprintf("%d: %s\n", i, item))
	}
	return builder.String()
}

