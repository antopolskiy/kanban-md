package board_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestMutationsRefuseUnsafeFrontmatterWithoutFilesOrLogs(t *testing.T) {
	operations := map[string]func(*config.Config) error{
		"title edit": func(cfg *config.Config) error {
			_, err := board.Edit(cfg, 1, "", false, func(tk *task.Task) (bool, error) {
				tk.Title = "Changed sample"
				return true, nil
			}, time.Now())
			return err
		},
		"claim": func(cfg *config.Config) error {
			_, err := board.Edit(cfg, 1, "agent-a", false, func(tk *task.Task) (bool, error) {
				now := time.Now()
				tk.ClaimedBy, tk.ClaimedAt = "agent-a", &now
				return true, nil
			}, time.Now())
			return err
		},
		"block": func(cfg *config.Config) error {
			_, err := board.Edit(cfg, 1, "", false, func(tk *task.Task) (bool, error) {
				tk.Blocked, tk.BlockReason = true, "waiting"
				return true, nil
			}, time.Now())
			return err
		},
		"unblock": func(cfg *config.Config) error {
			_, err := board.Edit(cfg, 1, "", false, func(tk *task.Task) (bool, error) {
				tk.Blocked, tk.BlockReason = false, ""
				return true, nil
			}, time.Now())
			return err
		},
		"release": func(cfg *config.Config) error {
			_, err := board.Edit(cfg, 1, "", true, func(tk *task.Task) (bool, error) {
				tk.ClaimedBy, tk.ClaimedAt = "", nil
				return true, nil
			}, time.Now())
			return err
		},
		"move": func(cfg *config.Config) error {
			_, err := board.Move(cfg, board.MoveParams{ID: 1, NewStatus: "in-progress", Claimant: "agent-a", SetClaim: true}, time.Now())
			return err
		},
		"archive": func(cfg *config.Config) error {
			_, err := board.Archive(cfg, 1, "", time.Now())
			return err
		},
		"delete": func(cfg *config.Config) error {
			_, err := board.Delete(cfg, 1, "", time.Now())
			return err
		},
		"handoff": func(cfg *config.Config) error {
			_, err := board.Handoff(cfg, board.HandoffParams{ID: 1, Claimant: "agent-a", Release: true, Note: "review note", BlockReason: "waiting"}, time.Now())
			return err
		},
		"pick and move": func(cfg *config.Config) error {
			_, _, _, err := board.PickAndClaim(cfg, board.PickAndClaimParams{Claimant: "agent-a", StatusFilter: "todo", MoveTarget: "in-progress"}, time.Now())
			return err
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			cfg, dir := setupMutateBoard(t)
			path := filepath.Join(cfg.TasksPath(), "001-generic-sample.md")
			content := []byte("---\nid: 1\ntitle: Generic sample\nstatus: todo\npriority: medium\ncreated: 2026-08-12T10:00:00Z\nupdated: 2026-08-12T10:00:00Z\nestimate: &shared 4h\ncustom_copy: *shared\n---\nOriginal body\n")
			if name == "release" {
				content = bytes.Replace(content, []byte("estimate:"), []byte("claimed_by: agent-a\nclaimed_at: "+time.Now().Format(time.RFC3339)+"\nestimate:"), 1)
			}
			if name == "unblock" {
				content = bytes.Replace(content, []byte("estimate:"), []byte("blocked: true\nblock_reason: waiting\nestimate:"), 1)
			}
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := operation(cfg); err == nil || !strings.Contains(err.Error(), "cannot preserve") {
				t.Fatalf("mutation error = %v, want preservation refusal", err)
			}
			written, err := os.ReadFile(path) //nolint:gosec // test-owned task path
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(written, content) {
				t.Errorf("refusal changed task:\n%s", written)
			}
			if _, err = os.Stat(filepath.Join(cfg.TasksPath(), "001-changed-sample.md")); !os.IsNotExist(err) {
				t.Errorf("refusal created rename target: %v", err)
			}
			if _, err = os.Stat(filepath.Join(dir, "activity.jsonl")); !os.IsNotExist(err) {
				t.Errorf("refusal wrote activity log: %v", err)
			}
		})
	}
}
