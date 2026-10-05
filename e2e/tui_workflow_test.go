//go:build !windows

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedTUIWorkflowBoard(t *testing.T) string {
	t.Helper()
	dir := initBoard(t)
	mustCreateTask(t, dir, "Milestone", "--priority", "critical")
	mustCreateTask(t, dir, "Epic", "--status", "todo", "--parent", "1", "--tags", "Backend")
	mustCreateTask(t, dir, "Story", "--status", "in-progress", "--parent", "2", "--tags", "Backend")
	mustCreateTask(t, dir, "Done child", "--status", "done", "--parent", "2")
	return dir
}

func tuiWorkflowBytes(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path) //nolint:gosec // isolated E2E board
		if err == nil {
			files[path] = data
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func requireTUIWorkflowBytes(t *testing.T, dir string, before map[string][]byte) {
	t.Helper()
	after := tuiWorkflowBytes(t, dir)
	if len(before) != len(after) {
		t.Fatal("read-only TUI workflow changed the board file set")
	}
	for path, data := range before {
		if !bytes.Equal(data, after[path]) {
			t.Fatalf("read-only TUI workflow changed %s", path)
		}
	}
}

func TestE2E_TUIWorkflowTagAndExactDepth(t *testing.T) {
	dir := seedTUIWorkflowBoard(t)
	before := tuiWorkflowBytes(t, dir)
	session := startTUIProcessWithOptions(t, dir, tuiProcessOptions{args: []string{"--hide-empty-columns"}})
	session.waitForOutput("level[all]")
	checkpoint := session.checkpoint()
	session.pressKeys("/")
	session.typeText("backend")
	session.pressKeys("enter")
	session.waitForOutputSince(checkpoint, "2 cards | level[all]")
	checkpoint = session.checkpoint()
	session.pressKeys("v")
	session.waitForOutputSince(checkpoint, "0 cards | level[0]")
	checkpoint = session.checkpoint()
	session.pressKeys("v")
	session.waitForOutputSince(checkpoint, "1 card | level[1]")
	checkpoint = session.checkpoint()
	session.pressKeys("/")
	session.typeText("v")
	session.waitForOutputSince(checkpoint, "level[1] | /backendv")
	checkpoint = session.checkpoint()
	session.pressKeys("esc")
	session.waitForOutputSince(checkpoint, "1 card | level[1]")
	session.pressKeys("q")
	session.waitForExit()
	requireTUIWorkflowBytes(t, dir, before)
}

func TestE2E_TUIWorkflowRelationsHistoryOutsideFilters(t *testing.T) {
	dir := seedTUIWorkflowBoard(t)
	before := tuiWorkflowBytes(t, dir)
	session := startTUIProcessWithOptions(t, dir, tuiProcessOptions{args: []string{"--hide-empty-columns"}})
	session.waitForOutput("level[all]")
	session.pressKeys("v", "v", "v")
	session.waitForOutput("level[2]")
	session.pressKeys("/")
	session.typeText("#3 ")
	session.pressKeys("enter")
	checkpoint := session.checkpoint()
	session.pressKeys("enter")
	session.waitForOutputSince(checkpoint, "Task #3: Story")
	checkpoint = session.checkpoint()
	session.pressKeys("tab", "tab", "enter") // root-first focus, then direct epic
	session.waitForOutputSince(checkpoint, "Task #2: Epic")
	checkpoint = session.checkpoint()
	session.pressKeys("tab", "enter")
	session.waitForOutputSince(checkpoint, "Task #1: Milestone")
	checkpoint = session.checkpoint()
	session.pressKeys("esc")
	session.waitForOutputSince(checkpoint, "Task #2: Epic")
	checkpoint = session.checkpoint()
	session.pressKeys("backspace")
	session.waitForOutputSince(checkpoint, "Task #3: Story")
	checkpoint = session.checkpoint()
	session.pressKeys("q")
	session.waitForOutputSince(checkpoint, "1 card | level[2]")
	checkpoint = session.checkpoint()
	session.pressKeys("enter")
	session.waitForOutputSince(checkpoint, "Task #3: Story")
	session.pressKeys("q", "q")
	session.waitForExit()
	requireTUIWorkflowBytes(t, dir, before)
}

func TestE2E_TUIWorkflowMouseRelationAndBack(t *testing.T) {
	dir := initBoard(t)
	mustCreateTask(t, dir, "Milestone")
	mustCreateTask(t, dir, "Story", "--parent", "1")
	before := tuiWorkflowBytes(t, dir)
	session := startTUIProcessWithOptions(t, dir, tuiProcessOptions{
		args: []string{"--mouse", "--hide-empty-columns"},
	})
	session.waitForRawOutput("\x1b[?1002h")
	session.waitForOutput("level[all]")
	session.pressKeys("/")
	session.typeText("#2 ")
	detailCheckpoint := session.checkpoint()
	session.pressKeys("enter", "enter")
	session.waitForOutput("Task #2: Story")
	session.waitForOutput("Ancestry")
	session.waitForOutputSettled()
	// Locate the row in the newly rendered detail. Canonical default metadata
	// such as class can add lines before ancestry.
	detailOutput := session.outputSince(detailCheckpoint)
	header := strings.Index(detailOutput, "Task #2: Story")
	if header < 0 {
		t.Fatal("detail header is missing from the output checkpoint")
	}
	ancestorRow := -1
	for row, line := range strings.Split(detailOutput[header:], "\n") {
		if strings.Contains(line, "↑ #1 [backlog] Milestone") {
			ancestorRow = row
			break
		}
	}
	if ancestorRow < 0 {
		t.Fatal("ancestor row is missing from the rendered detail")
	}
	for _, x10 := range []bool{false, true} {
		checkpoint := session.checkpoint()
		if x10 {
			session.clickX10(2, ancestorRow)
		} else {
			session.clickSGR(2, ancestorRow)
		}
		session.waitForOutputSince(checkpoint, "Task #1: Milestone")
		checkpoint = session.checkpoint()
		if x10 {
			session.clickX10(1, 39)
		} else {
			session.clickSGR(1, 39)
		}
		session.waitForOutputSince(checkpoint, "Task #2: Story")
	}
	session.pressKeys("q", "q")
	session.waitForExit()
	requireTUIWorkflowBytes(t, dir, before)
}
