package date

import (
	"encoding/json"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func TestNew(t *testing.T) {
	d := New(2026, time.February, 14)
	if got := d.String(); got != "2026-02-14" {
		t.Errorf("New(2026, Feb, 14).String() = %q, want %q", got, "2026-02-14")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"2026-02-14", "2026-02-14", false},
		{"2000-01-01", "2000-01-01", false},
		{"not-a-date", "", true},
		{"02-14-2026", "", true},
		{"2026/02/14", "", true},
	}
	for _, tt := range tests {
		d, err := Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && d.String() != tt.want {
			t.Errorf("Parse(%q).String() = %q, want %q", tt.input, d.String(), tt.want)
		}
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	type wrapper struct {
		Due Date `yaml:"due"`
	}

	original := wrapper{Due: New(2026, time.March, 15)}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal error: %v", err)
	}

	if got := string(data); got != "due: \"2026-03-15\"\n" {
		t.Errorf("yaml.Marshal = %q", got)
	}

	var decoded wrapper
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal error: %v", err)
	}

	if decoded.Due.String() != "2026-03-15" {
		t.Errorf("yaml round-trip = %q, want %q", decoded.Due.String(), "2026-03-15")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		Due Date `json:"due"`
	}

	original := wrapper{Due: New(2026, time.March, 15)}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	if got := string(data); got != `{"due":"2026-03-15"}` {
		t.Errorf("json.Marshal = %q", got)
	}

	var decoded wrapper
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.Due.String() != "2026-03-15" {
		t.Errorf("json round-trip = %q, want %q", decoded.Due.String(), "2026-03-15")
	}
}

func TestToday(t *testing.T) {
	d := Today()
	now := time.Now()
	expected := now.Format("2006-01-02")
	if d.String() != expected {
		t.Errorf("Today() = %q, want %q", d.String(), expected)
	}
}
