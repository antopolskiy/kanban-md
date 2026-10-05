package task

import (
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/property"
)

func propertyTask(t *testing.T, extra string) Task {
	t.Helper()
	var result Task
	if err := yaml.Unmarshal([]byte("id: 1\ntitle: Example\nstatus: backlog\npriority: medium\n"+extra), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func propertyValue(t *testing.T, raw string) property.Scalar {
	t.Helper()
	value, err := property.ParseLiteral(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPropertyCopyOnWriteAndKinds(t *testing.T) {
	original := propertyTask(t, "type: story\nrank: 20\nnullish: null\nnested: [1]\n")
	copyTask := original
	if _, err := copyTask.SetPropertyScalar("type", propertyValue(t, "epic")); err != nil {
		t.Fatal(err)
	}
	if _, err := copyTask.ClearProperty("rank"); err != nil {
		t.Fatal(err)
	}
	value, _ := original.PropertyScalar("type")
	if value.JSONLiteral() != `"story"` {
		t.Fatal("copy mutated original")
	}
	_, state := original.PropertyScalar("rank")
	if state != PropertySupported {
		t.Fatal("clear mutated original")
	}
	for key, want := range map[string]PropertyState{"missing": PropertyMissing, "nested": PropertyUnsupported, "nullish": PropertySupported} {
		_, got := original.PropertyScalar(key)
		if got != want {
			t.Errorf("%s state=%d", key, got)
		}
	}
	for key := range canonicalTaskYAMLKeys {
		if property.ValidateKey(key) == nil {
			t.Errorf("canonical key not reserved: %s", key)
		}
	}
}

func TestPropertyFinalGraphRefusalAndRecovery(t *testing.T) {
	task := propertyTask(t, "rank: &rank 20\nreference: *rank\n")
	unchanged := task
	if changed, err := task.SetPropertyScalar("rank", propertyValue(t, "20.0")); err != nil || changed {
		t.Fatal("equal set replaced anchor")
	}
	if _, err := task.SetPropertyScalar("rank", propertyValue(t, "30")); err != nil {
		t.Fatal(err)
	}
	if _, err := yaml.Marshal(task); err == nil {
		t.Fatal("orphaned alias write allowed")
	}
	if _, err := yaml.Marshal(unchanged); err != nil {
		t.Fatal("copy lost original bindings")
	}
	if _, err := task.ClearProperty("reference"); err != nil {
		t.Fatal(err)
	}
	if _, err := yaml.Marshal(task); err != nil {
		t.Fatalf("safe recovery refused: %v", err)
	}
}

func TestPropertyCanonicalAliasRecoveryAndOwnershipRefusal(t *testing.T) {
	var task Task
	if err := yaml.Unmarshal([]byte("id: 1\ntitle: &title Example\nstatus: backlog\npriority: medium\nforeign: *title\n"), &task); err != nil {
		t.Fatal(err)
	}
	if _, err := yaml.Marshal(task); err == nil {
		t.Fatal("canonical alias write allowed")
	}
	if _, err := task.ClearProperty("foreign"); err != nil {
		t.Fatal(err)
	}
	if _, err := yaml.Marshal(task); err != nil {
		t.Fatal(err)
	}
	root := propertyTask(t, "other: 1\n")
	if err := yaml.Unmarshal([]byte("!custom\nid: 1\ntitle: Example\nstatus: backlog\npriority: medium\nother: 1\n"), &root); err != nil {
		t.Fatal(err)
	}
	if _, err := root.ClearProperty("other"); err != nil {
		t.Fatal(err)
	}
	if _, err := yaml.Marshal(root); err == nil || !strings.Contains(err.Error(), "custom root") {
		t.Fatal("root ownership refusal lost")
	}
}

func TestCompatSelectedScalars(t *testing.T) {
	task, err := Read(filepath.Join(v1FixtureDir, "010-selected-scalars.md"))
	if err != nil {
		t.Fatal(err)
	}
	for key, literal := range map[string]string{"type": `"story"`, "reading_order": "18446744073709551617", "numeric_string": `"20"`, "flag": "false", "empty_string": `""`, "nullish": "null"} {
		value, state := task.PropertyScalar(key)
		if state != PropertySupported || value.JSONLiteral() != literal {
			t.Fatalf("%s=%s state=%d", key, value.JSONLiteral(), state)
		}
	}
	path := filepath.Join(t.TempDir(), "010-selected-scalars.md")
	if writeErr := Write(path, task); writeErr != nil {
		t.Fatal(writeErr)
	}
	again, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := task.PropertyScalar("reading_order")
	after, _ := again.PropertyScalar("reading_order")
	if !before.Equal(after) {
		t.Fatal("selected scalar changed on round trip")
	}
}
