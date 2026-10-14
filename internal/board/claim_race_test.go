package board

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestPickAndClaim_ConcurrentClaimers(t *testing.T) {
	dir := t.TempDir()
	cfg := config.NewDefault(dir)
	cfg.SetDir(dir)
	if err := os.MkdirAll(cfg.TasksPath(), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := task.Write(filepath.Join(cfg.TasksPath(), "001-only.md"), &task.Task{
		ID: 1, Title: "only", Status: config.DefaultStatus,
	}); err != nil {
		t.Fatal(err)
	}

	var entered atomic.Int32
	release := make(chan struct{})
	var once sync.Once
	openGate := func() { once.Do(func() { close(release) }) }
	beforeClaimPersist = func() {
		if entered.Add(1) == 2 {
			openGate()
		}
		select {
		case <-release:
		case <-time.After(500 * time.Millisecond):
			openGate()
		}
	}
	t.Cleanup(func() { beforeClaimPersist = nil })

	type result struct {
		claimedBy string
		err       error
	}
	results := make([]result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for i, name := range []string{"agent-a", "agent-b"} {
		go func(i int, name string) {
			defer wg.Done()
			picked, _, _, err := PickAndClaim(cfg, PickAndClaimParams{Claimant: name}, time.Now())
			claimedBy := ""
			if picked != nil {
				claimedBy = picked.ClaimedBy
			}
			results[i] = result{claimedBy: claimedBy, err: err}
		}(i, name)
	}
	wg.Wait()

	wins := 0
	var winner string
	for _, r := range results {
		if r.err == nil {
			wins++
			winner = r.claimedBy
			continue
		}
		var ce *clierr.Error
		if !errors.As(r.err, &ce) || ce.Code != clierr.NothingToPick {
			t.Fatalf("loser error = %v, want NOTHING_TO_PICK", r.err)
		}
	}
	if wins != 1 {
		t.Fatalf("successful claims = %d, want 1", wins)
	}

	saved, err := task.Read(filepath.Join(cfg.TasksPath(), "001-only.md"))
	if err != nil {
		t.Fatal(err)
	}
	if saved.ClaimedBy != winner {
		t.Fatalf("saved claim = %q, want %q", saved.ClaimedBy, winner)
	}
}
