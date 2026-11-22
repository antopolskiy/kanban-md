package tui

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestDecodeMacClipboard(t *testing.T) {
	pngData := imageTestPNG(t)
	for _, tt := range []struct {
		name   string
		json   string
		images bool
		text   string
		image  bool
		fails  bool
	}{
		{name: clipboardImage, json: `{"kind":"image","data":"` + base64.StdEncoding.EncodeToString(pngData) + `"}`, images: true, image: true},
		{name: clipboardText, json: `{"kind":"text","text":"hello\nworld"}`, text: "hello\nworld"},
		{name: clipboardEmpty, json: `{"kind":"empty"}`},
		{name: "text-only image refusal", json: `{"kind":"image","data":"aGk="}`, fails: true},
		{name: "conversion error", json: `{"kind":"error"}`, fails: true},
		{name: "oversized image", json: `{"kind":"too-large"}`, fails: true},
		{name: "bad base64", json: `{"kind":"image","data":"!"}`, images: true, fails: true},
		{name: "empty image", json: `{"kind":"image","data":""}`, images: true, fails: true},
		{name: "invalid JSON", json: `not JSON`, fails: true},
		{name: "unknown result", json: `{"kind":"unknown"}`, fails: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeMacClipboard([]byte(tt.json), tt.images)
			if (err != nil) != tt.fails {
				t.Fatalf("error=%v, want failure=%v", err, tt.fails)
			}
			if got.text != tt.text || (len(got.image) != 0) != tt.image {
				t.Fatalf("wrong clipboard representation: text=%q, image=%v", got.text, len(got.image) != 0)
			}
			if tt.image && !bytes.Equal(got.image, pngData) {
				t.Fatal("clipboard image bytes changed")
			}
		})
	}
}

func TestMacClipboardReaderSelectsModeAndReportsFailure(t *testing.T) {
	for _, images := range []bool{true, false} {
		wantMode := clipboardText
		if images {
			wantMode = clipboardImage
		}
		_, err := readMacClipboard(t.Context(), images, func(_ context.Context, mode string) ([]byte, error) {
			if mode != wantMode {
				t.Fatalf("mode=%q, want %q", mode, wantMode)
			}
			return nil, context.DeadlineExceeded
		})
		if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "Ctrl+O") {
			t.Fatalf("missing failure and fallback guidance: %v", err)
		}
	}
}

func TestClipboardOutputIsBounded(t *testing.T) {
	var out limitedClipboardOutput
	if _, err := out.Write(bytes.Repeat([]byte{'a'}, clipboardOutputLimit)); err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte{'b'}); err == nil || out.Len() != clipboardOutputLimit {
		t.Fatal("clipboard writer exceeded its output limit")
	}
	if _, err := decodeMacClipboard(bytes.Repeat([]byte{'a'}, clipboardOutputLimit+1), true); err == nil {
		t.Fatal("oversized clipboard output accepted")
	}
}
