package tui

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	imageTestTitle  = "title"
	imageTestBody   = "body"
	imageTestPath   = "path"
	imageTestSearch = "search"
)

func imageTestPNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func imageKey(b *Board, key tea.KeyType) tea.Cmd {
	_, cmd := b.Update(tea.KeyMsg{Type: key})
	return cmd
}

func runImageCommand(t *testing.T, b *Board, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected an asynchronous paste/import command")
	}
	msg := cmd()
	_, _ = b.Update(msg)
	return msg
}

func imageReferencePath(t *testing.T, cfg *config.Config, text string) string {
	t.Helper()
	start := strings.Index(text, "[screenshot](")
	if start < 0 {
		t.Fatalf("missing image reference in %q", text)
	}
	link := strings.SplitN(text[start+len("[screenshot]("):], ")", 2)[0]
	return filepath.Join(cfg.TasksPath(), filepath.FromSlash(link))
}

func TestClipboardImageInsertsAtTitleAndBodyCursor(t *testing.T) {
	for _, step := range []int{stepTitle, stepBody} {
		t.Run(map[int]string{stepTitle: imageTestTitle, stepBody: imageTestBody}[step], func(t *testing.T) {
			b, cfg := newWorkflowBoard(t, nil)
			b.handleCreateStart()
			b.createTitleInput.SetValue("A task")
			b.createStep = step
			b.focusCreateField()
			data := imageTestPNG(t)
			b.clipboardRead = func(_ context.Context, images bool) (clipboardContent, error) {
				if !images {
					t.Fatal("title/body paste did not request an image")
				}
				return clipboardContent{image: data}, nil
			}
			if step == stepTitle {
				b.createTitleInput.SetValue("αβ end")
				b.createTitleInput.SetCursor(2)
			} else {
				b.createBodyInput.SetValue("αβ end")
				b.createBodyInput.SetCursor(2)
			}
			msg := runImageCommand(t, b, imageKey(b, tea.KeyCtrlV))
			_, _ = b.Update(msg) // duplicate delivery must not insert twice
			text := b.createTitleInput.Value()
			if step == stepBody {
				text = b.createBodyInput.Value()
			}
			if !strings.HasPrefix(text, "αβ[screenshot](") || !strings.HasSuffix(text, ") end") || strings.Count(text, "[screenshot]") != 1 {
				t.Fatalf("image was not inserted once at the cursor: %q", text)
			}
			stored, err := os.ReadFile(imageReferencePath(t, cfg, text))
			if err != nil || !bytes.Equal(stored, data) {
				t.Fatalf("stored image differs: %v", err)
			}
			if cfg.NextID != 1 || !strings.Contains(b.View(), "Image saved; link inserted") {
				t.Fatal("image import changed the task ID or hid confirmation")
			}
			imageKey(b, tea.KeyEnter)
			path, err := task.FindByID(cfg.TasksPath(), 1)
			if err != nil {
				t.Fatal(err)
			}
			created, err := task.Read(path)
			if err != nil || !strings.Contains(created.Title+created.Body, text) {
				t.Fatalf("saved task lost reference: %v", err)
			}
		})
	}
}

func TestCanceledPasteCannotChangeLaterDraft(t *testing.T) {
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	b.clipboardRead = func(context.Context, bool) (clipboardContent, error) {
		return clipboardContent{image: imageTestPNG(t)}, nil
	}
	cmd := imageKey(b, tea.KeyCtrlV)
	imageKey(b, tea.KeyTab)
	if b.createStep != stepTitle {
		t.Fatal("pending paste allowed changing fields")
	}
	imageKey(b, tea.KeyEscape)
	b.handleCreateStart()
	b.createTitleInput.SetValue("Later draft")
	runImageCommand(t, b, cmd)
	if b.createTitleInput.Value() != "Later draft" || b.err != nil {
		t.Fatal("canceled paste changed a later draft")
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir(), "attachments")); !os.IsNotExist(err) {
		t.Fatalf("canceled paste created stored images: %v", err)
	}
}

func TestFileImportPreservesDraftAndRetainsSavedImageOnCancel(t *testing.T) {
	b, cfg := newWorkflowBoard(t, nil)
	data := imageTestPNG(t)
	source := filepath.Join(t.TempDir(), "a screenshot.png")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	b.handleCreateStart()
	b.createTitleInput.SetValue("Draft")
	b.createTitleInput.SetCursor(5)
	imageKey(b, tea.KeyCtrlO)
	if !strings.Contains(b.View(), "Import image path") {
		t.Fatal("missing path import prompt")
	}
	b.imagePathInput.SetValue(source)
	runImageCommand(t, b, imageKey(b, tea.KeyEnter))
	storedPath := imageReferencePath(t, cfg, b.createTitleInput.Value())
	if b.imagePrompt || !strings.HasPrefix(b.createTitleInput.Value(), "Draft[screenshot]") {
		t.Fatal("import did not return to the originating cursor")
	}
	imageKey(b, tea.KeyEscape)
	stored, err := os.ReadFile(storedPath) //nolint:gosec // test-owned imported image
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("cancel removed the imported image: %v", err)
	}
	entries, err := os.ReadDir(cfg.TasksPath())
	if err != nil || len(entries) != 0 {
		t.Fatal("import created a task before saving")
	}
}

