package board

import (
	"math"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestMetricsEmpty(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	m := ComputeMetrics(cfg, nil, now)

	if m.Throughput7d != 0 {
		t.Errorf("Throughput7d = %d, want 0", m.Throughput7d)
	}
	if m.Throughput30d != 0 {
		t.Errorf("Throughput30d = %d, want 0", m.Throughput30d)
	}
	if m.AvgLeadTimeHours != nil {
		t.Errorf("AvgLeadTimeHours = %v, want nil", m.AvgLeadTimeHours)
	}
	if m.AvgCycleTimeHours != nil {
		t.Errorf("AvgCycleTimeHours = %v, want nil", m.AvgCycleTimeHours)
	}
	if m.FlowEfficiency != nil {
		t.Errorf("FlowEfficiency = %v, want nil", m.FlowEfficiency)
	}
	if len(m.AgingItems) != 0 {
		t.Errorf("AgingItems = %d, want 0", len(m.AgingItems))
	}
}

func TestMetricsThroughput(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	comp3dAgo := now.AddDate(0, 0, -3)
	comp10dAgo := now.AddDate(0, 0, -10)
	comp60dAgo := now.AddDate(0, 0, -60)

	tasks := []*task.Task{
		{ID: 1, Status: "done", Completed: &comp3dAgo, Created: comp60dAgo},
		{ID: 2, Status: "done", Completed: &comp10dAgo, Created: comp60dAgo},
		{ID: 3, Status: "done", Completed: &comp60dAgo, Created: comp60dAgo},
		{ID: 4, Status: "todo"},
	}

	m := ComputeMetrics(cfg, tasks, now)

	if m.Throughput7d != 1 {
		t.Errorf("Throughput7d = %d, want 1", m.Throughput7d)
	}
	if m.Throughput30d != 2 {
		t.Errorf("Throughput30d = %d, want 2", m.Throughput30d)
	}
}

func TestMetricsLeadAndCycleTime(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// Task completed in 48 hours, cycle time 24 hours.
	created := now.Add(-48 * time.Hour)
	started := now.Add(-24 * time.Hour)
	completed := now

	tasks := []*task.Task{
		{ID: 1, Status: "done", Created: created, Started: &started, Completed: &completed},
	}

	m := ComputeMetrics(cfg, tasks, now)

	if m.AvgLeadTimeHours == nil {
		t.Fatal("AvgLeadTimeHours is nil")
	}
	if math.Abs(*m.AvgLeadTimeHours-48) > 0.01 {
		t.Errorf("AvgLeadTimeHours = %.2f, want 48", *m.AvgLeadTimeHours)
	}

	if m.AvgCycleTimeHours == nil {
		t.Fatal("AvgCycleTimeHours is nil")
	}
	if math.Abs(*m.AvgCycleTimeHours-24) > 0.01 {
		t.Errorf("AvgCycleTimeHours = %.2f, want 24", *m.AvgCycleTimeHours)
	}
}

func TestMetricsFlowEfficiency(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	created := now.Add(-100 * time.Hour)
	started := now.Add(-50 * time.Hour)
	completed := now

	tasks := []*task.Task{
		{ID: 1, Status: "done", Created: created, Started: &started, Completed: &completed},
	}

	m := ComputeMetrics(cfg, tasks, now)

	if m.FlowEfficiency == nil {
		t.Fatal("FlowEfficiency is nil")
	}
	// cycle=50h, lead=100h → efficiency=0.5
	if math.Abs(*m.FlowEfficiency-0.5) > 0.01 {
		t.Errorf("FlowEfficiency = %.3f, want 0.5", *m.FlowEfficiency)
	}
}

func TestMetricsAgingItems(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	started48hAgo := now.Add(-48 * time.Hour)
	started24hAgo := now.Add(-24 * time.Hour)
	completed := now

	tasks := []*task.Task{
		{ID: 1, Status: "in-progress", Started: &started48hAgo, Title: "Aging task"},
		{ID: 2, Status: "review", Started: &started24hAgo, Title: "Review task"},
		{ID: 3, Status: "done", Started: &started48hAgo, Completed: &completed, Title: "Done task"},
		{ID: 4, Status: "backlog", Title: "Not started"},
	}

	m := ComputeMetrics(cfg, tasks, now)

	if len(m.AgingItems) != 2 {
		t.Fatalf("AgingItems = %d, want 2", len(m.AgingItems))
	}
	if m.AgingItems[0].ID != 1 {
		t.Errorf("AgingItems[0].ID = %d, want 1", m.AgingItems[0].ID)
	}
	if math.Abs(m.AgingItems[0].AgeHours-48) > 0.01 {
		t.Errorf("AgingItems[0].AgeHours = %.2f, want 48", m.AgingItems[0].AgeHours)
	}
}

func TestMetricsNoTimestampsGraceful(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tasks := []*task.Task{
		{ID: 1, Status: "backlog", Priority: "medium"},
		{ID: 2, Status: "todo", Priority: "high"},
	}

	m := ComputeMetrics(cfg, tasks, now)

	if m.AvgLeadTimeHours != nil {
		t.Errorf("AvgLeadTimeHours = %v, want nil (no completed tasks)", m.AvgLeadTimeHours)
	}
	if m.AvgCycleTimeHours != nil {
		t.Errorf("AvgCycleTimeHours = %v, want nil", m.AvgCycleTimeHours)
	}
	if m.FlowEfficiency != nil {
		t.Errorf("FlowEfficiency = %v, want nil", m.FlowEfficiency)
	}
}

func TestMetricsMultipleTasksAverages(t *testing.T) {
	cfg := config.NewDefault("Test")
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// Task 1: lead=100h, cycle=50h
	created1 := now.Add(-100 * time.Hour)
	started1 := now.Add(-50 * time.Hour)
	// Task 2: lead=200h, cycle=100h
	created2 := now.Add(-200 * time.Hour)
	started2 := now.Add(-100 * time.Hour)

	tasks := []*task.Task{
		{ID: 1, Status: "done", Created: created1, Started: &started1, Completed: &now},
		{ID: 2, Status: "done", Created: created2, Started: &started2, Completed: &now},
	}

	m := ComputeMetrics(cfg, tasks, now)

	// Avg lead = (100+200)/2 = 150
	if m.AvgLeadTimeHours == nil || math.Abs(*m.AvgLeadTimeHours-150) > 0.01 {
		t.Errorf("AvgLeadTimeHours = %v, want 150", m.AvgLeadTimeHours)
	}
	// Avg cycle = (50+100)/2 = 75
	if m.AvgCycleTimeHours == nil || math.Abs(*m.AvgCycleTimeHours-75) > 0.01 {
		t.Errorf("AvgCycleTimeHours = %v, want 75", m.AvgCycleTimeHours)
	}
	// Flow efficiency = 75/150 = 0.5
	if m.FlowEfficiency == nil || math.Abs(*m.FlowEfficiency-0.5) > 0.01 {
		t.Errorf("FlowEfficiency = %v, want 0.5", m.FlowEfficiency)
	}
}
