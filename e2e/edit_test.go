package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const preservationClaimFlag = "--claim"

// ---------------------------------------------------------------------------
// Edit tests
// ---------------------------------------------------------------------------

func TestEditFields(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Edit target", "--tags", "original")

	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, task taskJSON)
	}{
		{
			name: "change status",
			args: []string{"--status", statusTodo},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if task.Status != statusTodo {
					t.Errorf("Status = %q, want %q", task.Status, statusTodo)
				}
			},
		},
		{
			name: "change priority",
			args: []string{"--priority", "high"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if task.Priority != priorityHigh {
					t.Errorf("Priority = %q, want %q", task.Priority, priorityHigh)
				}
			},
		},
		{
			name: "add tag",
			args: []string{"--add-tag", "newtag"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				found := false
				for _, tag := range task.Tags {
					if tag == "newtag" {
						found = true
					}
				}
				if !found {
					t.Errorf("Tags %v missing %q", task.Tags, "newtag")
				}
			},
		},
		{
			name: "set due date",
			args: []string{"--due", "2026-06-15"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if task.Due != "2026-06-15" {
					t.Errorf("Due = %q, want %q", task.Due, "2026-06-15")
				}
			},
		},
		{
			name: "set body",
			args: []string{"--body", "Updated body content"},
			check: func(t *testing.T, task taskJSON) {
				t.Helper()
				if !strings.Contains(task.Body, "Updated body content") {
					t.Errorf("Body = %q, want %q", task.Body, "Updated body content")
				}
			},
		},
	}

	// These run sequentially on the same task — each edit builds on previous state.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editArgs := append([]string{"edit", "1"}, tt.args...)
			var task taskJSON
			r := runKanbanJSON(t, kanbanDir, &task, editArgs...)
			if r.exitCode != 0 {
				t.Fatalf("edit failed: %s", r.stderr)
			}
			tt.check(t, task)
		})
	}
}

func TestEditTitleRename(t *testing.T) {
	kanbanDir := initBoard(t)
	original := mustCreateTask(t, kanbanDir, "Original title")

	var updated taskJSON
	runKanbanJSON(t, kanbanDir, &updated, "edit", "1", "--title", "New title")

	if updated.Title != "New title" {
		t.Errorf("Title = %q, want %q", updated.Title, "New title")
	}

	// Old file removed.
	if _, err := os.Stat(original.File); !os.IsNotExist(err) {
		t.Errorf("old file %q still exists", original.File)
	}

	// New file exists with correct slug.
	if _, err := os.Stat(updated.File); err != nil {
		t.Errorf("new file %q not found: %v", updated.File, err)
	}
	if !strings.Contains(filepath.Base(updated.File), "new-title") {
		t.Errorf("filename %q missing 'new-title'", filepath.Base(updated.File))
	}
}

func TestMutationsPreserveUnknownFrontmatter(t *testing.T) {
	kanbanDir := initBoard(t)
	created := mustCreateTask(t, kanbanDir, "Generic sample")
	data, err := os.ReadFile(created.File)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	closing := strings.LastIndex(content, "---\n")
	if closing <= 0 {
		t.Fatalf("task has no closing frontmatter delimiter:\n%s", content)
	}
	content = content[:closing] + `custom_value: retained
huge: 18446744073709551617
mixed:
  plain: retained
  tagged: !integration 001
  binary: !!binary aGVsbG8=
  source: &source {17: retained}
  copy: *source
` + content[closing:]
	if err = os.WriteFile(created.File, []byte(content), 0o600); err != nil { //nolint:gosec,nolintlint // test binary returned this task path
		t.Fatal(err)
	}

	const listCommand = "list"
	for _, args := range [][]string{
		{listCommand},
		{"--compact", listCommand},
		{"--json", listCommand},
		{"--json", "show", "1"},
		{"--json", "board"},
	} {
		r := runKanban(t, kanbanDir, args...)
		if r.exitCode != 0 {
			t.Fatalf("%v failed: %s", args, r.stderr)
		}
		if strings.Contains(r.stdout, "custom_value") || strings.Contains(r.stdout, "retained") {
			t.Errorf("%v exposed custom frontmatter:\n%s", args, r.stdout)
		}
	}

	var edited taskJSON
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--title", "Changed sample")
	if r.exitCode != 0 {
		t.Fatalf("edit failed: %s", r.stderr)
	}
	assertCustomValuePreserved(t, edited.File)
	if strings.Contains(r.stdout, "custom_value") || strings.Contains(r.stdout, "mixed") || strings.Contains(r.stdout, "huge") {
		t.Errorf("JSON edit exposed custom frontmatter:\n%s", r.stdout)
	}

	for _, args := range [][]string{
		{"move", "1", statusTodo},
		{"pick", preservationClaimFlag, claimTestAgent, "--status", statusTodo, "--no-body"},
		{"edit", "1", "--release"},
		{"pick", preservationClaimFlag, claimTestAgent, "--status", statusTodo, "--no-body"},
		{"handoff", "1", preservationClaimFlag, claimTestAgent, "--note", "Ready for review"},
		{"archive", "1", preservationClaimFlag, claimTestAgent},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if r = runKanban(t, kanbanDir, args...); r.exitCode != 0 {
				t.Fatalf("%v failed: %s", args, r.stderr)
			}
			assertCustomValuePreserved(t, edited.File)
		})
	}
}

