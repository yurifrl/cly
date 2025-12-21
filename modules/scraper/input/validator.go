package input

import (
	"fmt"
	"regexp"
)

// Validator validates product IDs
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateID validates a single product ID
func (v *Validator) ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("product ID cannot be empty")
	}

	// AliExpress product IDs are typically 13-16 digits
	if !regexp.MustCompile(`^\d{10,20}$`).MatchString(id) {
		return fmt.Errorf("invalid product ID format: %s (expected 10-20 digits)", id)
	}

	return nil
}

// ValidateIDs validates multiple product IDs
func (v *Validator) ValidateIDs(ids []string) ([]string, []error) {
	var validIDs []string
	var errors []error

	for _, id := range ids {
		if err := v.ValidateID(id); err != nil {
			errors = append(errors, err)
		} else {
			validIDs = append(validIDs, id)
		}
	}

	return validIDs, errors
}
