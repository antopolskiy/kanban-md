package board

import (
	"fmt"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func scalarTask(t *testing.T, id int, extra string) *task.Task {
	t.Helper()
	var result task.Task
	if err := yaml.Unmarshal([]byte(fmt.Sprintf("id: %d\ntitle: Child\nstatus: backlog\npriority: medium\nparent: 100\n%s", id, extra)), &result); err != nil {
		t.Fatal(err)
	}
	return &result
}

func TestPropertyFilterAndOrderedTypedGroups(t *testing.T) {
	cfg := config.NewDefault("test")
	for _, raw := range []string{"milestone", "epic", "story", "bug"} {
		value, _ := property.ParseLiteral(raw)
		cfg.GroupOrders = ensureGroupOrder(cfg.GroupOrders)
		cfg.GroupOrders["property:type"] = append(cfg.GroupOrders["property:type"], value)
	}
	tasks := []*task.Task{scalarTask(t, 1, "type: bug\n"), scalarTask(t, 2, "type: epic\n"), scalarTask(t, 3, "type: epicc\n"), scalarTask(t, 4, "type: null\n"), scalarTask(t, 5, ""), scalarTask(t, 6, "type: [story]\n"), scalarTask(t, 7, "type: 20\n"), scalarTask(t, 8, "type: \"20\"\n"), scalarTask(t, 9, "type: 20.0\n")}
	grouped := GroupBy(tasks, "property:type", cfg)
	var labels []string
	for _, group := range grouped.Groups {
		labels = append(labels, group.Key)
	}
	want := []string{`"epic"`, `"bug"`, `"20"`, `"epicc"`, `20`, `null`, `(unclassified)`}
	if !slices.Equal(labels, want) {
		t.Fatalf("groups=%v want %v", labels, want)
	}
	if grouped.Groups[4].Total != 2 {
		t.Fatal("equal numbers split groups")
	}
	filter, _ := property.ParseAssignment("type=null")
	got := Filter(tasks, FilterOptions{Properties: []property.Assignment{filter}})
	if len(got) != 1 || got[0].ID != 4 {
		t.Fatal("null matched absence")
	}
	filter, _ = property.ParseAssignment(`type="20"`)
	got = Filter(tasks, FilterOptions{Properties: []property.Assignment{filter}, Statuses: []string{"backlog"}})
	if len(got) != 1 || got[0].ID != 8 {
		t.Fatal("numeric string coerced")
	}
}

func ensureGroupOrder(orders config.PropertyGroupOrders) config.PropertyGroupOrders {
	if orders == nil {
		return config.PropertyGroupOrders{}
	}
	return orders
}

func TestPropertyChildOrderAndCallerIsolation(t *testing.T) {
	cfg := config.NewDefault("test")
	cfg.Children.DetailSort = "property:rank"
	tasks := []*task.Task{scalarTask(t, 8, "rank: 9007199254740993\n"), scalarTask(t, 7, "rank: 9007199254740992\n"), scalarTask(t, 6, "rank: 20.0\n"), scalarTask(t, 5, "rank: 20\n"), scalarTask(t, 4, "rank: -1\n"), scalarTask(t, 3, "rank: \"0\"\n"), scalarTask(t, 2, "rank: .nan\n"), scalarTask(t, 1, ""), scalarTask(t, 9, "rank: 0\n")}
	original := slices.Clone(tasks)
	summary := SummarizeChildren(tasks, 100, cfg, false)
	var ids []int
	for _, child := range summary.Children {
		ids = append(ids, child.ID)
	}
	if !slices.Equal(ids, []int{4, 9, 5, 6, 7, 8, 1, 2, 3}) {
		t.Fatalf("child IDs=%v", ids)
	}
	if !slices.Equal(tasks, original) {
		t.Fatal("caller tasks reordered")
	}
	cfg.Children.DetailSort = ""
	summary = SummarizeChildren(tasks, 100, cfg, false)
	for i, child := range summary.Children {
		if child.ID != i+1 {
			t.Fatal("default order changed")
		}
	}
}
