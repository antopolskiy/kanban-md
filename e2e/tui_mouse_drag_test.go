//go:build !windows

package e2e_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/config"
)

func TestE2E_TUIMouseDrag_MovesTaskWithSGRAndX10(t *testing.T) {
	for _, protocol := range []string{"SGR", "X10"} {
		t.Run(protocol, func(t *testing.T) {
			kanbanDir := initBoardWithSeededTasks(t)
			session := startTUIProcessWithOptions(t, kanbanDir, tuiProcessOptions{
				args: []string{"--mouse"},
			})
			session.waitForOutput("mouse:click/double-click/wheel")

			switch protocol {
			case "SGR":
				session.dragSGR(1, 2, 25, 0)
			case "X10":
				session.dragX10(1, 2, 25, 0)
			}
			waitForTask(t, kanbanDir, 1, func(tk taskJSON) bool {
				return tk.Status == statusTodo
			})

			// The moved task remains selected in its destination.
			checkpoint := session.checkpoint()
			session.pressKeys("enter")
			session.waitForOutputSince(checkpoint, "Task #1: Task A")
			session.pressKeys("q", "q")
			session.waitForExit()
		})
	}
}

func TestE2E_TUIMouseDrag_RejectsFullWIPColumn(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Move blocked by WIP", "--priority", "high")
	mustCreateTask(t, kanbanDir, "Occupies todo", "--status", statusTodo)

	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WIPLimits = map[string]int{statusTodo: 1}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	session := startTUIProcessWithOptions(t, kanbanDir, tuiProcessOptions{
		args: []string{"--mouse"},
	})
	session.waitForOutput("mouse:click/double-click/wheel")
	checkpoint := session.checkpoint()
	session.dragSGR(1, 2, 25, 0)
	session.waitForOutputSince(checkpoint, "WIP limit")
	assertTaskStatus(t, kanbanDir, 1, statusBacklog)
	assertKeyboardResponsiveAfterRejectedDrag(t, session)
}

func TestE2E_TUIMouseDrag_RejectsTaskClaimedByAnotherActor(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Claimed elsewhere", "--priority", "high")
	result := runKanban(t, kanbanDir, "edit", "1", "--claim", "other-agent")
	if result.exitCode != 0 {
		t.Fatalf("claiming task failed: %s", result.stderr)
	}

	session := startTUIProcessWithOptions(t, kanbanDir, tuiProcessOptions{
		args: []string{"--mouse"},
	})
	session.waitForOutput("mouse:click/double-click/wheel")
	checkpoint := session.checkpoint()
	session.dragX10(1, 2, 25, 0)
	session.waitForOutputSince(checkpoint, "claimed by")
	assertTaskStatus(t, kanbanDir, 1, statusBacklog)
	assertKeyboardResponsiveAfterRejectedDrag(t, session)
}

func assertTaskStatus(t *testing.T, kanbanDir string, id int, want string) {
	t.Helper()
	var tk taskJSON
	result := runKanbanJSON(t, kanbanDir, &tk, "show", strconv.Itoa(id))
	if result.exitCode != 0 {
		t.Fatalf("showing task #%d failed: %s", id, result.stderr)
	}
	if tk.Status != want {
		t.Fatalf("task #%d status=%q, want %q", id, tk.Status, want)
	}
}

func assertKeyboardResponsiveAfterRejectedDrag(t *testing.T, session *tuiSession) {
	t.Helper()
	checkpoint := session.checkpoint()
	session.pressKeys("enter")
	session.waitForOutputSince(checkpoint, "Task #1:")
	if strings.Contains(session.outputSince(checkpoint), "Move #1") {
		t.Fatal("stale drag hint remained after rejected move")
	}
	session.pressKeys("q", "q")
	session.waitForExit()
}
