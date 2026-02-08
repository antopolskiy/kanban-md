package task

import (
	"github.com/antopolskiy/kanban-md/internal/clierr"
)

// ValidateStatus checks that a status is in the allowed list.
func ValidateStatus(status string, allowed []string) error {
	for _, s := range allowed {
		if s == status {
			return nil
		}
	}
	return clierr.Newf(clierr.InvalidStatus, "invalid status %q", status).
		WithDetails(map[string]any{
			"status":  status,
			"allowed": allowed,
		})
}

// ValidatePriority checks that a priority is in the allowed list.
func ValidatePriority(priority string, allowed []string) error {
	for _, p := range allowed {
		if p == priority {
			return nil
		}
	}
	return clierr.Newf(clierr.InvalidPriority, "invalid priority %q", priority).
		WithDetails(map[string]any{
			"priority": priority,
			"allowed":  allowed,
		})
}

// ValidateDate returns a CLIError for invalid date input.
func ValidateDate(field, input string, err error) *clierr.Error {
	return clierr.Newf(clierr.InvalidDate, "invalid %s date: %v", field, err).
		WithDetails(map[string]any{
			"field": field,
			"input": input,
		})
}

// ValidateTaskID returns a CLIError for invalid task ID input.
func ValidateTaskID(input string) *clierr.Error {
	return clierr.Newf(clierr.InvalidTaskID, "invalid task ID %q", input).
		WithDetails(map[string]any{"input": input})
}

// ValidateSelfReference returns a CLIError for self-referencing dependency.
func ValidateSelfReference(id int) *clierr.Error {
	return clierr.Newf(clierr.SelfReference, "task cannot depend on itself (ID %d)", id).
		WithDetails(map[string]any{"id": id})
}

// ValidateDependencyNotFound returns a CLIError for missing dependency.
func ValidateDependencyNotFound(depID int) *clierr.Error {
	return clierr.Newf(clierr.DependencyNotFound, "dependency task #%d not found", depID).
		WithDetails(map[string]any{"id": depID})
}

// ValidateWIPLimit returns a CLIError for WIP limit violations.
func ValidateWIPLimit(status string, limit, current int) *clierr.Error {
	return clierr.Newf(clierr.WIPLimitExceeded,
		"WIP limit reached for %q (%d/%d)", status, current, limit).
		WithDetails(map[string]any{
			"status":  status,
			"limit":   limit,
			"current": current,
		})
}

// ValidateBoundaryError returns a CLIError for boundary moves.
func ValidateBoundaryError(id int, status, direction string) *clierr.Error {
	return clierr.Newf(clierr.BoundaryError,
		"task #%d is already at the %s status (%s)", id, direction, status).
		WithDetails(map[string]any{
			"id":        id,
			"status":    status,
			"direction": direction,
		})
}

// ValidateClass checks that a class is in the allowed list.
func ValidateClass(class string, allowed []string) error {
	for _, c := range allowed {
		if c == class {
			return nil
		}
	}
	return clierr.Newf(clierr.InvalidClass, "invalid class %q", class).
		WithDetails(map[string]any{
			"class":   class,
			"allowed": allowed,
		})
}

// ValidateTaskClaimed returns a CLIError when a task is claimed by another agent.
func ValidateTaskClaimed(id int, claimedBy, remaining string) *clierr.Error {
	return clierr.Newf(clierr.TaskClaimed,
		"task #%d is claimed by %q (expires in %s). Use --force to override",
		id, claimedBy, remaining).
		WithDetails(map[string]any{
			"id":         id,
			"claimed_by": claimedBy,
			"remaining":  remaining,
		})
}

// ValidateClassWIPExceeded returns a CLIError for class-level WIP limit violations.
func ValidateClassWIPExceeded(class string, limit, current int) *clierr.Error {
	return clierr.Newf(clierr.ClassWIPExceeded,
		"%s WIP limit reached (%d/%d board-wide)", class, current, limit).
		WithDetails(map[string]any{
			"class":   class,
			"limit":   limit,
			"current": current,
		})
}

// FormatDueDate returns a CLIError for invalid due date input.
func FormatDueDate(input string, err error) *clierr.Error {
	return ValidateDate("due", input, err)
}
