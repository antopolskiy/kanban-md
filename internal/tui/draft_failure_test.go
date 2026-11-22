package tui

import (
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
