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
	Statuses     []StatusConfig `yaml:"statuses"`
	Priorities   []string       `yaml:"priorities"`
	Defaults     DefaultsConfig `yaml:"defaults"`
	WIPLimits    map[string]int `yaml:"wip_limits,omitempty"`
	ClaimTimeout string         `yaml:"claim_timeout,omitempty"`
	Classes      []ClassConfig  `yaml:"classes,omitempty"`
	TUI          TUIConfig      `yaml:"tui,omitempty"`
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

// AgeThreshold maps a duration threshold to an ANSI color code.
// Tasks older than the threshold (in their current status) render in this color.
type AgeThreshold struct {
	After string `yaml:"after" json:"after"` // duration string, e.g. "1h", "24h", "7d"
	Color string `yaml:"color" json:"color"` // ANSI 256 color code, e.g. "34", "226", "196"
}

// TUIConfig holds TUI-specific display settings.
type TUIConfig struct {
	TitleLines       int            `yaml:"title_lines,omitempty"`
	AgeThresholds    []AgeThreshold `yaml:"age_thresholds,omitempty"`
	HideEmptyColumns bool           `yaml:"hide_empty_columns,omitempty"`
}

// StatusConfig defines a status column and its enforcement rules.
type StatusConfig struct {
	Name         string `yaml:"name" json:"name"`
	RequireClaim bool   `yaml:"require_claim,omitempty" json:"require_claim,omitempty"`
	ShowDuration *bool  `yaml:"show_duration,omitempty" json:"show_duration,omitempty"`
}

// UnmarshalYAML allows StatusConfig to be parsed from either a plain string
// (old format: "backlog") or a mapping (new format: {name: backlog, require_claim: true}).
// This provides seamless backward compatibility with v6 configs.
func (s *StatusConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		s.Name = value.Value
		return nil
	}
	type plain StatusConfig
	return value.Decode((*plain)(s))
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
		Statuses:     append([]StatusConfig{}, DefaultStatuses...),
		Priorities:   append([]string{}, DefaultPriorities...),
		Classes:      append([]ClassConfig{}, DefaultClasses...),
		ClaimTimeout: DefaultClaimTimeout,
		TUI: TUIConfig{
			TitleLines:       DefaultTitleLines,
			AgeThresholds:    append([]AgeThreshold{}, DefaultAgeThresholds...),
			HideEmptyColumns: DefaultHideEmptyColumns,
		},
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

// StatusNames returns the ordered list of status name strings.
func (c *Config) StatusNames() []string {
	names := make([]string, len(c.Statuses))
	for i, s := range c.Statuses {
		names[i] = s.Name
	}
	return names
}

// StatusRequiresClaim returns true if the given status has require_claim set.
func (c *Config) StatusRequiresClaim(status string) bool {
	for _, s := range c.Statuses {
		if s.Name == status {
			return s.RequireClaim
		}
	}
	return false
}

