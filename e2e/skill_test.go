package e2e_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Skill command tests
// ---------------------------------------------------------------------------

// runKanbanNoDir runs the binary without the --dir flag (for skill commands).
func runKanbanNoDir(t *testing.T, dir string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...) //nolint:noctx,gosec // e2e test binary, args are test-controlled
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	r := result{stdout: stdout.String(), stderr: stderr.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			r.exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running kanban-md: %v", err)
		}
	}
	return r
}

func TestSkillShow(t *testing.T) {
	dir := t.TempDir()
	r := runKanbanNoDir(t, dir, "skill", "show", "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("skill show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "kanban-md") {
		t.Errorf("skill show output should contain skill content, got:\n%s", r.stdout)
	}
}

func TestSkillShowAll(t *testing.T) {
	dir := t.TempDir()
	r := runKanbanNoDir(t, dir, "skill", "show")
	if r.exitCode != 0 {
		t.Fatalf("skill show all failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "=== kanban-md ===") {
		t.Error("skill show should include kanban-md header")
	}
	if !strings.Contains(r.stdout, "=== kanban-based-development ===") {
		t.Error("skill show should include kanban-based-development header")
	}
}

func TestSkillInstallAndCheck(t *testing.T) {
	dir := t.TempDir()
	// Create a .claude directory so the agent is detected.
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Install skills for claude agent.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("skill install failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("expected file paths in output, got:\n%s", r.stdout)
	}

	// Check that SKILL.md files were created.
	skillMD := filepath.Join(dir, ".claude", "skills", "kanban-md", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("kanban-md SKILL.md not created: %v", err)
	}
	devSkillMD := filepath.Join(dir, ".claude", "skills", "kanban-based-development", "SKILL.md")
	if _, err := os.Stat(devSkillMD); err != nil {
		t.Errorf("kanban-based-development SKILL.md not created: %v", err)
	}

	// Check that references subdir was created.
	refPath := filepath.Join(dir, ".claude", "skills", "kanban-md", "references", "json-schemas.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("references/json-schemas.md not created: %v", err)
	}

	// Run check — should be up to date.
	r = runKanbanNoDir(t, dir, "skill", "check", "--agent", "claude")
	if r.exitCode != 0 {
		t.Errorf("skill check should pass (exit 0), got exit %d: %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "up to date") {
		t.Errorf("expected 'up to date' in check output, got:\n%s", r.stdout)
	}
}

func TestSkillInstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// First install.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("first install failed: %s", r.stderr)
	}

	// Second install — should skip (already installed).
	r = runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("second install failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "skipped") || !strings.Contains(r.stdout, "up to date") {
		// Either "skipped" or "up to date" is acceptable.
		if !strings.Contains(r.stdout, "skipped") && !strings.Contains(r.stdout, "up to date") {
			t.Errorf("expected skip message on re-install, got:\n%s", r.stdout)
		}
	}
}

func TestSkillInstallForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// First install.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("first install failed: %s", r.stderr)
	}

	// Force re-install — should overwrite.
	r = runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude", "--force")
	if r.exitCode != 0 {
		t.Fatalf("force install failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("force install should re-install, got:\n%s", r.stdout)
	}
}

func TestSkillInstallPath(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "custom-skills")

	// Install to a custom path — directory doesn't exist yet, should be created.
	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir)
	if r.exitCode != 0 {
		t.Fatalf("skill install --path failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("expected file paths in output, got:\n%s", r.stdout)
	}

	// Verify files were created at the custom path.
	skillMD := filepath.Join(targetDir, "kanban-md", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("SKILL.md not created at custom path: %v", err)
	}
	refPath := filepath.Join(targetDir, "kanban-md", "references", "json-schemas.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("references not created at custom path: %v", err)
	}
}

func TestSkillInstallPathSelectSkill(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "skills")

	// Install only kanban-md skill to custom path.
	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir, "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("skill install --path --skill failed: %s", r.stderr)
	}

	// kanban-md should exist, kanban-based-development should not.
	if _, err := os.Stat(filepath.Join(targetDir, "kanban-md", "SKILL.md")); err != nil {
		t.Error("kanban-md SKILL.md not created")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "kanban-based-development", "SKILL.md")); err == nil {
		t.Error("kanban-based-development should not be installed with --skill kanban-md")
	}
}

func TestSkillInstallCreatesNestedDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a .cursor directory (parent exists, but skills/ subdir doesn't).
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o750); err != nil {
		t.Fatal(err)
	}

	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "cursor")
	if r.exitCode != 0 {
		t.Fatalf("install for cursor failed: %s", r.stderr)
	}

	// Verify the nested skills directory was created.
	skillMD := filepath.Join(dir, ".cursor", "skills", "kanban-md", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Errorf("cursor skill not installed: %v", err)
	}
}

func TestSkillInstallExistingDirPreserved(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Create an existing file in the skills dir that we don't manage.
	userFile := filepath.Join(dir, ".claude", "skills", "my-custom-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("custom content"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Install kanban-md skills.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	// The user's custom skill should not be touched.
	data, err := os.ReadFile(userFile) //nolint:gosec // test reads from known temp directory
	if err != nil {
		t.Fatalf("user's custom skill deleted: %v", err)
	}
	if string(data) != "custom content" {
		t.Errorf("user's custom skill modified: %q", string(data))
	}
}

func TestSkillInstallSymlinkNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "skills"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Create a symlink where kanban-md/ would go, pointing to an external dir.
	externalDir := filepath.Join(dir, "external-data")
	if err := os.MkdirAll(externalDir, 0o750); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(externalDir, "important.txt")
	if err := os.WriteFile(sentinel, []byte("do not delete"), 0o600); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(dir, ".claude", "skills", "kanban-md")
	if err := os.Symlink(externalDir, symlinkPath); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	// Install should still succeed (writes through the symlink), but must NOT
	// delete the symlink target or the sentinel file.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude", "--force")
	if r.exitCode != 0 {
		t.Fatalf("install with symlink failed: %s", r.stderr)
	}

	// The sentinel file in the external dir must survive.
	data, err := os.ReadFile(sentinel) //nolint:gosec // test reads from known temp directory
	if err != nil {
		t.Fatalf("sentinel file in symlink target deleted: %v", err)
	}
	if string(data) != "do not delete" {
		t.Errorf("sentinel file corrupted: %q", string(data))
	}

	// The symlink itself should still be a symlink.
	fi, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("symlink gone: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("kanban-md is no longer a symlink after install")
	}
}

