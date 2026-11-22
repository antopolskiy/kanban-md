//go:build !windows

package tui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/task"
)

const terminalTestImageName = "clipboard-2026-11-22-162239-C8CD00A5.png"

func terminalTestImage(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), terminalTestImageName)
	data := imageTestPNG(t)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func pasteTerminalPath(b *Board, path string) tea.Cmd {
	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(path), Paste: true})
	return cmd
}

func TestTerminalImagePasteCopiesIntoActiveBoard(t *testing.T) {
	for _, step := range []int{stepTitle, stepBody} {
		t.Run(map[int]string{stepTitle: imageTestTitle, stepBody: imageTestBody}[step], func(t *testing.T) {
			b, cfg := newWorkflowBoard(t, nil)
			b.handleCreateStart()
			b.createTitleInput.SetValue("Evidence task")
			b.createStep = step
			b.focusCreateField()
			if step == stepTitle {
				b.createTitleInput.SetValue("Before  after")
				b.createTitleInput.SetCursor(7)
			} else {
				b.createBodyInput.SetValue("Before  after")
				b.createBodyInput.SetCursor(7)
			}
			b.clipboardRead = func(context.Context, bool) (clipboardContent, error) {
				t.Fatal("terminal image paste must read the supplied file, not a later system clipboard")
				return clipboardContent{}, nil
			}
			source, data := terminalTestImage(t)
			runImageCommand(t, b, pasteTerminalPath(b, source))
			text := b.createTitleInput.Value()
			if step == stepBody {
				text = b.createBodyInput.Value()
			}
			if !strings.HasPrefix(text, "Before [screenshot](") || !strings.HasSuffix(text, ") after") || strings.Contains(text, source) {
				t.Fatalf("temporary image path was not replaced at the cursor: %q", text)
			}
			storedPath := imageReferencePath(t, cfg, text)
			if filepath.Dir(storedPath) != filepath.Join(cfg.Dir(), "attachments") {
				t.Fatalf("image saved outside active board: %s", storedPath)
			}
			if err := os.Remove(source); err != nil {
				t.Fatal(err)
			}
			stored, err := os.ReadFile(storedPath) //nolint:gosec // test-owned board attachment
			if err != nil || !bytes.Equal(stored, data) {
				t.Fatalf("image did not survive terminal temporary-file cleanup: %v", err)
			}
			imageKey(b, tea.KeyEnter)
			path, err := task.FindByID(cfg.TasksPath(), 1)
			if err != nil {
				t.Fatal(err)
			}
			created, err := task.Read(path)
			if err != nil || !strings.Contains(created.Title+created.Body, text) {
				t.Fatalf("task lost the durable reference: %v", err)
			}
		})
	}
}

func TestTerminalImagePathRecognition(t *testing.T) {
	for _, tt := range []struct {
		name string
		text string
		path string
	}{
		{"cmux path", "/var/folders/4z/example/T/" + terminalTestImageName, "/var/folders/4z/example/T/" + terminalTestImageName},
		{"trailing separator", "/tmp/" + terminalTestImageName + " ", "/tmp/" + terminalTestImageName},
		{"escaped directory", "/tmp/a\\ screenshot\\ \\(copy\\)/" + terminalTestImageName, "/tmp/a screenshot (copy)/" + terminalTestImageName},
		{"ordinary image path", "/tmp/screenshot.png", ""},
		{"ordinary clipboard name", "/tmp/clipboard-not-a-terminal-image.png", ""},
		{"prose", "See /tmp/" + terminalTestImageName, ""},
		{"multiple files", "/tmp/first.png /tmp/" + terminalTestImageName, ""},
		{"newline", "/tmp/\n" + terminalTestImageName, ""},
		{"incomplete escape", "/tmp/" + terminalTestImageName + "\\", ""},
		{"relative path", terminalTestImageName, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path, matched := terminalImagePath(tt.text)
			if matched != (tt.path != "") || (matched && path != tt.path) {
				t.Fatalf("terminalImagePath(%q)=%q/%v, want %q", tt.text, path, matched, tt.path)
			}
		})
	}
}

