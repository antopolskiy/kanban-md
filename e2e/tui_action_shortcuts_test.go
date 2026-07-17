//go:build !windows

package e2e_test

import (
	"testing"
)

func TestE2E_TUI_MoveTask_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("m")
	session.waitForOutput("Move #1 to:")
	session.pressKeys("j")
	session.pressKeys("enter")
	session.waitForOutput(tuiBoardStatus)

	waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
		return tk.Status == "todo"
	})
	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_MoveTaskCancel_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("m")
	session.waitForOutput("Move #1 to:")
	session.pressKeys("esc")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()

	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list")
	if r.exitCode != 0 {
		t.Fatalf("listing tasks failed: %s", r.stderr)
	}
	if got := len(tasks); got != 4 {
		t.Fatalf("expected 4 tasks after canceling move, got %d", got)
	}
}

func TestE2E_TUI_MoveTaskQuickKeys_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("n")
	waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
		return tk.Status == "todo"
	})

	session.pressKeys("l")
	session.pressKeys("p")
	waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
		return tk.Status == "backlog"
	})

	session.waitForOutput(tuiBoardStatus)
	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_PriorityKeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("+")
	waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
		return tk.Priority == "critical"
	})

	session.pressKeys("-")
	waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
		return tk.Priority == "high"
	})

	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_DeleteTask_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("d")
	session.waitForOutput("Delete task?")
	session.pressKeys("y")

	waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
		return tk.Status == statusArchived
	})
	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_DeleteTaskCancel_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("d")
	session.waitForOutput("Delete task?")
	session.pressKeys("n")
	session.waitForOutput(tuiBoardStatus)

	var remaining []taskJSON
	r := runKanbanJSON(t, kanbanDir, &remaining, "list")
	if r.exitCode != 0 {
		t.Fatalf("listing tasks failed: %s", r.stderr)
	}
	if got := len(remaining); got != 4 {
		t.Fatalf("expected 4 visible tasks after canceling delete, got %d", got)
	}

	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_DeleteTaskFromSecondRow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("j")
	session.pressKeys("d")
	session.waitForOutput("Delete task?")
	session.pressKeys("y")

	waitForTask(t, kanbanDir, 2, func(tk taskJSON) bool {
		return tk.Status == statusArchived
	})
	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_DetailKeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("enter")
	session.waitForOutput("Status:")
	session.pressKeys("q")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_DetailFromSecondTaskViaNavigation(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("j")
	session.pressKeys("enter")
	session.waitForOutput("Task B")
	session.pressKeys("q")
	session.waitForOutput(tuiBoardStatus)
	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_HelpFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("?")
	session.waitForOutput("Keyboard Shortcuts")
	session.pressKeys("q")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_NavigateColumnsAndRows_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("l")
	session.pressKeys("h")
	session.pressKeys("j")
	session.pressKeys("k")
	session.pressKeys("h")
	session.pressKeys("l")
	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_EscapeExits(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("esc")
	session.waitForExit()

	_ = runKanbanJSON(t, kanbanDir, &struct{}{}, "board")
}

func TestE2E_TUI_CtrlCExits(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("ctrl+c")
	session.waitForExit()

	_ = runKanbanJSON(t, kanbanDir, &struct{}{}, "board")
}

func TestE2E_TUI_Edit_And_CloseWithBackspace(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("e")
	session.waitForOutput("Edit task #1")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("backspace")
	session.pressKeys("esc")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show task failed: %s", r.stderr)
	}
	if task.Title != "Task A" {
		t.Errorf("expected title unchanged after escaped edit, got %q", task.Title)
	}
}
