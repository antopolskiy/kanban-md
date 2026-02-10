package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/skill"
)

const (
	testVersion        = "v1.0.0"
	testVersionUpgrade = "v2.0.0"
	agentClaude        = "claude"
)

// setupSkillProject creates a temp directory with .git, installs a skill for claude,
// sets version, and changes cwd. Returns the project root.
func setupSkillProject(t *testing.T, ver string) {
	t.Helper()
	savedVersion := version
	version = ver
	t.Cleanup(func() { version = savedVersion })

	projectRoot := t.TempDir()
	claude := skill.AgentByName(agentClaude)
	if claude == nil {
		t.Skip("claude agent not found")
	}
	baseDir := claude.SkillPath(projectRoot, false)
	if err := skill.Install("kanban-md", baseDir, testVersion); err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(projectRoot, ".git")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(projectRoot)
}

// --- resolveAgentList tests ---

func TestResolveAgentList_AllAgents(t *testing.T) {
	// Empty filter returns all agents.
	agents := resolveAgentList(nil)
	if len(agents) == 0 {
		t.Fatal("expected non-empty agent list")
	}
	if len(agents) != len(skill.Agents()) {
		t.Errorf("len = %d, want %d", len(agents), len(skill.Agents()))
	}
}

func TestResolveAgentList_SpecificAgent(t *testing.T) {
	agents := resolveAgentList([]string{agentClaude})
	if len(agents) != 1 {
		t.Fatalf("len = %d, want 1", len(agents))
	}
	if agents[0].Name != agentClaude {
		t.Errorf("agent name = %q, want %q", agents[0].Name, "claude")
	}
}

func TestResolveAgentList_MultipleAgents(t *testing.T) {
	agents := resolveAgentList([]string{agentClaude, "codex"})
	if len(agents) != 2 {
		t.Fatalf("len = %d, want 2", len(agents))
	}
}

func TestResolveAgentList_UnknownAgent(t *testing.T) {
	agents := resolveAgentList([]string{"unknown-agent"})
	if len(agents) != 0 {
		t.Errorf("expected empty list for unknown agent, got %d", len(agents))
	}
}

func TestResolveAgentList_MixedKnownUnknown(t *testing.T) {
	agents := resolveAgentList([]string{agentClaude, "unknown-agent"})
	if len(agents) != 1 {
		t.Fatalf("len = %d, want 1 (only known agent)", len(agents))
	}
	if agents[0].Name != agentClaude {
		t.Errorf("agent name = %q, want %q", agents[0].Name, "claude")
	}
}

// --- resolveSkills tests (with explicit filter) ---

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringSlice("skill", nil, "")
	return cmd
}

func TestResolveSkills_AllSkills(t *testing.T) {
	// Empty filter in non-interactive mode returns all skills.
	skills, err := resolveSkills(newSkillCmd(), nil)
	if err != nil {
		t.Fatalf("resolveSkills error: %v", err)
	}
	if len(skills) != len(skill.AvailableSkills) {
		t.Errorf("len = %d, want %d", len(skills), len(skill.AvailableSkills))
	}
}

func TestResolveSkills_SpecificSkill(t *testing.T) {
	skills, err := resolveSkills(newSkillCmd(), []string{"kanban-md"})
	if err != nil {
		t.Fatalf("resolveSkills error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("len = %d, want 1", len(skills))
	}
	if skills[0].Name != "kanban-md" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "kanban-md")
	}
}

func TestResolveSkills_UnknownSkill(t *testing.T) {
	_, err := resolveSkills(newSkillCmd(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown skill")
	}
}

// --- relativePath tests ---

func TestRelativePath_Simple(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "home", "user", "project")
	target := filepath.Join(base, "skills", "SKILL.md")
	got := relativePath(base, target)
	want := filepath.Join("skills", "SKILL.md")
	if got != want {
		t.Errorf("relativePath = %q, want %q", got, want)
	}
}

func TestRelativePath_SamePath(t *testing.T) {
	got := relativePath("/home/user", "/home/user")
	if got != "." {
		t.Errorf("relativePath = %q, want %q", got, ".")
	}
}

