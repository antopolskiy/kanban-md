package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Install error paths ---

func TestInstall_InvalidSkillName(t *testing.T) {
	tmp := t.TempDir()
	err := Install("nonexistent-skill", tmp, "1.0.0")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
	if !strings.Contains(err.Error(), "reading embedded skill") {
		t.Errorf("error = %v, want to contain 'reading embedded skill'", err)
	}
}

func TestInstall_MkdirError(t *testing.T) {
	// Use an invalid base directory that can't be created.
	err := Install("kanban-md", "/dev/null/invalid", "1.0.0")
	if err == nil {
		t.Fatal("expected error when output directory can't be created")
	}
}

// --- injectVersionComment edge cases ---

func TestInjectVersionComment_NoClosingFrontmatter(t *testing.T) {
	// Frontmatter opens with --- but never closes.
	input := "---\nname: test\nno closing delimiter\n"
	result := string(injectVersionComment([]byte(input), "1.0.0"))

	// Should prepend the version comment since no closing --- found.
	if !strings.HasPrefix(result, "<!-- kanban-md-skill-version: 1.0.0 -->") {
		t.Errorf("expected version comment prepended when no closing ---:\n%s", result)
	}
}

func TestInjectVersionComment_EmptyContent(t *testing.T) {
	result := string(injectVersionComment([]byte(""), "1.0.0"))

	if !strings.HasPrefix(result, "<!-- kanban-md-skill-version: 1.0.0 -->") {
		t.Errorf("expected version comment prepended for empty content:\n%s", result)
	}
}

func TestInjectVersionComment_SingleLine(t *testing.T) {
	input := "# Just a title"
	result := string(injectVersionComment([]byte(input), "1.0.0"))

	if !strings.HasPrefix(result, "<!-- kanban-md-skill-version: 1.0.0 -->") {
		t.Errorf("expected version comment prepended for single line:\n%s", result)
	}
}

// --- IsOutdated edge cases ---

func TestIsOutdated_NotInstalled(t *testing.T) {
	if IsOutdated("/nonexistent/SKILL.md", "1.0.0") {
		t.Error("not installed should return false")
	}
}

func TestIsOutdated_SameVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "SKILL.md")
	content := "<!-- kanban-md-skill-version: 1.0.0 -->\n# Test\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if IsOutdated(path, "1.0.0") {
		t.Error("same version should not be outdated")
	}
}

func TestIsOutdated_DifferentVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "SKILL.md")
	content := "<!-- kanban-md-skill-version: 0.9.0 -->\n# Test\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if !IsOutdated(path, "1.0.0") {
		t.Error("different version should be outdated")
	}
}

// --- FindInstalledSkills ---

func TestFindInstalledSkills_NoSkills(t *testing.T) {
	tmp := t.TempDir()
	found := FindInstalledSkills(tmp)
	if len(found) != 0 {
		t.Errorf("FindInstalledSkills empty dir = %v, want empty", found)
	}
}

// --- Agent methods ---

func TestAgent_ProjectPath(t *testing.T) {
	a := AgentByName("claude")
	if a == nil {
		t.Fatal("AgentByName(claude) = nil")
	}

	got := a.ProjectPath("/home/user/project")
	want := filepath.Join("/home/user/project", ".claude/skills")
	if got != want {
		t.Errorf("ProjectPath = %q, want %q", got, want)
	}
}

func TestAgent_SkillPath_Global(t *testing.T) {
	a := AgentByName("claude")
	if a == nil {
		t.Fatal("AgentByName(claude) = nil")
	}

	// With global=true, should return global path.
	got := a.SkillPath("/some/root", true)
	if got == "" {
		t.Error("SkillPath(global=true) should not be empty")
	}
	if got == a.ProjectPath("/some/root") {
		t.Error("global SkillPath should differ from project path")
	}
}

func TestAgent_SkillPath_Project(t *testing.T) {
	a := AgentByName("claude")
	if a == nil {
		t.Fatal("AgentByName(claude) = nil")
	}

	got := a.SkillPath("/some/root", false)
	want := a.ProjectPath("/some/root")
	if got != want {
		t.Errorf("SkillPath(global=false) = %q, want %q", got, want)
	}
}
