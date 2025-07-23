package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ResumeValidationConfig holds configuration for resume-specific validation
type ResumeValidationConfig struct {
	RequireAllFields bool // Whether all fields are required
	ValidateEmail    bool // Whether to perform email format validation
}

// DefaultResumeValidationConfig returns a default validation configuration for resumes
func DefaultResumeValidationConfig() *ResumeValidationConfig {
	return &ResumeValidationConfig{
		RequireAllFields: true,
		ValidateEmail:    true,
	}
}

// ResumeValidator provides validation functionality specific to resume data
type ResumeValidator struct {
	config *ResumeValidationConfig
}

// NewResumeValidator creates a new resume validator with the specified configuration
func NewResumeValidator(config *ResumeValidationConfig) *ResumeValidator {
	if config == nil {
		config = DefaultResumeValidationConfig()
	}
	return &ResumeValidator{config: config}
}

// Validate implements structured.ValidatorInterface for ResumeData
func (v *ResumeValidator) Validate(data *ResumeData) error {
	if data == nil {
		return fmt.Errorf("resume data cannot be nil")
	}

	var validationErrors []string

	// Validate required fields
	if v.config.RequireAllFields {
		if strings.TrimSpace(data.Name) == "" {
			validationErrors = append(validationErrors, "name field is required but empty or missing")
		}

		if strings.TrimSpace(data.Email) == "" {
			validationErrors = append(validationErrors, "email field is required but empty or missing")
		}

		if data.Experience == nil {
			validationErrors = append(validationErrors, "experience field is required but missing")
		}

		if data.SkillIndexes == nil {
			validationErrors = append(validationErrors, "skill_indexes field is required but missing")
		}
	}

	// Validate email format
	if v.config.ValidateEmail && strings.TrimSpace(data.Email) != "" {
		if err := v.validateEmailFormat(data.Email); err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}

	// Validate experience entries
	if data.Experience != nil {
		for i, exp := range data.Experience {
			if strings.TrimSpace(exp.Title) == "" {
				validationErrors = append(validationErrors, fmt.Sprintf("experience[%d].title is required but empty or missing", i))
			}
			if strings.TrimSpace(exp.Company) == "" {
				validationErrors = append(validationErrors, fmt.Sprintf("experience[%d].company is required but empty or missing", i))
			}
		}
	}

	// Return combined validation errors if any exist
	if len(validationErrors) > 0 {
		return fmt.Errorf("resume validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// validateEmailFormat validates email format using regex
func (v *ResumeValidator) validateEmailFormat(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(strings.TrimSpace(email)) {
		return fmt.Errorf("email field contains invalid email format")
	}
	return nil
}

// InvoiceValidationConfig holds configuration for invoice-specific validation
type InvoiceValidationConfig struct {
	RequireAllFields bool     // Whether all fields are required
	MinAmount        float64  // Minimum valid amount
	MaxAmount        float64  // Maximum valid amount
	ValidCurrencies  []string // List of valid currencies
}

// DefaultInvoiceValidationConfig returns a default validation configuration for invoices
func DefaultInvoiceValidationConfig() *InvoiceValidationConfig {
	return &InvoiceValidationConfig{
		RequireAllFields: true,
		MinAmount:        0.01,
		MaxAmount:        1000000.00,
		ValidCurrencies:  []string{"USD", "EUR", "GBP", "CAD"},
	}
}

// InvoiceValidator provides validation functionality specific to invoice data
type InvoiceValidator struct {
	config *InvoiceValidationConfig
}

// NewInvoiceValidator creates a new invoice validator with the specified configuration
func NewInvoiceValidator(config *InvoiceValidationConfig) *InvoiceValidator {
	if config == nil {
		config = DefaultInvoiceValidationConfig()
	}
	return &InvoiceValidator{config: config}
}

// Validate implements structured.ValidatorInterface for InvoiceData
func (v *InvoiceValidator) Validate(data *InvoiceData) error {
	if data == nil {
		return fmt.Errorf("invoice data cannot be nil")
	}

	var validationErrors []string

	// Validate required fields
	if v.config.RequireAllFields {
		if strings.TrimSpace(data.InvoiceNumber) == "" {
			validationErrors = append(validationErrors, "invoice_number field is required but empty or missing")
		}

		if strings.TrimSpace(data.VendorName) == "" {
			validationErrors = append(validationErrors, "vendor_name field is required but empty or missing")
		}

		if strings.TrimSpace(data.Date) == "" {
			validationErrors = append(validationErrors, "date field is required but empty or missing")
		}
	}

	// Validate amount ranges
	if data.TotalAmount < v.config.MinAmount {
		validationErrors = append(validationErrors, fmt.Sprintf("total_amount %.2f is below minimum %.2f", data.TotalAmount, v.config.MinAmount))
	}

	if data.TotalAmount > v.config.MaxAmount {
		validationErrors = append(validationErrors, fmt.Sprintf("total_amount %.2f exceeds maximum %.2f", data.TotalAmount, v.config.MaxAmount))
	}

	// Validate line items
	if data.LineItems != nil {
		for i, item := range data.LineItems {
			if strings.TrimSpace(item.Description) == "" {
				validationErrors = append(validationErrors, fmt.Sprintf("line_items[%d].description is required but empty or missing", i))
			}
			if item.Quantity <= 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("line_items[%d].quantity must be greater than 0", i))
			}
			if item.UnitPrice < 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("line_items[%d].unit_price cannot be negative", i))
			}
		}
	}

	// Return combined validation errors if any exist
	if len(validationErrors) > 0 {
		return fmt.Errorf("invoice validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}