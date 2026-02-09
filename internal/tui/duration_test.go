package tui

import (
	"testing"
	"time"
)

func TestHumanDuration(t *testing.T) {
	const (
		day  = 24 * time.Hour
		week = 7 * day
	)

	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "<1m"},
		{30 * time.Second, "<1m"},
		{time.Minute, "1m"},
		{5 * time.Minute, "5m"},
		{59 * time.Minute, "59m"},
		{time.Hour, "1h"},
		{2 * time.Hour, "2h"},
		{23 * time.Hour, "23h"},
		{day, "1d"},
		{3 * day, "3d"},
		{6 * day, "6d"},
		{week, "1w"},
		{2 * week, "2w"},
		{29 * day, "4w"},
		{30 * day, "1mo"},
		{60 * day, "2mo"},
		{364 * day, "12mo"},
		{365 * day, "1y"},
		{730 * day, "2y"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := humanDuration(tt.d)
			if got != tt.want {
				t.Errorf("humanDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
