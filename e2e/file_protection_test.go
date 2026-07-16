package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// File protection for claimed tasks — OS-level chmod enforcement
//
// When a task is claimed via the CLI, its file should be made read-only
// (0o444) so that external agents cannot modify it directly. The CLI
// itself elevates permissions temporarily when writing claimed files.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Core: Claiming makes file read-only, releasing restores write
// ---------------------------------------------------------------------------

func TestFileProtection_ClaimMakesFileReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Protect me", "--status", statusTodo)

	// Claim the task via pick.
	runKanban(t, kanbanDir, "pick", "--claim", claimTestAgent, "--json")

	// Verify the file is now read-only (0o444).
	taskPath := filepath.Join(kanbanDir, "tasks", "001-protect-me.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat task file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("claimed file should be read-only, got permissions %o", perm)
	}
}

func TestFileProtection_ReleaseMakesFileWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Release me", "--status", statusTodo)

	// Claim then release (--release bypasses claim checks, no --claim needed).
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)
	runKanban(t, kanbanDir, "edit", "1", "--release")

	// Verify the file is writable again (0o600).
	taskPath := filepath.Join(kanbanDir, "tasks", "001-release-me.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat task file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 == 0 {
		t.Errorf("released file should be writable, got permissions %o", perm)
	}
}

func TestFileProtection_CreateWithClaimIsReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Create a task with --claim.
	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "create", "Claimed at birth", "--claim", claimTestAgent)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-claimed-at-birth.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat task file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("file created with --claim should be read-only, got permissions %o", perm)
	}
}

func TestFileProtection_CreateWithoutClaimIsWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No claim")

	taskPath := filepath.Join(kanbanDir, "tasks", "001-no-claim.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat task file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 == 0 {
		t.Errorf("unclaimed file should be writable, got permissions %o", perm)
	}
}

// ---------------------------------------------------------------------------
// CLI can still modify claimed files (owner operations)
// ---------------------------------------------------------------------------

func TestFileProtection_OwnerCanEditClaimedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Owner edit", "--status", statusTodo)

	// Claim and move to in-progress.
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Owner edits the task — CLI should handle chmod dance internally.
	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--priority", priorityHigh, "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("owner edit failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Priority != priorityHigh {
		t.Errorf("priority = %q, want %q", task.Priority, priorityHigh)
	}

	// File should still be read-only after the edit.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-owner-edit.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat task file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("claimed file should remain read-only after owner edit, got %o", perm)
	}
}

func TestFileProtection_OwnerCanMoveClaimedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Owner move", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Move from in-progress → review.
	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", statusReview, "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("owner move failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Status != statusReview {
		t.Errorf("status = %q, want %q", task.Status, statusReview)
	}

	// File should still be read-only (still claimed).
	taskPath := filepath.Join(kanbanDir, "tasks", "001-owner-move.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat task file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("claimed file should remain read-only after owner move, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Direct file modification is blocked for claimed files
// ---------------------------------------------------------------------------

func TestFileProtection_DirectWriteToClaimedFileBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Direct write test", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Simulate an external agent trying to write directly.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-direct-write-test.md")
	err := os.WriteFile(taskPath, []byte("rogue content"), 0o600)
	if err == nil {
		t.Fatal("direct write to claimed file should fail with permission denied")
	}
	if !os.IsPermission(err) {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestFileProtection_DirectReadOfClaimedFileAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Readable claimed", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Reading should still work (444 allows read for everyone).
	taskPath := filepath.Join(kanbanDir, "tasks", "001-readable-claimed.md")
	data, err := os.ReadFile(taskPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading claimed file should succeed: %v", err)
	}
	if !strings.Contains(string(data), "claimed_by:") {
		t.Error("claimed file should still be readable with claim data")
	}
}

// ---------------------------------------------------------------------------
// Title rename preserves protection
// ---------------------------------------------------------------------------

