// Package clierr defines structured error types for CLI commands.
// Errors carry a machine-readable code, a human-readable message,
// and optional details for agent consumption.
package clierr

import (
	"fmt"
	"strconv"
)

// Error code constants — uppercase, underscore-separated, stable across minor versions.
const (
	TaskNotFound       = "TASK_NOT_FOUND"
	BoardNotFound      = "BOARD_NOT_FOUND"
	BoardAlreadyExists = "BOARD_ALREADY_EXISTS"
	InvalidInput       = "INVALID_INPUT"
	InvalidStatus      = "INVALID_STATUS"
	InvalidPriority    = "INVALID_PRIORITY"
	InvalidDate        = "INVALID_DATE"
	InvalidTaskID      = "INVALID_TASK_ID"
	WIPLimitExceeded   = "WIP_LIMIT_EXCEEDED"
	DependencyNotFound = "DEPENDENCY_NOT_FOUND"
	SelfReference      = "SELF_REFERENCE"
	NoChanges          = "NO_CHANGES"
	BoundaryError      = "BOUNDARY_ERROR"
	StatusConflict     = "STATUS_CONFLICT"
	ConfirmationReq    = "CONFIRMATION_REQUIRED"
	TaskClaimed        = "TASK_CLAIMED"
	InvalidClass       = "INVALID_CLASS"
	ClassWIPExceeded   = "CLASS_WIP_EXCEEDED"
	NothingToPick      = "NOTHING_TO_PICK"
	InvalidGroupBy     = "INVALID_GROUP_BY"
	InternalError      = "INTERNAL_ERROR"
)

// Error represents a structured CLI error with a machine-readable code.
type Error struct {
	Code    string
	Message string
	Details map[string]any
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Message }

// New creates an Error with the given code and message.
func New(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf creates an Error with a formatted message.
func Newf(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// WithDetails returns the error with the given details map attached.
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// ExitCode returns 2 for InternalError, 1 for all others.
func (e *Error) ExitCode() int {
	if e.Code == InternalError {
		return 2 //nolint:mnd // exit code 2 for internal errors
	}
	return 1
}

// SilentError signals an exit code without additional output.
// Used by batch operations where results are already written to stdout.
type SilentError struct {
	Code int
}

// Error implements the error interface.
func (e *SilentError) Error() string { return "exit " + strconv.Itoa(e.Code) }
