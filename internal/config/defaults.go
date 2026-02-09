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
	// DefaultClass is the default class of service for new tasks.
	DefaultClass = "standard"
	// DefaultClaimTimeout is the default claim expiration as a duration string.
	DefaultClaimTimeout = "1h"
	// DefaultTitleLines is the default number of title lines in TUI cards.
	DefaultTitleLines = 1

	// ConfigFileName is the name of the config file within the kanban directory.
	ConfigFileName = "config.yml"

	// CurrentVersion is the current config schema version.
	CurrentVersion = 6

	// ArchivedStatus is the reserved status name for soft-deleted tasks.
	ArchivedStatus = "archived"
)

// Default slice values for a new board (slices cannot be const).
var (
	DefaultStatuses = []string{
		"backlog",
		"todo",
		"in-progress",
		"review",
		"done",
		ArchivedStatus,
	}

	DefaultPriorities = []string{
		"low",
		"medium",
		"high",
		"critical",
	}

	// DefaultAgeThresholds defines the default progressive color thresholds
	// for task duration display in the TUI. Tasks are colored based on how
	// long they've been in their current status.
	DefaultAgeThresholds = []AgeThreshold{
		{After: "0s", Color: "242"},   // dim gray (fresh)
		{After: "1h", Color: "34"},    // green
		{After: "24h", Color: "226"},  // yellow
		{After: "72h", Color: "208"},  // orange
		{After: "168h", Color: "196"}, // red (1 week)
	}

	// DefaultClasses defines the default classes of service.
	DefaultClasses = []ClassConfig{
		{Name: "expedite", WIPLimit: 1, BypassColumnWIP: true},
		{Name: "fixed-date"},
		{Name: "standard"},
		{Name: "intangible"},
	}
)
