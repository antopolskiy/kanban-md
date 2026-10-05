package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/property"
)

func TestPropertyWorkflow(t *testing.T) {
	dir := initBoard(t)
	created := runKanban(t, dir, "create", "Parent", "--set-property", "type=milestone", "--json")
	if created.exitCode != 0 {
		t.Fatalf("create: %s", created.stderr)
	}
	if strings.Contains(created.stdout, `"properties"`) {
		t.Fatal("authoring leaked properties into default JSON")
	}
	child := runKanban(t, dir, "create", "Child", "--parent", "1", "--set-property", "type=epicc", "--set-property", "reading_order=9007199254740993")
	if child.exitCode != 0 {
		t.Fatalf("create child: %s", child.stderr)
	}
	edit := runKanban(t, dir, "edit", "2", "--set-property", "type=epic")
	if edit.exitCode != 0 {
		t.Fatalf("edit: %s", edit.stderr)
	}
	list := runKanban(t, dir, "list", "--property", "type=epic", "--show-property", "type", "--json")
	var records []struct {
		ID         int                        `json:"id"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if list.exitCode != 0 || json.Unmarshal([]byte(list.stdout), &records) != nil || len(records) != 1 || records[0].ID != 2 || string(records[0].Properties["type"]) != `"epic"` {
		t.Fatalf("list: %s %s", list.stdout, list.stderr)
	}
	shown := runKanban(t, dir, "show", "1", "--json", "--show-property", "reading_order")
	if shown.exitCode != 0 || !strings.Contains(shown.stdout, "9007199254740993") {
		t.Fatalf("show: %s %s", shown.stdout, shown.stderr)
	}
	bad := runKanban(t, dir, "edit", "2", "--set-property", "status=done")
	if bad.exitCode == 0 {
		t.Fatal("owned key accepted")
	}
	conflict := runKanban(t, dir, "edit", "2", "--set-property", "type=story", "--clear-property", "type")
	if conflict.exitCode == 0 {
		t.Fatal("set/clear conflict accepted")
	}
	configurePropertyViews(t, dir)
	checkPropertyViews(t, dir)
	checkPropertyLifecycle(t, dir)
	checkPropertyAliasRefusal(t, dir)
	checkPropertyRecovery(t, dir)
}

func configurePropertyViews(t *testing.T, dir string) {
	t.Helper()
	third := runKanban(t, dir, "create", "Other child", "--parent", "1", "--set-property", "type=story", "--set-property", "reading_order=20")
	if third.exitCode != 0 {
		t.Fatal(third.stderr)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Children.DetailSort = "property:reading_order"
	cfg.Display.CompactFields = config.FieldList{"status", "property:type"}
	cfg.GroupOrders = config.PropertyGroupOrders{}
	for _, literal := range []string{"milestone", "epic", "story", "bug"} {
		value, parseErr := property.ParseLiteral(literal)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		cfg.GroupOrders["property:type"] = append(cfg.GroupOrders["property:type"], value)
	}
	if saveErr := cfg.Save(); saveErr != nil {
		t.Fatal(saveErr)
	}
}

func checkPropertyViews(t *testing.T, dir string) {
	t.Helper()
	configured := runKanban(t, dir, "show", "1", "--json")
	var detail struct {
		Children []struct {
			ID int `json:"id"`
		} `json:"children"`
	}
	if configured.exitCode != 0 || json.Unmarshal([]byte(configured.stdout), &detail) != nil || len(detail.Children) != 2 || detail.Children[0].ID != 3 || strings.Contains(configured.stdout, `"properties"`) {
		t.Fatalf("configured default JSON: %s %s", configured.stdout, configured.stderr)
	}
	compact := runKanban(t, dir, "list", "--compact")
	if compact.exitCode != 0 || strings.Count(compact.stdout, "\n") != 3 || !strings.Contains(compact.stdout, `type="epic"`) {
		t.Fatalf("compact: %s %s", compact.stdout, compact.stderr)
	}
	groups := runKanban(t, dir, "board", "--group-by", "property:type", "--json")
	if groups.exitCode != 0 || !strings.Contains(groups.stdout, `\"milestone\"`) || strings.Contains(groups.stdout, `"properties"`) {
		t.Fatalf("groups: %s %s", groups.stdout, groups.stderr)
	}
}

func checkPropertyLifecycle(t *testing.T, dir string) {
	t.Helper()
	detach := runKanban(t, dir, "edit", "2", "--clear-parent")
	if detach.exitCode != 0 {
		t.Fatal(detach.stderr)
	}
	retained := runKanban(t, dir, "show", "2", "--json", "--show-property", "reading_order")
	if !strings.Contains(retained.stdout, "9007199254740993") {
		t.Fatal("detach cleared property")
	}
	reattach := runKanban(t, dir, "edit", "2", "--parent", "1")
	if reattach.exitCode != 0 {
		t.Fatal(reattach.stderr)
	}
	claimed := runKanban(t, dir, "edit", "2", "--claim", "owner")
	if claimed.exitCode != 0 {
		t.Fatal(claimed.stderr)
	}
	refused := runKanban(t, dir, "edit", "2", "--set-property", "reading_order=10")
	if refused.exitCode == 0 {
		t.Fatal("property edit bypassed claim")
	}
	release := runKanban(t, dir, "edit", "2", "--release")
	if release.exitCode != 0 {
		t.Fatal(release.stderr)
	}
}

func checkPropertyAliasRefusal(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "tasks", "002-*.md"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("task paths=%v %v", paths, err)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	anchored := strings.Replace(string(data), "reading_order: 9007199254740993", "reading_order: &rank 9007199254740993\nrelated: *rank\nprivate: [hidden]", 1)
	if anchored == string(data) {
		t.Fatalf("rank spelling missing: %s", data)
	}
	if writeErr := os.WriteFile(paths[0], []byte(anchored), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	unsafe := runKanban(t, dir, "edit", "2", "--title", "Changed title", "--set-property", "reading_order=30")
	if unsafe.exitCode == 0 {
		t.Fatal("orphaned alias write allowed")
	}
	after, err := os.ReadFile(paths[0])
	if err != nil || string(after) != anchored {
		t.Fatal("refused edit changed task bytes")
	}
	unsupported := runKanban(t, dir, "show", "2", "--json", "--show-property", "related", "--show-property", "private")
	if unsupported.exitCode != 0 || !strings.Contains(unsupported.stderr, "not a supported scalar") || strings.Contains(unsupported.stdout, "hidden") || !strings.Contains(unsupported.stdout, `"properties": {}`) {
		t.Fatalf("unsupported projection: %s %s", unsupported.stdout, unsupported.stderr)
	}
}

func checkPropertyRecovery(t *testing.T, dir string) {
	t.Helper()
	recovered := runKanban(t, dir, "edit", "2", "--clear-property", "related", "--set-property", "reading_order=30")
	if recovered.exitCode != 0 {
		t.Fatal(recovered.stderr)
	}
	clearResult := runKanban(t, dir, "edit", "2", "--clear-parent", "--clear-property", "reading_order")
	if clearResult.exitCode != 0 {
		t.Fatal(clearResult.stderr)
	}
	cleared := runKanban(t, dir, "show", "2", "--json", "--show-property", "reading_order")
	if !strings.Contains(cleared.stdout, `"properties": {}`) {
		t.Fatal("explicit clear left rank")
	}
}
