package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Claim timeout / enforcement e2e tests
// ---------------------------------------------------------------------------

// writeTaskFile writes a raw task markdown file into the tasks directory.
// The filename follows the convention: 001-<slug>.md (zero-padded ID + slug).
// The title is extracted from the YAML frontmatter.
func writeTaskFile(t *testing.T, kanbanDir string, id int, content string) {
	t.Helper()
	// Extract title from frontmatter for the slug.
	slug := "task"
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "title:") {
			slug = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			slug = strings.ToLower(slug)
			slug = strings.ReplaceAll(slug, " ", "-")
			break
		}
	}
	filename := fmt.Sprintf("%03d-%s.md", id, slug)
	taskPath := filepath.Join(kanbanDir, "tasks", filename)
	if err := os.WriteFile(taskPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing task file %d: %v", id, err)
	}
}

// setConfigClaimTimeout rewrites the claim_timeout in config.yml.

// setConfigClaimTimeout rewrites the claim_timeout in config.yml.
func setConfigClaimTimeout(t *testing.T, kanbanDir, timeout string) {
	t.Helper()
	cfgPath := filepath.Join(kanbanDir, "config.yml")
	data, err := os.ReadFile(cfgPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	// Replace existing claim_timeout line or append if missing.
	if strings.Contains(content, "claim_timeout:") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "claim_timeout:") {
				lines[i] = "claim_timeout: " + timeout
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		content += "claim_timeout: " + timeout + "\n"
	}
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

// bumpNextID updates next_id in config.yml so the CLI knows the next available ID.

// bumpNextID updates next_id in config.yml so the CLI knows the next available ID.
func bumpNextID(t *testing.T, kanbanDir string, nextID int) {
	t.Helper()
	cfgPath := filepath.Join(kanbanDir, "config.yml")
	data, err := os.ReadFile(cfgPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "next_id:") {
			lines[i] = "next_id: " + strconv.Itoa(nextID)
		}
	}
	content = strings.Join(lines, "\n")
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

func TestClaimBlocksOtherAgentMove(t *testing.T) {
	kanbanDir := initBoard(t)

	// Create a task and claim it by writing the file directly with a recent timestamp.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Another agent tries to move — should fail.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode == 0 {
		t.Fatal("move should fail when task is claimed by another agent")
	}
	if !strings.Contains(r.stderr, "claimed") && !strings.Contains(r.stdout, "claimed") {
		t.Errorf("error should mention claim, got stdout=%q stderr=%q", r.stdout, r.stderr)
	}
}

func TestClaimBlocksOtherAgentEdit(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Edit without matching --claim should fail.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low")
	if r.exitCode == 0 {
		t.Fatal("edit should fail when task is claimed by another agent")
	}
}

func TestClaimBlocksOtherAgentDelete(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Delete should fail when task is claimed.
	r := runKanban(t, kanbanDir, "delete", "1")
	if r.exitCode == 0 {
		t.Fatal("delete should fail when task is claimed by another agent")
	}
}

