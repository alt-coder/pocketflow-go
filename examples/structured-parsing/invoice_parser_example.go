package main

import (
	"context"
	"fmt"

	"github.com/alt-coder/pocketflow-go/core"
	"github.com/alt-coder/pocketflow-go/llm"
	"github.com/alt-coder/pocketflow-go/structured"
)

// InvoiceData represents the structured output from invoice parsing
type InvoiceData struct {
	InvoiceNumber string        `yaml:"invoice_number" json:"invoice_number" description:"Invoice number or ID"`
	Date          string        `yaml:"date" json:"date" description:"Invoice date"`
	VendorName    string        `yaml:"vendor_name" json:"vendor_name" description:"Name of the vendor or company"`
	TotalAmount   float64       `yaml:"total_amount" json:"total_amount" description:"Total amount due"`
	LineItems     []InvoiceItem `yaml:"line_items" json:"line_items" description:"List of invoice line items"`
}

// InvoiceItem represents a single line item on an invoice
type InvoiceItem struct {
	Description string  `yaml:"description" json:"description" description:"Item description"`
	Quantity    int     `yaml:"quantity" json:"quantity" description:"Quantity of items"`
	UnitPrice   float64 `yaml:"unit_price" json:"unit_price" description:"Price per unit"`
	Total       float64 `yaml:"total" json:"total" description:"Total price for this line item"`
}

// InvoiceParserConfig holds configuration for invoice parsing
type InvoiceParserConfig struct {
	*structured.StructuredConfig
	Currency         string                   // Expected currency (e.g., "USD", "EUR")
	ValidationConfig *InvoiceValidationConfig // Invoice-specific validation config
}

// InvoiceParserState represents the shared state for invoice parsing workflow
type InvoiceParserState struct {
	InvoiceFilePath string                 `json:"invoice_file_path"`
	Currency        string                 `json:"currency,omitempty"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// InvoiceParserNode demonstrates how easy it is to create new structured parsing nodes
type InvoiceParserNode struct {
	*structured.StructuredNode[InvoiceData]
	config *InvoiceParserConfig
}

// NewInvoiceParserNode creates a new invoice parser node
func NewInvoiceParserNode(provider llm.LLMProvider, config *InvoiceParserConfig) (*InvoiceParserNode, error) {
	if provider == nil {
		return nil, fmt.Errorf("llm provider cannot be nil")
	}

	if config == nil {
		config = DefaultInvoiceParserConfig()
	}

	// Create an invoice-specific validator
	validator := NewInvoiceValidator(config.ValidationConfig)

	baseNode, err := structured.NewStructuredNode(provider, config.StructuredConfig, validator)
	if err != nil {
		return nil, fmt.Errorf("failed to create base node: %w", err)
	}

	return &InvoiceParserNode{
		StructuredNode: baseNode,
		config:         config,
	}, nil
}

// Prep extracts invoice file path from state
func (i *InvoiceParserNode) Prep(state *InvoiceParserState) []string {
	if state.InvoiceFilePath == "" {
		return []string{}
	}
	return []string{state.InvoiceFilePath}
}

// Exec parses the invoice using the structured framework
func (i *InvoiceParserNode) Exec(filePath string) (structured.ParseResult[InvoiceData], error) {
	ctx := context.Background()

	// Create additional context about currency expectations
	currencyContext := fmt.Sprintf("Expected currency: %s. Convert all amounts to this currency if needed.", i.config.Currency)

	// Parse from file using the structured framework - that's it!
	return i.ParseFromFile(ctx, filePath, currencyContext)
}

// Post handles the results and stores them in state
func (i *InvoiceParserNode) Post(state *InvoiceParserState, prepRes []string, execResults ...structured.ParseResult[InvoiceData]) core.Action {
	if state.Context == nil {
		state.Context = make(map[string]interface{})
	}
	for num, execResult := range execResults {
		// Display results
		if execResult.Data != nil {
			// Store result in state
			state.Context[fmt.Sprintf("%d", num)] = execResult.Data
			fmt.Println(num)
			i.displayInvoiceResults(execResult.Data)
		}
	}
	if len(state.Context)==0 {
		return core.ActionFailure
	}

	return core.ActionSuccess
}

// displayInvoiceResults shows the parsed invoice data
func (i *InvoiceParserNode) displayInvoiceResults(data *InvoiceData) {
	fmt.Println("\n=== Invoice Parsing Results ===")
	fmt.Printf("Invoice Number: %s\n", data.InvoiceNumber)
	fmt.Printf("Date: %s\n", data.Date)
	fmt.Printf("Vendor: %s\n", data.VendorName)
	fmt.Printf("Total Amount: %.2f %s\n", data.TotalAmount, i.config.Currency)

	fmt.Println("\nLine Items:")
	for i, item := range data.LineItems {
		fmt.Printf("  %d. %s (Qty: %d, Unit: %.2f, Total: %.2f)\n",
			i+1, item.Description, item.Quantity, item.UnitPrice, item.Total)
	}
	fmt.Println("===============================")
}

// ExecFallback provides fallback behavior
func (i *InvoiceParserNode) ExecFallback(err error) structured.ParseResult[InvoiceData] {
	return i.CreateFallbackResult(err)
}

// DefaultInvoiceParserConfig returns a default configuration
func DefaultInvoiceParserConfig() *InvoiceParserConfig {
	return &InvoiceParserConfig{
		StructuredConfig: structured.DefaultBaseConfig(),
		Currency:         "USD",
		ValidationConfig: DefaultInvoiceValidationConfig(),
	}
}