func TestMutationsRefuseUnknownAliasToCanonicalField(t *testing.T) {
	kanbanDir := initBoard(t)
	created := mustCreateTask(t, kanbanDir, "Generic sample")
	r := runKanban(t, kanbanDir, "edit", "1", preservationClaimFlag, claimTestAgent)
	if r.exitCode != 0 {
		t.Fatalf("claim failed: %s", r.stderr)
	}

	data, err := os.ReadFile(created.File)
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(
		string(data),
		"claimed_by: "+claimTestAgent,
		"claimed_by: &shared "+claimTestAgent+"\ncustom_copy: *shared",
		1,
	)
	if content == string(data) {
		t.Fatalf("claimed task does not contain claimed_by:\n%s", data)
	}
	if err = os.Chmod(created.File, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(created.File, []byte(content), 0o600); err != nil { //nolint:gosec,nolintlint // test binary returned this task path
		t.Fatal(err)
	}
	logPath := filepath.Join(kanbanDir, "activity.jsonl")
	beforeLog, err := os.ReadFile(logPath) //nolint:gosec // test-owned activity log
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"edit", "1", "--release"},
		{"edit", "1", preservationClaimFlag, claimTestAgent, "--title", "Changed sample"},
	} {
		r = runKanban(t, kanbanDir, args...)
		if r.exitCode == 0 || !strings.Contains(r.stderr, "cannot preserve") {
			t.Fatalf("%v result = %d %s, want preservation refusal", args, r.exitCode, r.stderr)
		}
		assertTaskBytes(t, created.File, []byte(content))
		assertTaskBytes(t, logPath, beforeLog)
	}
	if _, err = os.Stat(filepath.Join(filepath.Dir(created.File), "001-changed-sample.md")); !os.IsNotExist(err) {
		t.Errorf("refused CLI edit created destination: %v", err)
	}
	var unchanged taskJSON
	r = runKanbanJSON(t, kanbanDir, &unchanged, "show", "1")
	if r.exitCode != 0 || unchanged.ClaimedBy != claimTestAgent {
		t.Fatalf("show after refusal: %s, ClaimedBy=%q", r.stderr, unchanged.ClaimedBy)
	}
	for _, command := range []string{"list", "board"} {
		if r = runKanban(t, kanbanDir, command); r.exitCode != 0 {
			t.Errorf("ordinary %s failed: %s", command, r.stderr)
		}
	}
}

func assertCustomValuePreserved(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test helper receives a task path from the test binary
	if err != nil {
		t.Fatal(err)
	}
	const want = "custom_value: retained"
	if !strings.Contains(string(data), want) {
		t.Errorf("%s does not contain %q:\n%s", path, want, data)
	}
	var document yaml.Node
	frontmatter := strings.SplitN(string(data)[4:], "\n---", 2)[0]
	if err = yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		t.Fatal(err)
	}
	root := document.Content[0]
	huge := extraYAMLValue(t, root, "huge")
	if huge.Value != "18446744073709551617" || huge.ShortTag() != "!!float" {
		t.Errorf("huge scalar changed to %q %s", huge.Value, huge.ShortTag())
	}
	mixed := extraYAMLValue(t, root, "mixed")
	for _, wantScalar := range []struct{ key, value, tag string }{
		{"plain", "retained", "!!str"},
		{"tagged", "001", "!integration"},
		{"binary", "aGVsbG8=", "!!binary"},
	} {
		node := extraYAMLValue(t, mixed, wantScalar.key)
		if node.Value != wantScalar.value || node.ShortTag() != wantScalar.tag {
			t.Errorf("mixed.%s changed to %q %s", wantScalar.key, node.Value, node.ShortTag())
		}
	}
	if extraYAMLValue(t, mixed, "copy").Alias != extraYAMLValue(t, mixed, "source") {
		t.Error("mixed alias lost its binding")
	}
}

func extraYAMLValue(t *testing.T, mapping *yaml.Node, key string) *yaml.Node {
	t.Helper()
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	t.Fatalf("missing retained property %q", key)
	return nil
}

func TestEditNoChanges(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Stable task")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1")
	if errResp.Code != "NO_CHANGES" {
		t.Errorf("code = %q, want NO_CHANGES", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "no changes") {
		t.Errorf("error = %q, want 'no changes'", errResp.Error)
	}
}