func TestFileProtection_TitleRenamePreservesProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Original title", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Rename the task (triggers WriteAndRename).
	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--title", "Renamed title", "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("rename failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Old file should be gone.
	oldPath := filepath.Join(kanbanDir, "tasks", "001-original-title.md")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old file should be removed after rename")
	}

	// New file should be read-only.
	newPath := filepath.Join(kanbanDir, "tasks", "001-renamed-title.md")
	info, err := os.Stat(newPath)
	if err != nil {
		t.Fatalf("stat renamed file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("renamed claimed file should be read-only, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Expired claim restores writability
// ---------------------------------------------------------------------------

func TestFileProtection_ExpiredClaimFileBecomesWritableOnAccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Manually write a claimed file that's read-only but has an expired claim.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired protection
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Make the file read-only to simulate a previously-claimed state.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-expired-protection.md")
	if err := os.Chmod(taskPath, 0o444); err != nil { //nolint:gosec // intentionally set read-only for test
		t.Fatalf("chmod: %v", err)
	}

	// A read-only CLI operation (show) should detect the expired claim
	// and restore writability via self-healing.
	r := runKanban(t, kanbanDir, "show", "1", "--json")
	if r.exitCode != 0 {
		t.Fatalf("show with expired+readonly file failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// File should be writable now since the claim is expired.
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 == 0 {
		t.Errorf("expired claim file should be writable after self-healing, got %o", perm)
	}
}

func TestFileProtection_ExpiredClaimFileReprotectedOnNewClaim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Write a task with an expired claim.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Reclaim expired
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// A new agent claims — expired claim should be overridden, file becomes read-only.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", "new-agent", "--json")
	if r.exitCode != 0 {
		t.Fatalf("move with expired claim failed (exit %d): %s", r.exitCode, r.stderr)
	}

	taskPath := filepath.Join(kanbanDir, "tasks", "001-reclaim-expired.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("newly claimed file should be read-only, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Self-healing: claimed file with wrong permissions gets fixed on read
// ---------------------------------------------------------------------------

func TestFileProtection_SelfHealingOnCLIRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Create a claimed task file with writable permissions (simulates git pull).
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Git pulled task
status: in-progress
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: some-agent
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// File is writable (0o600 from writeTaskFile). This is "wrong" for a claimed file.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-git-pulled-task.md")

	// A CLI read operation (list, show) should detect the mismatch and fix it.
	r := runKanban(t, kanbanDir, "show", "1", "--json")
	if r.exitCode != 0 {
		t.Fatalf("show failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// After the CLI read, the file should now be 444.
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("self-healing should have made claimed file read-only, got %o", perm)
	}
}

func TestFileProtection_SelfHealingListFixesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Two claimed tasks, both with wrong (writable) permissions.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Task one
status: in-progress
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-a
claimed_at: 2099-01-01T00:00:00Z
---
`)
	writeTaskFile(t, kanbanDir, 2, `---
id: 2
title: Task two
status: in-progress
priority: medium
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-b
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 3)

	// Run list to trigger self-healing.
	r := runKanban(t, kanbanDir, "list", "--json")
	if r.exitCode != 0 {
		t.Fatalf("list failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Both files should now be read-only.
	for _, name := range []string{"001-task-one.md", "002-task-two.md"} {
		taskPath := filepath.Join(kanbanDir, "tasks", name)
		info, err := os.Stat(taskPath)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		perm := info.Mode().Perm()
		if perm&0o200 != 0 {
			t.Errorf("%s: self-healing should make claimed file read-only, got %o", name, perm)
		}
	}
}

// ---------------------------------------------------------------------------
// Unclaimed file with read-only permissions gets fixed (reverse self-heal)
// ---------------------------------------------------------------------------

func TestFileProtection_SelfHealingUnclaimedReadOnlyGetFixed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Write an unclaimed task file, then manually make it read-only.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Orphan readonly
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-orphan-readonly.md")
	if err := os.Chmod(taskPath, 0o444); err != nil { //nolint:gosec // intentionally set read-only for test
		t.Fatalf("chmod: %v", err)
	}

	// CLI should detect the mismatch (unclaimed but read-only) and fix.
	r := runKanban(t, kanbanDir, "show", "1", "--json")
	if r.exitCode != 0 {
		t.Fatalf("show failed (exit %d): %s", r.exitCode, r.stderr)
	}

	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 == 0 {
		t.Errorf("unclaimed file should be writable after self-healing, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Pick with --claim sets protection on the picked task
// ---------------------------------------------------------------------------

func TestFileProtection_PickWithClaimSetsProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Pickable", "--status", statusTodo)

	r := runKanban(t, kanbanDir, "pick", "--claim", claimTestAgent, "--json")
	if r.exitCode != 0 {
		t.Fatalf("pick failed (exit %d): %s", r.exitCode, r.stderr)
	}

	taskPath := filepath.Join(kanbanDir, "tasks", "001-pickable.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("picked+claimed file should be read-only, got %o", perm)
	}
}

func TestFileProtection_PickWithMoveAndClaimSetsProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Pick and move", "--status", statusTodo)

	var picked taskJSON
	r := runKanbanJSON(t, kanbanDir, &picked, "pick", "--claim", claimTestAgent, "--move", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("pick --move failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if picked.Status != statusInProgress {
		t.Errorf("status = %q, want %q", picked.Status, statusInProgress)
	}

	taskPath := filepath.Join(kanbanDir, "tasks", "001-pick-and-move.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("picked+moved+claimed file should be read-only, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Move with --claim on unclaimed task sets protection
// ---------------------------------------------------------------------------

func TestFileProtection_MoveWithClaimSetsProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Move claim", "--status", statusTodo)

	// File starts writable (unclaimed).
	taskPath := filepath.Join(kanbanDir, "tasks", "001-move-claim.md")
	infoBefore, _ := os.Stat(taskPath)
	if infoBefore.Mode().Perm()&0o200 == 0 {
		t.Fatal("precondition: unclaimed file should be writable")
	}

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("file should be read-only after move --claim, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Delete of claimed file works (CLI handles chmod before delete)
// ---------------------------------------------------------------------------

func TestFileProtection_DeleteAfterReleaseWorks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Delete me", "--status", statusTodo)

	// Claim, then release (delete doesn't have --claim flag).
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Verify it's read-only while claimed.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-delete-me.md")
	info, _ := os.Stat(taskPath)
	if info.Mode().Perm()&0o200 != 0 {
		t.Fatal("precondition: claimed file should be read-only")
	}

	// Release first, then delete.
	runKanban(t, kanbanDir, "edit", "1", "--release")
	r := runKanban(t, kanbanDir, "delete", "1", "--yes")
	if r.exitCode != 0 {
		t.Fatalf("delete after release failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Batch operations preserve file protection
// ---------------------------------------------------------------------------

func TestFileProtection_BatchMoveWithClaimSetsProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Batch one", "--status", statusTodo)
	mustCreateTask(t, kanbanDir, "Batch two", "--status", statusTodo)

	r := runKanban(t, kanbanDir, "--json", "move", "1,2", "in-progress", "--claim", claimTestAgent)

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}
	for _, res := range results {
		if !res.OK {
			t.Errorf("batch move failed for task %d: %s", res.ID, res.Error)
		}
	}

	// Both files should be read-only.
	for _, name := range []string{"001-batch-one.md", "002-batch-two.md"} {
		taskPath := filepath.Join(kanbanDir, "tasks", name)
		info, err := os.Stat(taskPath)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		perm := info.Mode().Perm()
		if perm&0o200 != 0 {
			t.Errorf("%s: batch-claimed file should be read-only, got %o", name, perm)
		}
	}
}

// ---------------------------------------------------------------------------
// Full lifecycle: claim → edit → move → release → writable
// ---------------------------------------------------------------------------

func TestFileProtection_FullLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Lifecycle", "--status", statusTodo)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-lifecycle.md")
	assertPerm := func(step string, wantReadOnly bool) {
		t.Helper()
		info, err := os.Stat(taskPath)
		if err != nil {
			t.Fatalf("%s: stat: %v", step, err)
		}
		isReadOnly := info.Mode().Perm()&0o200 == 0
		if isReadOnly != wantReadOnly {
			t.Errorf("%s: read-only = %v, want %v (perm %o)", step, isReadOnly, wantReadOnly, info.Mode().Perm())
		}
	}

	// 1. Unclaimed — writable.
	assertPerm("initial", false)

	// 2. Claim via move to in-progress — read-only.
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)
	assertPerm("after claim", true)

	// 3. Owner edits — still read-only after.
	runKanban(t, kanbanDir, "edit", "1", "--priority", priorityHigh, "--claim", claimTestAgent)
	assertPerm("after edit", true)

	// 4. Owner moves to review — still read-only.
	runKanban(t, kanbanDir, "move", "1", statusReview, "--claim", claimTestAgent)
	assertPerm("after move to review", true)

	// 5. Release — writable (--release bypasses claim checks).
	runKanban(t, kanbanDir, "edit", "1", "--release")
	assertPerm("after release", false)

	// 6. Re-claim — read-only again.
	runKanban(t, kanbanDir, "edit", "1", "--claim", "new-agent")
	assertPerm("after re-claim", true)
}

// ---------------------------------------------------------------------------
// Edge case: CLI show/list on read-only file works
// ---------------------------------------------------------------------------

func TestFileProtection_ShowWorksOnReadOnlyFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Show test", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// show should work fine on a read-only file.
	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show on claimed/read-only file failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.ClaimedBy != claimTestAgent {
		t.Errorf("claimed_by = %q, want %q", task.ClaimedBy, claimTestAgent)
	}
}

func TestFileProtection_ListWorksWithReadOnlyFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Listed", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// list should work fine.
	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list")
	if r.exitCode != 0 {
		t.Fatalf("list with read-only files failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

// ---------------------------------------------------------------------------
// Edge case: multiple claims/releases cycle permissions correctly
// ---------------------------------------------------------------------------

func TestFileProtection_RepeatedClaimReleaseCycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Cycling", "--status", statusTodo)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-cycling.md")

	for i := 0; i < 3; i++ {
		// Claim → read-only.
		runKanban(t, kanbanDir, "edit", "1", "--claim", claimTestAgent)
		info, _ := os.Stat(taskPath)
		if info.Mode().Perm()&0o200 != 0 {
			t.Errorf("cycle %d: claimed file should be read-only, got %o", i, info.Mode().Perm())
		}

		// Release → writable (--release bypasses claim checks).
		runKanban(t, kanbanDir, "edit", "1", "--release")
		info, _ = os.Stat(taskPath)
		if info.Mode().Perm()&0o200 == 0 {
			t.Errorf("cycle %d: released file should be writable, got %o", i, info.Mode().Perm())
		}
	}
}

// ---------------------------------------------------------------------------
// Edge case: handoff between agents transitions protection
// ---------------------------------------------------------------------------

func TestFileProtection_HandoffBetweenAgentsKeepsProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Handoff", "--status", statusTodo)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-handoff.md")

	// Agent A claims.
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", "agent-a")

	info, _ := os.Stat(taskPath)
	if info.Mode().Perm()&0o200 != 0 {
		t.Fatal("precondition: claimed by agent-a should be read-only")
	}

	// Agent A releases (--release bypasses claim checks).
	runKanban(t, kanbanDir, "edit", "1", "--release")
	info, _ = os.Stat(taskPath)
	if info.Mode().Perm()&0o200 == 0 {
		t.Fatal("after release should be writable")
	}

	// Agent B claims.
	runKanban(t, kanbanDir, "edit", "1", "--claim", "agent-b")
	info, _ = os.Stat(taskPath)
	if info.Mode().Perm()&0o200 != 0 {
		t.Error("claimed by agent-b should be read-only")
	}
}

// ---------------------------------------------------------------------------
// Handoff command: writes to claimed task files
// ---------------------------------------------------------------------------

func TestFileProtection_HandoffCommandPreservesProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Handoff task", "--status", statusTodo)

	// Claim and move to in-progress.
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-handoff-task.md")

	// Handoff moves to review, writes note — should work on read-only file.
	r := runKanban(t, kanbanDir, "handoff", "1", "--claim", claimTestAgent, "--note", "Done with this")
	if r.exitCode != 0 {
		t.Fatalf("handoff failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// File should still be read-only (still claimed after handoff without --release).
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("handed-off file should remain read-only (still claimed), got %o", perm)
	}
}

func TestFileProtection_HandoffWithReleaseRestoresWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Handoff release", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-handoff-release.md")

	// Handoff with --release should clear the claim and restore writability.
	r := runKanban(t, kanbanDir, "handoff", "1", "--claim", claimTestAgent, "--note", "Handing off", "--release")
	if r.exitCode != 0 {
		t.Fatalf("handoff --release failed (exit %d): %s", r.exitCode, r.stderr)
	}

	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 == 0 {
		t.Errorf("handoff with --release should make file writable, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Archive command: writes to task files (potentially claimed)
// ---------------------------------------------------------------------------

func TestFileProtection_ArchiveClaimedFileWorks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Archive me", "--status", statusTodo)

	// Claim and move to in-progress.
	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	taskPath := filepath.Join(kanbanDir, "tasks", "001-archive-me.md")
	info, _ := os.Stat(taskPath)
	if info.Mode().Perm()&0o200 != 0 {
		t.Fatal("precondition: claimed file should be read-only")
	}

	// The claimant should be able to archive the read-only claimed file.
	// The CLI must handle the chmod dance after claim validation succeeds.
	r := runKanban(t, kanbanDir, "archive", "1", "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("archive claimed file failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Verify the task was archived (still has claim, but status changed).
	var task taskJSON
	r2 := runKanbanJSON(t, kanbanDir, &task, "show", "1")
	if r2.exitCode != 0 {
		t.Fatalf("show failed: %s", r2.stderr)
	}
	if task.Status != statusArchived {
		t.Errorf("status = %q, want %q", task.Status, statusArchived)
	}
}

// ---------------------------------------------------------------------------
// Auto-repair: EnsureConsistency must handle read-only claimed files
// ---------------------------------------------------------------------------

func TestFileProtection_AutoRepairHandlesReadOnlyClaimedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Write a claimed task with a mismatched filename (ID 1 but filename says 005).
	// This triggers auto-repair's repairFilenameMismatches().
	tasksDir := filepath.Join(kanbanDir, "tasks")
	mismatchPath := filepath.Join(tasksDir, "005-wrong-name.md")
	content := `---
id: 1
title: Wrong name
status: in-progress
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: some-agent
claimed_at: 2099-01-01T00:00:00Z
---
`
	if err := os.WriteFile(mismatchPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing task: %v", err)
	}
	// Make it read-only (simulates a previously-claimed file).
	if err := os.Chmod(mismatchPath, 0o444); err != nil { //nolint:gosec // intentionally set read-only for test
		t.Fatalf("chmod: %v", err)
	}
	bumpNextID(t, kanbanDir, 2)

	// Any CLI command triggers loadConfig() → EnsureConsistency().
	// Auto-repair should be able to rename the file despite it being read-only.
	r := runKanban(t, kanbanDir, "list", "--json")
	if r.exitCode != 0 {
		t.Fatalf("list (with auto-repair) failed (exit %d): stderr=%s stdout=%s", r.exitCode, r.stderr, r.stdout)
	}

	// Verify the file was renamed correctly.
	correctPath := filepath.Join(tasksDir, "001-wrong-name.md")
	if _, err := os.Stat(correctPath); os.IsNotExist(err) {
		t.Error("auto-repair should have renamed file to 001-wrong-name.md")
	}

	// Verify the old file is gone.
	if _, err := os.Stat(mismatchPath); !os.IsNotExist(err) {
		t.Error("old mismatched file should be removed after repair")
	}

	// Repaired file should still be read-only (still claimed).
	info, err := os.Stat(correctPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("repaired claimed file should still be read-only, got %o", perm)
	}
}

func TestFileProtection_AutoRepairDuplicateIDOnReadOnlyFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Two tasks with the same ID — one claimed (read-only), one unclaimed.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed original
status: in-progress
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-a
claimed_at: 2099-01-01T00:00:00Z
---
`)
	// Make the claimed one read-only.
	claimedPath := filepath.Join(kanbanDir, "tasks", "001-claimed-original.md")
	if err := os.Chmod(claimedPath, 0o444); err != nil { //nolint:gosec // intentionally set read-only for test
		t.Fatalf("chmod: %v", err)
	}

	// Write a second file with the same ID 1 but different filename.
	dupPath := filepath.Join(kanbanDir, "tasks", "099-duplicate.md")
	dupContent := `---
id: 1
title: Duplicate task
status: todo
priority: low
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`
	if err := os.WriteFile(dupPath, []byte(dupContent), 0o600); err != nil {
		t.Fatalf("writing dup: %v", err)
	}
	bumpNextID(t, kanbanDir, 2)

	// Auto-repair should reassign the duplicate's ID without failing on the
	// read-only claimed file.
	r := runKanban(t, kanbanDir, "list", "--json")
	if r.exitCode != 0 {
		t.Fatalf("list (with auto-repair of duplicates) failed (exit %d): stderr=%s stdout=%s",
			r.exitCode, r.stderr, r.stdout)
	}

	// The claimed original should still exist unchanged.
	if _, err := os.Stat(claimedPath); os.IsNotExist(err) {
		t.Error("claimed original file should still exist")
	}
}

// ---------------------------------------------------------------------------
// ReadAllLenient: must not fail on read-only task files
// ---------------------------------------------------------------------------

func TestFileProtection_ReadAllLenientWorksWithReadOnlyFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)

	// Create a mix of claimed (read-only) and unclaimed (writable) tasks.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: in-progress
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-x
claimed_at: 2099-01-01T00:00:00Z
---
`)
	writeTaskFile(t, kanbanDir, 2, `---
id: 2
title: Unclaimed task
status: todo
priority: medium
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 3)

	// Make the claimed file read-only.
	claimedPath := filepath.Join(kanbanDir, "tasks", "001-claimed-task.md")
	if err := os.Chmod(claimedPath, 0o444); err != nil { //nolint:gosec // intentionally set read-only for test
		t.Fatalf("chmod: %v", err)
	}

	// Commands that use ReadAllLenient (list, pick, move WIP check) should work.
	var tasks []taskJSON
	r := runKanbanJSON(t, kanbanDir, &tasks, "list")
	if r.exitCode != 0 {
		t.Fatalf("list failed with mix of permissions (exit %d): %s", r.exitCode, r.stderr)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}

	// Pick should also work — it reads all tasks to find the best candidate.
	r = runKanban(t, kanbanDir, "pick", "--claim", "new-agent", "--json")
	if r.exitCode != 0 {
		t.Fatalf("pick with read-only files failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Edit --body on claimed file: body append goes through task.Write
// ---------------------------------------------------------------------------

func TestFileProtection_EditBodyOnClaimedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Body edit", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Edit body on a claimed (read-only) file.
	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--body", "New body content", "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("edit --body on claimed file failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Body != "New body content" {
		t.Errorf("body = %q, want %q", task.Body, "New body content")
	}

	// File should remain read-only.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-body-edit.md")
	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0o200 != 0 {
		t.Errorf("claimed file should remain read-only after body edit, got %o", perm)
	}
}

// ---------------------------------------------------------------------------
// Metrics/summary/context: read-only operations should not be affected
// ---------------------------------------------------------------------------

func TestFileProtection_MetricsWorksWithReadOnlyFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Metrics task", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Metrics (read-only command) should work with read-only files.
	r := runKanban(t, kanbanDir, "metrics")
	if r.exitCode != 0 {
		t.Fatalf("metrics with read-only files failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestFileProtection_BoardSummaryWorksWithReadOnlyFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board task", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// Board summary should work with read-only files.
	r := runKanban(t, kanbanDir, "board")
	if r.exitCode != 0 {
		t.Fatalf("board with read-only files failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

// ---------------------------------------------------------------------------
// Batch edit with --claim sets protection on all files
// ---------------------------------------------------------------------------

func TestFileProtection_BatchEditWithClaimSetsProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Batch edit one", "--status", statusTodo)
	mustCreateTask(t, kanbanDir, "Batch edit two", "--status", statusTodo)

	// Batch edit with --claim should protect both files.
	r := runKanban(t, kanbanDir, "--json", "edit", "1,2", "--claim", claimTestAgent)

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}
	for _, res := range results {
		if !res.OK {
			t.Errorf("batch edit failed for task %d: %s", res.ID, res.Error)
		}
	}

	for _, name := range []string{"001-batch-edit-one.md", "002-batch-edit-two.md"} {
		taskPath := filepath.Join(kanbanDir, "tasks", name)
		info, err := os.Stat(taskPath)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		perm := info.Mode().Perm()
		if perm&0o200 != 0 {
			t.Errorf("%s: batch-edited+claimed file should be read-only, got %o", name, perm)
		}
	}
}

// ---------------------------------------------------------------------------
// Batch handoff on claimed files
// ---------------------------------------------------------------------------

func TestFileProtection_BatchHandoffPreservesProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Batch handoff one", "--status", statusTodo)
	mustCreateTask(t, kanbanDir, "Batch handoff two", "--status", statusTodo)

	// Claim both and move to in-progress.
	runKanban(t, kanbanDir, "move", "1,2", "in-progress", "--claim", claimTestAgent)

	// Batch handoff should work on read-only files.
	r := runKanban(t, kanbanDir, "--json", "handoff", "1,2", "--claim", claimTestAgent, "--note", "Done")

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}
	for _, res := range results {
		if !res.OK {
			t.Errorf("batch handoff failed for task %d: %s", res.ID, res.Error)
		}
	}

	// Both files should still be read-only (still claimed).
	for _, name := range []string{"001-batch-handoff-one.md", "002-batch-handoff-two.md"} {
		taskPath := filepath.Join(kanbanDir, "tasks", name)
		info, err := os.Stat(taskPath)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		perm := info.Mode().Perm()
		if perm&0o200 != 0 {
			t.Errorf("%s: batch-handoff file should remain read-only, got %o", name, perm)
		}
	}
}

// ---------------------------------------------------------------------------
// Batch archive on claimed files
// ---------------------------------------------------------------------------

func TestFileProtection_BatchArchiveClaimedFilesWorks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Batch archive one", "--status", statusTodo)
	mustCreateTask(t, kanbanDir, "Batch archive two", "--status", statusTodo)

	// Claim both and move to in-progress.
	runKanban(t, kanbanDir, "move", "1,2", "in-progress", "--claim", claimTestAgent)

	// Verify precondition: both files are read-only.
	for _, name := range []string{"001-batch-archive-one.md", "002-batch-archive-two.md"} {
		taskPath := filepath.Join(kanbanDir, "tasks", name)
		info, _ := os.Stat(taskPath)
		if info.Mode().Perm()&0o200 != 0 {
			t.Fatalf("precondition: %s should be read-only", name)
		}
	}

	// Batch archive should handle read-only files when the claimant is supplied.
	r := runKanban(t, kanbanDir, "--json", "archive", "1,2", "--claim", claimTestAgent)

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}
	for _, res := range results {
		if !res.OK {
			t.Errorf("batch archive failed for task %d: %s", res.ID, res.Error)
		}
	}
}

// ---------------------------------------------------------------------------
// Direct file operations that bypass chmod (known limitations)
// ---------------------------------------------------------------------------

func TestFileProtection_DirectDeleteNotBlockedByChmod(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Deletable", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// On Unix, os.Remove works on 444 files because it's a directory operation.
	// This is a known limitation — chmod doesn't prevent file deletion.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-deletable.md")
	err := os.Remove(taskPath)
	if err != nil {
		// If this fails, chmod is stronger than expected on this OS — good!
		t.Skipf("os.Remove blocked on read-only file (unexpected but fine): %v", err)
	}
	// If we get here, the file was deleted despite being 444. This documents
	// the known limitation.
	if _, statErr := os.Stat(taskPath); !os.IsNotExist(statErr) {
		t.Error("file should have been deleted")
	}
}

func TestFileProtection_DirectRenameNotBlockedByChmod(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks differ on Windows")
	}
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Renamable", "--status", statusTodo)

	runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)

	// On Unix, os.Rename works on 444 files because it's a directory operation.
	// This is a known limitation.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-renamable.md")
	renamedPath := filepath.Join(kanbanDir, "tasks", "001-rogue-rename.md")
	err := os.Rename(taskPath, renamedPath)
	if err != nil {
		t.Skipf("os.Rename blocked on read-only file (unexpected but fine): %v", err)
	}
	if _, statErr := os.Stat(renamedPath); os.IsNotExist(statErr) {
		t.Error("file should have been renamed")
	}
}
