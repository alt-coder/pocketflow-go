# Structured Parsing Framework

This package provides a clean, generic framework for creating structured parsing nodes that use LLMs to extract structured data from unstructured text. It focuses on core functionality while allowing domain-specific validation to be implemented in the application layer.

## Key Components

### 1. Parser (`parser.go`)
- **`Parser`**: Core parsing functionality with LLM integration
- **`ParseWithStructuredPrompt[T]`**: Automatically generates prompts based on struct tags
- **`ParseWithPrompt[T]`**: Uses custom prompts for parsing
- **`ExtractYAMLFromResponse`** / **`ExtractJSONFromResponse`**: Response parsing utilities

### 2. Validation (`validation.go`)
- **`ValidatorInterface[T]`**: Generic interface for type-safe validation
- **`NoOpValidator[T]`**: Default validator that performs minimal validation
- **`ValidateIndexes`**: Common utility for validating numeric indexes

### 3. Base Node (`base_node.go`)
- **`BaseNode[T]`**: Generic, embeddable base functionality for structured parsing nodes
- **Type-safe methods**: `ParseFromFile()`, `ParseFromText()`, `ValidateResult()`
- **`CreateFallbackResult()`**: Fallback handling
- **Utility functions**: `FormatIndexedList`, `MapIndexesToValues`, etc.

## Usage Examples

### Basic Structured Parsing Node

```go
// 1. Define your data structure with description tags
type PersonData struct {
    Name  string `yaml:"name" json:"name" description:"Full name of the person"`
    Email string `yaml:"email" json:"email" description:"Email address"`
    Age   int    `yaml:"age" json:"age" description:"Age in years"`
}

// 2. Create a node that embeds generic BaseNode[T]
type PersonParserNode struct {
    *structured.BaseNode[PersonData]
    config *PersonParserConfig
}

// 3. Simple constructor with optional custom validator
func NewPersonParserNode(provider llm.LLMProvider, config *PersonParserConfig) (*PersonParserNode, error) {
    // Use nil for default NoOpValidator, or provide custom validator
    baseNode, err := structured.NewBaseNode[PersonData](provider, config.BaseConfig, nil)
    if err != nil {
        return nil, err
    }
    return &PersonParserNode{BaseNode: baseNode, config: config}, nil
}

// 4. Minimal implementation with type safety
func (p *PersonParserNode) Exec(inputText string) (structured.ParseResult[PersonData], error) {
    ctx := context.Background()
    result, err := p.ParseFromText(ctx, inputText)
    if err != nil {
        return result, err
    }
    return result, p.ValidateResult(result)
}
```

### Advanced Example with Custom Domain-Specific Validation

```go
// Custom validator for domain-specific rules (implemented in examples/)
type ResumeValidator struct {
    config *ResumeValidationConfig
}

func (v *ResumeValidator) Validate(data *ResumeData) error {
    // Domain-specific validation logic here
    if strings.TrimSpace(data.Email) != "" {
        emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
        if !emailRegex.MatchString(data.Email) {
            return fmt.Errorf("invalid email format")
        }
    }
    return nil
}

// Node with custom validator
type ResumeParserNode struct {
    *structured.BaseNode[ResumeData]
    config *ResumeParserConfig
}

func NewResumeParserNode(provider llm.LLMProvider, config *ResumeParserConfig) (*ResumeParserNode, error) {
    // Inject custom validator
    validator := NewResumeValidator(config.ValidationConfig)
    baseNode, err := structured.NewBaseNode[ResumeData](provider, config.BaseConfig, validator)
    if err != nil {
        return nil, err
    }
    return &ResumeParserNode{BaseNode: baseNode, config: config}, nil
}

func (r *ResumeParserNode) Exec(prepResult PrepResult) (structured.ParseResult[ResumeData], error) {
    ctx := context.Background()
    
    // Add custom context for better parsing
    skillsContext := fmt.Sprintf("Target Skills:\n%s", 
        structured.FormatIndexedList(prepResult.TargetSkills))
    
    // Parse using structured framework with type safety
    result, err := r.ParseFromFile(ctx, prepResult.FilePath, skillsContext)
    if err != nil {
        return result, err
    }
    
    // Framework handles validation automatically via injected validator
    if err := r.ValidateResult(result); err != nil {
        return structured.ParseResult[ResumeData]{Data: nil, Error: err}, err
    }
    
    // Additional domain-specific validation using framework utilities
    if result.Data != nil && result.Data.SkillIndexes != nil {
        err := structured.ValidateIndexes(result.Data.SkillIndexes, 
            len(prepResult.TargetSkills)-1, "skill_indexes")
        if err != nil {
            return structured.ParseResult[ResumeData]{Data: nil, Error: err}, err
        }
    }
    
    return result, nil
}
```

## Configuration

### BaseConfig
Core parsing configuration:

```go
config := &structured.BaseConfig{
    Config: &structured.Config{
        MaxRetries: 3,
        Timeout:    30 * time.Second,
    },
}
```

### Custom Node Configuration
Extend BaseConfig for node-specific settings:

```go
type ResumeParserConfig struct {
    *structured.BaseConfig
    TargetSkills       []string                    // Node-specific configuration
    ValidationConfig   *ResumeValidationConfig     // Domain-specific validation
}
```

## Validation Architecture

The framework uses a clean separation between generic framework functionality and domain-specific validation:

### Framework Level (structured package)
- **`ValidatorInterface[T]`**: Generic validation contract
- **`NoOpValidator[T]`**: Minimal default validation
- **`ValidateIndexes`**: Common utility functions

### Application Level (examples package)
- **Domain-specific validators**: `ResumeValidator`, `InvoiceValidator`, etc.
- **Custom validation rules**: Email formats, business logic, data constraints
- **Validation configuration**: Domain-specific validation settings

## Struct Tags

The framework uses struct tags to generate better prompts:

- **`description`**: Explains the field purpose in generated prompts
- **`yaml`** / **`json`**: Controls serialization format

```go
type UserData struct {
    Name     string `yaml:"name" json:"name" description:"User's full name"`
    Email    string `yaml:"email" json:"email" description:"Email address"`
    Bio      string `yaml:"bio" json:"bio" description:"User biography"`
}
```

## Benefits

1. **Type Safety**: Compile-time guarantees with `BaseNode[T]`
2. **Clean Separation**: Framework vs. domain-specific concerns
3. **Flexible Validation**: Inject custom validators or use defaults
4. **Reduced Boilerplate**: ~80% less code for new structured parsing nodes
5. **Consistent API**: All nodes follow the same patterns
6. **Easy Testing**: Generic types make mocking and testing simpler
7. **Extensible**: Easy to add new parsing nodes with minimal code

## Migration from Manual Implementation

To migrate existing structured parsing nodes:

1. **Make BaseNode Generic**: Change `BaseNode` to `BaseNode[YourDataType]`
2. **Create Domain Validators**: Move validation logic to examples package
3. **Inject Validators**: Pass custom validators to `NewBaseNode[T]`
4. **Use Type-Safe Methods**: Call methods directly on BaseNode instance
5. **Simplify Configuration**: Remove framework-specific validation config

The framework maintains the same three-phase execution model (Prep/Exec/Post) while providing better type safety and cleaner separation of concerns.