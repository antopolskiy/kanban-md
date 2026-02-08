// Package output handles formatting CLI output as table, JSON, or compact.
package output

import (
	"os"
)

// Format represents an output format.
type Format int

const (
	// FormatAuto uses the default format (table).
	FormatAuto Format = iota
	// FormatJSON outputs JSON.
	FormatJSON
	// FormatTable outputs a human-readable table.
	FormatTable
	// FormatCompact outputs one-line-per-record compact format.
	FormatCompact
)

// Detect returns the appropriate format based on flags and environment.
// Default is table when no explicit format is set.
func Detect(jsonFlag, tableFlag, compactFlag bool) Format {
	if jsonFlag {
		return FormatJSON
	}
	if compactFlag {
		return FormatCompact
	}
	if tableFlag {
		return FormatTable
	}

	// Check environment variable.
	switch os.Getenv("KANBAN_OUTPUT") {
	case "json":
		return FormatJSON
	case "compact", "oneline":
		return FormatCompact
	case "table":
		return FormatTable
	}

	// Default: table.
	return FormatTable
}
