package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestWebCommandDefaults(t *testing.T) {
	cmd := newWebCommand()
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		t.Fatal(err)
	}
	if port != defaultWebPort {
		t.Fatalf("port = %d, want %d", port, defaultWebPort)
	}
	noBrowser, err := cmd.Flags().GetBool("no-browser")
	if err != nil {
		t.Fatal(err)
	}
	if noBrowser {
		t.Fatal("expected browser to open by default")
	}
	found := false
	for _, command := range rootCmd.Commands() {
		if command.Name() == "web" {
			found = true
		}
	}
	if !found {
		t.Fatal("web command is not registered")
	}
}

func TestWebRejectsBothColumnFlags(t *testing.T) {
	kanbanDir := setupBoard(t)
	old := flagDir
	flagDir = kanbanDir
	t.Cleanup(func() { flagDir = old })

	cmd := newWebCommand()
	if err := cmd.Flags().Set("hide-empty-columns", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("show-empty-columns", "true"); err != nil {
		t.Fatal(err)
	}
	err := runWeb(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot use") {
		t.Fatalf("error = %v", err)
	}
}

func TestWebInitRefused(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	old := flagDir
	flagDir = ""
	t.Cleanup(func() { flagDir = old })

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = writer.WriteString("n\n"); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin })

	err = runWeb(newWebCommand(), nil)
	if err == nil || !strings.Contains(err.Error(), "no board found") {
		t.Fatalf("error = %v", err)
	}
}
