package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	workflowStoryQuery = "#3 "
	workflowArchived   = "archived"
)

func workflowTasks() []*task.Task {
	return []*task.Task{
		{ID: 1, Title: "Milestone", Status: "backlog", Priority: "critical"},
		{ID: 2, Title: "Epic", Status: "todo", Priority: "high", Parent: parentID(1), Tags: []string{"Backend"}},
		{ID: 3, Title: "Story", Status: "in-progress", Priority: "medium", Parent: parentID(2), Tags: []string{"Backend"}},
		{ID: 4, Title: "Done child", Status: "done", Priority: "low", Parent: parentID(2)},
		{ID: 5, Title: "Archived epic", Status: workflowArchived, Priority: "medium", Parent: parentID(1)},
		{ID: 6, Title: "Child of archived epic", Status: "todo", Priority: "medium", Parent: parentID(5)},
		{ID: 7, Title: "Other root", Status: "done", Priority: "low"},
	}
}

func newWorkflowBoard(t *testing.T, tasks []*task.Task) (*Board, *config.Config) {
	t.Helper()
	cfg := config.NewDefault("Workflow test")
	cfg.SetDir(t.TempDir())
	if err := os.MkdirAll(cfg.TasksPath(), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	for _, tk := range tasks {
		if tk.Title == "" {
			tk.Title = fmt.Sprintf("Task %d", tk.ID)
		}
		if tk.Priority == "" {
			tk.Priority = "medium"
		}
		tk.Created = mouseTestTime
		tk.Updated = mouseTestTime
		if err := task.Write(filepath.Join(cfg.TasksPath(), task.GenerateFilename(tk.ID, tk.Title)), tk); err != nil {
			t.Fatal(err)
		}
	}
	b := NewBoard(cfg)
	b.SetNow(func() time.Time { return mouseTestTime.Add(time.Hour) })
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return b, cfg
}

func workflowKey(b *Board, key string) {
	_, _ = b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
}

func requireDetailID(t *testing.T, b *Board, id int) {
	t.Helper()
	if b.view != viewDetail || b.detailTask == nil || b.detailTask.ID != id {
		t.Fatalf("detail view=%v task=%#v, want #%d", b.view, b.detailTask, id)
	}
}

func changeWorkflowTask(t *testing.T, cfg *config.Config, id int, change func(*task.Task)) {
	t.Helper()
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	change(tk)
	if err := task.Write(path, tk); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowDepthCycleAndSearchComposition(t *testing.T) {
	b, _ := newWorkflowBoard(t, workflowTasks())
	for _, tt := range []struct {
		level string
		count int
	}{
		{"0", 2}, {"1", 1}, {"2", 3}, {"all", 6},
	} {
		workflowKey(b, "v")
		if len(b.tasks) != tt.count || b.levelState() != "level["+tt.level+"]" {
			t.Fatalf("level=%s count=%d, want %s/%d", b.levelState(), len(b.tasks), tt.level, tt.count)
		}
	}
	workflowKey(b, "v")
	workflowKey(b, "v") // level 1
	workflowKey(b, "/")
	for _, r := range "BACKEND" {
		workflowKey(b, string(r))
	}
	workflowKey(b, "enter")
	if len(b.tasks) != 1 || b.tasks[0].ID != 2 {
		t.Fatalf("tag AND exact depth=%#v", b.tasks)
	}
	b.filterQuery = workflowStoryQuery
	b.loadTasks()
	if len(b.tasks) != 0 || *b.depthFilter != 1 {
		t.Fatal("ID query should compose with, not reset, exact depth")
	}
	b.filterQuery = ""
	workflowKey(b, "/")
	workflowKey(b, "v")
	if b.filterQuery != "v" || *b.depthFilter != 1 {
		t.Fatal("v in search input must type, not cycle")
	}
	if got := b.View(); !strings.Contains(got, "level[1]") {
		t.Fatalf("search bar hides active level:\n%s", got)
	}
	workflowKey(b, "esc")
	if b.filterQuery != "" || *b.depthFilter != 1 {
		t.Fatal("clearing text search should retain depth")
	}
}

func TestWorkflowInactiveColumnOffsetsClampAfterRebuild(t *testing.T) {
	for _, transition := range []string{"depth", "search", "reload"} {
		t.Run(transition, func(t *testing.T) {
			tasks := []*task.Task{
				{ID: 1, Title: "Backlog root", Status: "backlog"},
				{ID: 2, Title: "Unique level-one child", Status: "todo", Parent: parentID(1)},
			}
			for id := 3; id <= 22; id++ {
				tasks = append(tasks, &task.Task{ID: id, Title: fmt.Sprintf("Todo root %d", id), Status: "todo"})
			}
			b, cfg := newWorkflowBoard(t, tasks)
			b.SetMouseEnabled(true)
			b.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
			last := b.columns[1].tasks[len(b.columns[1].tasks)-1]
			b.selectTaskByID(last.ID)
			offset := b.currentColumn().scrollOff
			if offset == 0 {
				t.Fatal("fixture did not scroll the todo column")
			}
			b.selectTaskByID(1)
			b.loadTasks()
			if b.columns[1].scrollOff != offset {
				t.Fatal("rebuild must retain a still-valid inactive scroll position")
			}
			t.Logf("progress: %s reduces a todo column scrolled to %d", transition, offset)
			reduceInactiveWorkflowColumn(t, b, cfg, transition)
			if b.activeCol != 0 || len(b.columns[1].tasks) != 1 || b.columns[1].tasks[0].ID != 2 {
				t.Fatal("fixture did not leave the sole child in an inactive todo column")
			}
			for _, col := range b.columns {
				if col.scrollOff < 0 || col.scrollOff > max(0, len(col.tasks)-1) {
					t.Fatalf("%s retained invalid offset %d for %d tasks", col.status, col.scrollOff, len(col.tasks))
				}
			}
			view := ansi.Strip(b.View())
			if !strings.Contains(view, "#2 L1 Unique") || !strings.Contains(view, "level-one child") {
				t.Fatalf("the matching child's wrapped card is absent after the rebuild:\n%s", view)
			}
			if !slices.ContainsFunc(b.layout.cards, func(target cardTarget) bool { return target.taskID == 2 }) {
				t.Fatal("the matching child has no mouse target after the rebuild")
			}
		})
	}
}

func reduceInactiveWorkflowColumn(t *testing.T, b *Board, cfg *config.Config, transition string) {
	t.Helper()
	switch transition {
	case "depth":
		workflowKey(b, "v")
		workflowKey(b, "v")
	case "search":
		workflowKey(b, "/")
		for _, r := range "Unique level-one" {
			workflowKey(b, string(r))
		}
		workflowKey(b, "enter")
	case "reload":
		for id := 3; id <= 22; id++ {
			changeWorkflowTask(t, cfg, id, func(tk *task.Task) { tk.Status = workflowArchived })
		}
		b.Update(ReloadMsg{})
	}
}

func TestWorkflowFlatDefaultAndDepthReload(t *testing.T) {
	b, cfg := newWorkflowBoard(t, []*task.Task{{ID: 1, Title: "Flat", Status: "backlog", Priority: "medium"}})
	before := b.View()
	if strings.Contains(before, " L0") || strings.Contains(before, "level[") {
		t.Fatal("flat default must not gain hierarchy labels")
	}
	workflowKey(b, "v")
	if b.levelState() != "level[0]" || strings.Contains(b.View(), " L0") {
		t.Fatal("explicit flat depth view should filter without card labels")
	}
	workflowKey(b, "v")
	if b.View() != before {
		t.Fatal("returning to all must restore flat presentation")
	}
	changeWorkflowTask(t, cfg, 1, func(tk *task.Task) { tk.Parent = parentID(99) })
	b.Update(ReloadMsg{})
	if !strings.Contains(b.View(), "#1 L0") {
		t.Fatal("invalid parent should enable an explanatory hierarchy cue")
	}
}

func TestWorkflowDepthUnknownAndSkippedLevels(t *testing.T) {
	tasks := []*task.Task{
		{ID: 1, Status: workflowArchived},
		{ID: 2, Status: workflowArchived, Parent: parentID(1)},
		{ID: 3, Status: "todo", Priority: "high", Parent: parentID(2)},
		{ID: 4, Status: "todo", Priority: "medium", Parent: parentID(5)},
		{ID: 5, Status: "todo", Priority: "low", Parent: parentID(4)},
	}
	b, _ := newWorkflowBoard(t, tasks)
	workflowKey(b, "v")
	if b.depthFilter == nil || *b.depthFilter != 2 || len(b.tasks) != 1 {
		t.Fatal("cycle should skip absent active depths")
	}
	workflowKey(b, "v")
	if b.depthFilter == nil || *b.depthFilter != unknownDepth || len(b.tasks) != 2 || !strings.Contains(b.View(), "L?") {
		t.Fatal("unknown-depth tasks must have a distinct last choice and cue")
	}
	workflowKey(b, "v")
	if b.depthFilter != nil || len(b.tasks) != 3 {
		t.Fatal("unknown depth must return to all")
	}
}

func TestWorkflowDepthReloadAndHiddenColumnsKeepSelectionID(t *testing.T) {
	b, cfg := newWorkflowBoard(t, workflowTasks())
	b.SetHideEmptyColumns(true)
	b.selectTaskByID(3)
	depth := 2
	b.depthFilter = &depth
	b.reloadKeepingSelection()
	if tk := b.selectedTask(); tk == nil || tk.ID != 3 {
		t.Fatalf("same selected ID lost when columns disappeared: %#v", tk)
	}
	b.Update(ReloadMsg{})
	if b.depthFilter == nil || *b.depthFilter != 2 {
		t.Fatal("existing depth should survive reload")
	}
	for _, id := range []int{3, 4, 6} {
		changeWorkflowTask(t, cfg, id, func(tk *task.Task) { tk.Status = workflowArchived })
	}
	b.Update(ReloadMsg{})
	if b.depthFilter != nil || !strings.Contains(b.View(), "level[all]") {
		t.Fatal("vanished active depth should reset visibly to all")
	}
}

func TestWorkflowRelationsRootFirstHistoryAndFilters(t *testing.T) {
	b, _ := newWorkflowBoard(t, workflowTasks())
	b.filterQuery = workflowStoryQuery
	depth := 2
	b.depthFilter = &depth
	b.loadTasks()
	b.SetHideEmptyColumns(true)
	b.handleEnter()
	requireDetailID(t, b, 3)
	workflowKey(b, "tab")
	if b.detailFocusID != 1 {
		t.Fatal("first Tab should focus the root ancestor")
	}
	workflowKey(b, "tab")
	workflowKey(b, "enter")
	requireDetailID(t, b, 2)
	workflowKey(b, "tab")
	workflowKey(b, "enter")
	requireDetailID(t, b, 1)
	workflowKey(b, "esc")
	requireDetailID(t, b, 2)
	if b.detailFocusID != 1 {
		t.Fatal("Back did not restore focus by ID")
	}
	workflowKey(b, "backspace")
	requireDetailID(t, b, 3)
	if b.detailFocusID != 2 {
		t.Fatal("Back did not restore story focus")
	}
	workflowKey(b, "q")
	if b.view != viewBoard || b.filterQuery != workflowStoryQuery || *b.depthFilter != 2 || b.selectedTask().ID != 3 {
		t.Fatal("closing the chain lost original board state")
	}
	b.handleEnter()
	workflowKey(b, "shift+tab")
	if b.detailFocusID != 2 {
		t.Fatal("Shift+Tab should activate the last relation")
	}
	workflowKey(b, "tab")
	if b.detailFocusID != 1 {
		t.Fatal("Tab should wrap in reading order")
	}
	workflowKey(b, "esc")
	if b.view != viewBoard {
		t.Fatal("Back from first detail should return to board")
	}
}

func TestWorkflowRelationReloadFocusAndUnavailableHistory(t *testing.T) {
	b, cfg := newWorkflowBoard(t, workflowTasks())
	b.selectTaskByID(3)
	b.handleEnter()
	workflowKey(b, "tab")
	workflowKey(b, "tab")
	workflowKey(b, "enter") // epic
	workflowKey(b, "tab")
	workflowKey(b, "enter") // milestone
	changeWorkflowTask(t, cfg, 2, func(tk *task.Task) { tk.Status = workflowArchived })
	b.Update(ReloadMsg{})
	workflowKey(b, "esc")
	requireDetailID(t, b, 3) // unavailable epic frame is skipped
	if b.detailFocusID != 0 {
		t.Fatal("unavailable archived focused relation should become inactive")
	}
	if !strings.Contains(b.View(), "[archived] Epic") {
		t.Fatal("archived ancestor should remain visible context")
	}
	workflowKey(b, "tab")
	if b.detailFocusID != 1 {
		t.Fatal("archived ancestor should be skipped while its active ancestor stays openable")
	}
	changeWorkflowTask(t, cfg, 3, func(tk *task.Task) { tk.Parent = parentID(7) })
	b.Update(ReloadMsg{})
	if b.detailFocusID != 0 {
		t.Fatal("reparenting must clear stale focus instead of retargeting")
	}
	changeWorkflowTask(t, cfg, 3, func(tk *task.Task) { tk.Status = workflowArchived })
	b.Update(ReloadMsg{})
	if b.view != viewBoard || b.detailTask != nil || len(b.detailHistory) != 0 {
		t.Fatal("archiving the open task must close the complete chain")
	}
}

func TestWorkflowDeletedHistoryFramesAndOpenTask(t *testing.T) {
	b, cfg := newWorkflowBoard(t, workflowTasks())
	b.selectTaskByID(3)
	b.handleEnter()
	workflowKey(b, "tab")
	workflowKey(b, "tab")
	workflowKey(b, "enter")
	workflowKey(b, "tab")
	workflowKey(b, "enter")
	for _, id := range []int{2, 3} {
		path, err := task.FindByID(cfg.TasksPath(), id)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		b.Update(ReloadMsg{})
		if id == 2 {
			workflowKey(b, "esc")
			requireDetailID(t, b, 3)
			if b.detailFocusID != 0 || !strings.Contains(b.View(), "missing parent #2") {
				t.Fatal("deleted history frame must be skipped, retaining the missing parent ID")
			}
		} else if b.view != viewBoard || b.detailTask != nil || len(b.detailHistory) != 0 {
			t.Fatal("deleting the open task must close details and history")
		}
	}
}

func TestWorkflowTagFilterKeepsDirectChildContext(t *testing.T) {
	b, _ := newWorkflowBoard(t, workflowTasks())
	b.filterQuery = "backend"
	depth := 1
	b.depthFilter = &depth
	b.reloadKeepingSelection()
	b.selectTaskByID(2)
	b.handleEnter()
	if got := b.View(); !strings.Contains(got, "Children (1/2 done)") || !strings.Contains(got, "Done child") {
		t.Fatalf("tag/depth filters must not hide direct child context:\n%s", got)
	}
}

func TestWorkflowRelationViewportAndStableReload(t *testing.T) {
	tasks := workflowTasks()
	for id := 10; id < 43; id++ {
		tasks = append(tasks, &task.Task{
			ID: id, Title: fmt.Sprintf("Child %d with a long wrapped description", id),
			Status: "todo", Priority: "low", Parent: parentID(2),
		})
	}
	b, cfg := newWorkflowBoard(t, tasks)
	b.Update(tea.WindowSizeMsg{Width: 32, Height: 12})
	b.selectTaskByID(2)
	b.handleEnter()
	workflowKey(b, "shift+tab")
	if b.detailFocusID != 42 || !strings.Contains(b.View(), "Child 42") {
		t.Fatal("focus should scroll the final off-screen wrapped child into view")
	}
	saved := b.detailScrollOff
	workflowKey(b, "enter")
	requireDetailID(t, b, 42)
	workflowKey(b, "esc")
	requireDetailID(t, b, 2)
	if b.detailScrollOff != saved || b.detailFocusID != 42 {
		t.Fatal("history should restore the prior viewport and relation ID")
	}
	changeWorkflowTask(t, cfg, 42, func(tk *task.Task) { tk.Title = "Renamed child" })
	b.Update(ReloadMsg{})
	if b.detailFocusID != 42 {
		t.Fatal("renaming must retain focus by ID")
	}
	workflowKey(b, "g")
	if b.detailScrollOff != 0 || b.detailFocusID != 42 {
		t.Fatal("scroll keys must not become relation cursor keys")
	}
	workflowKey(b, "G")
	_ = b.View()
	if b.detailScrollOff <= 0 || b.detailFocusID != 42 {
		t.Fatal("bottom scroll should preserve focus")
	}
}

func workflowFileBytes(t *testing.T, cfg *config.Config) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(cfg.Dir(), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path) //nolint:gosec // isolated test board
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

func TestWorkflowReadOnlyDoesNotWriteFiles(t *testing.T) {
	b, cfg := newWorkflowBoard(t, workflowTasks())
	before := workflowFileBytes(t, cfg)
	workflowKey(b, "v")
	workflowKey(b, "v")
	workflowKey(b, "/")
	for _, r := range "backend" {
		workflowKey(b, string(r))
	}
	workflowKey(b, "enter")
	b.SetHideEmptyColumns(true)
	b.handleEnter()
	workflowKey(b, "tab")
	workflowKey(b, "enter")
	workflowKey(b, "esc")
	workflowKey(b, "q")
	b.Update(ReloadMsg{})
	after := workflowFileBytes(t, cfg)
	if len(before) != len(after) {
		t.Fatal("read-only interaction changed the file set")
	}
	for path, data := range before {
		if !bytes.Equal(data, after[path]) {
			t.Fatalf("read-only interaction changed %s", path)
		}
	}
}
