package prompt

import (
	"strings"
	"testing"
)

// Test struct with yaml and description tags
type Person struct {
	Name       string   `yaml:"name" json:"name" description:"Full name of the person"`
	Age        int      `yaml:"age" json:"age" description:"Age in years"`
	Email      string   `yaml:"email" json:"email" description:"Email address"`
	Skills     []string `yaml:"skills" json:"skills" description:"List of technical skills"`
	Address    Address  `yaml:"address" json:"address" description:"Home address information"`
	IsEmployed bool     `yaml:"is_employed" json:"is_employed" description:"Current employment status"`
}

type Address struct {
	Street  string `yaml:"street" json:"street" description:"Street address"`
	City    string `yaml:"city" json:"city" description:"City name"`
	Country string `yaml:"country" json:"country" description:"Country name"`
}

// Test struct without yaml tags
type SimpleStruct struct {
	ID   int    `json:"id" description:"Unique identifier"`
	Name string `json:"name" description:"Name field"`
}

func TestGenerateStructuredPrompt(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() string
		contains []string
	}{
		{
			name:     "Person struct with yaml tags",
			testFunc: func() string { return GenerateStructuredPrompt[Person]() },
			contains: []string{
				"YAML format",
				"name:",
				"age:",
				"email:",
				"skills:",
				"address:",
				"is_employed:",
				"Full name of the person",
				"Age in years",
				"List of technical skills",
				"Home address information",
			},
		},
		{
			name:     "SimpleStruct without yaml tags",
			testFunc: func() string { return GenerateStructuredPrompt[SimpleStruct]() },
			contains: []string{
				"JSON format",
				"\"id\":",
				"\"name\":",
				"Unique identifier",
				"Name field",
			},
		},
		{
			name:     "Pointer to struct",
			testFunc: func() string { return GenerateStructuredPrompt[*Person]() },
			contains: []string{
				"YAML format",
				"name:",
				"Full name of the person",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.testFunc()

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected prompt to contain '%s', but it didn't.\nFull prompt:\n%s", expected, result)
				}
			}
		})
	}
}

func TestValidateStructForPrompt(t *testing.T) {
	tests := []struct {
		name      string
		testFunc  func() error
		shouldErr bool
	}{
		{
			name:      "Valid struct with yaml tags",
			testFunc:  func() error { return ValidateStructForPrompt[Person]() },
			shouldErr: false,
		},
		{
			name:      "Valid struct without yaml tags",
			testFunc:  func() error { return ValidateStructForPrompt[SimpleStruct]() },
			shouldErr: false,
		},
		{
			name:      "Invalid non-struct type",
			testFunc:  func() error { return ValidateStructForPrompt[string]() },
			shouldErr: true,
		},
		{
			name:      "Valid pointer to struct",
			testFunc:  func() error { return ValidateStructForPrompt[*Person]() },
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestHasYamlTags(t *testing.T) {
	tests := []struct {
		name     string
		testType interface{}
		expected bool
	}{
		{
			name:     "Struct with yaml tags",
			testType: Person{},
			expected: true,
		},
		{
			name:     "Struct without yaml tags",
			testType: SimpleStruct{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to test the internal function, so we'll create a wrapper
			switch  tt.testType.(type) {
			case Person:
				result := GenerateStructuredPrompt[Person]()
				hasYaml := strings.Contains(result, "YAML format")
				if hasYaml != tt.expected {
					t.Errorf("Expected hasYamlTags to be %v, got %v", tt.expected, hasYaml)
				}
			case SimpleStruct:
				result := GenerateStructuredPrompt[SimpleStruct]()
				hasYaml := strings.Contains(result, "YAML format")
				if hasYaml != tt.expected {
					t.Errorf("Expected hasYamlTags to be %v, got %v", tt.expected, hasYaml)
				}
			}
		})
	}
}
