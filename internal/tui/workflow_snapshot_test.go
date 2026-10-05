package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

func TestSnapshot_WorkflowHierarchy(t *testing.T) {
	b := setupParentChildrenBoard(t)
	assertGolden(t, "workflow_hierarchy", b.View())
}

func TestSnapshot_WorkflowDepth(t *testing.T) {
	b := setupParentChildrenBoard(t)
	b.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	b = sendKey(b, "v")
	b = sendKey(b, "v")
	assertGolden(t, "workflow_depth", b.View())
}

func TestSnapshot_WorkflowRelationPath(t *testing.T) {
	b := setupParentChildrenBoard(t)
	b.SetHideEmptyColumns(true)
	b = typeSearch(b, "#5 ")
	b = sendKey(b, "enter")
	b = sendKey(b, "enter")
	b = sendSpecialKey(b, tea.KeyTab)
	assertGolden(t, "workflow_relation_path", b.View())
}

func TestSnapshot_WorkflowRelationNarrow(t *testing.T) {
	b := setupParentChildrenBoard(t)
	b.SetHideEmptyColumns(true)
	b = typeSearch(b, "#5 ")
	b.Update(tea.WindowSizeMsg{Width: 36, Height: 20})
	b = sendKey(b, "enter")
	b = sendKey(b, "enter")
	b = sendSpecialKey(b, tea.KeyShiftTab)
	assertGolden(t, "workflow_relation_narrow", b.View())
}

func TestSnapshot_WorkflowDepthUnknown(t *testing.T) {
	b, cfg := setupTestBoard(t)
	for _, id := range []int{1, 2} {
		path, err := task.FindByID(cfg.TasksPath(), id)
		if err != nil {
			t.Fatal(err)
		}
		tk, err := task.Read(path)
		if err != nil {
			t.Fatal(err)
		}
		parent := 3 - id
		tk.Parent = &parent
		if err := task.Write(path, tk); err != nil {
			t.Fatal(err)
		}
	}
	b.Update(tui.ReloadMsg{})
	b = sendKey(b, "v")
	b = sendKey(b, "v")
	assertGolden(t, "workflow_depth_unknown", b.View())
}
