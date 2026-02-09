package cmd

import (
	"bytes"
	"testing"
)

// saveRootUse saves rootCmd.Use and restores it on cleanup.
// runCompletion modifies rootCmd.Use to match os.Args[0].
func saveRootUse(t *testing.T) {
	t.Helper()
	saved := rootCmd.Use
	t.Cleanup(func() { rootCmd.Use = saved })
}

func TestRunCompletion_Bash(t *testing.T) {
	saveRootUse(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	err := runCompletion(completionCmd, []string{"bash"})
	if err != nil {
		t.Fatalf("runCompletion(bash) error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty bash completion output")
	}
}

func TestRunCompletion_Zsh(t *testing.T) {
	saveRootUse(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	err := runCompletion(completionCmd, []string{"zsh"})
	if err != nil {
		t.Fatalf("runCompletion(zsh) error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty zsh completion output")
	}
}

func TestRunCompletion_Fish(t *testing.T) {
	saveRootUse(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	err := runCompletion(completionCmd, []string{"fish"})
	if err != nil {
		t.Fatalf("runCompletion(fish) error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty fish completion output")
	}
}

func TestRunCompletion_Powershell(t *testing.T) {
	saveRootUse(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	err := runCompletion(completionCmd, []string{"powershell"})
	if err != nil {
		t.Fatalf("runCompletion(powershell) error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty powershell completion output")
	}
}
