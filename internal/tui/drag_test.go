package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var dragMoveTime = time.Date(2026, 7, 16, 15, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // fixed mutation clock

const (
	dragStatusBacklog = "backlog"
	dragStatusTodo    = "todo"
)

func newDragFilesystemBoard(
	t *testing.T,
	configure func(*config.Config),
	tasks ...*task.Task,
) (*Board, *config.Config) {
	t.Helper()
	kanbanDir := filepath.Join(t.TempDir(), "kanban")
	if err := os.MkdirAll(filepath.Join(kanbanDir, "tasks"), 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewDefault("Drag Test")
	cfg.SetDir(kanbanDir)
	if configure != nil {
		configure(cfg)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	for _, tk := range tasks {
		if tk.Priority == "" {
			tk.Priority = "medium"
		}
		if tk.Created.IsZero() {
			tk.Created = mouseTestTime
		}
		if tk.Updated.IsZero() {
			tk.Updated = mouseTestTime
		}
		path := filepath.Join(cfg.TasksPath(), task.GenerateFilename(tk.ID, tk.Title))
		if err := task.Write(path, tk); err != nil {
			t.Fatal(err)
		}
	}

	b := NewBoard(cfg)
	b.SetNow(func() time.Time { return dragMoveTime })
	b.SetMouseEnabled(true)
	b.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	_ = b.View()
	return b, cfg
}

func columnTargetForStatus(t *testing.T, b *Board, status string) columnTarget {
	t.Helper()
	_ = b.View()
	for _, target := range b.layout.columns {
		if target.status == status {
			return target
		}
	}
	t.Fatalf("status %q has no rendered column target: %#v", status, b.layout.columns)
	return columnTarget{}
}

func dragTask(
	t *testing.T,
	b *Board,
	taskID int,
	status string,
	releaseButton tea.MouseButton,
	withMotion bool,
) {
	t.Helper()
	source := targetForTask(t, b, taskID)
	destination := columnTargetForStatus(t, b, status)
	sourceX, sourceY := source.rect.x0+1, source.rect.y0
	destinationX, destinationY := destination.rect.x0+1, destination.rect.y0

	_, _ = b.Update(tea.MouseMsg{
		X: sourceX, Y: sourceY,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	if withMotion {
		_, _ = b.Update(tea.MouseMsg{
			X: destinationX, Y: destinationY,
			Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
		})
		_ = b.View()
	}
	_, _ = b.Update(tea.MouseMsg{
		X: destinationX, Y: destinationY,
		Button: releaseButton, Action: tea.MouseActionRelease,
	})
	_ = b.View()
}

func readDragTask(t *testing.T, cfg *config.Config, id int) *task.Task {
	t.Helper()
	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := task.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func TestMouseDragDirectReleaseMovesWithSGRAndX10(t *testing.T) {
	for _, tt := range []struct {
		name          string
		releaseButton tea.MouseButton
	}{
		{name: "SGR", releaseButton: tea.MouseButtonLeft},
		{name: "X10", releaseButton: tea.MouseButtonNone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b, cfg := newDragFilesystemBoard(t, nil, &task.Task{
				ID: 1, Title: "Drag me", Status: dragStatusBacklog,
			})
			dragTask(t, b, 1, dragStatusTodo, tt.releaseButton, false)

			moved := readDragTask(t, cfg, 1)
			if moved.Status != dragStatusTodo {
				t.Fatalf("status=%q, want todo", moved.Status)
			}
			if selected := b.selectedTask(); selected == nil || selected.ID != 1 {
				t.Fatalf("moved task not selected: %#v", selected)
			}
			if b.currentColumn() == nil || b.currentColumn().status != dragStatusTodo {
				t.Fatalf("active destination column=%#v, want todo", b.currentColumn())
			}
		})
	}
}

func TestMouseDragMovesTaskUnderPointerNotKeyboardSelection(t *testing.T) {
	b, cfg := newDragFilesystemBoard(t, nil,
		&task.Task{ID: 1, Title: "Keyboard selection", Status: dragStatusBacklog, Priority: "high"},
		&task.Task{ID: 2, Title: "Pointer selection", Status: dragStatusBacklog, Priority: "low"},
	)
	if selected := b.selectedTask(); selected == nil || selected.ID != 1 {
		t.Fatalf("initial keyboard selection=%#v, want task #1", selected)
	}

	dragTask(t, b, 2, dragStatusTodo, tea.MouseButtonLeft, false)
	if got := readDragTask(t, cfg, 1).Status; got != dragStatusBacklog {
		t.Fatalf("keyboard-selected task moved to %q", got)
	}
	if got := readDragTask(t, cfg, 2).Status; got != dragStatusTodo {
		t.Fatalf("pointer task status=%q, want todo", got)
	}
	if selected := b.selectedTask(); selected == nil || selected.ID != 2 {
		t.Fatalf("moved pointer task not selected: %#v", selected)
	}
}

func TestMouseDragWholeRenderedDestinationColumnIsDropZone(t *testing.T) {
	tests := []struct {
		name            string
		destinationTask bool
		dropY           int
	}{
		{name: "header", dropY: 0},
		{name: "empty placeholder", dropY: 1},
		{name: "visible empty area", dropY: 10},
		{name: "destination card", destinationTask: true, dropY: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []*task.Task{{ID: 1, Title: "Move me", Status: dragStatusBacklog, Priority: "high"}}
			if tt.destinationTask {
				tasks = append(tasks, &task.Task{
					ID: 2, Title: "Existing destination card", Status: dragStatusTodo, Priority: "medium",
				})
			}
			b, cfg := newDragFilesystemBoard(t, nil, tasks...)
			source := targetForTask(t, b, 1)
			destination := columnTargetForStatus(t, b, dragStatusTodo)

			_, _ = b.Update(tea.MouseMsg{
				X: source.rect.x0 + 1, Y: source.rect.y0,
				Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
			})
			_, _ = b.Update(tea.MouseMsg{
				X: destination.rect.x0 + 1, Y: tt.dropY,
				Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
			})

			if got := readDragTask(t, cfg, 1).Status; got != dragStatusTodo {
				t.Fatalf("drop at y=%d changed status to %q", tt.dropY, got)
			}
		})
	}
}

func TestMouseDragMotionShowsDestinationAndMoveHint(t *testing.T) {
	b, _ := newDragFilesystemBoard(t, nil, &task.Task{
		ID: 1, Title: "Drag me", Status: dragStatusBacklog,
	})
	source := targetForTask(t, b, 1)
	destination := columnTargetForStatus(t, b, dragStatusTodo)

	_, _ = b.Update(tea.MouseMsg{
		X: source.rect.x0 + 1, Y: source.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	_, _ = b.Update(tea.MouseMsg{
		X: destination.rect.x0 + 1, Y: destination.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	view := b.View()

	if !strings.Contains(view, "→ todo") {
		t.Fatalf("destination header is not highlighted:\n%s", view)
	}
	if !strings.Contains(view, "Move #1 → todo — release to move") {
		t.Fatalf("drag move hint missing:\n%s", view)
	}
	if strings.Count(view, "#1") != 2 {
		t.Fatalf("expected one card and one status hint, got %d #1 occurrences:\n%s",
			strings.Count(view, "#1"), view)
	}
}

func TestMouseDragCanceledReleaseZonesDoNotMutate(t *testing.T) {
	tests := []struct {
		name string
		act  func(t *testing.T, b *Board)
	}{
		{
			name: "source column outside original card",
			act: func(t *testing.T, b *Board) {
				source := targetForTask(t, b, 1)
				_, _ = b.Update(tea.MouseMsg{
					X: source.rect.x0 + 1, Y: source.rect.y0,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
				})
				_, _ = b.Update(tea.MouseMsg{
					X: source.rect.x0 + 1, Y: 0,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
				})
			},
		},
		{
			name: "leave and return to source card",
			act: func(t *testing.T, b *Board) {
				source := targetForTask(t, b, 1)
				destination := columnTargetForStatus(t, b, dragStatusTodo)
				_, _ = b.Update(tea.MouseMsg{
					X: source.rect.x0 + 1, Y: source.rect.y0,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
				})
				_, _ = b.Update(tea.MouseMsg{
					X: destination.rect.x0 + 1, Y: destination.rect.y0,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
				})
				_, _ = b.Update(tea.MouseMsg{
					X: source.rect.x0 + 1, Y: source.rect.y0,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
				})
				_, _ = b.Update(tea.MouseMsg{
					X: source.rect.x0 + 1, Y: source.rect.y0,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
				})
			},
		},
		{
			name: "footer",
			act: func(t *testing.T, b *Board) {
				pressTask(t, b)
				_, _ = b.Update(tea.MouseMsg{
					X: 1, Y: b.height - 1,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
				})
			},
		},
		{
			name: "error line",
			act: func(t *testing.T, b *Board) {
				b.err = errors.New("existing error")
				_ = b.View()
				pressTask(t, b)
				_, _ = b.Update(tea.MouseMsg{
					X: 1, Y: b.height - 2,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
				})
			},
		},
		{
			name: "outside board",
			act: func(t *testing.T, b *Board) {
				pressTask(t, b)
				_, _ = b.Update(tea.MouseMsg{
					X: -1, Y: -1,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
				})
			},
		},
		{
			name: "unused right side",
			act: func(t *testing.T, b *Board) {
				b.Update(tea.WindowSizeMsg{Width: 300, Height: 40})
				_ = b.View()
				pressTask(t, b)
				_, _ = b.Update(tea.MouseMsg{
					X: 299, Y: 1,
					Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, cfg := newDragFilesystemBoard(t, nil, &task.Task{
				ID: 1, Title: "Stay put", Status: dragStatusBacklog,
			})
			tt.act(t, b)
			if got := readDragTask(t, cfg, 1).Status; got != dragStatusBacklog {
				t.Fatalf("canceled drag changed status to %q", got)
			}
		})
	}
}

func pressTask(t *testing.T, b *Board) {
	t.Helper()
	source := targetForTask(t, b, 1)
	_, _ = b.Update(tea.MouseMsg{
		X: source.rect.x0 + 1, Y: source.rect.y0,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
}

func TestMouseDragHiddenAndArchivedStatusesAreNotTargets(t *testing.T) {
	b, cfg := newDragFilesystemBoard(t, func(cfg *config.Config) {
		cfg.TUI.HideEmptyColumns = true
	}, &task.Task{ID: 1, Title: "Stay put", Status: dragStatusBacklog})
	if len(b.layout.columns) != 1 || b.layout.columns[0].status != dragStatusBacklog {
		t.Fatalf("hidden-column layout=%#v, want only backlog", b.layout.columns)
	}

	pressTask(t, b)
	_, _ = b.Update(tea.MouseMsg{
		X: 80, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
	})
	if got := readDragTask(t, cfg, 1).Status; got != dragStatusBacklog {
		t.Fatalf("hidden/archived drop changed status to %q", got)
	}
	for _, col := range b.layout.columns {
		if col.status == config.ArchivedStatus {
			t.Fatal("archived status was rendered as a drop target")
		}
	}
}

func TestMouseDragCanceledByResizeReloadAndDifferentButton(t *testing.T) {
	tests := []struct {
		name string
		act  func(b *Board)
	}{
		{
			name: "resize",
			act: func(b *Board) {
				b.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			},
		},
		{
			name: "reload",
			act: func(b *Board) {
				b.Update(ReloadMsg{})
			},
		},
		{
			name: "sort change",
			act: func(b *Board) {
				b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
			},
		},
		{
			name: "search change",
			act: func(b *Board) {
				b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
			},
		},
		{
			name: "view change",
			act: func(b *Board) {
				b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			},
		},
		{
			name: "wheel",
			act: func(b *Board) {
				b.Update(tea.MouseMsg{
					X: 1, Y: 1, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
				})
			},
		},
		{
			name: "different button",
			act: func(b *Board) {
				b.Update(tea.MouseMsg{
					X: 25, Y: 1, Button: tea.MouseButtonRight, Action: tea.MouseActionPress,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, cfg := newDragFilesystemBoard(t, nil, &task.Task{
				ID: 1, Title: "Stay put", Status: dragStatusBacklog,
			})
			pressTask(t, b)
			tt.act(b)
			_ = b.View()
			_, _ = b.Update(tea.MouseMsg{
				X: 25, Y: 0,
				Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
			})
			if got := readDragTask(t, cfg, 1).Status; got != dragStatusBacklog {
				t.Fatalf("%s cancellation changed status to %q", tt.name, got)
			}
		})
	}
}

func TestMouseDragSuccessfulMoveAppliesClaimTimestampsAndActivity(t *testing.T) {
	b, cfg := newDragFilesystemBoard(t, nil, &task.Task{
		ID: 1, Title: "Claim and move", Status: dragStatusBacklog,
	})
	dragTask(t, b, 1, "in-progress", tea.MouseButtonLeft, true)

	moved := readDragTask(t, cfg, 1)
	if moved.Status != "in-progress" {
		t.Fatalf("status=%q, want in-progress", moved.Status)
	}
	if moved.ClaimedBy != tuiClaimant() {
		t.Fatalf("claimed_by=%q, want %q", moved.ClaimedBy, tuiClaimant())
	}
	if moved.ClaimedAt == nil || !moved.ClaimedAt.Equal(dragMoveTime) {
		t.Fatalf("claimed_at=%v, want %v", moved.ClaimedAt, dragMoveTime)
	}
	if !moved.Updated.Equal(dragMoveTime) {
		t.Fatalf("updated=%v, want %v", moved.Updated, dragMoveTime)
	}
	if moved.Started == nil {
		t.Fatal("move out of backlog did not set started timestamp")
	}

	entries, err := board.ReadLog(cfg.Dir(), board.LogFilterOptions{
		Action: "move",
		TaskID: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Detail != "backlog -> in-progress" {
		t.Fatalf("move activity entries=%#v", entries)
	}
}

func TestMouseDragMoveToDoneSetsCompletionTimestamp(t *testing.T) {
	b, cfg := newDragFilesystemBoard(t, nil, &task.Task{
		ID: 1, Title: "Complete me", Status: dragStatusBacklog,
	})
	dragTask(t, b, 1, "done", tea.MouseButtonLeft, false)
	moved := readDragTask(t, cfg, 1)
	if moved.Completed == nil || moved.Started == nil {
		t.Fatalf("direct move to done timestamps: started=%v completed=%v", moved.Started, moved.Completed)
	}
}

func TestMouseDragRejectedMovesReloadAndKeepTaskSelected(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		configure func(*config.Config)
		tasks     []*task.Task
		target    string
		errorText string
	}{
		{
			name: "column WIP",
			configure: func(cfg *config.Config) {
				cfg.WIPLimits = map[string]int{dragStatusTodo: 1}
			},
			tasks: []*task.Task{
				{ID: 1, Title: "Blocked by WIP", Status: dragStatusBacklog},
				{ID: 2, Title: "Occupies todo", Status: dragStatusTodo},
			},
			target:    dragStatusTodo,
			errorText: "WIP limit",
		},
		{
			name: "class WIP",
			tasks: []*task.Task{
				{ID: 1, Title: "Second expedite", Status: dragStatusBacklog, Class: "expedite"},
				{ID: 2, Title: "Existing expedite", Status: dragStatusTodo, Class: "expedite"},
			},
			target:    dragStatusTodo,
			errorText: "expedite WIP",
		},
		{
			name: "claim conflict",
			tasks: []*task.Task{
				{
					ID: 1, Title: "Claimed elsewhere", Status: dragStatusBacklog,
					ClaimedBy: "other-agent", ClaimedAt: &now,
				},
			},
			target:    dragStatusTodo,
			errorText: "claimed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, cfg := newDragFilesystemBoard(t, tt.configure, tt.tasks...)
			dragTask(t, b, 1, tt.target, tea.MouseButtonLeft, true)

			if got := readDragTask(t, cfg, 1).Status; got != dragStatusBacklog {
				t.Fatalf("rejected move changed status to %q", got)
			}
			if b.err == nil || !strings.Contains(strings.ToLower(b.err.Error()), strings.ToLower(tt.errorText)) {
				t.Fatalf("error=%v, want text %q", b.err, tt.errorText)
			}
			if selected := b.selectedTask(); selected == nil || selected.ID != 1 {
				t.Fatalf("rejected move did not keep task selected: %#v", selected)
			}
			// The model remains responsive after the error reload.
			b.handleNavigation(keyRight)
			if b.activeCol != 1 {
				t.Fatalf("keyboard navigation unresponsive after error, activeCol=%d", b.activeCol)
			}
		})
	}
}
