package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestStatusBarPrioritizesCardsHelpAndMouse(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
	})

	const actions = "create edit move +/- priority delete sort[priority↓] / search quit"
	tests := []struct {
		name         string
		mouseEnabled bool
		filter       string
		want         string
	}{
		{
			name: "keyboard only",
			want: " 4 cards | ? help | " + actions,
		},
		{
			name:         "mouse enabled",
			mouseEnabled: true,
			want:         " 4 cards | ? help | mouse | " + actions,
		},
		{
			name:         "mouse and filter",
			mouseEnabled: true,
			filter:       "bug",
			want:         ` 4 cards | ? help | mouse | filter:"bug" | ` + actions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Board{
				cfg:          config.NewDefault("Board title must not appear"),
				tasks:        make([]*task.Task, 4),
				width:        200,
				sortField:    "priority",
				sortReverse:  true,
				mouseEnabled: tt.mouseEnabled,
				filterQuery:  tt.filter,
			}

			if got := b.renderStatusBar(); got != tt.want {
				t.Fatalf("status bar:\n got: %q\nwant: %q", got, tt.want)
			}
			for _, unwanted := range []string{
				"Board title must not appear",
				"n/p",
				"c:create",
				"m:move",
			} {
				if strings.Contains(b.renderStatusBar(), unwanted) {
					t.Fatalf("status bar contains removed text %q: %q", unwanted, b.renderStatusBar())
				}
			}
		})
	}
}

func TestStatusBarKeepsHelpVisibleWhenNarrow(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
	})

	b := &Board{
		cfg:          config.NewDefault("Narrow"),
		tasks:        make([]*task.Task, 4),
		width:        20,
		sortField:    "priority",
		sortReverse:  true,
		mouseEnabled: true,
	}

	if got := b.renderStatusBar(); got != " 4 cards | ? help..." {
		t.Fatalf("narrow status bar=%q, want task count and help before truncation", got)
	}
}

func TestStatusBarHighlightsShortcutCharacters(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
	})

	b := &Board{
		cfg:         config.NewDefault("Highlights"),
		tasks:       []*task.Task{{}},
		width:       200,
		sortField:   "priority",
		sortReverse: true,
	}
	got := b.renderStatusBar()
	expectedStyle := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Foreground(lipgloss.Color("252"))

	for _, shortcut := range []string{"?", "c", "e", "m", "+/-", "d", "s", "/", "q"} {
		rendered := expectedStyle.Render(shortcut)
		if !strings.Contains(got, rendered) {
			t.Errorf("shortcut %q is not highlighted in %q", shortcut, got)
		}
	}
}