// --- exitCodeError tests ---

func TestExitCodeError_Error(t *testing.T) {
	e := &exitCodeError{code: 1}
	got := e.Error()
	if got != "exit code 1" {
		t.Errorf("Error() = %q, want %q", got, "exit code 1")
	}
}

func TestExitCodeError_ZeroCode(t *testing.T) {
	e := &exitCodeError{code: 0}
	got := e.Error()
	if got != "exit code 0" {
		t.Errorf("Error() = %q, want %q", got, "exit code 0")
	}
}

// --- findProjectRoot tests ---

func TestFindProjectRoot_GitDir(t *testing.T) {
	// Create a temp directory with a .git directory.
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory to search from.
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)

	got, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot error: %v", err)
	}
	if got != root {
		t.Errorf("findProjectRoot = %q, want %q", got, root)
	}
}

func TestFindProjectRoot_NoGitDir(t *testing.T) {
	// In a temp directory with no .git, should return cwd.
	dir := t.TempDir()
	t.Chdir(dir)

	got, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot error: %v", err)
	}
	// Should return cwd since there's no .git anywhere up the temp tree.
	// It walks to root and falls back to cwd.
	if got != dir {
		t.Errorf("findProjectRoot = %q, want %q (cwd fallback)", got, dir)
	}
}

// --- CheckSkillStaleness tests ---

func TestCheckSkillStaleness_DevVersion(t *testing.T) {
	// When version is "dev", should not print anything.
	savedVersion := version
	version = "dev"
	t.Cleanup(func() { version = savedVersion })

	r, w := captureStderr(t)

	CheckSkillStaleness(t.TempDir())

	got := drainPipe(t, r, w)
	if got != "" {
		t.Errorf("expected no output for dev version, got: %s", got)
	}
}

func TestCheckSkillStaleness_NoSkillsInstalled(t *testing.T) {
	savedVersion := version
	version = testVersion
	t.Cleanup(func() { version = savedVersion })

	r, w := captureStderr(t)

	// Empty project root with no installed skills.
	CheckSkillStaleness(t.TempDir())

	got := drainPipe(t, r, w)
	if got != "" {
		t.Errorf("expected no output when no skills installed, got: %s", got)
	}
}

func TestCheckSkillStaleness_OutdatedSkill(t *testing.T) {
	// setupSkillProject installs to SkillPath; CheckSkillStaleness uses ProjectPath.
	// Set up manually for the staleness check's specific path logic.
	savedVersion := version
	version = testVersionUpgrade
	t.Cleanup(func() { version = savedVersion })

	projectRoot := t.TempDir()
	claude := skill.AgentByName(agentClaude)
	if claude == nil {
		t.Skip("claude agent not found")
	}
	baseDir := claude.ProjectPath(projectRoot)
	if err := skill.Install("kanban-md", baseDir, testVersion); err != nil {
		t.Fatalf("Install error: %v", err)
	}

	r, w := captureStderr(t)
	CheckSkillStaleness(projectRoot)
	got := drainPipe(t, r, w)

	if !containsSubstring(got, "outdated") {
		t.Errorf("expected 'outdated' warning, got: %s", got)
	}
}

func TestCheckSkillStaleness_UpToDate(t *testing.T) {
	savedVersion := version
	version = testVersion
	t.Cleanup(func() { version = savedVersion })

	// Install a skill at the current version.
	projectRoot := t.TempDir()
	claude := skill.AgentByName(agentClaude)
	if claude == nil {
		t.Skip("claude agent not found")
	}
	baseDir := claude.ProjectPath(projectRoot)
	if err := skill.Install("kanban-md", baseDir, testVersion); err != nil {
		t.Fatalf("Install error: %v", err)
	}

	r, w := captureStderr(t)

	CheckSkillStaleness(projectRoot)

	got := drainPipe(t, r, w)
	if got != "" {
		t.Errorf("expected no output when up to date, got: %s", got)
	}
}

// --- installToPath tests ---

