package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAvailableSkills(t *testing.T) {
	names := Names()
	if len(names) != 2 {
		t.Fatalf("Names() = %v, want 2 entries", names)
	}
	if names[0] != "kanban-md" {
		t.Errorf("Names()[0] = %q, want %q", names[0], "kanban-md")
	}
	if names[1] != "kanban-based-development" {
		t.Errorf("Names()[1] = %q, want %q", names[1], "kanban-based-development")
	}
}

func TestEmbeddedSkillsReadable(t *testing.T) {
	for _, s := range AvailableSkills {
		// embed.FS always uses forward slashes, even on Windows.
		p := "skills/" + s.Name + "/SKILL.md"
		data, err := skillsFS.ReadFile(p)
		if err != nil {
			t.Errorf("reading embedded %s: %v", p, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded %s is empty", p)
		}
	}
}

func TestEmbeddedKanbanMDHasReferences(t *testing.T) {
	// embed.FS always uses forward slashes, even on Windows.
	p := "skills/kanban-md/references/json-schemas.md"
	data, err := skillsFS.ReadFile(p)
	if err != nil {
		t.Fatalf("reading embedded %s: %v", p, err)
	}
	if len(data) == 0 {
		t.Fatal("json-schemas.md is empty")
	}
}

func TestAgentRegistry(t *testing.T) {
	agents := Agents()
	if len(agents) != 4 {
		t.Fatalf("Agents() returned %d, want 4", len(agents))
	}

	names := AllAgentNames()
	expected := [4]string{"claude", "codex", "cursor", "openclaw"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("AllAgentNames()[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestAgentByName(t *testing.T) {
	a := AgentByName("claude")
	if a == nil {
		t.Fatal("AgentByName(claude) = nil")
	}
	if a.DisplayName != "Claude Code" {
		t.Errorf("DisplayName = %q, want %q", a.DisplayName, "Claude Code")
	}
	if a.ProjectDir != ".claude/skills" {
		t.Errorf("ProjectDir = %q, want %q", a.ProjectDir, ".claude/skills")
	}

	if AgentByName("nonexistent") != nil {
		t.Error("AgentByName(nonexistent) should return nil")
	}
}

func TestDetectAgents(t *testing.T) {
	tmp := t.TempDir()

	// Create .claude/ directory.
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	detected := DetectAgents(tmp)
	// Should detect Claude Code and OpenClaw (always detected).
	var names []string
	for _, a := range detected {
		names = append(names, a.Name)
	}
	if !contains(names, "claude") {
		t.Error("expected claude to be detected")
	}
	if !contains(names, "openclaw") {
		t.Error("expected openclaw to be detected (always present)")
	}
	if contains(names, "codex") {
		t.Error("codex should not be detected without .agents/ dir")
	}
}

func TestVersionComment(t *testing.T) {
	comment := VersionComment("1.2.3")
	want := "<!-- kanban-md-skill-version: 1.2.3 -->"
	if comment != want {
		t.Errorf("VersionComment = %q, want %q", comment, want)
	}
}

func TestInstalledVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "SKILL.md")

	content := "---\nname: test\n---\n<!-- kanban-md-skill-version: 0.19.0 -->\n# Test\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	ver := InstalledVersion(path)
	if ver != "0.19.0" {
		t.Errorf("InstalledVersion = %q, want %q", ver, "0.19.0")
	}
}

func TestInstalledVersionMissing(t *testing.T) {
	ver := InstalledVersion("/nonexistent/path/SKILL.md")
	if ver != "" {
		t.Errorf("InstalledVersion(nonexistent) = %q, want empty", ver)
	}
}

func TestInstalledVersionNoComment(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "SKILL.md")

	content := "---\nname: test\n---\n# Test\nNo version here.\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	ver := InstalledVersion(path)
	if ver != "" {
		t.Errorf("InstalledVersion(no comment) = %q, want empty", ver)
	}
}

func TestIsOutdated(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "SKILL.md")

	content := "---\nname: test\n---\n<!-- kanban-md-skill-version: 0.18.0 -->\n# Test\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if !IsOutdated(path, "0.19.0") {
		t.Error("should be outdated (0.18.0 vs 0.19.0)")
	}
	if IsOutdated(path, "0.18.0") {
		t.Error("should not be outdated (same version)")
	}
}

func TestInstall(t *testing.T) {
	tmp := t.TempDir()

	err := Install("kanban-md", tmp, "1.0.0")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Check SKILL.md was written with version comment.
	skillMD := filepath.Join(tmp, "kanban-md", "SKILL.md")
	data, err := os.ReadFile(skillMD) //nolint:gosec // test reads from known temp directory
	if err != nil {
		t.Fatalf("reading installed SKILL.md: %v", err)
	}
	if !strings.Contains(string(data), "<!-- kanban-md-skill-version: 1.0.0 -->") {
		t.Error("installed SKILL.md missing version comment")
	}
	if !strings.Contains(string(data), "kanban-md") {
		t.Error("installed SKILL.md missing expected content")
	}

	// Check references subdirectory.
	refPath := filepath.Join(tmp, "kanban-md", "references", "json-schemas.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("references/json-schemas.md not installed: %v", err)
	}

	// Check version is readable.
	ver := InstalledVersion(skillMD)
	if ver != "1.0.0" {
		t.Errorf("InstalledVersion = %q, want %q", ver, "1.0.0")
	}
}

func TestInstallBothSkills(t *testing.T) {
	tmp := t.TempDir()

	for _, s := range AvailableSkills {
		if err := Install(s.Name, tmp, "2.0.0"); err != nil {
			t.Fatalf("Install(%s): %v", s.Name, err)
		}
	}

	found := FindInstalledSkills(tmp)
	if len(found) != 2 {
		t.Errorf("FindInstalledSkills found %d, want 2", len(found))
	}
}

func TestInstallOverwrite(t *testing.T) {
	tmp := t.TempDir()

	// Install v1.
	if err := Install("kanban-md", tmp, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	// Install v2 (overwrite).
	if err := Install("kanban-md", tmp, "2.0.0"); err != nil {
		t.Fatal(err)
	}

	skillMD := filepath.Join(tmp, "kanban-md", "SKILL.md")
	ver := InstalledVersion(skillMD)
	if ver != "2.0.0" {
		t.Errorf("after overwrite, version = %q, want %q", ver, "2.0.0")
	}
}

func TestInjectVersionCommentWithFrontmatter(t *testing.T) {
	input := "---\nname: test\ndescription: a test\n---\n# Title\nBody\n"
	result := string(injectVersionComment([]byte(input), "1.0.0"))

	if !strings.Contains(result, "---\n<!-- kanban-md-skill-version: 1.0.0 -->") {
		t.Errorf("version comment not injected after frontmatter:\n%s", result)
	}
	// Frontmatter should still be intact.
	if !strings.HasPrefix(result, "---\n") {
		t.Error("frontmatter opening lost")
	}
}

func TestInjectVersionCommentWithoutFrontmatter(t *testing.T) {
	input := "# Title\nNo frontmatter\n"
	result := string(injectVersionComment([]byte(input), "1.0.0"))

	if !strings.HasPrefix(result, "<!-- kanban-md-skill-version: 1.0.0 -->\n") {
		t.Errorf("version comment not prepended:\n%s", result)
	}
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
