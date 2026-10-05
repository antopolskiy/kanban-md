// ---
// relationships:
//   verifies: frontmatter
// ---

package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	frontmatterTestReference = "sample-17"
	frontmatterTestRetained  = "retained"
	frontmatterTestStatus    = "todo"
	frontmatterTestTitle     = "Changed sample"
)

func TestWritePreservesAdditionalSemanticValues(t *testing.T) {
	path := writeRawTask(t, `---
custom_scalar: "001" # presentation is not retained
custom_float: 2.0
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
custom_sequence:
  - alpha
  - alpha
  - 3
  - true
  - null
custom_mapping:
  defaults:
    enabled: true
  nested:
    enabled: true
    references: [one, two]
---

Body
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Title = frontmatterTestTitle
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	values := readFrontmatterValues(t, path)
	if got := values["custom_scalar"]; got != "001" {
		t.Errorf("custom_scalar = %#v, want %q", got, "001")
	}
	if got := values["custom_float"]; got != float64(2) {
		t.Errorf("custom_float = %#v (%T), want float64(2)", got, got)
	}
	wantSequence := []any{"alpha", "alpha", 3, true, nil}
	if got := values["custom_sequence"]; !reflect.DeepEqual(got, wantSequence) {
		t.Errorf("custom_sequence = %#v, want %#v", got, wantSequence)
	}
	wantMapping := map[string]any{
		"defaults": map[string]any{"enabled": true},
		"nested": map[string]any{
			"enabled":    true,
			"references": []any{"one", "two"},
		},
	}
	if got := values["custom_mapping"]; !reflect.DeepEqual(got, wantMapping) {
		t.Errorf("custom_mapping = %#v, want %#v", got, wantMapping)
	}
	if got := values["title"]; got != frontmatterTestTitle {
		t.Errorf("title = %#v, want changed canonical value", got)
	}
}

func TestWritePreservesAdditionalPropertiesWithTaggedAndAliasedYAML(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
estimate: 4h
custom_source: &shared 4h
custom_copy: *shared
custom_anchor: &extra anchored
custom_tagged: !integration 001
custom_binary: !!binary aGVsbG8=
custom_sequence:
  - plain
  - &item anchored
  - *item
custom_mapping:
  nested: !integration value
custom_supported: retained # comments are discarded without dropping the value
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Estimate = ""
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	values := readFrontmatterValues(t, path)
	if _, present := values["estimate"]; present {
		t.Error("cleared estimate remains in frontmatter")
	}
	if got := values["custom_supported"]; got != frontmatterTestRetained {
		t.Errorf("custom_supported = %#v, want retained", got)
	}
}

func TestWriteRefusesTopLevelMergeValues(t *testing.T) {
	path := writeRawTask(t, `---
defaults: &defaults
  external_reference: merged
  merged_reference: sample-17
<<: *defaults
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
external_reference: explicit
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Priority = "high"
	if err = Write(path, tk); err == nil || !strings.Contains(err.Error(), "top-level YAML merge") {
		t.Fatalf("Write() error = %v, want merge refusal", err)
	}
}

func TestWritePreservesQuotedMergeKeyAcrossRepeatedWrites(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
defaults: &defaults
  external_reference: ignored
"<<":
  external_reference: sample-17
---
`)

	for i := 0; i < 2; i++ {
		tk, err := Read(path)
		if err != nil {
			t.Fatalf("Read() cycle %d error: %v", i+1, err)
		}
		tk.Priority = "high"
		if err = Write(path, tk); err != nil {
			t.Fatalf("Write() cycle %d error: %v", i+1, err)
		}
	}

	values := readFrontmatterValues(t, path)
	want := map[string]any{"external_reference": "sample-17"}
	if got := values["<<"]; !reflect.DeepEqual(got, want) {
		t.Errorf("quoted merge-key property = %#v, want %#v", got, want)
	}
	if _, merged := values["external_reference"]; merged {
		t.Errorf("quoted merge-key property was hoisted into the task: %#v", values)
	}
}

func TestWritePreservesPropertiesWithDecoratedKeySyntax(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
key_source: &key custom_alias_key
!integration custom_tagged_key: tagged-key value
&anchored_key custom_anchored_key: anchored-key value
!!str custom_explicit_key: explicit-key value
custom_mapping:
  *key: nested alias-key value
  !integration nested_tagged_key: nested tagged-key value
custom_supported: retained
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Title = frontmatterTestTitle
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	values := readFrontmatterValues(t, path)
	if got := values["title"]; got != frontmatterTestTitle {
		t.Errorf("title = %#v, want changed canonical value", got)
	}
	if got := values["custom_supported"]; got != frontmatterTestRetained {
		t.Errorf("custom_supported = %#v, want retained", got)
	}
	root := readFrontmatterNode(t, path)
	for _, want := range []struct{ key, value string }{
		{"custom_tagged_key", "tagged-key value"},
		{"custom_anchored_key", "anchored-key value"},
		{"custom_explicit_key", "explicit-key value"},
	} {
		assertScalarNode(t, mappingValue(t, root, want.key), want.value, "!!str")
	}
	mapping := mappingValue(t, root, "custom_mapping")
	if mapping.Content[0].Alias != mappingValue(t, root, "key_source") {
		t.Error("nested alias key lost its binding")
	}
}

func TestWriteToleratesAdditionalAliasExpansion(t *testing.T) {
	var content strings.Builder
	content.WriteString("---\nid: 1\ntitle: Generic sample\nstatus: todo\npriority: medium\n")
	content.WriteString("created: 2026-08-12T10:00:00Z\nupdated: 2026-08-12T10:00:00Z\ncustom:\n")
	content.WriteString("  - &a0 [x, x]\n")
	for i := 1; i <= 24; i++ {
		fmt.Fprintf(&content, "  - &a%d [*a%d, *a%d]\n", i, i-1, i-1)
	}
	content.WriteString("---\n")
	path := writeRawTask(t, content.String())

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Title = frontmatterTestTitle
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	rewritten, err := Read(path)
	if err != nil {
		t.Fatalf("Read() rewritten task error: %v", err)
	}
	if rewritten.Title != frontmatterTestTitle {
		t.Errorf("Title = %q, want changed canonical value", rewritten.Title)
	}
	sequence := mappingValue(t, readFrontmatterNode(t, path), "custom")
	if len(sequence.Content) != 25 {
		t.Fatalf("alias sequence has %d entries, want 25", len(sequence.Content))
	}
	for i := 1; i < len(sequence.Content); i++ {
		for _, alias := range sequence.Content[i].Content {
			if alias.Kind != yaml.AliasNode || alias.Alias != sequence.Content[i-1] {
				t.Errorf("alias expansion level %d lost its original binding", i)
			}
		}
	}
}

func TestWriteKeepsCanonicalFieldsAuthoritative(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
assignee: sample-user
custom_value: retained
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Assignee = ""
	tk.Estimate = "2h"
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	values := readFrontmatterValues(t, path)
	if _, present := values["assignee"]; present {
		t.Error("cleared assignee remains in frontmatter")
	}
	if got := values["estimate"]; got != "2h" {
		t.Errorf("estimate = %#v, want 2h", got)
	}
	if got := values["custom_value"]; got != frontmatterTestRetained {
		t.Errorf("custom_value = %#v, want retained", got)
	}
}

func TestWriteToleratesAdditionalMappingWithNonStringKey(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
custom_mapping:
  nested:
    1: unsupported
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Title = frontmatterTestTitle
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	rewritten, err := Read(path)
	if err != nil {
		t.Fatalf("Read() rewritten task error: %v", err)
	}
	if rewritten.Title != frontmatterTestTitle {
		t.Errorf("Title = %q, want changed canonical value", rewritten.Title)
	}
	nested := mappingValue(t, mappingValue(t, readFrontmatterNode(t, path), "custom_mapping"), "nested")
	assertScalarNode(t, nested.Content[0], "1", "!!int")
	assertScalarNode(t, nested.Content[1], "unsupported", "!!str")
}

func TestWriteToleratesAdditionalPropertyWithNonStringKey(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
1: unsupported
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Title = frontmatterTestTitle
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	rewritten, err := Read(path)
	if err != nil {
		t.Fatalf("Read() rewritten task error: %v", err)
	}
	if rewritten.Title != frontmatterTestTitle {
		t.Errorf("Title = %q, want changed canonical value", rewritten.Title)
	}
	assertScalarNode(t, mappingValue(t, readFrontmatterNode(t, path), "1"), "unsupported", "!!str")
}

func TestInMemoryTaskKeepsCanonicalYAMLAndAdditionalPropertiesOutOfJSON(t *testing.T) {
	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	tk := &Task{
		ID:       1,
		Title:    "Generic sample",
		Status:   frontmatterTestStatus,
		Priority: "medium",
		Created:  now,
		Updated:  now,
	}

	wantYAML, err := yaml.Marshal((*taskYAML)(tk))
	if err != nil {
		t.Fatal(err)
	}
	gotYAML, err := yaml.Marshal(tk)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotYAML, wantYAML) {
		t.Errorf("in-memory YAML changed:\ngot:\n%s\nwant:\n%s", gotYAML, wantYAML)
	}

	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
custom_value: retained
---
`)
	loaded, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "custom_value") || strings.Contains(string(encoded), "extraProperties") {
		t.Errorf("JSON exposed additional frontmatter: %s", encoded)
	}
	valueYAML, err := yaml.Marshal(*loaded)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(valueYAML), "custom_value: retained") {
		t.Errorf("marshaling a Task value dropped additional frontmatter:\n%s", valueYAML)
	}
}

