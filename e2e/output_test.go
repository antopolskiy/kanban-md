package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Output format tests
// ---------------------------------------------------------------------------

func TestTextOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Text output test")

	// Table list.
	r := runKanban(t, kanbanDir, "--table", "list")
	if r.exitCode != 0 {
		t.Fatalf("table list failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Text output test") {
		t.Errorf("table list missing task title in stdout")
	}

	// Table show.
	r = runKanban(t, kanbanDir, "--table", "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("table show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Task #1") {
		t.Errorf("show missing 'Task #1' in stdout")
	}

	// Table create.
	r = runKanban(t, kanbanDir, "--table", "create", "Another task")
	if r.exitCode != 0 {
		t.Fatalf("table create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #2") {
		t.Errorf("create missing 'Created task #2' in stdout")
	}
}

// ---------------------------------------------------------------------------
// Blocked state tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Default output format tests (table is always the default)
// ---------------------------------------------------------------------------

func TestDefaultOutputIsTable(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Default output task", "--tags", "test,demo")

	// Default output should be table (even when piped/non-TTY).
	r := runKanban(t, kanbanDir, "list")
	if r.exitCode != 0 {
		t.Fatalf("list failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "ID") || !strings.Contains(r.stdout, "STATUS") {
		t.Errorf("default list should be table with headers, got:\n%s", r.stdout)
	}

	// Create should also default to table (message) output.
	r = runKanban(t, kanbanDir, "create", "Another task")
	if r.exitCode != 0 {
		t.Fatalf("create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #2") {
		t.Errorf("default create should show message, got:\n%s", r.stdout)
	}
}

func TestTableFlagOutputList(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Table output task", "--tags", "test,demo")

	r := runKanban(t, kanbanDir, "--table", "list")
	if r.exitCode != 0 {
		t.Fatalf("list failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "ID") || !strings.Contains(r.stdout, "STATUS") {
		t.Errorf("table output missing headers:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "TAGS") {
		t.Errorf("table output missing TAGS column:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "test,demo") {
		t.Errorf("table output missing tag values:\n%s", r.stdout)
	}
}

func TestTableFlagOutputCreate(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "create", "New task via table")
	if r.exitCode != 0 {
		t.Fatalf("create failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Created task #1") {
		t.Errorf("table create missing confirmation:\n%s", r.stdout)
	}
}

func TestTableFlagOutputMove(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Movable task")

	r := runKanban(t, kanbanDir, "--table", "move", "1", statusTodo)
	if r.exitCode != 0 {
		t.Fatalf("move failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Moved task #1") {
		t.Errorf("table move missing confirmation:\n%s", r.stdout)
	}
}

func TestTableFlagOutputDelete(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Deletable task")

	r := runKanban(t, kanbanDir, "--table", "delete", "1", "--yes")
	if r.exitCode != 0 {
		t.Fatalf("delete failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Deleted task #1") {
		t.Errorf("table delete missing confirmation:\n%s", r.stdout)
	}
}

func TestTableFlagOutputBoard(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board task")

	r := runKanban(t, kanbanDir, "--table", "board")
	if r.exitCode != 0 {
		t.Fatalf("board failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "STATUS") || !strings.Contains(r.stdout, "COUNT") {
		t.Errorf("table board missing headers:\n%s", r.stdout)
	}
}

func TestVersionDefault(t *testing.T) {
	// Binary built without ldflags should report "dev".
	r := runKanban(t, t.TempDir(), "--version")
	if r.exitCode != 0 {
		t.Fatalf("--version failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "dev") {
		t.Errorf("default version should contain 'dev', got: %s", r.stdout)
	}
}

func TestVersionLdflags(t *testing.T) {
	// Build a separate binary with ldflags to verify version injection works.
	tmp := t.TempDir()
	binName := "kanban-md-version-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	versionBin := filepath.Join(tmp, binName)
	wantVersion := "1.2.3-test"

	//nolint:gosec,noctx // building test binary with ldflags
	build := exec.Command("go", "build",
		"-ldflags", "-X github.com/antopolskiy/kanban-md/cmd.version="+wantVersion,
		"-o", versionBin, "../cmd/kanban-md")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("building binary with ldflags: %v", err)
	}

	//nolint:gosec,noctx // running test binary
	cmd := exec.Command(versionBin, "--version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running --version: %v", err)
	}

	if !strings.Contains(string(out), wantVersion) {
		t.Errorf("version should contain %q, got: %s", wantVersion, string(out))
	}
}

func TestTableFlagOutputMetrics(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--table", "metrics")
	if r.exitCode != 0 {
		t.Fatalf("metrics failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Throughput") {
		t.Errorf("table metrics missing throughput:\n%s", r.stdout)
	}
}

func TestREADMEDocumentsAllCommands(t *testing.T) {
	readmePath := filepath.Join("..", "README.md")
	data, err := os.ReadFile(readmePath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("reading README: %v", err)
	}
	readme := string(data)

	// Every user-facing command must have a ### `command` section.
	commands := []string{
		"init", "create", "list", "show", "edit", "move", "delete",
		"board", "pick", "metrics", "log", "config", "context",
	}
	for _, cmd := range commands {
		heading := "### `" + cmd + "`"
		if !strings.Contains(readme, heading) {
			t.Errorf("README missing command section: %s", heading)
		}
	}

	// Key flags that must be documented somewhere in the README.
	requiredFlags := map[string][]string{
		"init":   {"--wip-limit"},
		"create": {"--parent", "--depends-on"},
		"edit": {
			"--started", "--clear-started", "--completed", "--clear-completed",
			"--parent", "--clear-parent", "--add-dep", "--remove-dep",
			"--block", "--unblock", "--claim", "--release", "--class",
		},
		"move":    {"--claim"},
		"list":    {"--blocked", "--not-blocked", "--parent", "--unblocked", "--unclaimed", "--claimed-by", "--class", "--group-by"},
		"board":   {"--group-by"},
		"pick":    {"--claim", "--status", "--move", "--tags", "--no-body"},
		"metrics": {"--since"},
		"log":     {"--since", "--limit", "--action", "--task"},
	}
	for cmd, flags := range requiredFlags {
		for _, flag := range flags {
			// Flag should appear in the README (in the command's section or flags table).
			if !strings.Contains(readme, "`"+flag+"`") {
				t.Errorf("README missing flag %s for command %s", flag, cmd)
			}
		}
	}

	// Config example must show current schema version.
	if !strings.Contains(readme, "version: 3") {
		t.Error("README config example still shows old version (should be version: 3)")
	}

	// Config example must mention wip_limits.
	if !strings.Contains(readme, "wip_limits") {
		t.Error("README config example missing wip_limits field")
	}
}

// ---------------------------------------------------------------------------
// Compact output format tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Compact output format tests
// ---------------------------------------------------------------------------

func TestCompactOutputList(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact list task", "--tags", "test,demo", "--priority", "high")

	r := runKanban(t, kanbanDir, "--compact", "list")
	if r.exitCode != 0 {
		t.Fatalf("compact list failed: %s", r.stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(r.stdout), "#1 [") {
		t.Errorf("compact list should start with '#1 [', got:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Compact list task") {
		t.Errorf("compact list missing title:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "(test, demo)") {
		t.Errorf("compact list missing tags:\n%s", r.stdout)
	}
}

func TestCompactOutputShow(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact show task", "--priority", "high")

	r := runKanban(t, kanbanDir, "--compact", "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("compact show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "#1 [backlog/high] Compact show task") {
		t.Errorf("compact show missing header:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "created:") {
		t.Errorf("compact show missing timestamps:\n%s", r.stdout)
	}
}

func TestCompactOutputBoard(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Board compact task")

	r := runKanban(t, kanbanDir, "--compact", "board")
	if r.exitCode != 0 {
		t.Fatalf("compact board failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "tasks)") {
		t.Errorf("compact board missing task count:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "backlog:") {
		t.Errorf("compact board missing status line:\n%s", r.stdout)
	}
}

func TestCompactOutputMetrics(t *testing.T) {
	kanbanDir := initBoard(t)

	r := runKanban(t, kanbanDir, "--compact", "metrics")
	if r.exitCode != 0 {
		t.Fatalf("compact metrics failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Throughput:") {
		t.Errorf("compact metrics missing throughput:\n%s", r.stdout)
	}
}

func TestCompactOutputLog(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Log compact task")

	r := runKanban(t, kanbanDir, "--compact", "log")
	if r.exitCode != 0 {
		t.Fatalf("compact log failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "create #1") {
		t.Errorf("compact log missing create entry:\n%s", r.stdout)
	}
}

func TestOnelineAlias(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Oneline alias task")

	compact := runKanban(t, kanbanDir, "--compact", "list")
	oneline := runKanban(t, kanbanDir, "--oneline", "list")

	if compact.stdout != oneline.stdout {
		t.Errorf("--oneline should produce same output as --compact\ncompact:\n%s\noneline:\n%s",
			compact.stdout, oneline.stdout)
	}
}

func TestCompactEnvVar(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Env compact task")

	r := runKanbanEnv(t, kanbanDir, []string{"KANBAN_OUTPUT=compact"}, "list")
	if r.exitCode != 0 {
		t.Fatalf("env compact list failed: %s", r.stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(r.stdout), "#1 [") {
		t.Errorf("KANBAN_OUTPUT=compact should produce compact output, got:\n%s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Skill command tests
// ---------------------------------------------------------------------------

// runKanbanNoDir runs the binary without the --dir flag (for skill commands).
