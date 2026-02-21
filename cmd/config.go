package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or modify board configuration",
	Long:  `View the full configuration, get a specific key, or set a writable value.`,
	RunE:  runConfigShow,
}

var configGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2), //nolint:mnd // key and value
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

// configAccessor describes how to get and set a config key.
type configAccessor struct {
	get      func(*config.Config) any
	set      func(*config.Config, string) error
	writable bool
}

func configAccessors() map[string]configAccessor {
	accessors := baseConfigAccessors()
	addExtendedConfigAccessors(accessors)
	return accessors
}

func baseConfigAccessors() map[string]configAccessor {
	return map[string]configAccessor{
		"board.name": {
			get:      func(c *config.Config) any { return c.Board.Name },
			set:      func(c *config.Config, v string) error { c.Board.Name = v; return nil },
			writable: true,
		},
		"board.description": {
			get:      func(c *config.Config) any { return c.Board.Description },
			set:      func(c *config.Config, v string) error { c.Board.Description = v; return nil },
			writable: true,
		},
		"statuses": {
			get: func(c *config.Config) any { return c.StatusNames() },
		},
		"priorities": {
			get: func(c *config.Config) any { return c.Priorities },
		},
		"defaults.status": {
			get: func(c *config.Config) any { return c.Defaults.Status },
			set: func(c *config.Config, v string) error {
				if config.IndexOf(c.StatusNames(), v) < 0 {
					return clierr.Newf(clierr.InvalidInput,
						"invalid default status %q; allowed: %s", v, strings.Join(c.StatusNames(), ", "))
				}
				c.Defaults.Status = v
				return nil
			},
			writable: true,
		},
		"defaults.priority": {
			get: func(c *config.Config) any { return c.Defaults.Priority },
			set: func(c *config.Config, v string) error {
				if config.IndexOf(c.Priorities, v) < 0 {
					return clierr.Newf(clierr.InvalidInput,
						"invalid default priority %q; allowed: %s", v, strings.Join(c.Priorities, ", "))
				}
				c.Defaults.Priority = v
				return nil
			},
			writable: true,
		},
		"tasks_dir": {
			get: func(c *config.Config) any { return c.TasksDir },
		},
		"next_id": {
			get: func(c *config.Config) any { return c.NextID },
		},
		"version": {
			get: func(c *config.Config) any { return c.Version },
		},
		"wip_limits": {
			get: func(c *config.Config) any {
				if c.WIPLimits == nil {
					return map[string]int{}
				}
				return c.WIPLimits
			},
		},
	}
}

func addExtendedConfigAccessors(accessors map[string]configAccessor) {
	accessors["defaults.class"] = configAccessor{
		get: func(c *config.Config) any { return c.Defaults.Class },
		set: func(c *config.Config, v string) error {
			if v == "" {
				c.Defaults.Class = ""
				return nil
			}
			if c.ClassByName(v) == nil {
				return clierr.Newf(clierr.InvalidInput,
					"invalid default class %q; allowed: %s", v, strings.Join(c.ClassNames(), ", "))
			}
			c.Defaults.Class = v
			return nil
		},
		writable: true,
	}
	accessors["claim_timeout"] = configAccessor{
		get: func(c *config.Config) any { return c.ClaimTimeout },
		set: func(c *config.Config, v string) error {
			if _, err := time.ParseDuration(v); err != nil {
				return clierr.Newf(clierr.InvalidInput,
					"invalid claim_timeout %q: %v", v, err)
			}
			c.ClaimTimeout = v
			return nil
		},
		writable: true,
	}
	accessors["classes"] = configAccessor{
		get: func(c *config.Config) any { return c.Classes },
	}
	accessors["tui.title_lines"] = configAccessor{
		get: func(c *config.Config) any { return c.TUI.TitleLines },
		set: func(c *config.Config, v string) error {
			n, err := strconv.Atoi(v)
			if err != nil {
				return clierr.Newf(clierr.InvalidInput,
					"invalid tui.title_lines %q: must be an integer", v)
			}
			c.TUI.TitleLines = n
			return nil // validation handles range check
		},
		writable: true,
	}
	accessors["tui.hide_empty_columns"] = configAccessor{
		get: func(c *config.Config) any { return c.TUI.HideEmptyColumns },
		set: func(c *config.Config, v string) error {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return clierr.Newf(clierr.InvalidInput,
					"invalid tui.hide_empty_columns %q: must be true or false", v)
			}
			c.TUI.HideEmptyColumns = b
			return nil
		},
		writable: true,
	}
	accessors["tui.age_thresholds"] = configAccessor{
		get: func(c *config.Config) any { return c.TUI.AgeThresholds },
	}
}

// allConfigKeys returns config keys in display order.
func allConfigKeys() []string {
	return []string{
		"version",
		"board.name",
		"board.description",
		"tasks_dir",
		"statuses",
		"priorities",
		"defaults.status",
		"defaults.priority",
		"defaults.class",
		"wip_limits",
		"claim_timeout",
		"classes",
		"tui.title_lines",
		"tui.hide_empty_columns",
		"tui.age_thresholds",
		"next_id",
	}
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	accessors := configAccessors()

	if outputFormat() == output.FormatJSON {
		m := make(map[string]any, len(accessors))
		for _, key := range allConfigKeys() {
			m[key] = accessors[key].get(cfg)
		}
		return output.JSON(os.Stdout, m)
	}

	// Table mode: key-value pairs.
	for _, key := range allConfigKeys() {
		val := accessors[key].get(cfg)
		fmt.Fprintf(os.Stdout, "%-20s %v\n", key, formatConfigValue(val))
	}
	return nil
}

func runConfigGet(_ *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	key := args[0]
	accessors := configAccessors()
	acc, ok := accessors[key]
	if !ok {
		return clierr.Newf(clierr.InvalidInput, "unknown config key %q", key)
	}

	val := acc.get(cfg)

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, val)
	}

	fmt.Fprintln(os.Stdout, formatConfigValue(val))
	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	key, value := args[0], args[1]
	accessors := configAccessors()
	acc, ok := accessors[key]
	if !ok {
		return clierr.Newf(clierr.InvalidInput, "unknown config key %q", key)
	}
	if !acc.writable {
		return clierr.Newf(clierr.InvalidInput, "config key %q is read-only", key)
	}

	if err := acc.set(cfg, value); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, map[string]any{"key": key, "value": acc.get(cfg)})
	}

	output.Messagef(os.Stdout, "Set %s = %v", key, formatConfigValue(acc.get(cfg)))
	return nil
}

func formatConfigValue(val any) string {
	switch v := val.(type) {
	case []string:
		return strings.Join(v, ", ")
	case map[string]int:
		if len(v) == 0 {
			return "--"
		}
		parts := make([]string, 0, len(v))
		for k, n := range v {
			parts = append(parts, fmt.Sprintf("%s=%d", k, n))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}
