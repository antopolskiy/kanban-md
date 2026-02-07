package task

import (
	"errors"
	"fmt"
)

// Validation errors.
var (
	ErrInvalidStatus   = errors.New("invalid status")
	ErrInvalidPriority = errors.New("invalid priority")
)

// ValidateStatus checks that a status is in the allowed list.
func ValidateStatus(status string, allowed []string) error {
	for _, s := range allowed {
		if s == status {
			return nil
		}
	}
	return fmt.Errorf("%w: %q (allowed: %v)", ErrInvalidStatus, status, allowed)
}

// ValidatePriority checks that a priority is in the allowed list.
func ValidatePriority(priority string, allowed []string) error {
	for _, p := range allowed {
		if p == priority {
			return nil
		}
	}
	return fmt.Errorf("%w: %q (allowed: %v)", ErrInvalidPriority, priority, allowed)
}
