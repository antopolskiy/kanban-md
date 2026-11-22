package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestCreateFailureKeepsDraftForRetry(t *testing.T) {
	b, cfg := newWorkflowBoard(t, []*task.Task{{ID: 1, Title: "Existing task", Status: config.DefaultStatus}})
	cfg.WIPLimits = map[string]int{config.DefaultStatus: 1}
	b.handleCreateStart()
	b.createTitleInput.SetValue("Keep this draft")
	b.createBodyInput.SetValue("Keep this evidence reference")
	_, _ = b.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if b.view != viewCreate || b.createTitleInput.Value() != "Keep this draft" ||
		b.createBodyInput.Value() != "Keep this evidence reference" {
		t.Fatal("failed create discarded the editing draft")
	}
	if b.err == nil || !strings.Contains(b.View(), "WIP limit") {
		t.Fatalf("failed create did not show the error in the wizard: %v", b.err)
	}
}

func TestClaimRefusalKeepsEditDraft(t *testing.T) {
	claimed := time.Now()
	b, _ := newWorkflowBoard(t, []*task.Task{{
		ID: 1, Title: "Claimed task", Status: config.DefaultStatus, ClaimedBy: "another-agent", ClaimedAt: &claimed,
	}})
	b.handleEditStart()
	b.createTitleInput.SetValue("Keep the unsaved edit")
	_, _ = b.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if b.view != viewCreate || b.createTitleInput.Value() != "Keep the unsaved edit" {
		t.Fatal("claim refusal discarded the editing draft")
	}
	if b.err == nil || !strings.Contains(b.View(), "another-agent") {
		t.Fatalf("claim refusal did not show the error in the wizard: %v", b.err)
	}
}

func TestPartialCreateFailureDoesNotOfferDuplicateDraftRetry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	b.createTitleInput.SetValue("Partially saved task")
	path := cfg.ConfigPath()
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	imageKey(b, tea.KeyEnter)
	if b.view != viewBoard || b.selectedTask() == nil || b.selectedTask().ID != 1 {
		t.Fatal("a persisted task remained an unsaved create draft after partial failure")
	}
	if b.err == nil || !strings.Contains(b.View(), "saving config") {
		t.Fatal("partial create error was cleared while returning to the saved task")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	// The old draft must no longer be submittable as a second filename for #1.
	b.createTitleInput.SetValue("Changed after failure")
	imageKey(b, tea.KeyEnter)
	entries, err := filepath.Glob(filepath.Join(cfg.TasksPath(), "001-*.md"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("partial create retry produced duplicate IDs: %v, %v", entries, err)
	}
}