// StatusShowDuration returns whether the given status column should display
// task age/duration. If not explicitly configured, returns true (show by default).
func (c *Config) StatusShowDuration(status string) bool {
	for _, s := range c.Statuses {
		if s.Name == status {
			if s.ShowDuration == nil {
				return true
			}
			return *s.ShowDuration
		}
	}
	return true
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
	names := c.StatusNames()
	if len(names) < 2 { //nolint:mnd // minimum 2 statuses for a kanban board
		return fmt.Errorf("%w: at least 2 statuses are required", ErrInvalid)
	}
	if hasDuplicates(names) {
		return fmt.Errorf("%w: statuses contain duplicates", ErrInvalid)
	}
	if len(c.Priorities) < 1 {
		return fmt.Errorf("%w: at least 1 priority is required", ErrInvalid)
	}
	if hasDuplicates(c.Priorities) {
		return fmt.Errorf("%w: priorities contain duplicates", ErrInvalid)
	}
	if !contains(names, c.Defaults.Status) {
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
	if err := c.validateClaimTimeout(); err != nil {
		return err
	}
	if err := c.validateTUI(); err != nil {
		return err
	}
	if c.NextID < 1 {
		return fmt.Errorf("%w: next_id must be >= 1", ErrInvalid)
	}
	return nil
}

func (c *Config) validateWIPLimits() error {
	names := c.StatusNames()
	for status, limit := range c.WIPLimits {
		if !contains(names, status) {
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

func (c *Config) validateClaimTimeout() error {
	if c.ClaimTimeout != "" {
		if _, err := time.ParseDuration(c.ClaimTimeout); err != nil {
			return fmt.Errorf("%w: invalid claim_timeout %q: %w", ErrInvalid, c.ClaimTimeout, err)
		}
	}
	return nil
}

func (c *Config) validateTUI() error {
	const minTitleLines, maxTitleLines = 1, 3
	if c.TUI.TitleLines < minTitleLines || c.TUI.TitleLines > maxTitleLines {
		return fmt.Errorf("%w: tui.title_lines must be between %d and %d",
			ErrInvalid, minTitleLines, maxTitleLines)
	}
	for i, at := range c.TUI.AgeThresholds {
		if _, err := time.ParseDuration(at.After); err != nil {
			return fmt.Errorf("%w: tui.age_thresholds[%d].after %q: %w", ErrInvalid, i, at.After, err)
		}
		if at.Color == "" {
			return fmt.Errorf("%w: tui.age_thresholds[%d].color is required", ErrInvalid, i)
		}
	}
	return nil
}

// AgeThresholdsDuration returns the age thresholds as parsed durations with color codes,
// sorted by duration ascending. Returns DefaultAgeThresholds parsed if none are configured.
func (c *Config) AgeThresholdsDuration() []struct {
	After time.Duration
	Color string
} {
	thresholds := c.TUI.AgeThresholds
	if len(thresholds) == 0 {
		thresholds = DefaultAgeThresholds
	}
	result := make([]struct {
		After time.Duration
		Color string
	}, 0, len(thresholds))
	for _, at := range thresholds {
		d, err := time.ParseDuration(at.After)
		if err != nil {
			continue
		}
		result = append(result, struct {
			After time.Duration
			Color string
		}{After: d, Color: at.Color})
	}
	return result
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

// TitleLines returns the configured number of title lines for TUI cards.
// Returns DefaultTitleLines if the value is unset (zero).
func (c *Config) TitleLines() int {
	if c.TUI.TitleLines == 0 {
		return DefaultTitleLines
	}
	return c.TUI.TitleLines
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

// Init creates a new kanban board in the given directory with default settings.
// It creates the kanban directory, tasks subdirectory, and config file.
func Init(dir, name string) (*Config, error) {
	const dirMode = 0o750

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	cfg := NewDefault(name)
	cfg.SetDir(absDir)

	if err := os.MkdirAll(cfg.TasksPath(), dirMode); err != nil {
		return nil, fmt.Errorf("creating tasks directory: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("writing config: %w", err)
	}

	return cfg, nil
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

// IsTerminalStatus returns true if the given status is a terminal status.
// Both the "done" status (immediately before archived) and "archived" itself
// are considered terminal. If the board has no archived status, the last
// status is terminal (backward-compatible behavior).
func (c *Config) IsTerminalStatus(s string) bool {
	names := c.StatusNames()
	if len(names) == 0 {
		return false
	}
	if s == ArchivedStatus {
		return true
	}
	lastIdx := len(names) - 1
	if names[lastIdx] == ArchivedStatus && lastIdx > 0 {
		return s == names[lastIdx-1]
	}
	return s == names[lastIdx]
}

// IsArchivedStatus returns true if the given status is the archived status.
func (c *Config) IsArchivedStatus(s string) bool {
	return s == ArchivedStatus && contains(c.StatusNames(), ArchivedStatus)
}

// BoardStatuses returns the statuses that should appear as board columns,
// excluding the archived status.
func (c *Config) BoardStatuses() []string {
	names := c.StatusNames()
	result := make([]string, 0, len(names))
	for _, s := range names {
		if s != ArchivedStatus {
			result = append(result, s)
		}
	}
	return result
}

// ActiveStatuses returns statuses that are neither terminal nor archived,
// i.e. statuses where work is happening. Used by pick to determine default
// candidate pools.
func (c *Config) ActiveStatuses() []string {
	names := c.StatusNames()
	result := make([]string, 0, len(names))
	for _, s := range names {
		if !c.IsTerminalStatus(s) {
			result = append(result, s)
		}
	}
	return result
}

// StatusIndex returns the index of a status in the configured order, or -1.
func (c *Config) StatusIndex(status string) int {
	return IndexOf(c.StatusNames(), status)
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
