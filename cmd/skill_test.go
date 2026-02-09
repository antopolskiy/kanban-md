package cmd

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel() selectModel {
	return selectModel{
		prompt: "Test prompt:",
		items: []menuItem{
			{label: "Item A", description: "desc A", selected: true},
			{label: "Item B", description: "desc B", selected: false},
			{label: "Item C", description: "desc C", selected: true},
		},
	}
}

func sendTestKey(m selectModel, key string) selectModel {
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return result.(selectModel)
}

func sendTestSpecialKey(m selectModel, keyType tea.KeyType) selectModel {
	result, _ := m.Update(tea.KeyMsg{Type: keyType})
	return result.(selectModel)
}

func TestSelectModel_EnterConfirms(t *testing.T) {
	m := newTestModel()
	m = sendTestSpecialKey(m, tea.KeyEnter)

	if !m.done {
		t.Error("expected done to be true after enter")
	}
	if m.canceled {
		t.Error("expected canceled to be false after enter")
	}
}

func TestSelectModel_EscCancels(t *testing.T) {
	m := newTestModel()
	m = sendTestSpecialKey(m, tea.KeyEscape)

	if !m.done {
		t.Error("expected done to be true after esc")
	}
	if !m.canceled {
		t.Error("expected canceled to be true after esc")
	}
}

func TestSelectModel_CtrlCCancels(t *testing.T) {
	m := newTestModel()
	m = sendTestSpecialKey(m, tea.KeyCtrlC)

	if !m.done {
		t.Error("expected done to be true after ctrl+c")
	}
	if !m.canceled {
		t.Error("expected canceled to be true after ctrl+c")
	}
}

func TestSelectModel_QCancels(t *testing.T) {
	m := newTestModel()
	m = sendTestKey(m, "q")

	if !m.done {
		t.Error("expected done to be true after q")
	}
	if !m.canceled {
		t.Error("expected canceled to be true after q")
	}
}

func TestSelectModel_CancelClearsSelections(t *testing.T) {
	m := newTestModel()

	// Items A and C are pre-selected. After cancel, multiSelect should return nil.
	// Simulate what multiSelect does: check canceled flag.
	m = sendTestSpecialKey(m, tea.KeyEscape)

	if !m.canceled {
		t.Fatal("expected canceled to be true")
	}

	// Verify items still have their selected state (model doesn't mutate them),
	// but the canceled flag tells multiSelect to return nil.
	if !m.items[0].selected {
		t.Error("item selections should not be mutated by cancel")
	}
}

func TestSelectModel_Navigation(t *testing.T) {
	m := newTestModel()

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Move down.
	m = sendTestKey(m, "j")
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}

	// Move down again.
	m = sendTestKey(m, "j")
	if m.cursor != 2 {
		t.Errorf("cursor after j = %d, want 2", m.cursor)
	}

	// At bottom, stay there.
	m = sendTestKey(m, "j")
	if m.cursor != 2 {
		t.Errorf("cursor at bottom after j = %d, want 2", m.cursor)
	}

	// Move up.
	m = sendTestKey(m, "k")
	if m.cursor != 1 {
		t.Errorf("cursor after k = %d, want 1", m.cursor)
	}
}

func TestSelectModel_SpaceToggles(t *testing.T) {
	m := newTestModel()

	// Item A is selected, toggle it off.
	m = sendTestKey(m, " ")
	if m.items[0].selected {
		t.Error("expected item A to be deselected after space")
	}

	// Toggle it back on.
	m = sendTestKey(m, " ")
	if !m.items[0].selected {
		t.Error("expected item A to be selected after second space")
	}
}

func TestSelectModel_DoneIgnoresInput(t *testing.T) {
	m := newTestModel()
	m = sendTestSpecialKey(m, tea.KeyEnter)

	// After done, further input should be ignored.
	before := m.cursor
	m = sendTestKey(m, "j")
	if m.cursor != before {
		t.Error("input should be ignored after done")
	}
}

func TestSelectModel_ViewShowsHelpWithEsc(t *testing.T) {
	m := newTestModel()
	view := m.View()

	if !containsSubstring(view, "esc cancel") {
		t.Errorf("view should mention esc cancel, got:\n%s", view)
	}
}

func TestSelectModel_ViewEmptyWhenDone(t *testing.T) {
	m := newTestModel()
	m = sendTestSpecialKey(m, tea.KeyEnter)

	if m.View() != "" {
		t.Error("view should be empty when done")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
