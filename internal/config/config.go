package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/clierr"
)

const fileMode = 0o600

// Sentinel errors.
var (
	ErrNotFound = errors.New("no kanban board found (run 'kanban-md init' to create one)")
	ErrInvalid  = errors.New("invalid config")
)

// Config represents the kanban board configuration.
type Config struct {
	Version      int            `yaml:"version"`
	Board        BoardConfig    `yaml:"board"`
	TasksDir     string         `yaml:"tasks_dir"`
	Statuses     []string       `yaml:"statuses"`
	Priorities   []string       `yaml:"priorities"`
	Defaults     DefaultsConfig `yaml:"defaults"`
	WIPLimits    map[string]int `yaml:"wip_limits,omitempty"`
	ClaimTimeout string         `yaml:"claim_timeout,omitempty"`
	Classes      []ClassConfig  `yaml:"classes,omitempty"`
	NextID       int            `yaml:"next_id"`

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
	Class    string `yaml:"class,omitempty"`
}

// ClassConfig defines a class of service and its WIP rules.
type ClassConfig struct {
	Name            string `yaml:"name" json:"name"`
	WIPLimit        int    `yaml:"wip_limit,omitempty" json:"wip_limit,omitempty"`
	BypassColumnWIP bool   `yaml:"bypass_column_wip,omitempty" json:"bypass_column_wip,omitempty"`
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
		Version:      CurrentVersion,
		Board:        BoardConfig{Name: name},
		TasksDir:     DefaultTasksDir,
		Statuses:     append([]string{}, DefaultStatuses...),
		Priorities:   append([]string{}, DefaultPriorities...),
		Classes:      append([]ClassConfig{}, DefaultClasses...),
		ClaimTimeout: DefaultClaimTimeout,
		Defaults: DefaultsConfig{
			Status:   DefaultStatus,
			Priority: DefaultPriority,
			Class:    DefaultClass,
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
	if err := c.validateWIPLimits(); err != nil {
		return err
	}
	if err := c.validateClasses(); err != nil {
		return err
	}
	if c.ClaimTimeout != "" {
		if _, err := time.ParseDuration(c.ClaimTimeout); err != nil {
			return fmt.Errorf("%w: invalid claim_timeout %q: %w", ErrInvalid, c.ClaimTimeout, err)
		}
	}
	if c.NextID < 1 {
		return fmt.Errorf("%w: next_id must be >= 1", ErrInvalid)
	}
	return nil
}

func (c *Config) validateWIPLimits() error {
	for status, limit := range c.WIPLimits {
		if !contains(c.Statuses, status) {
			return fmt.Errorf("%w: wip_limits references unknown status %q", ErrInvalid, status)
		}
		if limit < 0 {
			return fmt.Errorf("%w: wip_limits for %q must be >= 0", ErrInvalid, status)
		}
	}
	return nil
}

func (c *Config) validateClasses() error {
	if len(c.Classes) == 0 {
		return nil // classes are optional
	}
	seen := make(map[string]bool, len(c.Classes))
	for _, cl := range c.Classes {
		if cl.Name == "" {
			return fmt.Errorf("%w: class name is required", ErrInvalid)
		}
		if seen[cl.Name] {
			return fmt.Errorf("%w: duplicate class name %q", ErrInvalid, cl.Name)
		}
		seen[cl.Name] = true
		if cl.WIPLimit < 0 {
			return fmt.Errorf("%w: class %q wip_limit must be >= 0", ErrInvalid, cl.Name)
		}
	}
	if c.Defaults.Class != "" && !seen[c.Defaults.Class] {
		return fmt.Errorf("%w: default class %q not in classes list", ErrInvalid, c.Defaults.Class)
	}
	return nil
}

// WIPLimit returns the WIP limit for a status, or 0 (unlimited).
func (c *Config) WIPLimit(status string) int {
	if c.WIPLimits == nil {
		return 0
	}
	return c.WIPLimits[status]
}

// ClaimTimeoutDuration parses the claim_timeout string into a time.Duration.
// Returns 0 (no expiry) if the field is empty or unparseable.
func (c *Config) ClaimTimeoutDuration() time.Duration {
	if c.ClaimTimeout == "" {
		return 0
	}
	d, err := time.ParseDuration(c.ClaimTimeout)
	if err != nil {
		return 0
	}
	return d
}

// ClassByName returns the ClassConfig for the given name, or nil if not found.
func (c *Config) ClassByName(name string) *ClassConfig {
	for i := range c.Classes {
		if c.Classes[i].Name == name {
			return &c.Classes[i]
		}
	}
	return nil
}

// ClassNames returns the list of configured class names in order.
func (c *Config) ClassNames() []string {
	names := make([]string, len(c.Classes))
	for i, cl := range c.Classes {
		names[i] = cl.Name
	}
	return names
}

// ClassIndex returns the index of a class name in the configured order, or -1.
func (c *Config) ClassIndex(class string) int {
	for i, cl := range c.Classes {
		if cl.Name == class {
			return i
		}
	}
	return -1
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

	// Migrate old config versions forward before validating.
	oldVersion := cfg.Version
	if err := migrate(&cfg); err != nil {
		return nil, err
	}

	// Persist migrated config so future loads skip re-migration.
	if cfg.Version != oldVersion {
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("saving migrated config: %w", err)
		}
	}

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
			return "", clierr.New(clierr.BoardNotFound,
				"no kanban board found (run 'kanban-md init' to create one)")
		}
		dir = parent
	}
}

// IsTerminalStatus returns true if the given status is the last in the configured order.
func (c *Config) IsTerminalStatus(s string) bool {
	if len(c.Statuses) == 0 {
		return false
	}
	return s == c.Statuses[len(c.Statuses)-1]
}

// StatusIndex returns the index of a status in the configured order, or -1.
func (c *Config) StatusIndex(status string) int {
	return IndexOf(c.Statuses, status)
}

// PriorityIndex returns the index of a priority in the configured order, or -1.
func (c *Config) PriorityIndex(priority string) int {
	return IndexOf(c.Priorities, priority)
}

func contains(slice []string, item string) bool {
	return IndexOf(slice, item) >= 0
}

// IndexOf returns the index of item in slice, or -1 if not found.
func IndexOf(slice []string, item string) int {
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
