package config

import (
	"errors"
	"testing"
)

func TestValidateTUI_NegativeNarrowThreshold(t *testing.T) {
	cfg := NewDefault("Test")
	cfg.TUI.NarrowThreshold = -1

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative narrow_threshold")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
	if want := "narrow_threshold"; !containsStr(err.Error(), want) {
		t.Errorf("error = %v, want to contain %q", err, want)
	}
}

func TestValidateTUI_NarrowThresholdAcceptsZeroAndPositive(t *testing.T) {
	cfg := NewDefault("Test")
	for _, v := range []int{0, 1, 40, 80, 200} {
		cfg.TUI.NarrowThreshold = v
		if err := cfg.Validate(); err != nil {
			t.Errorf("narrow_threshold %d should be valid, got %v", v, err)
		}
	}
}
