package config

import (
	"errors"
	"testing"
)

func TestMigrateCurrentVersionNoop(t *testing.T) {
	cfg := NewDefault("Test")
	if err := migrate(cfg); err != nil {
		t.Errorf("migrate() current version: %v", err)
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
}

func TestMigrateNewerVersionErrors(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = CurrentVersion + 1

	err := migrate(cfg)
	if err == nil {
		t.Fatal("migrate() newer version: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("migrate() error = %v, want ErrInvalid", err)
	}
}

func TestMigrateZeroVersionErrors(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = 0

	err := migrate(cfg)
	if err == nil {
		t.Fatal("migrate() version 0: expected error, got nil")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("migrate() error = %v, want ErrInvalid", err)
	}
}

func TestMigrateNegativeVersionErrors(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.Version = -1

	err := migrate(cfg)
	if err == nil {
		t.Fatal("migrate() negative version: expected error, got nil")
	}
}