func TestEditClearDue(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Dated task", "--due", "2026-03-15")

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--clear-due")

	if task.Due != "" {
		t.Errorf("Due = %q, want empty (cleared)", task.Due)
	}
}

func TestEditAppendBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Append target", "--body", "initial body")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--append-body", "appended note")
	if r.exitCode != 0 {
		t.Fatalf("edit failed: %s", r.stderr)
	}
	if !strings.Contains(task.Body, "initial body") {
		t.Errorf("Body should contain original text, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "appended note") {
		t.Errorf("Body should contain appended text, got %q", task.Body)
	}
}

func TestEditAppendBody_ToEmptyBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Empty body target")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--append-body", "first note")
	if r.exitCode != 0 {
		t.Fatalf("edit failed: %s", r.stderr)
	}
	if task.Body != "first note" {
		t.Errorf("Body = %q, want %q", task.Body, "first note")
	}
}

func TestEditAppendBody_WithTimestamp(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Timestamp target", "--body", "existing")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1", "-a", "progress update", "-t")
	if r.exitCode != 0 {
		t.Fatalf("edit failed: %s", r.stderr)
	}
	if !strings.Contains(task.Body, "existing") {
		t.Errorf("Body should contain original text, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "[[") || !strings.Contains(task.Body, "]]") {
		t.Errorf("Body should contain timestamp markers, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "progress update") {
		t.Errorf("Body should contain appended text, got %q", task.Body)
	}
}

func TestEditAppendBody_MultipleAppends(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Multi-append target")

	var task taskJSON
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--append-body", "note 1")
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--append-body", "note 2")
	runKanbanJSON(t, kanbanDir, &task, "edit", "1", "--append-body", "note 3")

	if !strings.Contains(task.Body, "note 1") {
		t.Errorf("Body should contain note 1, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "note 2") {
		t.Errorf("Body should contain note 2, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "note 3") {
		t.Errorf("Body should contain note 3, got %q", task.Body)
	}
}

func TestEditBodyAndAppendBodyConflict(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Conflict target")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--body", "replace", "--append-body", "append")
	if errResp.Code != codeStatusConflict {
		t.Errorf("code = %q, want STATUS_CONFLICT", errResp.Code)
	}
}

// ---------------------------------------------------------------------------
// Move tests
// ---------------------------------------------------------------------------

func TestEditStatusRespectsWIPLimit(t *testing.T) {
	kanbanDir := initBoardWithWIP(t, 1)

	mustCreateTask(t, kanbanDir, "Task A", "--status", "in-progress")
	mustCreateTask(t, kanbanDir, "Task B")

	// Edit task B status to in-progress should fail.
	errResp := runKanbanJSONError(t, kanbanDir, "edit", "2", "--status", "in-progress", "--claim", claimTestAgent)
	if errResp.Code != codeWIPLimitExceeded {
		t.Errorf("code = %q, want WIP_LIMIT_EXCEEDED", errResp.Code)
	}
}

func TestEditStartedManualBackfill(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--started", "2026-01-15")
	if r.exitCode != 0 {
		t.Fatalf("edit --started failed: %s", r.stderr)
	}
	started, ok := edited["started"].(string)
	if !ok || !strings.HasPrefix(started, "2026-01-15") {
		t.Errorf("started = %v, want prefix 2026-01-15", edited["started"])
	}
}

func TestEditCompletedManualBackfill(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--completed", "2026-02-01")
	if r.exitCode != 0 {
		t.Fatalf("edit --completed failed: %s", r.stderr)
	}
	completed, ok := edited["completed"].(string)
	if !ok || !strings.HasPrefix(completed, "2026-02-01") {
		t.Errorf("completed = %v, want prefix 2026-02-01", edited["completed"])
	}
}

func TestEditClearStarted(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--started", "2026-01-15")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--clear-started")
	if r.exitCode != 0 {
		t.Fatalf("edit --clear-started failed: %s", r.stderr)
	}
	if edited["started"] != nil {
		t.Errorf("started = %v, want nil after clear", edited["started"])
	}
}

func TestEditClearCompleted(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "edit", "1", "--completed", "2026-02-01")

	var edited map[string]interface{}
	r := runKanbanJSON(t, kanbanDir, &edited, "edit", "1", "--clear-completed")
	if r.exitCode != 0 {
		t.Fatalf("edit --clear-completed failed: %s", r.stderr)
	}
	if edited["completed"] != nil {
		t.Errorf("completed = %v, want nil after clear", edited["completed"])
	}
}

func TestEditStartedAndClearStartedConflict(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1", "--started", "2026-01-15", "--clear-started")
	if errResp.Code != codeStatusConflict {
		t.Errorf("code = %q, want STATUS_CONFLICT", errResp.Code)
	}
	if !strings.Contains(errResp.Error, "cannot use") {
		t.Errorf("error = %q, want conflict error", errResp.Error)
	}
}
