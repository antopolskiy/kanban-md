package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/antopolskiy/kanban-md/internal/attachment"
)

type pasteState struct {
	id      uint64
	pending bool
	cancel  context.CancelFunc
	view    view
	step    int
	prompt  bool
}

type pasteFinishedMsg struct {
	id      uint64
	content clipboardContent
	err     error
}

// cancelPaste invalidates even replies that were queued just before cancellation.
func (b *Board) cancelPaste() {
	if b.paste.cancel != nil {
		b.paste.cancel()
	}
	b.paste.id++
	b.paste.pending = false
	b.paste.cancel = nil
}

func (b *Board) beginPaste(read func(context.Context) (clipboardContent, error)) tea.Cmd {
	b.cancelPaste()
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	b.paste = pasteState{
		id: b.paste.id, pending: true, cancel: cancel,
		view: b.view, step: b.createStep, prompt: b.imagePrompt,
	}
	id := b.paste.id
	b.imageNotice = "Reading clipboard or image… (esc:cancel)"
	return func() tea.Msg {
		defer cancel()
		content, err := read(ctx)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return pasteFinishedMsg{id: id, content: content, err: err}
	}
}

func (b *Board) pasteClipboard(images bool) tea.Cmd {
	read := b.clipboardRead
	if read == nil {
		read = readSystemClipboard
	}
	return b.beginPaste(func(ctx context.Context) (clipboardContent, error) {
		return read(ctx, images)
	})
}

func (b *Board) handleCreateAttachmentKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	images := b.createStep == stepTitle || b.createStep == stepBody
	switch msg.String() {
	case keyCtrlV:
		if b.createStep != stepPriority {
			return b.pasteClipboard(images), true
		}
	case keyCtrlO:
		if images {
			b.openImagePrompt()
			return nil, true
		}
	}
	return nil, false
}

func (b *Board) finishPaste(msg pasteFinishedMsg) (tea.Model, tea.Cmd) {
	if !b.paste.pending || b.paste.id != msg.id {
		return b, nil
	}
	target := b.paste
	b.cancelPaste()
	b.imageNotice = ""
	if b.view != target.view || b.createStep != target.step || b.imagePrompt != target.prompt {
		return b, nil
	}
	if msg.err != nil {
		b.err = fmt.Errorf("pasting: %w", msg.err)
		return b, nil
	}
	text := msg.content.text
	if len(msg.content.image) != 0 {
		return b.insertImage(msg.content.image)
	}
	if text == "" {
		b.err = fmt.Errorf("clipboard has no %s", b.clipboardExpected())
		return b, nil
	}
	b.insertPastedText(text)
	return b, nil
}

func (b *Board) clipboardExpected() string {
	if b.view == viewCreate && !b.imagePrompt && (b.createStep == stepTitle || b.createStep == stepBody) {
		return "image or text; use Ctrl+O to import an image file"
	}
	return clipboardText
}

func (b *Board) insertImage(data []byte) (tea.Model, tea.Cmd) {
	if b.view != viewCreate || (b.createStep != stepTitle && b.createStep != stepBody) {
		b.err = errors.New("images can be inserted only in the title and body")
		return b, nil
	}
	img, err := attachment.Read(bytes.NewReader(data))
	if err != nil {
		b.err = err
		return b, nil
	}
	ref, err := attachment.Import(b.cfg.Dir(), b.cfg.TasksPath(), img)
	if err != nil {
		b.err = err
		return b, nil
	}
	if b.imagePrompt {
		b.closeImagePrompt()
	}
	b.insertPastedText(ref)
	b.imageNotice = "Image saved; link inserted"
	return b, nil
}

func insertTextInput(input *textinput.Model, text string) {
	// A paste-marked rune message inserts at the cursor without invoking shortcuts.
	m, _ := input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text), Paste: true})
	*input = m
}

func (b *Board) insertPastedText(text string) {
	switch {
	case b.imagePrompt:
		insertTextInput(&b.imagePathInput, text)
	case b.view == viewSearch:
		insertTextInput(&b.searchInput, text)
		b.filterQuery = b.searchInput.Value()
		b.reloadKeepingSelection()
	case b.createStep == stepTitle:
		insertTextInput(&b.createTitleInput, text)
	case b.createStep == stepBody:
		b.createBodyInput.InsertString(text)
	case b.createStep == stepTags:
		insertTextInput(&b.createTagsInput, text)
	}
}

func (b *Board) openImagePrompt() {
	b.imagePathInput = textinput.New()
	b.imagePathInput.Prompt = ""
	b.imagePathInput.Width = b.createInputWidth("Path: ")
	b.imagePathInput.Focus()
	b.createTitleInput.Blur()
	b.createBodyInput.Blur()
	b.imagePrompt = true
	b.imageNotice = ""
}

func (b *Board) closeImagePrompt() {
	b.cancelPaste()
	b.imagePrompt = false
	b.imagePathInput.Blur()
	b.focusCreateField()
}

func (b *Board) handleImagePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		b.closeImagePrompt()
		return b, nil
	case keyCtrlV:
		return b, b.pasteClipboard(false)
	case keyEnter:
		path := b.imagePathInput.Value()
		return b, b.beginPaste(func(ctx context.Context) (clipboardContent, error) {
			data, err := readImagePath(ctx, path)
			return clipboardContent{image: data}, err
		})
	default:
		b.imagePathInput, _ = b.imagePathInput.Update(msg)
		return b, nil
	}
}

func readImagePath(ctx context.Context, path string) ([]byte, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		if path == "~" {
			path = homeDir
		} else {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("opening image: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("select a regular PNG, JPEG, or GIF image file")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(path) //nolint:gosec // explicit user-selected local image path
	if err != nil {
		return nil, fmt.Errorf("opening image: %w", err)
	}
	defer f.Close()
	img, err := attachment.Read(f)
	return img.Data, err
}

func (b *Board) viewImagePrompt() string {
	b.imagePathInput.Width = b.createInputWidth("Path: ")
	content := lipgloss.NewStyle().Bold(true).Render("Import image path") + "\n\n" +
		b.renderLabeledCreateInput("Path: ", b.imagePathInput.View()) + "\n\n" +
		dimStyle.Render("enter:import  esc:back")
	return b.imageFeedback(content)
}

func (b *Board) imageFeedback(content string) string {
	if b.imageNotice != "" {
		content += "\n" + dimStyle.Render(b.imageNotice)
	}
	if b.err != nil {
		content += "\n" + errorStyle.Render("Error: "+b.err.Error())
	}
	return content
}
