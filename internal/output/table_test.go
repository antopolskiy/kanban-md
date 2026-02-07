package output

import (
	"testing"
	"time"
)

func TestFormatDuration_Days(t *testing.T) {
	d := 50 * time.Hour
	got := FormatDuration(d)
	if got != "2d 2h" {
		t.Errorf("FormatDuration(50h) = %q, want %q", got, "2d 2h")
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	d := 3*time.Hour + 30*time.Minute
	got := FormatDuration(d)
	if got != "3h 30m" {
		t.Errorf("FormatDuration(3h30m) = %q, want %q", got, "3h 30m")
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	got := FormatDuration(0)
	if got != "0h 0m" {
		t.Errorf("FormatDuration(0) = %q, want %q", got, "0h 0m")
	}
}

func TestFormatDuration_ExactDays(t *testing.T) {
	d := 48 * time.Hour
	got := FormatDuration(d)
	if got != "2d 0h" {
		t.Errorf("FormatDuration(48h) = %q, want %q", got, "2d 0h")
	}
}
