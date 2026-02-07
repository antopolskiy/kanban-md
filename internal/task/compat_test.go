package task

import (
	"path/filepath"
	"testing"
)

const v1FixtureDir = "testdata/compat/v1/tasks"

func TestCompatV1TaskCoreFields(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "001-set-up-database.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task: %v", err)
	}

	if tk.ID != 1 {
		t.Errorf("ID = %d, want 1", tk.ID)
	}
	if tk.Title != "Set up database" {
		t.Errorf("Title = %q, want %q", tk.Title, "Set up database")
	}
	if tk.Status != "done" {
		t.Errorf("Status = %q, want %q", tk.Status, "done")
	}
	if tk.Priority != "high" {
		t.Errorf("Priority = %q, want %q", tk.Priority, "high")
	}
	if tk.Created.IsZero() {
		t.Error("Created is zero")
	}
	if tk.Updated.IsZero() {
		t.Error("Updated is zero")
	}
}

func TestCompatV1TaskOptionalFields(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "001-set-up-database.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task: %v", err)
	}

	if tk.Assignee != "alice" {
		t.Errorf("Assignee = %q, want %q", tk.Assignee, "alice")
	}
	if tk.Estimate != "4h" {
		t.Errorf("Estimate = %q, want %q", tk.Estimate, "4h")
	}

	wantTags := []string{"backend", "infrastructure"}
	if len(tk.Tags) != len(wantTags) {
		t.Fatalf("Tags len = %d, want %d", len(tk.Tags), len(wantTags))
	}
	for i, tag := range wantTags {
		if tk.Tags[i] != tag {
			t.Errorf("Tags[%d] = %q, want %q", i, tk.Tags[i], tag)
		}
	}

	if tk.Due == nil {
		t.Fatal("Due is nil, want 2026-02-01")
	}
	if tk.Due.String() != "2026-02-01" {
		t.Errorf("Due = %q, want %q", tk.Due.String(), "2026-02-01")
	}

	if tk.Body == "" {
		t.Error("Body is empty, want non-empty")
	}
}

func TestCompatV1TaskMinimalFields(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "002-design-api.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task with minimal fields: %v", err)
	}

	if tk.ID != 2 {
		t.Errorf("ID = %d, want 2", tk.ID)
	}
	if tk.Title != "Design API" {
		t.Errorf("Title = %q, want %q", tk.Title, "Design API")
	}

	// Optional fields should be zero values.
	if tk.Assignee != "" {
		t.Errorf("Assignee = %q, want empty", tk.Assignee)
	}
	if len(tk.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", tk.Tags)
	}
	if tk.Due != nil {
		t.Errorf("Due = %v, want nil", tk.Due)
	}
	if tk.Estimate != "" {
		t.Errorf("Estimate = %q, want empty", tk.Estimate)
	}
	if tk.Parent != nil {
		t.Errorf("Parent = %v, want nil", tk.Parent)
	}
	if len(tk.DependsOn) != 0 {
		t.Errorf("DependsOn = %v, want empty", tk.DependsOn)
	}
}

func TestCompatV1TaskWithDependencies(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "003-auth-flow.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task with dependencies: %v", err)
	}

	if tk.ID != 3 {
		t.Errorf("ID = %d, want 3", tk.ID)
	}

	// Parent field
	if tk.Parent == nil {
		t.Fatal("Parent is nil, want 2")
	}
	if *tk.Parent != 2 {
		t.Errorf("Parent = %d, want 2", *tk.Parent)
	}

	// DependsOn field
	if len(tk.DependsOn) != 1 {
		t.Fatalf("DependsOn len = %d, want 1", len(tk.DependsOn))
	}
	if tk.DependsOn[0] != 1 {
		t.Errorf("DependsOn[0] = %d, want 1", tk.DependsOn[0])
	}

	// No body
	if tk.Body != "" {
		t.Errorf("Body = %q, want empty", tk.Body)
	}
}

func TestCompatV1TaskBlockedFields(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "004-blocked-task.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 blocked task: %v", err)
	}

	if tk.ID != 4 {
		t.Errorf("ID = %d, want 4", tk.ID)
	}
	if !tk.Blocked {
		t.Error("Blocked = false, want true")
	}
	if tk.BlockReason != "waiting for API credentials" {
		t.Errorf("BlockReason = %q, want %q", tk.BlockReason, "waiting for API credentials")
	}
	if tk.Body == "" {
		t.Error("Body is empty, want non-empty")
	}
}

func TestCompatV1TaskMinimalNotBlocked(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "002-design-api.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task: %v", err)
	}

	// Tasks without blocked fields should default to not-blocked.
	if tk.Blocked {
		t.Error("Blocked = true, want false for task without blocked field")
	}
	if tk.BlockReason != "" {
		t.Errorf("BlockReason = %q, want empty", tk.BlockReason)
	}
}
