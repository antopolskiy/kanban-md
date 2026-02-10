package skill

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ---------------------------------------------------------------------------
// ReadEmbeddedSkill — basic read test (0% -> covered)
// ---------------------------------------------------------------------------

func TestReadEmbeddedSkill_KanbanMD(t *testing.T) {
	data, err := ReadEmbeddedSkill("kanban-md")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(kanban-md) error: %v", err)
	}
	if len(data) == 0 {
		t.Error("ReadEmbeddedSkill(kanban-md) returned empty content")
	}
}

func TestReadEmbeddedSkill_KanbanBasedDevelopment(t *testing.T) {
	data, err := ReadEmbeddedSkill("kanban-based-development")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(kanban-based-development) error: %v", err)
	}
	if len(data) == 0 {
		t.Error("ReadEmbeddedSkill(kanban-based-development) returned empty content")
	}
}

func TestReadEmbeddedSkill_Nonexistent(t *testing.T) {
	_, err := ReadEmbeddedSkill("nonexistent-skill")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

// ---------------------------------------------------------------------------
// GlobalPath — returns non-empty for normal agents
// ---------------------------------------------------------------------------

func TestGlobalPath_ReturnsNonEmpty(t *testing.T) {
	a := AgentByName("claude")
	if a == nil {
		t.Fatal("AgentByName(claude) = nil")
	}
	path := a.GlobalPath()
	if path == "" {
		t.Error("GlobalPath() for claude should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Install — write-protected target directory
// ---------------------------------------------------------------------------

func TestInstall_WriteProtectedTarget(t *testing.T) {
	tmp := t.TempDir()
	// Create the target directory, then make it read-only.
	target := filepath.Join(tmp, "readonly")
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatal(err)
	}

	// Pre-create the skill directory so MkdirAll succeeds,
	// then make the skill dir read-only so file writes fail.
	skillDir := filepath.Join(target, "kanban-md")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict directory writes on Windows")
	}

	if err := os.Chmod(skillDir, 0o444); err != nil { //nolint:gosec // intentionally restrict for test
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(skillDir, 0o750) }) //nolint:gosec // restore permissions in cleanup

	err := Install("kanban-md", target, "1.0.0")
	if err == nil {
		t.Fatal("expected error when target directory is write-protected")
	}
}

// ---------------------------------------------------------------------------
// FindInstalledSkills — partial installation
// ---------------------------------------------------------------------------

func TestFindInstalledSkills_PartialInstall(t *testing.T) {
	tmp := t.TempDir()

	// Only install one skill.
	if err := Install("kanban-md", tmp, "1.0.0"); err != nil {
		t.Fatalf("Install error: %v", err)
	}

	found := FindInstalledSkills(tmp)
	if len(found) != 1 {
		t.Errorf("FindInstalledSkills = %d, want 1 (only kanban-md)", len(found))
	}
	if _, ok := found["kanban-md"]; !ok {
		t.Error("expected kanban-md in found skills")
	}
}

// ---------------------------------------------------------------------------
// Names — returns correct count
// ---------------------------------------------------------------------------

func TestNames_Count(t *testing.T) {
	names := Names()
	if len(names) != len(AvailableSkills) {
		t.Errorf("Names() len = %d, want %d", len(names), len(AvailableSkills))
	}
}

// ---------------------------------------------------------------------------
// DetectAgents — empty directory
// ---------------------------------------------------------------------------

func TestDetectAgents_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	detected := DetectAgents(tmp)
	if len(detected) != 0 {
		t.Errorf("DetectAgents empty dir = %d, want 0", len(detected))
	}
}

// ---------------------------------------------------------------------------
// Agent.SkillPath — global-only with global=true
// ---------------------------------------------------------------------------

func TestSkillPath_GlobalOnlyGlobalTrue(t *testing.T) {
	oc := AgentByName("openclaw")
	if oc == nil {
		t.Fatal("AgentByName(openclaw) = nil")
	}

	// Even with global=true explicitly, should return global path.
	sp := oc.SkillPath("/some/root", true)
	if sp != oc.GlobalPath() {
		t.Errorf("SkillPath(true) = %q, want GlobalPath %q", sp, oc.GlobalPath())
	}
}

// ---------------------------------------------------------------------------
// injectVersionComment — frontmatter with only one line
// ---------------------------------------------------------------------------

func TestInjectVersionComment_MinimalFrontmatter(t *testing.T) {
	input := "---\n---\n# Title\n"
	result := string(injectVersionComment([]byte(input), "1.0.0"))

	// Version comment should appear after closing ---.
	idx := findSubstring(result, "<!-- kanban-md-skill-version: 1.0.0 -->")
	if idx < 0 {
		t.Errorf("version comment not found in result:\n%s", result)
	}
}

// findSubstring returns the index of substr in s, or -1 if not found.
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
