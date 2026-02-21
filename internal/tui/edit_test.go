package tui_test

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestEdit_DialogOpensAndCloses(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "e")
	v := b.View()
	if !containsStr(v, "Edit task #1") {
		t.Fatalf("expected edit dialog, got:\n%s", v)
	}
	if !containsStr(v, "Step 1/4: Title") {
		t.Fatal("expected title step in edit dialog")
	}
	if !containsStr(v, "enter:save") {
		t.Fatal("expected save hint in edit dialog")
	}

	b = sendSpecialKey(b, tea.KeyEscape)
	if containsStr(b.View(), "Edit task #1") {
		t.Error("expected board view after esc")
	}
}

func TestEdit_StatusBarShowsEditHint(t *testing.T) {
	b, _ := setupTestBoard(t)
	if !containsStr(b.View(), "e:edit") {
		t.Fatal("expected edit hint in status bar")
	}
}

func TestEdit_PrefillsExistingFields(t *testing.T) {
	b, cfg := setupTestBoard(t)

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	tk.Title = "Editable task"
	tk.Body = "Original body"
	tk.Priority = "critical"
	tk.Tags = []string{"alpha", "beta"}
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}

	b = sendKey(b, "r")
	b = sendKey(b, "e")
	v := b.View()
	if !containsStr(v, "Editable task") {
		t.Fatal("expected prefilled title")
	}

	b = sendSpecialKey(b, tea.KeyTab)
	v = b.View()
	if !containsStr(v, "Original body") {
		t.Fatal("expected prefilled body")
	}

	b = sendSpecialKey(b, tea.KeyTab)
	v = b.View()
	if !containsStr(v, "> critical") {
		t.Fatalf("expected critical priority selected, got:\n%s", v)
	}

	b = sendSpecialKey(b, tea.KeyTab)
	v = b.View()
	if !containsStr(v, "alpha,beta") {
		t.Fatal("expected prefilled tags")
	}
}

func TestEdit_EnterSavesFromTitleStep(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "e")
	b = typeText(b, " updated")
	_ = sendSpecialKey(b, tea.KeyEnter)

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Title != "Task A updated" {
		t.Errorf("title = %q, want %q", tk.Title, "Task A updated")
	}
	if !strings.Contains(filepath.Base(path), "task-a-updated") {
		t.Errorf("expected renamed file slug, got %q", filepath.Base(path))
	}
}

func TestEdit_FullWizardSavesUpdatedFields(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "e")
	for range len("Task A") {
		b = sendSpecialKey(b, tea.KeyBackspace)
	}
	b = typeText(b, "Edited task")

	b = sendSpecialKey(b, tea.KeyTab)
	b = typeText(b, "Updated body")

	b = sendSpecialKey(b, tea.KeyTab)
	b = sendKey(b, "j") // high -> critical

	b = sendSpecialKey(b, tea.KeyTab)
	b = typeText(b, "ui,editing")
	_ = sendSpecialKey(b, tea.KeyEnter)

	path, err := task.FindByID(cfg.TasksPath(), 1)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if tk.Title != "Edited task" {
		t.Errorf("title = %q, want %q", tk.Title, "Edited task")
	}
	if tk.Body != "Updated body\n" {
		t.Errorf("body = %q, want %q", tk.Body, "Updated body\\n")
	}
	if tk.Priority != "critical" {
		t.Errorf("priority = %q, want %q", tk.Priority, "critical")
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "ui" || tk.Tags[1] != "editing" {
		t.Errorf("tags = %v, want [ui editing]", tk.Tags)
	}
}
