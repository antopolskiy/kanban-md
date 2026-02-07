// Package task handles task files and their frontmatter.
package task

import (
	"time"

	"github.com/antopolskiy/kanban-md/internal/date"
)

// Task represents a kanban task parsed from a markdown file.
type Task struct {
	ID        int        `yaml:"id" json:"id"`
	Title     string     `yaml:"title" json:"title"`
	Status    string     `yaml:"status" json:"status"`
	Priority  string     `yaml:"priority" json:"priority"`
	Created   time.Time  `yaml:"created" json:"created"`
	Updated   time.Time  `yaml:"updated" json:"updated"`
	Assignee  string     `yaml:"assignee,omitempty" json:"assignee,omitempty"`
	Tags      []string   `yaml:"tags,omitempty" json:"tags,omitempty"`
	Due       *date.Date `yaml:"due,omitempty" json:"due,omitempty"`
	Estimate  string     `yaml:"estimate,omitempty" json:"estimate,omitempty"`
	Parent    *int       `yaml:"parent,omitempty" json:"parent,omitempty"`
	DependsOn []int      `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`

	// Body is the markdown content below the frontmatter (not in YAML).
	Body string `yaml:"-" json:"body,omitempty"`

	// File is the path to the task file (not in YAML).
	File string `yaml:"-" json:"file,omitempty"`
}