func TestInstallToPath_CreatesFiles(t *testing.T) {
	dir := t.TempDir()

	err := installToPath(dir, skill.AvailableSkills, true)
	if err != nil {
		t.Fatalf("installToPath error: %v", err)
	}

	// Verify skill files were created.
	for _, s := range skill.AvailableSkills {
		path := filepath.Join(dir, s.Name, "SKILL.md")
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("expected skill file at %s, got error: %v", path, statErr)
		}
	}
}

func TestInstallToPath_SkipsCurrent(t *testing.T) {
	savedVersion := version
	version = testVersion
	t.Cleanup(func() { version = savedVersion })

	dir := t.TempDir()

	// Install first time.
	if err := installToPath(dir, skill.AvailableSkills, true); err != nil {
		t.Fatalf("first installToPath error: %v", err)
	}

	// Install again without force — should skip (already at current version).
	r, w := captureStdout(t)
	err := installToPath(dir, skill.AvailableSkills, false)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("second installToPath error: %v", err)
	}
	if !containsSubstring(got, "skipped") {
		t.Errorf("expected 'skipped' in output, got: %s", got)
	}
}

// --- resolveAgents tests (non-interactive) ---

func TestResolveAgents_WithFilter(t *testing.T) {
	// Explicit --agent filter bypasses detection.
	agents := resolveAgents([]string{agentClaude}, t.TempDir(), false)
	if len(agents) != 1 {
		t.Fatalf("len = %d, want 1", len(agents))
	}
	if agents[0].Name != agentClaude {
		t.Errorf("agent = %q, want %q", agents[0].Name, "claude")
	}
}

func TestResolveAgents_NonInteractiveGlobal(t *testing.T) {
	// global=true, no filter, non-interactive → returns all agents.
	agents := resolveAgents(nil, t.TempDir(), true)
	if len(agents) == 0 {
		t.Fatal("expected non-empty agent list for global mode")
	}
	if len(agents) != len(skill.Agents()) {
		t.Errorf("len = %d, want %d", len(agents), len(skill.Agents()))
	}
}

func TestResolveAgents_NonInteractiveDetect(t *testing.T) {
	// non-interactive, not global → returns detected agents.
	// In a temp dir with no agent dirs, detected set may be empty.
	projectRoot := t.TempDir()
	agents := resolveAgents(nil, projectRoot, false)
	// Result depends on detection; just verify it doesn't panic.
	// For an empty project root, DetectAgents returns empty.
	_ = agents
}

func TestResolveAgents_NonInteractiveDetectWithClaude(t *testing.T) {
	// Create a fake .claude directory so DetectAgents finds it.
	projectRoot := t.TempDir()
	claudeDir := filepath.Join(projectRoot, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		t.Fatal(err)
	}

	agents := resolveAgents(nil, projectRoot, false)
	found := false
	for _, a := range agents {
		if a.Name == agentClaude {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected claude in detected agents")
	}
}

// --- resolveSkills additional tests ---

func TestResolveSkills_MultipleSkills(t *testing.T) {
	skills, err := resolveSkills(newSkillCmd(), []string{"kanban-md", "kanban-based-development"})
	if err != nil {
		t.Fatalf("resolveSkills error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("len = %d, want 2", len(skills))
	}
}

// --- runSkillCheck tests ---

func newSkillCheckCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "check"}
	cmd.Flags().StringSlice("agent", nil, "")
	cmd.Flags().Bool("global", false, "")
	return cmd
}

func TestRunSkillCheck_NoSkillsInstalled(t *testing.T) {
	savedVersion := version
	version = testVersion
	t.Cleanup(func() { version = savedVersion })

	// Empty project root with .git and no skills.
	projectRoot := t.TempDir()
	gitDir := filepath.Join(projectRoot, ".git")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(projectRoot)

	r, w := captureStdout(t)
	cmd := newSkillCheckCmd()
	// Use a specific agent to avoid checking global dirs with real installed skills.
	_ = cmd.Flags().Set("agent", agentClaude)

	err := runSkillCheck(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillCheck error: %v", err)
	}
	if !containsSubstring(got, "No kanban-md skills installed") {
		t.Errorf("expected 'No kanban-md skills installed', got: %s", got)
	}
}

