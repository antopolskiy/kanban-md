// Package config handles kanban board configuration.
package config

const (
	// DefaultDir is the default kanban directory name.
	DefaultDir = "kanban"
	// DefaultTasksDir is the default tasks subdirectory name.
	DefaultTasksDir = "tasks"
	// DefaultStatus is the default status for new tasks.
	DefaultStatus = "backlog"
	// DefaultPriority is the default priority for new tasks.
	DefaultPriority = "medium"

	// ConfigFileName is the name of the config file within the kanban directory.
	ConfigFileName = "config.yml"

	// CurrentVersion is the current config schema version.
	CurrentVersion = 2
)

// Default slice values for a new board (slices cannot be const).
var (
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
)
