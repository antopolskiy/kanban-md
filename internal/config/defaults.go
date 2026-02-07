// Package config handles kanban board configuration.
package config

// Default values for a new board.
var (
	DefaultDir      = "kanban"
	DefaultTasksDir = "tasks"

	DefaultStatuses = []string{
		"backlog",
		"todo",
		"in-progress",
		"review",
		"done",
	}

	DefaultPriorities = []string{
		"low",
		"medium",
		"high",
		"critical",
	}

	DefaultStatus   = "backlog"
	DefaultPriority = "medium"
)

const (
	// ConfigFileName is the name of the config file within the kanban directory.
	ConfigFileName = "config.yml"

	// CurrentVersion is the current config schema version.
	CurrentVersion = 1
)
