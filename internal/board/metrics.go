package board

import (
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
)

const (
	hoursPerDay = 24
	days7       = 7
	days30      = 30
)

// Metrics holds aggregate board metrics.
type Metrics struct {
	Throughput7d      int         `json:"throughput_7d"`
	Throughput30d     int         `json:"throughput_30d"`
	AvgLeadTimeHours  *float64    `json:"avg_lead_time_hours,omitempty"`
	AvgCycleTimeHours *float64    `json:"avg_cycle_time_hours,omitempty"`
	FlowEfficiency    *float64    `json:"flow_efficiency,omitempty"`
	AgingItems        []AgingItem `json:"aging_items,omitempty"`
}

// AgingItem represents a work item that has started but not completed.
type AgingItem struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	AgeHours float64 `json:"age_hours"`
}

// ComputeMetrics computes aggregate flow metrics from all tasks.
func ComputeMetrics(cfg *config.Config, tasks []*task.Task, now time.Time) Metrics {
	m := Metrics{}

	window7 := now.AddDate(0, 0, -days7)
	window30 := now.AddDate(0, 0, -days30)

	var leadSum, cycleSum float64
	var leadCount, cycleCount int

	for _, t := range tasks {
		if t.Completed != nil {
			if t.Completed.After(window7) {
				m.Throughput7d++
			}
			if t.Completed.After(window30) {
				m.Throughput30d++
			}

			leadHours := t.Completed.Sub(t.Created).Hours()
			leadSum += leadHours
			leadCount++

			if t.Started != nil {
				cycleHours := t.Completed.Sub(*t.Started).Hours()
				cycleSum += cycleHours
				cycleCount++
			}
		}

		// Aging: started but not completed, not in terminal status.
		if t.Started != nil && t.Completed == nil && !cfg.IsTerminalStatus(t.Status) {
			m.AgingItems = append(m.AgingItems, AgingItem{
				ID:       t.ID,
				Title:    t.Title,
				Status:   t.Status,
				AgeHours: now.Sub(*t.Started).Hours(),
			})
		}
	}

	if leadCount > 0 {
		avg := leadSum / float64(leadCount)
		m.AvgLeadTimeHours = &avg
	}
	if cycleCount > 0 {
		avg := cycleSum / float64(cycleCount)
		m.AvgCycleTimeHours = &avg
	}
	if m.AvgLeadTimeHours != nil && m.AvgCycleTimeHours != nil && *m.AvgLeadTimeHours > 0 {
		eff := *m.AvgCycleTimeHours / *m.AvgLeadTimeHours
		m.FlowEfficiency = &eff
	}

	return m
}
