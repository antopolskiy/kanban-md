//go:build !windows

package e2e_test

import (
	"strconv"
	"strings"
	"testing"
)

func TestE2E_TUI_CreateTask_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("c")
	session.waitForOutput("Create task in")

	session.typeText("Create task from TUI")
	session.pressKeys("tab")
	session.waitForOutput("Step 2/4")

	session.typeText("Created from interactive hotkeys")
	session.pressKeys("tab")
	session.waitForOutput("Step 3/4")

	session.pressKeys("tab")
	session.waitForOutput("Step 4/4")
	session.typeText("e2e,ui,hotkeys")
	session.pressKeys("enter")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()

	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list")
	if r.exitCode != 0 {
		t.Fatalf("listing tasks failed: %s", r.stderr)
	}

	var createdID int
	for _, task := range tasks {
		if task.Title == "Create task from TUI" {
			createdID = task.ID
			break
		}
	}
	if createdID == 0 {
		t.Fatalf("expected task created through TUI flow, got tasks: %#v", tasks)
	}

	var created taskJSON
	r = runKanbanJSON(t, kanbanDir, &created, "show", strconv.Itoa(createdID))
	if r.exitCode != 0 {
		t.Fatalf("show created task failed: %s", r.stderr)
	}

	if strings.TrimSuffix(created.Body, "\n") != "Created from interactive hotkeys" {
		t.Fatalf("expected body %q, got %q", "Created from interactive hotkeys", created.Body)
	}
	if !containsTag(created.Tags, "e2e") || !containsTag(created.Tags, "ui") || !containsTag(created.Tags, "hotkeys") {
		t.Fatalf("expected tags [e2e ui hotkeys], got %v", created.Tags)
	}
}

func TestE2E_TUI_CreateTaskWithoutBody(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("c")
	session.waitForOutput("Create task in")
	session.typeText("No body task")
	session.pressKeys("tab", "tab", "tab")
	session.pressKeys("enter")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()

	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list")
	if r.exitCode != 0 {
		t.Fatalf("listing tasks failed: %s", r.stderr)
	}

	for _, tsk := range tasks {
		if tsk.Title == "No body task" {
			return
		}
	}
	t.Fatalf("expected new task without body to be created")
}

func TestE2E_TUI_CreateCancel_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("c")
	session.waitForOutput("Create task in")
	session.pressKeys("esc")
	session.waitForOutput(tuiBoardStatus)

	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list")
	if r.exitCode != 0 {
		t.Fatalf("listing tasks failed: %s", r.stderr)
	}
	if got := len(tasks); got != 4 {
		t.Fatalf("expected 4 tasks after canceling create, got %d", got)
	}

	session.pressKeys("q")
	session.waitForExit()
}

func TestE2E_TUI_EditBackspaceFromTitleFlow(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("e")
	session.waitForOutput("Edit task #1")
	session.pressBackspaceRunes("Task A")
	session.typeText("Reworked")
	session.pressKeys("enter")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()

	var updated taskJSON
	r := runKanbanJSON(t, kanbanDir, &updated, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show edited task failed: %s", r.stderr)
	}
	if updated.Title != "Reworked" {
		t.Fatalf("title = %q, want %q", updated.Title, "Reworked")
	}
}

func TestE2E_TUI_EditTask_KeyboardFlow(t *testing.T) {
	kanbanDir := initBoard(t)
	seed := mustCreateTask(t, kanbanDir, "Edit target", "--body", "Original body", "--tag", "legacy")

	session := startTUIProcess(t, kanbanDir)
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("e")
	session.waitForOutput("Edit task #" + strconv.Itoa(seed.ID))

	session.pressBackspaceRunes(seed.Title)
	session.typeText("Edited task from TUI")
	session.pressKeys("tab")
	session.waitForOutput("Step 2/4")
	session.pressBackspaceRunes(seed.Body)
	session.typeText("Updated body text")
	session.pressKeys("tab")
	session.waitForOutput("Step 3/4")
	session.pressKeys("tab")
	session.waitForOutput("Step 4/4")
	session.pressBackspaceRunes(seed.Tags[0])
	session.typeText("e2e,updated")
	session.pressKeys("enter")
	session.waitForOutput(tuiBoardStatus)

	session.pressKeys("q")
	session.waitForExit()

	var updated taskJSON
	r := runKanbanJSON(t, kanbanDir, &updated, "show", strconv.Itoa(seed.ID))
	if r.exitCode != 0 {
		t.Fatalf("show edited task failed: %s", r.stderr)
	}

	if updated.Title != "Edited task from TUI" {
		t.Fatalf("expected title %q, got %q", "Edited task from TUI", updated.Title)
	}
	if strings.TrimSuffix(updated.Body, "\n") != "Updated body text" {
		t.Fatalf("expected body %q, got %q", "Updated body text", updated.Body)
	}
	if !containsTag(updated.Tags, "e2e") || !containsTag(updated.Tags, "updated") {
		t.Fatalf("expected edited tags [e2e updated], got %v", updated.Tags)
	}
}