func TestExpiredClaimAllowsMove(t *testing.T) {
	kanbanDir := initBoard(t)
	// Default claim_timeout is 1h. Set claimed_at to 2 hours in the past.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Move should succeed — claim has expired. --claim required for in-progress.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("move should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestExpiredClaimAllowsEdit(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Edit should succeed — claim has expired.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low")
	if r.exitCode != 0 {
		t.Fatalf("edit should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestExpiredClaimAllowsDelete(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Delete requires --yes in non-TTY for confirmation bypass.
	r := runKanban(t, kanbanDir, "delete", "1", "--yes")
	if r.exitCode != 0 {
		t.Fatalf("delete should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestActiveClaimNotExpiredBeforeTimeout(t *testing.T) {
	kanbanDir := initBoard(t)
	// Use a long timeout (10h) and a claim from 1 minute ago.
	setConfigClaimTimeout(t, kanbanDir, "10h")

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Recently claimed
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-busy
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Should still be blocked.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode == 0 {
		t.Fatal("move should fail — claim has not expired yet")
	}
}

func TestSameAgentCanMoveClaimed(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Same agent test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Same agent can move with --claim flag matching the current claimant.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("same agent should be able to move, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestSameAgentCanEditClaimed(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Same agent edit test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Same agent (via --claim) can edit.
	r := runKanban(t, kanbanDir, "edit", "1", "--priority", "low", "--claim", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("same agent should be able to edit, got exit %d: %s", r.exitCode, r.stderr)
	}
}

func TestClaimBlocksJSONError(t *testing.T) {
	kanbanDir := initBoard(t)

	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: JSON error test
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-alpha
claimed_at: 2099-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Verify structured JSON error output.
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "in-progress")
	if errResp.Code != codeTaskClaimed {
		t.Errorf("error code = %q, want TASK_CLAIMED", errResp.Code)
	}
	if errResp.Details["claimed_by"] != "agent-alpha" {
		t.Errorf("details.claimed_by = %v, want %q", errResp.Details["claimed_by"], "agent-alpha")
	}
}

func TestPickSkipsClaimedTasks(t *testing.T) {
	kanbanDir := initBoard(t)

	// Task 1: claimed (should be skipped).
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Claimed task
status: todo
priority: critical
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-other
claimed_at: 2099-01-01T00:00:00Z
---
`)
	// Task 2: unclaimed (should be picked).
	writeTaskFile(t, kanbanDir, 2, `---
id: 2
title: Available task
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 3)

	r := runKanban(t, kanbanDir, "pick", "--claim", "agent-me", "--json")
	if r.exitCode != 0 {
		t.Fatalf("pick failed (exit %d): %s", r.exitCode, r.stderr)
	}

	var picked taskJSON
	if err := json.Unmarshal([]byte(r.stdout), &picked); err != nil {
		t.Fatalf("parsing pick output: %v\nstdout: %s", err, r.stdout)
	}

	if picked.ID != 2 {
		t.Errorf("pick selected task #%d, want #2 (should skip claimed #1)", picked.ID)
	}
}

func TestPickSelectsExpiredClaimTask(t *testing.T) {
	kanbanDir := initBoard(t)

	// Task 1: expired claim (should be available for pick).
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim
status: todo
priority: critical
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	r := runKanban(t, kanbanDir, "pick", "--claim", "agent-me", "--json")
	if r.exitCode != 0 {
		t.Fatalf("pick should find expired-claim task, got exit %d: %s", r.exitCode, r.stderr)
	}

	var picked taskJSON
	if err := json.Unmarshal([]byte(r.stdout), &picked); err != nil {
		t.Fatalf("parsing pick output: %v\nstdout: %s", err, r.stdout)
	}

	if picked.ID != 1 {
		t.Errorf("pick selected task #%d, want #1 (expired claim should be available)", picked.ID)
	}
}

func TestExpiredClaimClearedAfterMutation(t *testing.T) {
	kanbanDir := initBoard(t)

	// Expired claim — move should succeed and clear the claim fields.
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim cleared
status: todo
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// --claim required for in-progress; the expired claim is cleared and replaced.
	r := runKanban(t, kanbanDir, "move", "1", "in-progress", "--claim", claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("move should succeed for expired claim, got exit %d: %s", r.exitCode, r.stderr)
	}

	// Read the task back and verify the old claim is replaced by the new one.
	var tk taskJSON
	r = runKanbanJSON(t, kanbanDir, &tk, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// Verify the expired claim is replaced by the new claim.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-expired-claim-cleared.md")
	data, err := os.ReadFile(taskPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading task file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "agent-old") {
		t.Error("task file should not contain old claim agent after expired claim is cleared")
	}
	if !strings.Contains(content, claimTestAgent) {
		t.Error("task file should contain new claim agent after move with --claim")
	}
}

// ---------------------------------------------------------------------------
// Handoff command tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Require-claim enforcement tests
// ---------------------------------------------------------------------------

func TestRequireClaimMoveToInProgressWithoutClaimFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task")

	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "in-progress")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

func TestRequireClaimMoveToReviewWithoutClaimFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task")

	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "review")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

func TestRequireClaimMoveToNonClaimStatusSucceeds(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task")

	// backlog, todo, done — none require claim.
	for _, status := range []string{statusTodo, "done"} {
		r := runKanban(t, kanbanDir, "--json", "move", "1", status)
		if r.exitCode != 0 {
			t.Errorf("move to %q without --claim failed (exit %d): %s", status, r.exitCode, r.stderr)
		}
	}
}

func TestRequireClaimMoveWithClaimSucceeds(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "in-progress", "--claim", "agent-x")
	if r.exitCode != 0 {
		t.Fatalf("move with --claim failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Status != statusInProgress {
		t.Errorf("status = %q, want %q", task.Status, statusInProgress)
	}
	if task.ClaimedBy != "agent-x" {
		t.Errorf("claimed_by = %q, want %q", task.ClaimedBy, "agent-x")
	}
}

func TestRequireClaimEditInClaimStatusWithoutClaimFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task", "--status", "in-progress")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--title", "Changed")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

func TestRequireClaimEditInNonClaimStatusSucceeds(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--title", "Changed")
	if r.exitCode != 0 {
		t.Fatalf("edit without --claim in backlog failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Title != "Changed" {
		t.Errorf("title = %q, want %q", task.Title, "Changed")
	}
}

func TestRequireClaimEditStatusToClaimStatusWithoutClaimFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Test task")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--status", "in-progress")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

// ---------------------------------------------------------------------------
// Multi-agent identity verification
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Multi-agent identity verification
// ---------------------------------------------------------------------------

func TestRequireClaimWrongAgentCannotMoveClaimedTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Claimed task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", "agent-alpha")

	// agent-beta tries to move → should fail (task claimed by agent-alpha).
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "review", "--claim", "agent-beta")
	if errResp.Code != codeTaskClaimed {
		t.Errorf("code = %q, want TASK_CLAIMED", errResp.Code)
	}
}