func TestSkillInstallOpenClawGlobalOnly(t *testing.T) {
	dir := t.TempDir()

	// Install for openclaw — should go to global dir, not project dir.
	// Use --path equivalent by checking that no "skills/" directory is created
	// in the project root.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "openclaw")
	if r.exitCode != 0 {
		t.Fatalf("openclaw install failed: %s", r.stderr)
	}

	// No skills/ directory should appear in the project root.
	if _, err := os.Stat(filepath.Join(dir, "skills")); err == nil {
		t.Error("openclaw install created skills/ in project root instead of global path")
	}

	// Output should reference .openclaw path.
	if !strings.Contains(r.stdout, ".openclaw") {
		t.Errorf("expected global .openclaw path in output, got:\n%s", r.stdout)
	}
}

func TestSkillInstallMultipleAgents(t *testing.T) {
	dir := t.TempDir()
	// Set up directories for two agents.
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o750); err != nil {
		t.Fatal(err)
	}

	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude,cursor")
	if r.exitCode != 0 {
		t.Fatalf("multi-agent install failed: %s", r.stderr)
	}

	// Both agents should have skills installed.
	for _, agentDir := range []string{".claude", ".cursor"} {
		skillMD := filepath.Join(dir, agentDir, "skills", "kanban-md", "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			t.Errorf("%s skill not installed: %v", agentDir, err)
		}
	}
}

func TestSkillInstallVersionCommentPresent(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "skills")

	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir, "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "kanban-md", "SKILL.md")) //nolint:gosec // test temp dir
	if err != nil {
		t.Fatalf("reading installed skill: %v", err)
	}

	if !strings.Contains(string(data), "<!-- kanban-md-skill-version:") {
		t.Error("installed SKILL.md missing version comment")
	}
}

func TestSkillUpdateAndCheck(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "skills")

	// Install with --path.
	r := runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir)
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	// Tamper with version to simulate outdated.
	skillMD := filepath.Join(targetDir, "kanban-md", "SKILL.md")
	oldContent := "---\nname: kanban-md\n---\n<!-- kanban-md-skill-version: 0.0.1 -->\n# old content\n"
	if err := os.WriteFile(skillMD, []byte(oldContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Re-install — kanban-md should be updated (0.0.1 → dev),
	// kanban-based-development may be skipped (already at dev).
	r = runKanbanNoDir(t, dir, "skill", "install", "--path", targetDir, "--skill", "kanban-md")
	if r.exitCode != 0 {
		t.Fatalf("re-install failed: %s", r.stderr)
	}
	// Should have updated since version differs.
	if strings.Contains(r.stdout, "skipped") {
		t.Errorf("expected install to update outdated skill, but it was skipped:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Installed 1 skill") {
		t.Errorf("expected 1 skill installed, got:\n%s", r.stdout)
	}
}

func TestSkillUpdateShowsPaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Install skills for claude agent.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	// Tamper with kanban-md version to simulate outdated.
	skillMD := filepath.Join(dir, ".claude", "skills", "kanban-md", "SKILL.md")
	oldContent := "---\nname: kanban-md\n---\n<!-- kanban-md-skill-version: 0.0.1 -->\n# old content\n"
	if err := os.WriteFile(skillMD, []byte(oldContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Update — kanban-md should update, kanban-based-development should be skipped.
	r = runKanbanNoDir(t, dir, "skill", "update", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("update failed: %s", r.stderr)
	}

	// Updated skill should show file path (SKILL.md) and version transition.
	if !strings.Contains(r.stdout, "SKILL.md") {
		t.Errorf("expected file path in update output, got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "0.0.1") {
		t.Errorf("expected old version in update output, got:\n%s", r.stdout)
	}

	// Skipped skill should show file path and "(skipped)".
	if !strings.Contains(r.stdout, "skipped") {
		t.Errorf("expected skipped message for up-to-date skill, got:\n%s", r.stdout)
	}

	// Summary line.
	if !strings.Contains(r.stdout, "Updated 1 skill") {
		t.Errorf("expected 'Updated 1 skill' summary, got:\n%s", r.stdout)
	}
}

func TestSkillUpdateAllUpToDate(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Install skills.
	r := runKanbanNoDir(t, dir, "skill", "install", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("install failed: %s", r.stderr)
	}

	// Update with everything already current.
	r = runKanbanNoDir(t, dir, "skill", "update", "--agent", "claude")
	if r.exitCode != 0 {
		t.Fatalf("update failed: %s", r.stderr)
	}

	// All should be skipped — output should show paths + "(skipped)" and the summary.
	if !strings.Contains(r.stdout, "skipped") {
		t.Errorf("expected skipped message for up-to-date skills, got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "All skills are already up to date.") {
		t.Errorf("expected 'All skills are already up to date.' summary, got:\n%s", r.stdout)
	}
}
