package e2e_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsafeConsistencyRepairsBlockCLIWithoutDestruction(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		t.Run(fmt.Sprintf("duplicate=%t", duplicate), func(t *testing.T) {
			kanbanDir := initBoard(t)
			created := mustCreateTask(t, kanbanDir, "Generic sample")
			original, err := os.ReadFile(created.File)
			if err != nil {
				t.Fatal(err)
			}
			closing := strings.LastIndex(string(original), "---\n")
			unsafe := []byte(string(original[:closing]) + "estimate: &shared 4h\ncustom_copy: *shared\n" + string(original[closing:]))
			unsafePath := filepath.Join(filepath.Dir(created.File), "099-mismatch.md")
			if err = os.WriteFile(unsafePath, unsafe, 0o600); err != nil { //nolint:gosec,nolintlint // test-owned temporary path
				t.Fatal(err)
			}
			if !duplicate {
				if err = os.Remove(created.File); err != nil {
					t.Fatal(err)
				}
			}
			logPath := filepath.Join(kanbanDir, "activity.jsonl")
			beforeLog, err := os.ReadFile(logPath) //nolint:gosec // test-owned activity log
			if err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{
				{"show", "1"},
				{"pick", preservationClaimFlag, claimTestAgent},
				{"edit", "1", "--title", "Changed sample"},
			} {
				r := runKanban(t, kanbanDir, args...)
				if r.exitCode == 0 || !strings.Contains(r.stderr, "rewriting task file") || !strings.Contains(r.stderr, "cannot preserve") {
					t.Fatalf("%v result = %d %s, want required repair refusal", args, r.exitCode, r.stderr)
				}
				assertTaskBytes(t, unsafePath, unsafe)
				if duplicate {
					assertTaskBytes(t, created.File, original)
				}
				assertTaskBytes(t, logPath, beforeLog)
			}
			entries, err := os.ReadDir(filepath.Dir(created.File))
			if err != nil {
				t.Fatal(err)
			}
			want := 1
			if duplicate {
				want++
			}
			if len(entries) != want {
				t.Errorf("refused consistency repair created files: %v", entries)
			}
		})
	}
}

func assertTaskBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path) //nolint:gosec // test-owned path
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("refusal changed %s:\n%s", path, got)
	}
}
