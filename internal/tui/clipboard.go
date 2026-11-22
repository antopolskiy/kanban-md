package tui

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"

	"github.com/antopolskiy/kanban-md/internal/attachment"
)

const (
	clipboardTimeout = 3 * time.Second
	clipboardImage   = "image"
	clipboardText    = "text"
	clipboardEmpty   = "empty"
	// Allow for base64 expansion and the small JSON response envelope.
	clipboardOutputLimit = (attachment.MaxBytes+2)/3*4 + 1024 //nolint:mnd // base64 encodes each three bytes as four characters
)

//go:embed clipboard_macos.js
var macClipboardScript string

type clipboardContent struct {
	text  string
	image []byte
}

type clipboardReader func(context.Context, bool) (clipboardContent, error)

func readSystemClipboard(ctx context.Context, images bool) (clipboardContent, error) {
	if runtime.GOOS == "darwin" {
		return readMacClipboard(ctx, images, runMacClipboard)
	}
	text, err := readTextClipboard(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return clipboardContent{}, ctx.Err()
		}
		return clipboardContent{}, errors.New("clipboard unavailable; use terminal text paste or Ctrl+O to import an image")
	}
	if len(text) > attachment.MaxBytes {
		return clipboardContent{}, errors.New("clipboard text exceeds 20 MiB")
	}
	if !utf8.ValidString(text) {
		return clipboardContent{}, errors.New("clipboard has no UTF-8 text; use Ctrl+O to import an image file")
	}
	return clipboardContent{text: text}, nil
}

func readTextClipboard(ctx context.Context) (string, error) {
	if runtime.GOOS == "windows" {
		return clipboard.ReadAll()
	}
	// Reuse the usual desktop tools, but bound execution and captured output.
	commands := [][]string{
		{"xclip", "-out", "-selection", "clipboard", "-target", "UTF8_STRING"},
		{"xsel", "--output", "--clipboard"},
		{"termux-clipboard-get"},
		{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw"},
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		commands = append([][]string{{"wl-paste", "--no-newline", "--type", clipboardText}}, commands...)
	}
	for _, args := range commands {
		path, err := exec.LookPath(args[0])
		if err != nil {
			continue
		}
		cmd := exec.CommandContext(ctx, path, args[1:]...) //nolint:gosec // fixed clipboard command choices, resolved on the user's PATH
		data, err := captureClipboardOutput(cmd)
		return string(data), err
	}
	return "", errors.New("no desktop text clipboard available")
}

func readMacClipboard(ctx context.Context, images bool, run func(context.Context, string) ([]byte, error)) (clipboardContent, error) {
	mode := clipboardText
	if images {
		mode = clipboardImage
	}
	data, err := run(ctx, mode)
	if err != nil {
		return clipboardContent{}, fmt.Errorf("reading macOS clipboard (use Ctrl+O to import a file): %w", err)
	}
	return decodeMacClipboard(data, images)
}

func decodeMacClipboard(data []byte, images bool) (clipboardContent, error) {
	var response struct {
		Kind string `json:"kind"`
		Data string `json:"data"`
		Text string `json:"text"`
	}
	if len(data) > clipboardOutputLimit {
		return clipboardContent{}, errors.New("clipboard output exceeds 20 MiB image limit")
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return clipboardContent{}, errors.New("invalid response from macOS clipboard")
	}
	switch response.Kind {
	case clipboardImage:
		if !images {
			return clipboardContent{}, errors.New("image paste is available only in the title and body")
		}
		decoded, err := base64.StdEncoding.DecodeString(response.Data)
		if err != nil || len(decoded) == 0 {
			return clipboardContent{}, errors.New("invalid image data from macOS clipboard")
		}
		if len(decoded) > attachment.MaxBytes {
			return clipboardContent{}, errors.New("clipboard image exceeds 20 MiB")
		}
		return clipboardContent{image: decoded}, nil
	case clipboardText:
		if len(response.Text) > attachment.MaxBytes {
			return clipboardContent{}, errors.New("clipboard text exceeds 20 MiB")
		}
		return clipboardContent{text: response.Text}, nil
	case clipboardEmpty:
		return clipboardContent{}, nil
	case "error":
		return clipboardContent{}, errors.New("could not convert clipboard image to PNG")
	case "too-large":
		return clipboardContent{}, errors.New("clipboard image exceeds 20 MiB")
	default:
		return clipboardContent{}, errors.New("invalid response from macOS clipboard")
	}
}

// limitedClipboardOutput bounds memory while osascript writes its response.
type limitedClipboardOutput struct {
	bytes.Buffer
}

func (w *limitedClipboardOutput) Write(p []byte) (int, error) {
	if len(p) > clipboardOutputLimit-w.Len() {
		return 0, errors.New("clipboard output exceeds 20 MiB image limit")
	}
	return w.Buffer.Write(p)
}

func runMacClipboard(ctx context.Context, mode string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", "-l", "JavaScript", "-", mode) //nolint:gosec // mode selects image/text for the fixed embedded script
	cmd.Stdin = strings.NewReader(macClipboardScript)
	data, err := captureClipboardOutput(cmd)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return data, err
}

func captureClipboardOutput(cmd *exec.Cmd) ([]byte, error) {
	var output limitedClipboardOutput
	cmd.Stdout = &output
	// Bound waiting for inherited pipes even if a subprocess outlives osascript.
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
