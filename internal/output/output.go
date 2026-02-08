// Package output handles formatting CLI output as table or JSON.
package output

import (
	"os"
)

// Format represents an output format.
type Format int

const (
	// FormatAuto detects based on TTY.
	FormatAuto Format = iota
	// FormatJSON outputs JSON.
	FormatJSON
	// FormatTable outputs a human-readable table.
	FormatTable
)

// Detect returns the appropriate format based on flags and environment.
// The default is table output. Use --json or KANBAN_OUTPUT=json for JSON.
func Detect(jsonFlag, tableFlag bool) Format {
	if jsonFlag {
		return FormatJSON
	}
	if tableFlag {
		return FormatTable
	}

	// Check environment variable.
	switch os.Getenv("KANBAN_OUTPUT") {
	case "json":
		return FormatJSON
	case "table":
		return FormatTable
	}

	return FormatTable
}
