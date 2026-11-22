//go:build !windows

package e2e_test

import (
	"bytes"
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestE2E_TUI_ImageReferencesSurviveRenameArchiveAndRelocation(t *testing.T) {
	boardDir := initBoard(t)
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "screenshot with spaces.png")
	if err := os.WriteFile(source, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	session := startTUIProcess(t, boardDir)
	session.waitForOutput(tuiBoardStatus)
	session.pressKeys("c")
	session.waitForOutput("Create task in")
	session.typeText("Evidence ")
	importTUIImage(t, session, source)
	session.pressKeys("tab")
	session.waitForOutput("Step 2/4")
	session.typeText("See the screenshot: ")
	importTUIImage(t, session, source)
	session.pressKeys("enter")
	session.waitForOutput(tuiBoardStatus)
	session.pressKeys("q")
	session.waitForExit()

	var created taskJSON
	r := runKanbanJSON(t, boardDir, &created, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show image task failed: %s", r.stderr)
	}
	if !strings.Contains(created.Title, "[screenshot](") || !strings.Contains(created.Body, "[screenshot](") {
		t.Fatalf("title/body references missing: %#v", created)
	}
	verifyTaskImages(t, created, data.Bytes())
	r = runKanban(t, boardDir, "edit", "1", "--title", "Renamed "+created.Title)
	if r.exitCode != 0 {
		t.Fatalf("rename failed: %s", r.stderr)
	}
	r = runKanban(t, boardDir, "archive", "1")
	if r.exitCode != 0 {
		t.Fatalf("archive failed: %s", r.stderr)
	}
	moved := filepath.Join(t.TempDir(), "moved-board")
	if err := os.Rename(boardDir, moved); err != nil {
		t.Fatal(err)
	}
	var relocated taskJSON
	r = runKanbanJSON(t, moved, &relocated, "show", "1")
	if r.exitCode != 0 || relocated.Status != statusArchived {
		t.Fatalf("relocated archived task could not be read: %s", r.stderr)
	}
	verifyTaskImages(t, relocated, data.Bytes())
	if relocated.Body != created.Body {
		t.Fatal("rename/archive/relocation changed the image reference")
	}
}

func importTUIImage(t *testing.T, session *tuiSession, source string) {
	t.Helper()
	checkpoint := session.checkpoint()
	session.pressKeys("ctrl+o")
	session.waitForOutputSince(checkpoint, "Import image path")
	if err := session.writeRaw([]byte("\x1b[200~" + source + "\x1b[201~")); err != nil {
		t.Fatal(err)
	}
	checkpoint = session.checkpoint()
	session.pressKeys("enter")
	session.waitForOutputSince(checkpoint, "Image saved; link inserted")
}

func verifyTaskImages(t *testing.T, task taskJSON, expected []byte) {
	t.Helper()
	links := regexp.MustCompile(`\[screenshot\]\(([^)]+)\)`).FindAllStringSubmatch(task.Title+"\n"+task.Body, -1)
	if len(links) != 2 {
		t.Fatalf("expected title and body image references, got %v", links)
	}
	for _, link := range links {
		relative, err := url.PathUnescape(link[1])
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(filepath.Dir(task.File), filepath.FromSlash(relative))
		stored, err := os.ReadFile(path) //nolint:gosec // test-owned image reference
		if err != nil || !bytes.Equal(stored, expected) {
			t.Fatalf("task reference did not resolve to original image: %v", err)
		}
	}
}