func TestRequireClaimCorrectAgentCanMoveClaimedTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Claimed task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", "agent-alpha")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "review", "--claim", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("correct agent move failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Status != statusReview {
		t.Errorf("status = %q, want %q", task.Status, "review")
	}
}

func TestRequireClaimWrongAgentCannotEditClaimedTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Claimed task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", "agent-alpha")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--title", "Changed", "--claim", "agent-beta")
	if errResp.Code != codeTaskClaimed {
		t.Errorf("code = %q, want TASK_CLAIMED", errResp.Code)
	}
}

func TestRequireClaimCorrectAgentCanEditClaimedTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Claimed task")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", "agent-alpha")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--title", "Changed", "--claim", "agent-alpha")
	if r.exitCode != 0 {
		t.Fatalf("correct agent edit failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Title != "Changed" {
		t.Errorf("title = %q, want %q", task.Title, "Changed")
	}
}

// ---------------------------------------------------------------------------
// Claim preservation through lifecycle
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Claim preservation through lifecycle
// ---------------------------------------------------------------------------

func TestRequireClaimFullLifecyclePreservesClaim(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Lifecycle task")

	// Move through: backlog → in-progress → review → done.
	// Claim should persist at every step.
	const agent = "lifecycle-agent"

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "in-progress", "--claim", agent)
	if r.exitCode != 0 {
		t.Fatalf("move to in-progress failed: %s", r.stderr)
	}
	if task.ClaimedBy != agent {
		t.Errorf("after in-progress: claimed_by = %q, want %q", task.ClaimedBy, agent)
	}

	r = runKanbanJSON(t, kanbanDir, &task, "move", "1", "review", "--claim", agent)
	if r.exitCode != 0 {
		t.Fatalf("move to review failed: %s", r.stderr)
	}
	if task.ClaimedBy != agent {
		t.Errorf("after review: claimed_by = %q, want %q", task.ClaimedBy, agent)
	}

	r = runKanbanJSON(t, kanbanDir, &task, "move", "1", "done", "--claim", agent)
	if r.exitCode != 0 {
		t.Fatalf("move to done failed: %s", r.stderr)
	}
	if task.ClaimedBy != agent {
		t.Errorf("after done: claimed_by = %q, want %q", task.ClaimedBy, agent)
	}
}