func TestTerminalImagePastePreservesTextInputs(t *testing.T) {
	source, _ := terminalTestImage(t)
	for _, target := range []string{imageTestTitle, "tags", imageTestSearch, imageTestPath} {
		t.Run(target, func(t *testing.T) {
			b, cfg := newWorkflowBoard(t, nil)
			b.handleCreateStart()
			switch target {
			case "tags":
				b.createStep = stepTags
				b.focusCreateField()
			case imageTestSearch:
				b.handleSearchStart()
			case imageTestPath:
				b.openImagePrompt()
			}
			if target == imageTestTitle {
				// Typed paths are ordinary text even if the file is a cmux image.
				_, _ = b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(source)})
			} else {
				pasteTerminalPath(b, source)
			}
			text := b.createTitleInput.Value() + b.createTagsInput.Value() + b.searchInput.Value() + b.imagePathInput.Value()
			if text != source || b.paste.pending {
				t.Fatalf("ordinary text input changed: %q", text)
			}
			if _, err := os.Stat(filepath.Join(cfg.Dir(), "attachments")); !os.IsNotExist(err) {
				t.Fatal("text-only input imported an image")
			}
		})
	}
}

func TestTerminalImagePasteCancelAndReadFailure(t *testing.T) {
	const laterDraft = "Terminal draft after cancellation"
	source, _ := terminalTestImage(t)
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	b.createTitleInput.SetValue("Keep draft")
	cmd := pasteTerminalPath(b, source)
	imageKey(b, tea.KeyEscape)
	b.handleCreateStart()
	b.createTitleInput.SetValue(laterDraft)
	runImageCommand(t, b, cmd)
	if b.createTitleInput.Value() != laterDraft || b.err != nil {
		t.Fatal("canceled terminal import changed a later draft")
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	runImageCommand(t, b, pasteTerminalPath(b, source))
	if b.err == nil || b.createTitleInput.Value() != laterDraft || b.paste.pending {
		t.Fatal("missing terminal image path was inserted as text or discarded the draft")
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "attachments")); !os.IsNotExist(err) {
		t.Fatal("canceled or failed paste wrote attachments")
	}
}

func TestTerminalImagePasteWithEscapedDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "clipboard images")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, terminalTestImageName)
	data := imageTestPNG(t)
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	escaped := strings.ReplaceAll(source, " ", "\\ ")
	runImageCommand(t, b, pasteTerminalPath(b, escaped))
	stored, err := os.ReadFile(imageReferencePath(t, cfg, b.createTitleInput.Value()))
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("escaped terminal path did not produce a board-local copy: %v", err)
	}
}

func TestTerminalPasteKeepsOrdinaryImagePathAsText(t *testing.T) {
	source, _ := terminalTestImage(t)
	ordinary := filepath.Join(filepath.Dir(source), "ordinary-image.png")
	if err := os.Rename(source, ordinary); err != nil {
		t.Fatal(err)
	}
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	pasteTerminalPath(b, ordinary)
	if b.createTitleInput.Value() != ordinary || b.paste.pending {
		t.Fatal("ordinary pasted image path was unexpectedly imported")
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "attachments")); !os.IsNotExist(err) {
		t.Fatal("ordinary path paste created an attachment")
	}
}

func TestTerminalImagePasteRejectsInvalidImage(t *testing.T) {
	source, _ := terminalTestImage(t)
	if err := os.WriteFile(source, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	b.createTitleInput.SetValue("Keep draft text")
	runImageCommand(t, b, pasteTerminalPath(b, source))
	if b.err == nil || b.createTitleInput.Value() != "Keep draft text" {
		t.Fatal("invalid image path changed the draft or hid the error")
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "attachments")); !os.IsNotExist(err) {
		t.Fatal("invalid image created an attachment")
	}
}
