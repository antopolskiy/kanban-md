package tui

import (
	"testing"
)

func TestWrapTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		maxWidth int
		maxLines int
		want     []string
	}{
		{
			name:     "short title fits one line",
			title:    "Fix bug",
			maxWidth: 20,
			maxLines: 2,
			want:     []string{"Fix bug"},
		},
		{
			name:     "single line mode truncates",
			title:    "This is a very long title that exceeds width",
			maxWidth: 15,
			maxLines: 1,
			want:     []string{"This is a ve..."},
		},
		{
			name:     "wraps at word boundary",
			title:    "Implement user authentication",
			maxWidth: 15,
			maxLines: 2,
			want:     []string{"Implement user", "authentication"},
		},
		{
			name:     "three lines",
			title:    "Add comprehensive integration test suite for the API",
			maxWidth: 15,
			maxLines: 3,
			want:     []string{"Add", "comprehensive", "integration ..."},
		},
		{
			name:     "long single word truncated",
			title:    "Supercalifragilisticexpialidocious task",
			maxWidth: 10,
			maxLines: 2,
			want:     []string{"Superca...", "task"},
		},
		{
			name:     "exact fit",
			title:    "Exact fit here",
			maxWidth: 14,
			maxLines: 2,
			want:     []string{"Exact fit here"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapTitle(tt.title, tt.maxWidth, tt.maxLines)
			if len(got) != len(tt.want) {
				t.Fatalf("wrapTitle() returned %d lines, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWrapTitle2(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		firstWidth int
		restWidth  int
		maxLines   int
		want       []string
	}{
		{
			name:       "short title fits first line",
			title:      "Fix bug",
			firstWidth: 12,
			restWidth:  20,
			maxLines:   2,
			want:       []string{"Fix bug"},
		},
		{
			name:       "continuation uses full width",
			title:      "Fix codecov showing unknown on badge",
			firstWidth: 15,
			restWidth:  22,
			maxLines:   2,
			want:       []string{"Fix codecov", "showing unknown on ..."},
		},
		{
			name:       "single line truncates at first width",
			title:      "A very long title that exceeds",
			firstWidth: 10,
			restWidth:  20,
			maxLines:   1,
			want:       []string{"A very ..."},
		},
		{
			name:       "three lines wider continuation",
			title:      "Add comprehensive integration test suite for the API",
			firstWidth: 10,
			restWidth:  20,
			maxLines:   3,
			want:       []string{"Add", "comprehensive", "integration test ..."},
		},
		{
			name:       "same width behaves like wrapTitle",
			title:      "Implement user authentication",
			firstWidth: 15,
			restWidth:  15,
			maxLines:   2,
			want:       []string{"Implement user", "authentication"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapTitle2(tt.title, tt.firstWidth, tt.restWidth, tt.maxLines)
			if len(got) != len(tt.want) {
				t.Fatalf("wrapTitle2() returned %d lines, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