func TestRequireClaimNotSilentlyClearedOnMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Preserved claim task")

	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", claimAgent1)
	runKanban(t, kanbanDir, "--json", "move", "1", "review", "--claim", claimAgent1)

	// Read the raw task file and verify claimed_by is present.
	taskPath := filepath.Join(kanbanDir, "tasks", "001-preserved-claim-task.md")
	data, err := os.ReadFile(taskPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading task file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "claimed_by: agent-1") {
		t.Error("claimed_by should be present in task file after move")
	}
	if !strings.Contains(content, "claimed_at:") {
		t.Error("claimed_at should be present in task file after move")
	}
}

func TestRequireClaimExplicitRelease(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Release test")

	// Claim during move.
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", claimAgent1)

	// Move to done — claim should persist.
	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "move", "1", "done", "--claim", claimAgent1)
	if task.ClaimedBy != claimAgent1 {
		t.Errorf("claimed_by after done = %q, want %q", task.ClaimedBy, claimAgent1)
	}

	// Only explicit release clears the claim.
	var released taskJSON
	r := runKanbanJSON(t, kanbanDir, &released, "edit", "1", "--release")
	if r.exitCode != 0 {
		t.Fatalf("edit --release failed (exit %d): stdout=%s stderr=%s", r.exitCode, r.stdout, r.stderr)
	}
	if released.ClaimedBy != "" {
		t.Errorf("claimed_by after release = %q, want empty", released.ClaimedBy)
	}
}

// ---------------------------------------------------------------------------
// --next/--prev with require_claim
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// --next/--prev with require_claim
// ---------------------------------------------------------------------------

func TestRequireClaimNextIntoClaimStatusFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Next test")
	runKanban(t, kanbanDir, "--json", "move", "1", statusTodo)

	// --next from todo → in-progress (requires claim).
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "--next")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

func TestRequireClaimNextWithClaimSucceeds(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Next test")
	runKanban(t, kanbanDir, "--json", "move", "1", statusTodo)

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "move", "1", "--next", "--claim", claimAgent1)
	if r.exitCode != 0 {
		t.Fatalf("--next with --claim failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if task.Status != statusInProgress {
		t.Errorf("status = %q, want %q", task.Status, statusInProgress)
	}
}

func TestRequireClaimPrevIntoClaimStatusFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Prev test")
	runKanban(t, kanbanDir, "--json", "move", "1", "done")

	// --prev from done → review (requires claim).
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "--prev")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

// ---------------------------------------------------------------------------
// Batch operations with require_claim
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Batch operations with require_claim
// ---------------------------------------------------------------------------

func TestRequireClaimBatchMoveWithoutClaimFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	r := runKanban(t, kanbanDir, "--json", "move", "1,2", "in-progress")

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}
	for i, res := range results {
		if res.OK {
			t.Errorf("results[%d].OK = true, want false (require_claim should block)", i)
		}
	}
}

