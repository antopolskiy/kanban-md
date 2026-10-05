package task

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const preservationCore = "id: 1\ntitle: Generic sample\nstatus: todo\npriority: medium\ncreated: 2026-08-12T10:00:00Z\nupdated: 2026-08-12T10:00:00Z\n"

const preservationFloatTag = "!!float"

func TestPreservationExactNumericNodes(t *testing.T) {
	path := writeRawTask(t, "---\n"+preservationCore+`huge: 18446744073709551617
negative: -9223372036854775809
decimal: 0.123456789012345678901234567890
exponent: 1.234567890123456789e+100
whole_float: 2.0
numeric_string: "18446744073709551617"
nested:
  sequence: [18446744073709551617, 0.123456789012345678901234567890]
---
`)
	for cycle := 0; cycle < 2; cycle++ {
		tk, err := Read(path)
		if err != nil {
			t.Fatal(err)
		}
		tk.Title = frontmatterTestTitle
		if err = Write(path, tk); err != nil {
			t.Fatal(err)
		}
		root := readFrontmatterNode(t, path)
		for _, want := range []struct{ key, value, tag string }{
			{"huge", "18446744073709551617", preservationFloatTag},
			{"negative", "-9223372036854775809", preservationFloatTag},
			{"decimal", "0.123456789012345678901234567890", preservationFloatTag},
			{"exponent", "1.234567890123456789e+100", preservationFloatTag},
			{"whole_float", "2.0", preservationFloatTag},
			{"numeric_string", "18446744073709551617", "!!str"},
		} {
			assertScalarNode(t, mappingValue(t, root, want.key), want.value, want.tag)
		}
		sequence := mappingValue(t, mappingValue(t, root, "nested"), "sequence")
		assertScalarNode(t, sequence.Content[0], "18446744073709551617", preservationFloatTag)
		assertScalarNode(t, sequence.Content[1], "0.123456789012345678901234567890", preservationFloatTag)
	}
}

func TestPreservationOpaqueMixedNodes(t *testing.T) {
	path := writeRawTask(t, "---\n"+preservationCore+`mixed:
  plain: retained
  tagged: !integration 001
  binary: !!binary aGVsbG8=
  sequence: [plain, !integration tagged]
  defaults: &defaults {enabled: true}
  copy: *defaults
  merged: {<<: *defaults, label: retained}
  17: !integration non-string-key
---
`)
	for cycle := 0; cycle < 2; cycle++ {
		tk, err := Read(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = Write(path, tk); err != nil {
			t.Fatal(err)
		}
		mixed := mappingValue(t, readFrontmatterNode(t, path), "mixed")
		assertScalarNode(t, mappingValue(t, mixed, "plain"), "retained", "!!str")
		assertScalarNode(t, mappingValue(t, mixed, "tagged"), "001", "!integration")
		assertScalarNode(t, mappingValue(t, mixed, "binary"), "aGVsbG8=", "!!binary")
		assertScalarNode(t, mappingValue(t, mixed, "sequence").Content[1], "tagged", "!integration")
		assertScalarNode(t, mappingValue(t, mixed, "17"), "non-string-key", "!integration")
		defaults := mappingValue(t, mixed, "defaults")
		copyNode := mappingValue(t, mixed, "copy")
		if copyNode.Kind != yaml.AliasNode || copyNode.Alias != defaults {
			t.Error("extra alias lost its binding")
		}
		merge := mappingValue(t, mappingValue(t, mixed, "merged"), "<<")
		if merge.Kind != yaml.AliasNode || merge.Alias != defaults {
			t.Error("nested merge lost its binding")
		}
	}
}

func TestPreservationRefusesUnsafeWritesBeforeTouchingFiles(t *testing.T) {
	for name, content := range map[string]string{
		"canonical alias":                preservationCore + "estimate: &shared 4h\ncustom_copy: *shared\n",
		"top-level merge":                preservationCore + "defaults: &defaults {assignee: sample-user}\n<<: *defaults\n",
		"alias key":                      preservationCore + "key_source: &key custom_key\n*key: value\n",
		"canonical key alias":            "&key id: 1\n" + strings.TrimPrefix(preservationCore, "id: 1\n") + "custom_key: *key\n",
		"root alias":                     "&root\n" + preservationCore + "custom_root: *root\n",
		"custom root tag":                "!integration\n" + preservationCore,
		"canonical anchor shadows extra": "custom_source: &shared original\n" + preservationCore + "estimate: &shared 4h\ncustom_copy: *shared\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := writeRawTask(t, "---\n"+content+"---\nBody\n")
			original, err := os.ReadFile(path) //nolint:gosec // test-owned temporary path
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(path, 0o400); err != nil {
				t.Fatal(err)
			}
			originalInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			tk, err := Read(path)
			if err != nil {
				t.Fatalf("ordinary Read must succeed: %v", err)
			}
			tk.Title = frontmatterTestTitle
			for attempt := 0; attempt < 2; attempt++ {
				if err = Write(path, tk); err == nil || !strings.Contains(err.Error(), "edit the frontmatter manually") {
					t.Errorf("Write error = %v, want actionable preservation refusal", err)
				}
				assertFileUnchanged(t, path, original)
				info, statErr := os.Stat(path)
				if statErr != nil {
					t.Fatal(statErr)
				}
				if info.Mode().Perm() != originalInfo.Mode().Perm() {
					t.Errorf("Write refusal changed mode to %o", info.Mode().Perm())
				}
			}
			newPath, err := WriteAndRename(path, tk, "Generic sample")
			if err == nil {
				t.Errorf("WriteAndRename succeeded: %s", newPath)
			}
			assertFileUnchanged(t, path, original)
			destination := filepath.Join(filepath.Dir(path), "001-changed-sample.md")
			if _, err = os.Stat(destination); !os.IsNotExist(err) {
				t.Errorf("refused rename created destination: %v", err)
			}
			if _, err = Read(path); err != nil {
				t.Errorf("Read after refusal: %v", err)
			}
		})
	}
}

