package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const fileMode = 0o600

// Sentinel errors.
var (
	ErrNotFound = errors.New("no kanban board found (run 'kanban-md init' to create one)")
	ErrInvalid  = errors.New("invalid config")
)

// Config represents the kanban board configuration.
type Config struct {
	Version    int            `yaml:"version"`
	Board      BoardConfig    `yaml:"board"`
	TasksDir   string         `yaml:"tasks_dir"`
	Statuses   []string       `yaml:"statuses"`
	Priorities []string       `yaml:"priorities"`
	Defaults   DefaultsConfig `yaml:"defaults"`
	NextID     int            `yaml:"next_id"`

	// dir is the absolute path to the kanban directory (not serialized).
	dir string `yaml:"-"`
}

// BoardConfig holds board metadata.
type BoardConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// DefaultsConfig holds default values for new tasks.
type DefaultsConfig struct {
	Status   string `yaml:"status"`
	Priority string `yaml:"priority"`
}

// Dir returns the absolute path to the kanban directory.
func (c *Config) Dir() string {
	return c.dir
}

// TasksPath returns the absolute path to the tasks directory.
func (c *Config) TasksPath() string {
	return filepath.Join(c.dir, c.TasksDir)
}

// ConfigPath returns the absolute path to the config file.
func (c *Config) ConfigPath() string {
	return filepath.Join(c.dir, ConfigFileName)
}

// NewDefault creates a Config with default values.
func NewDefault(name string) *Config {
	return &Config{
		Version:    CurrentVersion,
		Board:      BoardConfig{Name: name},
		TasksDir:   DefaultTasksDir,
		Statuses:   append([]string{}, DefaultStatuses...),
		Priorities: append([]string{}, DefaultPriorities...),
		Defaults: DefaultsConfig{
			Status:   DefaultStatus,
			Priority: DefaultPriority,
		},
		NextID: 1,
	}
}

// SetDir sets the kanban directory path on the config.
func (c *Config) SetDir(dir string) {
	c.dir = dir
}

// Validate checks the config for errors.
func (c *Config) Validate() error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("%w: unsupported version %d (expected %d)", ErrInvalid, c.Version, CurrentVersion)
	}
	if c.Board.Name == "" {
		return fmt.Errorf("%w: board.name is required", ErrInvalid)
	}
	if c.TasksDir == "" {
		return fmt.Errorf("%w: tasks_dir is required", ErrInvalid)
	}
	if len(c.Statuses) < 2 { //nolint:mnd // minimum 2 statuses for a kanban board
		return fmt.Errorf("%w: at least 2 statuses are required", ErrInvalid)
	}
	if hasDuplicates(c.Statuses) {
		return fmt.Errorf("%w: statuses contain duplicates", ErrInvalid)
	}
	if len(c.Priorities) < 1 {
		return fmt.Errorf("%w: at least 1 priority is required", ErrInvalid)
	}
	if hasDuplicates(c.Priorities) {
		return fmt.Errorf("%w: priorities contain duplicates", ErrInvalid)
	}
	if !contains(c.Statuses, c.Defaults.Status) {
		return fmt.Errorf("%w: default status %q not in statuses list", ErrInvalid, c.Defaults.Status)
	}
	if !contains(c.Priorities, c.Defaults.Priority) {
		return fmt.Errorf("%w: default priority %q not in priorities list", ErrInvalid, c.Defaults.Priority)
	}
	if c.NextID < 1 {
		return fmt.Errorf("%w: next_id must be >= 1", ErrInvalid)
	}
	return nil
}

// Save writes the config to its config file.
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(c.ConfigPath(), data, fileMode)
}

// Load reads and validates a config from the given kanban directory.
func Load(dir string) (*Config, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	path := filepath.Join(absDir, ConfigFileName)
	data, err := os.ReadFile(path) //nolint:gosec // config path from trusted source
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.dir = absDir

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// FindDir walks upward from startDir looking for a kanban directory
// containing config.yml. Returns the absolute path to the kanban directory.
func FindDir(startDir string) (string, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	dir := absStart
	for {
		candidate := filepath.Join(dir, DefaultDir, ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, DefaultDir), nil
		}

		// Also check if we're inside the kanban directory itself.
		candidate = filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// StatusIndex returns the index of a status in the configured order, or -1.
func (c *Config) StatusIndex(status string) int {
	return indexOf(c.Statuses, status)
}

// PriorityIndex returns the index of a priority in the configured order, or -1.
func (c *Config) PriorityIndex(priority string) int {
	return indexOf(c.Priorities, priority)
}

func contains(slice []string, item string) bool {
	return indexOf(slice, item) >= 0
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func hasDuplicates(slice []string) bool {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		if seen[s] {
			return true
		}
		seen[s] = true
	}
	return false
}
