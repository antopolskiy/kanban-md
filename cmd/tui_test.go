package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/tui"
)

// --- isBoardNotFound tests ---

func TestIsBoardNotFound_True(t *testing.T) {
	err := clierr.New(clierr.BoardNotFound, "no board")
	if !isBoardNotFound(err) {
		t.Error("expected true for BoardNotFound error")
	}
}

func TestIsBoardNotFound_FalseOtherCode(t *testing.T) {
	err := clierr.New(clierr.TaskNotFound, "no task")
	if isBoardNotFound(err) {
		t.Error("expected false for non-BoardNotFound clierr")
	}
}

func TestIsBoardNotFound_FalseGenericError(t *testing.T) {
	err := errors.New("something else")
	if isBoardNotFound(err) {
		t.Error("expected false for generic error")
	}
}

func TestIsBoardNotFound_FalseNil(t *testing.T) {
	if isBoardNotFound(nil) {
		t.Error("expected false for nil error")
	}
}

// --- offerInitTUI tests ---

func TestOfferInitTUI_AcceptDefault(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Pipe "\n" (empty answer = default = yes).
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("\n"))
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Capture stdout to avoid polluting test output.
	rOut, wOut := captureStdout(t)

	cfg, initErr := offerInitTUI()

	_ = drainPipe(t, rOut, wOut)

	if initErr != nil {
		t.Fatalf("offerInitTUI() error: %v", initErr)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Verify board was created.
	kanbanDir := filepath.Join(dir, config.DefaultDir)
	if _, statErr := os.Stat(filepath.Join(kanbanDir, "config.yml")); statErr != nil {
		t.Errorf("expected config.yml to exist in %s", kanbanDir)
	}
}

func TestOfferInitTUI_AcceptY(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("y\n"))
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	rOut, wOut := captureStdout(t)

	cfg, initErr := offerInitTUI()

	_ = drainPipe(t, rOut, wOut)

	if initErr != nil {
		t.Fatalf("offerInitTUI() error: %v", initErr)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestOfferInitTUI_AcceptYes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("yes\n"))
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	rOut, wOut := captureStdout(t)

	cfg, initErr := offerInitTUI()

	_ = drainPipe(t, rOut, wOut)

	if initErr != nil {
		t.Fatalf("offerInitTUI() error: %v", initErr)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestOfferInitTUI_Decline(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("n\n"))
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	rOut, wOut := captureStdout(t)

	_, initErr := offerInitTUI()

	_ = drainPipe(t, rOut, wOut)

	if initErr == nil {
		t.Fatal("expected error when user declines")
	}
	if !containsSubstring(initErr.Error(), "no board found") {
		t.Errorf("expected 'no board found' in error, got: %v", initErr)
	}
}

func TestOfferInitTUI_DeclineArbitrary(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("nope\n"))
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	rOut, wOut := captureStdout(t)

	_, initErr := offerInitTUI()

	_ = drainPipe(t, rOut, wOut)

	if initErr == nil {
		t.Fatal("expected error for non-yes answer")
	}
}

// --- RunTUI / runTUI tests ---

func TestRunTUI_ConfigLoadError_NonBoardNotFound(t *testing.T) {
	// Point flagDir to a directory that exists but has an invalid config.
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "kanban")
	if err := os.MkdirAll(kanbanDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Write an invalid config file so config.Load fails with a non-BoardNotFound error.
	if err := os.WriteFile(filepath.Join(kanbanDir, "config.yml"), []byte("invalid: [yaml: broken"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldFlagDir := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := runTUI(nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	// Should NOT be a BoardNotFound error — it should be a parse error.
	if isBoardNotFound(err) {
		t.Error("expected non-BoardNotFound error for invalid config")
	}
}

func TestRunTUI_ConfigLoadError_BoardNotFound_Decline(t *testing.T) {
	// Use a temp dir with no board.
	dir := t.TempDir()
	t.Chdir(dir)

	oldFlagDir := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = oldFlagDir })

	// Pipe "n\n" so offerInitTUI declines.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("n\n"))
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	rOut, wOut := captureStdout(t)

	runErr := runTUI(nil, nil)

	_ = drainPipe(t, rOut, wOut)

	if runErr == nil {
		t.Fatal("expected error when user declines init")
	}
	if !containsSubstring(runErr.Error(), "no board found") {
		t.Errorf("expected 'no board found' in error, got: %v", runErr)
	}
}

func TestRunTUI_PublicAPI_SetsDir(t *testing.T) {
	// RunTUI with an invalid dir should fail with a config error.
	dir := t.TempDir()
	kanbanDir := filepath.Join(dir, "nonexistent-kanban")

	oldFlagDir := flagDir
	t.Cleanup(func() { flagDir = oldFlagDir })

	err := RunTUI(kanbanDir)
	if err == nil {
		t.Fatal("expected error for non-existent kanban dir")
	}
	// Verify flagDir was set.
	if flagDir != kanbanDir {
		t.Errorf("flagDir = %q, want %q", flagDir, kanbanDir)
	}
}

func TestRunTUI_PublicAPI_EmptyDir(t *testing.T) {
	// RunTUI with empty dir should not overwrite flagDir.
	dir := t.TempDir()

	oldFlagDir := flagDir
	flagDir = dir
	t.Cleanup(func() { flagDir = oldFlagDir })

	// This will fail because dir has no config, but that's expected.
	_ = RunTUI("")

	if flagDir != dir {
		t.Errorf("flagDir changed to %q, expected it to stay %q", flagDir, dir)
	}
}

func newTUIFlagTestCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "tui"}
	cmd.Flags().Bool("hide-empty-columns", false, "")
	cmd.Flags().Bool("show-empty-columns", false, "")
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cmd
}

