package tui

import "testing"

func TestWrapLinesCap_NoLineLimit(t *testing.T) {
	if got := wrapLinesCap(noLineLimit); got != 8 {
		t.Errorf("wrapLinesCap(noLineLimit) = %d, want 8", got)
	}
}

func TestWrapTitle_NoLineLimitWrapsWithoutHugeAllocation(t *testing.T) {
	title := "This is a deliberately long title that should wrap into multiple lines when width is narrow"
	lines := wrapTitle(title, 20, noLineLimit)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped title to have multiple lines, got %d", len(lines))
	}
}
