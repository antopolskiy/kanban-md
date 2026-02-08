package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes data as indented JSON to the given writer.
func JSON(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

// ErrorResponse is the JSON envelope for structured error output.
type ErrorResponse struct {
	Error   string         `json:"error"`
	Code    string         `json:"code"`
	Details map[string]any `json:"details,omitempty"`
}

// JSONError writes a structured error to the given writer as JSON.
func JSONError(w io.Writer, code, msg string, details map[string]any) {
	resp := ErrorResponse{Error: msg, Code: code, Details: details}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp) // best-effort; if writer fails, nothing we can do
}

// BatchResult represents the outcome of a single operation within a batch.
type BatchResult struct {
	ID    int    `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Code  string `json:"code,omitempty"`
}
