package tui

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/attachment"
)

// Run explicitly after copying a screenshot or browser image on a macOS desktop.
func TestMacClipboardImageSmoke(t *testing.T) {
	if os.Getenv("KANBAN_MD_MAC_CLIPBOARD_SMOKE") != "1" {
		t.Skip("set KANBAN_MD_MAC_CLIPBOARD_SMOKE=1 with an image on the macOS clipboard")
	}
	ctx, cancel := context.WithTimeout(t.Context(), clipboardTimeout)
	defer cancel()
	content, err := readSystemClipboard(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	img, err := attachment.Read(bytes.NewReader(content.image))
	if err != nil || img.Format != "png" {
		t.Fatalf("macOS clipboard did not return a valid PNG image: %v", err)
	}
}