func TestRunSkillCheck_AllUpToDate(t *testing.T) {
	// Install at testVersion, then check with version=testVersion → up to date.
	setupSkillProject(t, testVersion)

	r, w := captureStdout(t)
	cmd := newSkillCheckCmd()
	_ = cmd.Flags().Set("agent", agentClaude)

	err := runSkillCheck(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillCheck error: %v", err)
	}
	if !containsSubstring(got, "ok") {
		t.Errorf("expected 'ok' for up-to-date skill, got: %s", got)
	}
}

func TestRunSkillCheck_Outdated(t *testing.T) {
	// Install at testVersion, then check with version=testVersionUpgrade → outdated.
	setupSkillProject(t, testVersionUpgrade)

	r, w := captureStdout(t)
	cmd := newSkillCheckCmd()
	_ = cmd.Flags().Set("agent", agentClaude)

	err := runSkillCheck(cmd, nil)
	got := drainPipe(t, r, w)

	if err == nil {
		t.Fatal("expected error for outdated skills")
	}
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exitCodeError, got %T", err)
	}
	if !containsSubstring(got, "x") {
		t.Errorf("expected 'x' marker for outdated, got: %s", got)
	}
}

// --- runSkillShow tests ---

func newSkillShowCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "show"}
	cmd.Flags().String("skill", "", "")
	return cmd
}

func TestRunSkillShow_SpecificSkill(t *testing.T) {
	r, w := captureStdout(t)
	cmd := newSkillShowCmd()
	_ = cmd.Flags().Set("skill", "kanban-md")

	err := runSkillShow(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillShow error: %v", err)
	}
	if !containsSubstring(got, "kanban") {
		t.Errorf("expected skill content, got: %s", got[:min(len(got), 200)])
	}
}

func TestRunSkillShow_AllSkills(t *testing.T) {
	r, w := captureStdout(t)
	cmd := newSkillShowCmd()
	// No filter — shows all skills.

	err := runSkillShow(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillShow error: %v", err)
	}
	// All skills should have separator headers.
	if !containsSubstring(got, "===") {
		t.Errorf("expected '===' separators for multi-skill output, got: %s", got[:min(len(got), 200)])
	}
}

// --- runSkillUpdate tests ---

func newSkillUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().StringSlice("agent", nil, "")
	cmd.Flags().Bool("global", false, "")
	return cmd
}

func TestRunSkillUpdate_NoSkillsInstalled(t *testing.T) {
	savedVersion := version
	version = testVersion
	t.Cleanup(func() { version = savedVersion })

	projectRoot := t.TempDir()
	gitDir := filepath.Join(projectRoot, ".git")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Chdir(projectRoot)

	r, w := captureStdout(t)
	cmd := newSkillUpdateCmd()
	_ = cmd.Flags().Set("agent", agentClaude)

	err := runSkillUpdate(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillUpdate error: %v", err)
	}
	if !containsSubstring(got, "up to date") {
		t.Errorf("expected 'up to date' message, got: %s", got)
	}
}

func TestRunSkillUpdate_UpdatesOutdated(t *testing.T) {
	// Install at testVersion, then update with version=testVersionUpgrade.
	setupSkillProject(t, testVersionUpgrade)

	r, w := captureStdout(t)
	cmd := newSkillUpdateCmd()
	_ = cmd.Flags().Set("agent", agentClaude)

	err := runSkillUpdate(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillUpdate error: %v", err)
	}
	if !containsSubstring(got, "Updated") {
		t.Errorf("expected 'Updated' in output, got: %s", got)
	}
}

func TestRunSkillUpdate_SkipsUpToDate(t *testing.T) {
	// Install at testVersion, then update with version=testVersion → already up to date.
	setupSkillProject(t, testVersion)

	r, w := captureStdout(t)
	cmd := newSkillUpdateCmd()
	_ = cmd.Flags().Set("agent", agentClaude)

	err := runSkillUpdate(cmd, nil)
	got := drainPipe(t, r, w)

	if err != nil {
		t.Fatalf("runSkillUpdate error: %v", err)
	}
	if !containsSubstring(got, "skipped") {
		t.Errorf("expected 'skipped' in output, got: %s", got)
	}
}
