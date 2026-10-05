package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
)

func TestPriorityRefusalRestoresUpdatedAndPriority(t *testing.T) {
	cfg := config.NewDefault("Refused priority")
	cfg.SetDir(t.TempDir())
	if err := os.MkdirAll(cfg.TasksPath(), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.TasksPath(), "001-generic-sample.md")
	content := []byte("---\nid: 1\ntitle: Generic sample\nstatus: backlog\npriority: medium\ncreated: 2026-08-12T10:00:00Z\nupdated: 2026-08-12T10:00:00Z\nestimate: &shared 4h\ncustom_copy: *shared\n---\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	b := NewBoard(cfg)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	selected := b.selectedTask()
	if selected == nil {
		t.Fatal("task with unsafe metadata was hidden")
	}
	oldPriority, oldUpdated := selected.Priority, selected.Updated
	b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if selected.Priority != oldPriority || !selected.Updated.Equal(oldUpdated) {
		t.Errorf("refusal left Priority=%q Updated=%s, want %q %s", selected.Priority, selected.Updated.Format(time.RFC3339Nano), oldPriority, oldUpdated.Format(time.RFC3339Nano))
	}
	if b.err == nil || !strings.Contains(b.err.Error(), "cannot preserve") || !strings.Contains(b.View(), "cannot preserve") {
		t.Errorf("missing preservation error feedback: %v", b.err)
	}
	written, err := os.ReadFile(path) //nolint:gosec // test-owned task path
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, content) {
		t.Errorf("refused priority edit changed task:\n%s", written)
	}
	if _, err = os.Stat(filepath.Join(cfg.Dir(), "activity.jsonl")); !os.IsNotExist(err) {
		t.Errorf("refused priority edit wrote activity log: %v", err)
	}
}
