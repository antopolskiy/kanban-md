package tui_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

func TestCreate_DialogOpensAndCloses(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Press 'c' to open create dialog.
	b = sendKey(b, "c")
	v := b.View()
	if !containsStr(v, "Create task in") {
		t.Error("expected create dialog, got:", v[:min(len(v), 200)])
	}

	// Press esc to cancel.
	b = sendSpecialKey(b, tea.KeyEscape)
	v = b.View()
	if containsStr(v, "Create task in") {
		t.Error("expected board view after esc, but create dialog is still showing")
	}
}

func TestCreate_TypesTitle(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "New task")

	v := b.View()
	if !containsStr(v, "New task") {
		t.Error("expected typed title in dialog")
	}
}

func TestCreate_BackspaceDeletesCharacter(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	// Type "AB" then backspace.
	for _, ch := range "AB" {
		m, _ := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		b = m.(*tui.Board)
	}
	b = sendSpecialKey(b, tea.KeyBackspace)

	v := b.View()
	// Should show "A" not "AB".
	if containsStr(v, "AB") {
		t.Error("backspace should have deleted the last character")
	}
}

// typeText sends each character as a rune key to the board.
func typeText(b *tui.Board, text string) *tui.Board {
	for _, ch := range text {
		m, _ := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		b = m.(*tui.Board)
	}
	return b
}

func TestCreate_EnterCreatesFromTitleStep(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Navigate to todo column (l once from backlog).
	b = sendKey(b, "l")

	// Open create dialog.
	b = sendKey(b, "c")

	// Type task title and Enter to create immediately.
	b = typeText(b, "My new task")
	b = sendSpecialKey(b, tea.KeyEnter)

	// Board should be back to normal view with the new task visible.
	v := b.View()
	if !containsStr(v, "My new task") {
		t.Error("expected new task in board view")
	}

	// Verify the task file was created.
	entries, err := os.ReadDir(cfg.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if containsStr(e.Name(), "my-new-task") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected task file to be created in tasks directory")
	}
}

func TestCreate_FullWizardCreatesTask(t *testing.T) {
	b, cfg := setupTestBoard(t)

	// Navigate to todo column.
	b = sendKey(b, "l")

	// Open create dialog.
	b = sendKey(b, "c")

	// Step 1: Title.
	v := b.View()
	if !containsStr(v, "Step 1/4: Title") {
		t.Fatal("expected step 1 (Title)")
	}
	b = typeText(b, "Wizard task")
	b = sendSpecialKey(b, tea.KeyTab) // advance to body

	// Step 2: Body.
	v = b.View()
	if !containsStr(v, "Step 2/4: Body") {
		t.Fatal("expected step 2 (Body)")
	}
	b = typeText(b, "Task description")
	b = sendSpecialKey(b, tea.KeyTab) // advance to priority

	// Step 3: Priority.
	v = b.View()
	if !containsStr(v, "Step 3/4: Priority") {
		t.Fatal("expected step 3 (Priority)")
	}

	b = sendKey(b, "j")               // move to next priority
	b = sendSpecialKey(b, tea.KeyTab) // advance to tags

	// Step 4: Tags.
	v = b.View()
	if !containsStr(v, "Step 4/4: Tags") {
		t.Fatal("expected step 4 (Tags)")
	}
	b = typeText(b, "test,wizard")
	b = sendSpecialKey(b, tea.KeyEnter) // create

	// Board should show the new task.
	v = b.View()
	if !containsStr(v, "Wizard task") {
		t.Error("expected new task in board view")
	}

	// Verify file was created.
	entries, err := os.ReadDir(cfg.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if containsStr(e.Name(), "wizard-task") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected task file to be created in tasks directory")
	}
}

func TestCreate_TabAdvancesToBodyStep(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Some title")
	b = sendSpecialKey(b, tea.KeyTab)

	v := b.View()
	if !containsStr(v, "Step 2/4: Body") {
		t.Error("expected body step after Tab on title")
	}
}

func TestCreate_EmptyTitleCancels(t *testing.T) {
	b, cfg := setupTestBoard(t)

	initialCount := countTaskFiles(t, cfg.TasksPath())

	// Open and immediately press enter (empty title).
	b = sendKey(b, "c")
	b = sendSpecialKey(b, tea.KeyEnter)

	// Should be back at board with no new task.
	v := b.View()
	if containsStr(v, "Create task in") {
		t.Error("expected board view after empty enter")
	}

	if got := countTaskFiles(t, cfg.TasksPath()); got != initialCount {
		t.Errorf("task count changed: %d -> %d, empty title should not create", initialCount, got)
	}
}