func TestRequireClaimBatchMoveWithClaimSucceeds(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	r := runKanban(t, kanbanDir, "--json", "move", "1,2", "in-progress", "--claim", claimAgent1)

	var results []batchResultJSON
	if err := json.Unmarshal([]byte(r.stdout), &results); err != nil {
		t.Fatalf("parsing batch results: %v\nstdout: %s", err, r.stdout)
	}
	for i, res := range results {
		if !res.OK {
			t.Errorf("results[%d].OK = false, want true", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Config edge cases
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Config edge cases
// ---------------------------------------------------------------------------

func TestRequireClaimCustomStatusesNoRequire(t *testing.T) {
	// Board with custom statuses — none have require_claim.
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	runKanban(t, kanbanDir, "init", "--statuses", "open,active,closed")
	mustCreateTask(t, kanbanDir, "Custom task")

	// Move to "active" without --claim should succeed.
	r := runKanban(t, kanbanDir, "--json", "move", "1", "active")
	if r.exitCode != 0 {
		t.Fatalf("move to active without --claim failed in custom board (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestRequireClaimNewBoardDefaults(t *testing.T) {
	kanbanDir := initBoard(t)

	// Read the generated config and verify require_claim defaults.
	configPath := filepath.Join(kanbanDir, "config.yml")
	data, err := os.ReadFile(configPath) //nolint:gosec // e2e test file
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "require_claim: true") {
		t.Error("default config should contain 'require_claim: true'")
	}
}

// ---------------------------------------------------------------------------
// Idempotent move and error message quality
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Idempotent move and error message quality
// ---------------------------------------------------------------------------

func TestRequireClaimIdempotentMoveSucceeds(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Idempotent test", "--status", "in-progress")

	// Task is already at in-progress. Moving again is idempotent — no actual change.
	// This should succeed without --claim because no mutation occurs.
	r := runKanban(t, kanbanDir, "--json", "move", "1", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("idempotent move failed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestRequireClaimErrorMessageQuality(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Error test")

	r := runKanban(t, kanbanDir, "move", "1", "in-progress")
	if r.exitCode == 0 {
		t.Fatal("expected failure")
	}
	combined := r.stdout + r.stderr
	if !strings.Contains(combined, "in-progress") {
		t.Error("error should mention the status name")
	}
	if !strings.Contains(combined, "--claim") {
		t.Error("error should mention --claim flag")
	}
	if !strings.Contains(combined, "agent-name") {
		t.Error("error should mention 'agent-name' command for generating a name")
	}
}

func TestRequireClaimJSONErrorOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "JSON error test")

	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "in-progress")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
	if errResp.Details["status"] != "in-progress" {
		t.Errorf("details.status = %v, want %q", errResp.Details["status"], "in-progress")
	}
}

func TestRequireClaimCreateInClaimStatusSucceeds(t *testing.T) {
	// Creating a task directly in a require_claim status should work.
	// Create doesn't have claim semantics — it's setting initial state.
	kanbanDir := initBoard(t)
	task := mustCreateTask(t, kanbanDir, "Direct create", "--status", "in-progress")
	if task.Status != statusInProgress {
		t.Errorf("status = %q, want %q", task.Status, statusInProgress)
	}
}

func TestRequireClaimPickWithMoveToClaimStatus(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Pickable task", "--status", statusTodo)

	var picked taskJSON
	r := runKanbanJSON(t, kanbanDir, &picked, "pick", "--claim", "picker-agent", "--move", "in-progress")
	if r.exitCode != 0 {
		t.Fatalf("pick --claim --move in-progress failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if picked.Status != statusInProgress {
		t.Errorf("status = %q, want %q", picked.Status, statusInProgress)
	}
	if picked.ClaimedBy != "picker-agent" {
		t.Errorf("claimed_by = %q, want %q", picked.ClaimedBy, "picker-agent")
	}
}

func TestRequireClaimExpiredClaimStillRequiresClaim(t *testing.T) {
	kanbanDir := initBoard(t)
	writeTaskFile(t, kanbanDir, 1, `---
id: 1
title: Expired claim test
status: backlog
priority: high
created: 2026-01-01T00:00:00Z
updated: 2026-01-01T00:00:00Z
claimed_by: agent-old
claimed_at: 2020-01-01T00:00:00Z
---
`)
	bumpNextID(t, kanbanDir, 2)

	// Even though the claim is expired, require_claim still demands --claim.
	errResp := runKanbanJSONError(t, kanbanDir, "move", "1", "in-progress")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q (expired claim doesn't bypass require_claim)", errResp.Code, codeClaimRequired)
	}

	// With --claim, it succeeds.
	r := runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", "new-agent")
	if r.exitCode != 0 {
		t.Fatalf("move with --claim should succeed (exit %d): %s", r.exitCode, r.stderr)
	}
}

func TestRequireClaimReleaseBypassesChecks(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Release edit test")
	runKanban(t, kanbanDir, "--json", "move", "1", "in-progress", "--claim", claimAgent1)

	// Release the claim. --release bypasses claim check and require_claim.
	r := runKanban(t, kanbanDir, "--json", "edit", "1", "--release")
	if r.exitCode != 0 {
		t.Fatalf("release failed (exit %d): %s", r.exitCode, r.stderr)
	}

	// After release, task is still in in-progress but unclaimed.
	// Editing still requires --claim because the status has require_claim.
	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--title", "Changed")
	if errResp.Code != codeClaimRequired {
		t.Errorf("code = %q, want %q", errResp.Code, codeClaimRequired)
	}
}

// ---------------------------------------------------------------------------
// Concurrent create — duplicate ID prevention
// ---------------------------------------------------------------------------
