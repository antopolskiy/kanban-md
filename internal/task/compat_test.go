package task

import (
	"path/filepath"
	"reflect"
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

func TestCompatV1TaskWithTimestamps(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "005-with-timestamps.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task with timestamps: %v", err)
	}

	if tk.ID != 5 {
		t.Errorf("ID = %d, want 5", tk.ID)
	}
	if tk.Started == nil {
		t.Fatal("Started is nil, want non-nil")
	}
	if tk.Completed == nil {
		t.Fatal("Completed is nil, want non-nil")
	}
	// Verify the dates parsed correctly.
	if tk.Started.Year() != 2026 || tk.Started.Month() != 1 || tk.Started.Day() != 20 {
		t.Errorf("Started = %v, want 2026-01-20", tk.Started)
	}
	if tk.Completed.Year() != 2026 || tk.Completed.Month() != 2 || tk.Completed.Day() != 1 {
		t.Errorf("Completed = %v, want 2026-02-01", tk.Completed)
	}
}

func TestCompatV1TaskWithoutTimestamps(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "002-design-api.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task: %v", err)
	}

	// Tasks without timestamp fields should have nil Started/Completed.
	if tk.Started != nil {
		t.Errorf("Started = %v, want nil", tk.Started)
	}
	if tk.Completed != nil {
		t.Errorf("Completed = %v, want nil", tk.Completed)
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

func TestCompatV1TaskWithClaimAndClass(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "006-with-claim-and-class.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task with claim and class: %v", err)
	}

	if tk.ID != 6 {
		t.Errorf("ID = %d, want 6", tk.ID)
	}
	if tk.ClaimedBy != "agent-1" {
		t.Errorf("ClaimedBy = %q, want %q", tk.ClaimedBy, "agent-1")
	}
	if tk.ClaimedAt == nil {
		t.Fatal("ClaimedAt is nil, want non-nil")
	}
	if tk.ClaimedAt.Year() != 2026 || tk.ClaimedAt.Month() != 2 || tk.ClaimedAt.Day() != 8 {
		t.Errorf("ClaimedAt = %v, want 2026-02-08", tk.ClaimedAt)
	}
	if tk.Class != "expedite" {
		t.Errorf("Class = %q, want %q", tk.Class, "expedite")
	}
}

func TestCompatV1TaskPreservesOpaqueProperties(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "007-with-extra-properties.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task with extra properties: %v", err)
	}

	tk.Priority = "high"
	outputPath := filepath.Join(t.TempDir(), "007-with-extra-properties.md")
	if err = Write(outputPath, tk); err != nil {
		t.Fatalf("Write() v1 task with extra properties: %v", err)
	}

	values := readFrontmatterValues(t, outputPath)
	want := map[string]any{"reference": frontmatterTestReference}
	if got := values["custom_supported"]; !reflect.DeepEqual(got, want) {
		t.Errorf("custom_supported = %#v, want %#v", got, want)
	}
	if got := values["priority"]; got != "high" {
		t.Errorf("priority = %#v, want changed canonical value", got)
	}
	root := readFrontmatterNode(t, outputPath)
	assertScalarNode(t, mappingValue(t, root, "custom_tagged"), "sample-17", "!integration")
	source := mappingValue(t, root, "custom_source")
	if mappingValue(t, root, "custom_copy").Alias != source {
		t.Error("compatibility alias lost its binding")
	}
	assertScalarNode(t, mappingValue(t, mappingValue(t, root, "custom_non_string_mapping"), "1"), "unsupported", "!!str")
}

func TestCompatV1TaskPreservesExactNumericAndTaggedValues(t *testing.T) {
	tk, err := Read(filepath.Join(v1FixtureDir, "008-exact-extra-values.md"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "008-exact-extra-values.md")
	for cycle := 0; cycle < 2; cycle++ {
		if err = Write(path, tk); err != nil {
			t.Fatal(err)
		}
		root := readFrontmatterNode(t, path)
		assertScalarNode(t, mappingValue(t, root, "huge"), "18446744073709551617", preservationFloatTag)
		assertScalarNode(t, mappingValue(t, root, "decimal"), "0.123456789012345678901234567890", preservationFloatTag)
		assertScalarNode(t, mappingValue(t, mappingValue(t, root, "mixed"), "tagged"), "001", "!integration")
		tk, err = Read(path)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompatV1TaskUnsafeAliasIsReadableButRefusesWrite(t *testing.T) {
	tk, err := Read(filepath.Join(v1FixtureDir, "009-unsafe-extra-alias.md"))
	if err != nil {
		t.Fatal(err)
	}
	if tk.Estimate != "4h" {
		t.Errorf("Estimate = %q, want 4h", tk.Estimate)
	}
	if err = Write(filepath.Join(t.TempDir(), "009-unsafe-extra-alias.md"), tk); err == nil {
		t.Fatal("unsafe fixture write succeeded")
	}
}

func TestCompatV1TaskWithoutClaimAndClass(t *testing.T) {
	path := filepath.Join(v1FixtureDir, "002-design-api.md")
	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() v1 task: %v", err)
	}

	// Tasks without claim/class fields should have zero values.
	if tk.ClaimedBy != "" {
		t.Errorf("ClaimedBy = %q, want empty", tk.ClaimedBy)
	}
	if tk.ClaimedAt != nil {
		t.Errorf("ClaimedAt = %v, want nil", tk.ClaimedAt)
	}
	if tk.Class != "" {
		t.Errorf("Class = %q, want empty", tk.Class)
	}
}
