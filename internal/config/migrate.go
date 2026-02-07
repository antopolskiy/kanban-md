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
//
// Example: when adding v1→v2 migration:
//
//	var migrations = map[int]func(*Config) error{
//	    1: migrateV1ToV2,
//	}
var migrations = map[int]func(*Config) error{}
