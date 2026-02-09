package config

import "fmt"

// migrate upgrades a config from its current version to CurrentVersion.
// Each migration function transforms the config one version forward.
// Returns nil if no migration is needed (already at current version).
// Returns an error if the config version is newer than what this binary supports.
func migrate(cfg *Config) error {
	if cfg.Version == CurrentVersion {
		return nil
	}
	if cfg.Version > CurrentVersion {
		return fmt.Errorf(
			"%w: config version %d is newer than supported version %d (upgrade kanban-md)",
			ErrInvalid, cfg.Version, CurrentVersion,
		)
	}
	if cfg.Version < 1 {
		return fmt.Errorf("%w: config version %d is invalid", ErrInvalid, cfg.Version)
	}

	// Apply migrations sequentially: v1→v2, v2→v3, etc.
	for cfg.Version < CurrentVersion {
		fn, ok := migrations[cfg.Version]
		if !ok {
			return fmt.Errorf("%w: no migration path from version %d", ErrInvalid, cfg.Version)
		}
		if err := fn(cfg); err != nil {
			return fmt.Errorf("migrating config from v%d: %w", cfg.Version, err)
		}
	}

	return nil
}

// migrations maps each version to the function that migrates it to the next version.
// The migration function must increment cfg.Version after a successful migration.
var migrations = map[int]func(*Config) error{
	1: migrateV1ToV2,
	2: migrateV2ToV3,
	3: migrateV3ToV4,
	4: migrateV4ToV5,
	5: migrateV5ToV6,
}

// migrateV1ToV2 adds the wip_limits field (defaults to nil/empty = unlimited).
func migrateV1ToV2(cfg *Config) error { //nolint:unparam // signature must match migrations map type
	cfg.Version = 2
	return nil
}

// migrateV2ToV3 adds claim_timeout, classes of service, and defaults.class.
func migrateV2ToV3(cfg *Config) error { //nolint:unparam // signature must match migrations map type
	if cfg.ClaimTimeout == "" {
		cfg.ClaimTimeout = DefaultClaimTimeout
	}
	if len(cfg.Classes) == 0 {
		cfg.Classes = append([]ClassConfig{}, DefaultClasses...)
	}
	if cfg.Defaults.Class == "" {
		cfg.Defaults.Class = DefaultClass
	}
	cfg.Version = 3
	return nil
}

// migrateV3ToV4 adds the tui section with title_lines default.
func migrateV3ToV4(cfg *Config) error { //nolint:unparam // signature must match migrations map type
	if cfg.TUI.TitleLines == 0 {
		cfg.TUI.TitleLines = DefaultTitleLines
	}
	cfg.Version = 4
	return nil
}

// migrateV4ToV5 adds the tui.age_thresholds default.
func migrateV4ToV5(cfg *Config) error { //nolint:unparam // signature must match migrations map type
	if len(cfg.TUI.AgeThresholds) == 0 {
		cfg.TUI.AgeThresholds = append([]AgeThreshold{}, DefaultAgeThresholds...)
	}
	cfg.Version = 5
	return nil
}

// migrateV5ToV6 adds the "archived" status for soft-delete support.
func migrateV5ToV6(cfg *Config) error { //nolint:unparam // signature must match migrations map type
	if !contains(cfg.Statuses, ArchivedStatus) {
		cfg.Statuses = append(cfg.Statuses, ArchivedStatus)
	}
	cfg.Version = 6
	return nil
}
