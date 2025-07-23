package structured

import (
	"fmt"
)

// ValidatorInterface defines the contract for data validation
type ValidatorInterface[T any] interface {
	Validate(data *T) error
}

// NoOpValidator provides a validator that does no validation (always passes)
type NoOpValidator[T any] struct{}

// NewNoOpValidator creates a validator that performs no validation
func NewNoOpValidator[T any]() *NoOpValidator[T] {
	return &NoOpValidator[T]{}
}

// Validate implements ValidatorInterface but performs no validation
func (v *NoOpValidator[T]) Validate(data *T) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	return nil
}