func TestResolveHideEmptyColumns_DefaultFromConfig(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.TUI.HideEmptyColumns = true

	hide, err := resolveHideEmptyColumns(newTUIFlagTestCmd(t), cfg)
	if err != nil {
		t.Fatalf("resolveHideEmptyColumns() error: %v", err)
	}
	if !hide {
		t.Fatal("hide_empty_columns should come from config when no override flags are set")
	}
}

func TestResolveHideEmptyColumns_HideFlagOverridesConfig(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.TUI.HideEmptyColumns = false

	hide, err := resolveHideEmptyColumns(newTUIFlagTestCmd(t, "--hide-empty-columns"), cfg)
	if err != nil {
		t.Fatalf("resolveHideEmptyColumns() error: %v", err)
	}
	if !hide {
		t.Fatal("--hide-empty-columns should force hiding empty columns")
	}
}

func TestResolveHideEmptyColumns_ShowFlagOverridesConfig(t *testing.T) {
	cfg := config.NewDefault("Test")
	cfg.TUI.HideEmptyColumns = true

	hide, err := resolveHideEmptyColumns(newTUIFlagTestCmd(t, "--show-empty-columns"), cfg)
	if err != nil {
		t.Fatalf("resolveHideEmptyColumns() error: %v", err)
	}
	if hide {
		t.Fatal("--show-empty-columns should force showing empty columns")
	}
}

func TestResolveHideEmptyColumns_ConflictingFlags(t *testing.T) {
	cfg := config.NewDefault("Test")

	_, err := resolveHideEmptyColumns(newTUIFlagTestCmd(t, "--hide-empty-columns", "--show-empty-columns"), cfg)
	if err == nil {
		t.Fatal("expected error for conflicting empty-column override flags")
	}

	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) || cliErr.Code != clierr.StatusConflict {
		t.Fatalf("expected STATUS_CONFLICT cli error, got: %v", err)
	}
}

// --- startTUIWatcher tests ---

func TestStartTUIWatcher_CancelledContext(t *testing.T) {
	kanbanDir := setupBoard(t)
	cfg, err := config.Load(kanbanDir)
	if err != nil {
		t.Fatal(err)
	}

	model := tui.NewBoard(cfg)

	// The watcher.New call should succeed (valid paths), and Run
	// should return immediately because context is already canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	done := make(chan struct{})
	go func() {
		// Pass nil for Program — startTUIWatcher only uses p.Send which
		// won't be called because the context is already canceled.
		startTUIWatcher(ctx, model, nil)
		close(done)
	}()

	select {
	case <-done:
		// Success — returned promptly.
	case <-time.After(5 * time.Second):
		t.Fatal("startTUIWatcher did not return after context cancellation")
	}
}

func TestStartTUIWatcher_InvalidPath(t *testing.T) {
	// Create a config pointing to a non-existent directory so watcher.New fails.
	cfg := config.NewDefault("test")
	cfg.SetDir(filepath.Join(t.TempDir(), "nonexistent"))

	model := tui.NewBoard(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should return immediately because watcher.New fails (non-fatal).
	done := make(chan struct{})
	go func() {
		startTUIWatcher(ctx, model, nil)
		close(done)
	}()

	select {
	case <-done:
		// Success — returned promptly on watcher creation error.
	case <-time.After(5 * time.Second):
		t.Fatal("startTUIWatcher did not return after watcher creation error")
	}
}