func TestImageImportFailureKeepsPromptAndDraft(t *testing.T) {
	b, cfg := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	b.createTitleInput.SetValue("Keep this")
	imageKey(b, tea.KeyCtrlO)
	b.imagePathInput.SetValue("missing-image.png")
	runImageCommand(t, b, imageKey(b, tea.KeyEnter))
	if !b.imagePrompt || b.createTitleInput.Value() != "Keep this" || b.imagePathInput.Value() != "missing-image.png" {
		t.Fatal("failed import discarded the prompt or draft")
	}
	if b.err == nil || !strings.Contains(b.View(), "opening image") {
		t.Fatal("import error is not visible in the prompt")
	}
	imageKey(b, tea.KeyEscape)
	if b.view != viewCreate || b.imagePrompt {
		t.Fatal("canceling path prompt did not restore draft")
	}
	if err := os.WriteFile(filepath.Join(cfg.Dir(), "attachments"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	b.clipboardRead = func(context.Context, bool) (clipboardContent, error) {
		return clipboardContent{image: imageTestPNG(t)}, nil
	}
	runImageCommand(t, b, imageKey(b, tea.KeyCtrlV))
	if b.err == nil || b.createTitleInput.Value() != "Keep this" {
		t.Fatal("failed storage changed the title or hid the error")
	}
}

func TestTextPasteInEditorSearchAndPathPrompt(t *testing.T) {
	for _, target := range []string{imageTestTitle, imageTestBody, "tags", imageTestSearch, imageTestPath} {
		t.Run(target, func(t *testing.T) {
			b, cfg := newWorkflowBoard(t, nil)
			b.handleCreateStart()
			switch target {
			case imageTestBody:
				b.createStep = stepBody
			case "tags":
				b.createStep = stepTags
			case imageTestSearch:
				b.handleSearchStart()
			case imageTestPath:
				b.openImagePrompt()
			}
			if target != imageTestPath {
				b.focusCreateField()
			}
			b.clipboardRead = func(_ context.Context, images bool) (clipboardContent, error) {
				if images != (target == imageTestTitle || target == imageTestBody) {
					t.Fatal("wrong image capability for text input")
				}
				return clipboardContent{text: "/tmp/an-image.png"}, nil
			}
			runImageCommand(t, b, imageKey(b, tea.KeyCtrlV))
			text := b.createTitleInput.Value() + b.createBodyInput.Value() + b.createTagsInput.Value() + b.searchInput.Value() + b.imagePathInput.Value()
			if !strings.Contains(text, "/tmp/an-image.png") || b.err != nil {
				t.Fatalf("text paste did not reach %s: %v", target, b.err)
			}
			if _, err := os.Stat(filepath.Join(cfg.Dir(), "attachments")); !os.IsNotExist(err) {
				t.Fatal("pasted file path triggered an implicit image import")
			}
		})
	}
}

func TestBracketedTextPasteNeverReadsImageClipboard(t *testing.T) {
	b, _ := newWorkflowBoard(t, nil)
	b.handleCreateStart()
	b.clipboardRead = func(context.Context, bool) (clipboardContent, error) {
		t.Fatal("terminal text paste must not inspect the system clipboard")
		return clipboardContent{}, nil
	}
	_, _ = b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyCtrlO), Paste: true})
	if b.imagePrompt || b.createTitleInput.Value() != keyCtrlO {
		t.Fatal("pasted text activated an application shortcut")
	}
}

func TestClipboardReadFailureRetainsDraft(t *testing.T) {
	for _, failure := range []error{errors.New("unavailable"), context.DeadlineExceeded} {
		b, _ := newWorkflowBoard(t, nil)
		b.handleCreateStart()
		b.createTitleInput.SetValue("Retain")
		b.clipboardRead = func(context.Context, bool) (clipboardContent, error) {
			return clipboardContent{}, failure
		}
		runImageCommand(t, b, imageKey(b, tea.KeyCtrlV))
		if b.createTitleInput.Value() != "Retain" || b.err == nil || b.paste.pending {
			t.Fatal("clipboard failure lost draft or left input locked")
		}
	}
}

func TestReadImagePathUsesLiteralRelativePaths(t *testing.T) {
	data := imageTestPNG(t)
	dir := t.TempDir()
	name := "a screenshot.png"
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	got, err := readImagePath(t.Context(), name)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("relative path with spaces did not import: %v", err)
	}
	if _, err := readImagePath(t.Context(), "~"); err == nil || !strings.Contains(err.Error(), "regular") {
		t.Fatalf("home shorthand was not expanded to a directory: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := readImagePath(ctx, name); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled import ignored cancellation: %v", err)
	}
}