func TestPreservationAliasBindingsAndCanonicalClearing(t *testing.T) {
	path := writeRawTask(t, "---\n"+`custom_first: &shared original
estimate: *shared
`+preservationCore+`custom_copy_first: *shared
custom_second: &shared replacement
custom_copy_second: *shared
self: &self {again: *self}
!integration assignee: old-owner
!integration 17: tagged-key
&anchored_key custom_anchored_key: retained
---
`)
	tk, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Estimate != "original" {
		t.Fatalf("Estimate = %q", tk.Estimate)
	}
	for _, estimate := range []string{"changed", ""} {
		tk.Estimate = estimate
		tk.Assignee = ""
		if err = Write(path, tk); err != nil {
			t.Fatal(err)
		}
		root := readFrontmatterNode(t, path)
		assertPreservedAliasBindings(t, root)
		for i := 0; i < len(root.Content); i += 2 {
			key := root.Content[i]
			if key.Value == "assignee" || key.Value == "estimate" && estimate == "" {
				t.Errorf("cleared canonical key %q remains", key.Value)
			}
			if key.Value == "17" && key.Tag != "!integration" {
				t.Errorf("tagged key lost tag: %s", key.Tag)
			}
		}
		if estimate != "" {
			assertScalarNode(t, mappingValue(t, root, "estimate"), estimate, "!!str")
		}
		tk, err = Read(path)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assertPreservedAliasBindings(t *testing.T, root *yaml.Node) {
	t.Helper()
	first := mappingValue(t, root, "custom_first")
	second := mappingValue(t, root, "custom_second")
	if mappingValue(t, root, "custom_copy_first").Alias != first || mappingValue(t, root, "custom_copy_second").Alias != second {
		t.Error("duplicate anchors changed alias bindings")
	}
	self := mappingValue(t, root, "self")
	if mappingValue(t, self, "again").Alias != self {
		t.Error("self alias lost its binding")
	}
}

func TestPreservationTaskCopiesAndUnmarshalReset(t *testing.T) {
	path := writeRawTask(t, "---\n"+preservationCore+"custom_value: &source !integration retained\ncustom_copy: *source\n---\n")
	tk, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	copyTask := *tk
	for attempt := 0; attempt < 3; attempt++ {
		encoded, marshalErr := yaml.Marshal(copyTask)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if strings.Count(string(encoded), "custom_value:") != 1 || strings.Count(string(encoded), "title:") != 1 {
			t.Fatalf("marshal accumulated entries:\n%s", encoded)
		}
	}
	unsafe := preservationCore + "estimate: &source 4h\ncustom_copy: *source\n"
	if err = yaml.Unmarshal([]byte(unsafe), tk); err != nil {
		t.Fatal(err)
	}
	if _, err = yaml.Marshal(tk); err == nil {
		t.Fatal("unsafe unmarshal did not record refusal")
	}
	if err = yaml.Unmarshal([]byte(preservationCore+"new_extra: retained\n"), tk); err != nil {
		t.Fatal(err)
	}
	encoded, err := yaml.Marshal(tk)
	if err != nil {
		t.Fatalf("unmarshal did not reset refusal: %v", err)
	}
	if strings.Contains(string(encoded), "custom_copy") || !strings.Contains(string(encoded), "new_extra") {
		t.Errorf("unmarshal did not replace extras:\n%s", encoded)
	}
}

func TestPreservationUnsafeTasksRemainInOrdinaryReads(t *testing.T) {
	path := writeRawTask(t, "---\n"+preservationCore+"estimate: &source 4h\ncustom_copy: *source\n---\n")
	tasks, err := ReadAll(filepath.Dir(path))
	if err != nil || len(tasks) != 1 {
		t.Fatalf("ReadAll() = %d tasks, %v", len(tasks), err)
	}
	tasks, warnings, err := ReadAllLenient(filepath.Dir(path))
	if err != nil || len(tasks) != 1 || len(warnings) != 0 {
		t.Fatalf("ReadAllLenient() = %d tasks, %d warnings, %v", len(tasks), len(warnings), err)
	}
}

func readFrontmatterNode(t *testing.T, path string) *yaml.Node {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test-owned path
	if err != nil {
		t.Fatal(err)
	}
	fm, _, err := splitFrontmatter(data)
	if err != nil {
		t.Fatal(err)
	}
	var node yaml.Node
	if err = yaml.Unmarshal(fm, &node); err != nil {
		t.Fatal(err)
	}
	return node.Content[0]
}

func mappingValue(t *testing.T, mapping *yaml.Node, key string) *yaml.Node {
	t.Helper()
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	t.Fatalf("missing YAML key %q", key)
	return nil
}

func assertScalarNode(t *testing.T, node *yaml.Node, value, tag string) {
	t.Helper()
	if node.Kind != yaml.ScalarNode || node.Value != value || node.ShortTag() != tag {
		t.Errorf("scalar = %q %s (kind %d), want %q %s", node.Value, node.ShortTag(), node.Kind, value, tag)
	}
}

func assertFileUnchanged(t *testing.T, path string, original []byte) {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test-owned path
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, original) {
		t.Errorf("refused write changed bytes:\n%s", data)
	}
}
