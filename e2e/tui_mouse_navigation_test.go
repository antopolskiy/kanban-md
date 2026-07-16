//go:build !windows

package e2e_test

import (
	"fmt"
	"strings"
	"testing"
)

func TestE2E_TUIMouse_DoubleClickDetailBackAndKeyboardSync(t *testing.T) {
	kanbanDir := initBoardWithSeededTasks(t)
	session := startTUIProcessWithOptions(t, kanbanDir, tuiProcessOptions{
		args: []string{"--mouse"},
	})
	session.waitForRawOutput("\x1b[?1002h")
	session.waitForOutput("mouse:click/double-click/wheel")
	session.resize(120, 40)

	// The default 120-column board has five 24-cell columns. Task C is the
	// first card in the third rendered column.
	checkpoint := session.checkpoint()
	session.clickSGR(49, 2)
	session.clickSGR(49, 2)
	session.waitForOutputSince(checkpoint, "Task #3: Task C")

	checkpoint = session.checkpoint()
	session.clickX10(1, 39)
	session.waitForOutputSince(checkpoint, "mouse:click/double-click/wheel")

	// Back preserves the card selection, so keyboard Enter must reopen Task C.
	checkpoint = session.checkpoint()
	session.pressKeys("enter")
	session.waitForOutputSince(checkpoint, "Task #3: Task C")
	session.pressKeys("q", "q")
	session.waitForExit()
}

func TestE2E_TUIMouse_WheelRevealsTaskAndScrollsLongDetail(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Short first card", "--priority", "critical")
	mustCreateTask(t, kanbanDir,
		"A deliberately long second card title that wraps",
		"--priority", "high")

	var bodyLines []string
	bodyLines = append(bodyLines, "DETAIL-TOP-MARKER")
	for i := 1; i <= 45; i++ {
		bodyLines = append(bodyLines, fmt.Sprintf("detail line %02d", i))
	}
	bodyLines = append(bodyLines, "DETAIL-BOTTOM-MARKER")
	mustCreateTask(t, kanbanDir, "Newly revealed task",
		"--priority", "medium",
		"--body", strings.Join(bodyLines, "\n"))

	session := startTUIProcessWithOptions(t, kanbanDir, tuiProcessOptions{
		args: []string{"--mouse"},
		cols: 100,
		rows: 14,
	})
	session.waitForOutput("mouse:click/double-click/wheel")

	checkpoint := session.checkpoint()
	session.wheelSGR(1, 2, 1)
	session.wheelSGR(2, 2, 1)
	session.waitForOutputSince(checkpoint, "revealed task")

	// After the two wheel events, the second card starts below the up
	// indicator and the newly revealed third card begins at row 7.
	checkpoint = session.checkpoint()
	session.clickSGR(2, 8)
	session.clickSGR(2, 8)
	session.waitForOutputSince(checkpoint, "Task #3: Newly revealed task")

	checkpoint = session.checkpoint()
	for range 30 {
		session.wheelSGR(2, 5, 1)
	}
	session.waitForOutputSince(checkpoint, "DETAIL-BOTTOM-MARKER")

	// Extra wheel events remain clamped and the keyboard fallback still works.
	for range 5 {
		session.wheelSGR(2, 5, 1)
	}
	for range 30 {
		session.wheelSGR(2, 5, -1)
	}
	checkpoint = session.checkpoint()
	session.pressKeys("q")
	session.waitForOutputSince(checkpoint, "mouse:click/double-click/wheel")
	session.pressKeys("q")
	session.waitForExit()
}