func TestCreate_UsesCurrentColumnStatus(t *testing.T) {
	b, _ := setupTestBoard(t)

	// Navigate to in-progress column (l twice from backlog).
	b = sendKey(b, "l") // todo
	b = sendKey(b, "l") // in-progress

	// Open create dialog — should show "in-progress" in the prompt.
	b = sendKey(b, "c")
	v := b.View()
	if !containsStr(v, "in-progress") {
		t.Error("expected 'in-progress' in create dialog")
	}
}

func TestCreate_NextIDIncrements(t *testing.T) {
	b, cfg := setupTestBoard(t)

	initialNextID := cfg.NextID

	// Create a task using Enter to finish immediately from title.
	b = sendKey(b, "c")
	b = typeText(b, "Increment test")
	_ = sendSpecialKey(b, tea.KeyEnter)

	// Reload config and check NextID incremented.
	reloaded, err := config.Load(cfg.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.NextID != initialNextID+1 {
		t.Errorf("NextID = %d, want %d", reloaded.NextID, initialNextID+1)
	}
}

func TestCreate_SpaceInTitle(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")

	// Type with explicit space key.
	for _, ch := range "hello" {
		m, _ := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		b = m.(*tui.Board)
	}
	m, _ := b.Update(tea.KeyMsg{Type: tea.KeySpace})
	b = m.(*tui.Board)
	for _, ch := range "world" {
		m, _ = b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		b = m.(*tui.Board)
	}

	v := b.View()
	if !containsStr(v, "hello world") {
		t.Error("expected 'hello world' in dialog")
	}
}

func TestCreate_StatusBarShowsCreateHint(t *testing.T) {
	b, _ := setupTestBoard(t)
	v := b.View()
	if !containsStr(v, "c:create") {
		t.Error("expected 'c:create' in status bar")
	}
}

func TestCreate_ShiftTabNavigatesBack(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Title")
	b = sendSpecialKey(b, tea.KeyTab) // → body

	v := b.View()
	if !containsStr(v, "Step 2/4: Body") {
		t.Fatal("expected body step")
	}

	// Shift+Tab back to title.
	m, _ := b.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	b = m.(*tui.Board)

	v = b.View()
	if !containsStr(v, "Step 1/4: Title") {
		t.Error("expected title step after shift+tab")
	}
	// Title should still be preserved.
	if !containsStr(v, "Title") {
		t.Error("expected title text preserved after navigating back")
	}
}

func TestCreate_PrioritySelection(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Priority test")
	b = sendSpecialKey(b, tea.KeyTab) // → body
	b = sendSpecialKey(b, tea.KeyTab) // → priority

	v := b.View()
	if !containsStr(v, "Step 3/4: Priority") {
		t.Fatal("expected priority step")
	}
	// Should show priority options.
	if !containsStr(v, "low") {
		t.Error("expected 'low' in priority options")
	}
	if !containsStr(v, "high") {
		t.Error("expected 'high' in priority options")
	}
}

func TestCreate_EscCancelsFromAnyStep(t *testing.T) {
	b, cfg := setupTestBoard(t)
	initialCount := countTaskFiles(t, cfg.TasksPath())

	// Open wizard, advance to body step, then cancel.
	b = sendKey(b, "c")
	b = typeText(b, "Cancel test")
	b = sendSpecialKey(b, tea.KeyTab) // → body
	b = sendSpecialKey(b, tea.KeyEscape)

	v := b.View()
	if containsStr(v, "Create task in") {
		t.Error("expected board view after esc from body step")
	}

	if got := countTaskFiles(t, cfg.TasksPath()); got != initialCount {
		t.Errorf("task count changed: %d -> %d, cancel should not create", initialCount, got)
	}
}

func TestCreate_FullWizardTaskHasCorrectFields(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "c")

	// Title.
	b = typeText(b, "Field test")
	b = sendSpecialKey(b, tea.KeyTab) // → body

	// Body.
	b = typeText(b, "Description here")
	b = sendSpecialKey(b, tea.KeyTab) // → priority

	// Priority: default is medium (index 1). Move down to high (index 2).
	b = sendKey(b, "j")
	b = sendSpecialKey(b, tea.KeyTab) // → tags

	// Tags.
	b = typeText(b, "backend, api")
	_ = sendSpecialKey(b, tea.KeyEnter) // create

	// Read the created task and verify fields.
	entries, err := os.ReadDir(cfg.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if containsStr(e.Name(), "field-test") {
			tk, err := task.Read(filepath.Join(cfg.TasksPath(), e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if tk.Title != "Field test" {
				t.Errorf("title = %q, want %q", tk.Title, "Field test")
			}
			if tk.Body != "Description here\n" {
				t.Errorf("body = %q, want %q", tk.Body, "Description here\n")
			}
			if tk.Priority != "high" {
				t.Errorf("priority = %q, want %q", tk.Priority, "high")
			}
			if len(tk.Tags) != 2 || tk.Tags[0] != "backend" || tk.Tags[1] != "api" {
				t.Errorf("tags = %v, want [backend api]", tk.Tags)
			}
			return
		}
	}
	t.Error("task file not found")
}

func TestCreate_EnterCreatesFromBodyStep(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Quick create")
	b = sendSpecialKey(b, tea.KeyTab) // → body
	b = typeText(b, "Some body")
	b = sendSpecialKey(b, tea.KeyEnter) // create from body step

	v := b.View()
	if !containsStr(v, "Quick create") {
		t.Error("expected task in board after enter from body step")
	}

	// Verify body was saved.
	entries, err := os.ReadDir(cfg.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if containsStr(e.Name(), "quick-create") {
			tk, err := task.Read(filepath.Join(cfg.TasksPath(), e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if tk.Body != "Some body\n" {
				t.Errorf("body = %q, want %q", tk.Body, "Some body\n")
			}
			return
		}
	}
	t.Error("task file not found")
}

func TestCreate_EnterCreatesFromPriorityStep(t *testing.T) {
	b, cfg := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Priority create")
	b = sendSpecialKey(b, tea.KeyTab)   // → body
	b = sendSpecialKey(b, tea.KeyTab)   // → priority
	b = sendKey(b, "j")                 // select next priority
	b = sendSpecialKey(b, tea.KeyEnter) // create from priority step

	v := b.View()
	if !containsStr(v, "Priority create") {
		t.Error("expected task in board after enter from priority step")
	}

	// Verify task was created.
	entries, err := os.ReadDir(cfg.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if containsStr(e.Name(), "priority-create") {
			tk, err := task.Read(filepath.Join(cfg.TasksPath(), e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if tk.Priority != "high" {
				t.Errorf("priority = %q, want %q", tk.Priority, "high")
			}
			return
		}
	}
	t.Error("task file not found")
}

func TestCreate_TabDoesNotAdvancePastLastStep(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Tab test")
	b = sendSpecialKey(b, tea.KeyTab) // → body
	b = sendSpecialKey(b, tea.KeyTab) // → priority
	b = sendSpecialKey(b, tea.KeyTab) // → tags
	b = sendSpecialKey(b, tea.KeyTab) // should stay on tags

	v := b.View()
	if !containsStr(v, "Step 4/4: Tags") {
		t.Error("expected to stay on tags step after extra tab")
	}
}

func TestCreate_ShiftTabDoesNotGoPastFirstStep(t *testing.T) {
	b, _ := setupTestBoard(t)

	b = sendKey(b, "c")
	b = typeText(b, "Back test")

	// Shift+Tab on first step should stay on title.
	m, _ := b.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	b = m.(*tui.Board)

	v := b.View()
	if !containsStr(v, "Step 1/4: Title") {
		t.Error("expected to stay on title step after shift+tab")
	}
}

func TestCreate_ConsistentHints(t *testing.T) {
	b, _ := setupTestBoard(t)
	b = sendKey(b, "c")

	// Title step: should show tab:next and enter:create (no alt+enter).
	v := b.View()
	if !containsStr(v, "tab:next") {
		t.Error("expected 'tab:next' hint on title step")
	}
	if !containsStr(v, "enter:create") {
		t.Error("expected 'enter:create' hint on title step")
	}
	if containsStr(v, "alt+enter") {
		t.Error("should not show 'alt+enter' hint")
	}

	// Body step.
	b = sendSpecialKey(b, tea.KeyTab)
	v = b.View()
	if !containsStr(v, "tab:next") {
		t.Error("expected 'tab:next' hint on body step")
	}
	if !containsStr(v, "enter:create") {
		t.Error("expected 'enter:create' hint on body step")
	}
	if !containsStr(v, "shift+tab:back") {
		t.Error("expected 'shift+tab:back' hint on body step")
	}

	// Priority step.
	b = sendSpecialKey(b, tea.KeyTab)
	v = b.View()
	if !containsStr(v, "tab:next") {
		t.Error("expected 'tab:next' hint on priority step")
	}
	if !containsStr(v, "enter:create") {
		t.Error("expected 'enter:create' hint on priority step")
	}

	// Tags step.
	b = sendSpecialKey(b, tea.KeyTab)
	v = b.View()
	if !containsStr(v, "enter:create") {
		t.Error("expected 'enter:create' hint on tags step")
	}
	if !containsStr(v, "shift+tab:back") {
		t.Error("expected 'shift+tab:back' hint on tags step")
	}
}

func countTaskFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			count++
		}
	}
	return count
}
