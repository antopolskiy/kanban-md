package date

import (
	"encoding/json"
	"testing"

	"go.yaml.in/yaml/v3"
)

// --- UnmarshalJSON error paths ---

func TestUnmarshalJSON_NonStringValue(t *testing.T) {
	type wrapper struct {
		Due Date `json:"due"`
	}
	var w wrapper
	// JSON number instead of string.
	err := json.Unmarshal([]byte(`{"due": 12345}`), &w)
	if err == nil {
		t.Fatal("expected error for non-string JSON value")
	}
}

func TestUnmarshalJSON_InvalidDateString(t *testing.T) {
	type wrapper struct {
		Due Date `json:"due"`
	}
	var w wrapper
	err := json.Unmarshal([]byte(`{"due": "not-a-date"}`), &w)
	if err == nil {
		t.Fatal("expected error for invalid date string in JSON")
	}
}

func TestUnmarshalJSON_NullValue(t *testing.T) {
	type wrapper struct {
		Due Date `json:"due"`
	}
	var w wrapper
	// JSON null should fail (can't unmarshal null into string).
	err := json.Unmarshal([]byte(`{"due": null}`), &w)
	if err == nil {
		// null unmarshals to zero string "" which will fail Parse.
		// Either the json.Unmarshal step or Parse step should error.
		t.Fatal("expected error for null JSON value")
	}
}

// --- UnmarshalYAML error path ---

func TestUnmarshalYAML_InvalidDateString(t *testing.T) {
	type wrapper struct {
		Due Date `yaml:"due"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("due: not-a-date\n"), &w)
	if err == nil {
		t.Fatal("expected error for invalid date string in YAML")
	}
}

func TestUnmarshalYAML_EmptyString(t *testing.T) {
	type wrapper struct {
		Due Date `yaml:"due"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("due: \"\"\n"), &w)
	if err == nil {
		t.Fatal("expected error for empty date string in YAML")
	}
}

// --- MarshalJSON ---

func TestMarshalJSON_Value(t *testing.T) {
	d := New(2026, 3, 15)
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	if string(data) != `"2026-03-15"` {
		t.Errorf("MarshalJSON = %s, want %q", data, "2026-03-15")
	}
}

// --- MarshalYAML ---

func TestMarshalYAML_Value(t *testing.T) {
	d := New(2026, 3, 15)
	data, err := yaml.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalYAML error: %v", err)
	}
	if string(data) != "\"2026-03-15\"\n" {
		t.Errorf("MarshalYAML = %q, want %q", string(data), "\"2026-03-15\"\n")
	}
}
