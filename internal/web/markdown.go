package web

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

func sanitizeMarkdown(src string) (string, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return string(bluemonday.UGCPolicy().SanitizeBytes(buf.Bytes())), nil
}