func TestReadRejectsDuplicateCanonicalKeys(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
status: done
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
---
`)

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read() accepted duplicate canonical keys")
	}
	if !strings.Contains(err.Error(), "mapping key \"status\" already defined") {
		t.Fatalf("Read() error = %v", err)
	}
}

func TestWriteRefusesAliasBackedDuplicateKey(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
key_source: &key custom_value
custom_value: first
*key: second
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Priority = "high"
	if err = Write(path, tk); err == nil || !strings.Contains(err.Error(), "top-level alias key") {
		t.Fatalf("Write() error = %v, want alias-key refusal", err)
	}
}

func TestWriteToleratesMappingWithNestedAliasBackedDuplicateKey(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
key_source: &key custom_value
custom_mapping:
  custom_value: first
  *key: second
---
`)

	tk, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	tk.Title = frontmatterTestTitle
	if err = Write(path, tk); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	rewritten, err := Read(path)
	if err != nil {
		t.Fatalf("Read() rewritten task error: %v", err)
	}
	if rewritten.Title != frontmatterTestTitle {
		t.Errorf("Title = %q, want changed canonical value", rewritten.Title)
	}
}

func TestReadRejectsDuplicateAdditionalKeys(t *testing.T) {
	path := writeRawTask(t, `---
id: 1
title: Generic sample
status: todo
priority: medium
created: 2026-08-12T10:00:00Z
updated: 2026-08-12T10:00:00Z
custom_value: first
custom_value: second
---
`)

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read() accepted duplicate additional keys")
	}
	if !strings.Contains(err.Error(), "mapping key \"custom_value\" already defined") {
		t.Fatalf("Read() error = %v", err)
	}
}

func writeRawTask(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "001-generic-sample.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFrontmatterValues(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test-owned temporary path
	if err != nil {
		t.Fatal(err)
	}
	fm, _, err := splitFrontmatter(data)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err = yaml.Unmarshal(fm, &values); err != nil {
		t.Fatal(err)
	}
	return values
}
