package task

import (
	"errors"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/clierr"
)

func TestValidateStatus_Valid(t *testing.T) {
	err := ValidateStatus("todo", []string{"backlog", "todo", "done"})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateStatus_Invalid(t *testing.T) {
	err := ValidateStatus("invalid", []string{"backlog", "todo", "done"})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidStatus {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidStatus)
	}
	if cliErr.Details["status"] != "invalid" {
		t.Errorf("details[status] = %v, want %q", cliErr.Details["status"], "invalid")
	}
}

func TestValidatePriority_Valid(t *testing.T) {
	err := ValidatePriority("high", []string{"low", "medium", "high"})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidatePriority_Invalid(t *testing.T) {
	err := ValidatePriority("urgent", []string{"low", "medium", "high"})
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidPriority {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidPriority)
	}
	if cliErr.Details["priority"] != "urgent" {
		t.Errorf("details[priority] = %v, want %q", cliErr.Details["priority"], "urgent")
	}
}

func TestValidateDate(t *testing.T) {
	err := ValidateDate("started", "not-a-date", errors.New("parse error"))
	if err.Code != clierr.InvalidDate {
		t.Errorf("code = %q, want %q", err.Code, clierr.InvalidDate)
	}
	if err.Details["field"] != "started" {
		t.Errorf("details[field] = %v, want %q", err.Details["field"], "started")
	}
	if err.Details["input"] != "not-a-date" {
		t.Errorf("details[input] = %v, want %q", err.Details["input"], "not-a-date")
	}
}

func TestValidateTaskID(t *testing.T) {
	err := ValidateTaskID("abc")
	if err.Code != clierr.InvalidTaskID {
		t.Errorf("code = %q, want %q", err.Code, clierr.InvalidTaskID)
	}
	if err.Details["input"] != "abc" {
		t.Errorf("details[input] = %v, want %q", err.Details["input"], "abc")
	}
}

func TestValidateSelfReference(t *testing.T) {
	err := ValidateSelfReference(42)
	if err.Code != clierr.SelfReference {
		t.Errorf("code = %q, want %q", err.Code, clierr.SelfReference)
	}
	if err.Details["id"] != 42 {
		t.Errorf("details[id] = %v, want 42", err.Details["id"])
	}
}

func TestValidateDependencyNotFound(t *testing.T) {
	err := ValidateDependencyNotFound(99)
	if err.Code != clierr.DependencyNotFound {
		t.Errorf("code = %q, want %q", err.Code, clierr.DependencyNotFound)
	}
	if err.Details["id"] != 99 {
		t.Errorf("details[id] = %v, want 99", err.Details["id"])
	}
}

func TestValidateWIPLimit(t *testing.T) {
	err := ValidateWIPLimit("in-progress", 3, 3)
	if err.Code != clierr.WIPLimitExceeded {
		t.Errorf("code = %q, want %q", err.Code, clierr.WIPLimitExceeded)
	}
	if err.Details["status"] != "in-progress" {
		t.Errorf("details[status] = %v, want %q", err.Details["status"], "in-progress")
	}
	if err.Details["limit"] != 3 {
		t.Errorf("details[limit] = %v, want 3", err.Details["limit"])
	}
	if err.Details["current"] != 3 {
		t.Errorf("details[current] = %v, want 3", err.Details["current"])
	}
}

func TestValidateBoundaryError(t *testing.T) {
	err := ValidateBoundaryError(5, "done", "last")
	if err.Code != clierr.BoundaryError {
		t.Errorf("code = %q, want %q", err.Code, clierr.BoundaryError)
	}
	if err.Details["id"] != 5 {
		t.Errorf("details[id] = %v, want 5", err.Details["id"])
	}
	if err.Details["direction"] != "last" {
		t.Errorf("details[direction] = %v, want %q", err.Details["direction"], "last")
	}
}

func TestFormatDueDate(t *testing.T) {
	err := FormatDueDate("2026-13-01", errors.New("month out of range"))
	if err.Code != clierr.InvalidDate {
		t.Errorf("code = %q, want %q", err.Code, clierr.InvalidDate)
	}
	if err.Details["field"] != "due" {
		t.Errorf("details[field] = %v, want %q", err.Details["field"], "due")
	}
}

func TestCheckClaimAllowed(t *testing.T) {
	const defaultTimeout = time.Hour

	pastTime := time.Now().Add(-2 * time.Hour)
	recentTime := time.Now().Add(-10 * time.Minute)

	tests := []struct {
		name      string
		task      Task
		claimant  string
		force     bool
		timeout   time.Duration
		wantClear bool // expect ClaimedBy/ClaimedAt cleared
	}{
		{
			name:    "unclaimed task allows operation",
			task:    Task{ID: 1, ClaimedBy: ""},
			timeout: defaultTimeout,
		},
		{
			name:     "same agent allows operation",
			task:     Task{ID: 2, ClaimedBy: "agent-1", ClaimedAt: &recentTime},
			claimant: "agent-1",
			timeout:  defaultTimeout,
		},
		{
			name:      "expired claim allows operation and clears fields",
			task:      Task{ID: 3, ClaimedBy: "agent-1", ClaimedAt: &pastTime},
			claimant:  "agent-2",
			timeout:   defaultTimeout,
			wantClear: true,
		},
		{
			name:      "force overrides active claim and clears fields",
			task:      Task{ID: 5, ClaimedBy: "agent-1", ClaimedAt: &recentTime},
			claimant:  "agent-2",
			force:     true,
			timeout:   defaultTimeout,
			wantClear: true,
		},
		{
			name:    "force on unclaimed task is a no-op",
			task:    Task{ID: 10, ClaimedBy: ""},
			force:   true,
			timeout: defaultTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := tt.task
			if tt.task.ClaimedAt != nil {
				cp := *tt.task.ClaimedAt
				tk.ClaimedAt = &cp
			}

			err := CheckClaim(&tk, tt.claimant, tt.force, tt.timeout)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantClear {
				if tk.ClaimedBy != "" {
					t.Errorf("ClaimedBy = %q, want empty (cleared)", tk.ClaimedBy)
				}
				if tk.ClaimedAt != nil {
					t.Errorf("ClaimedAt = %v, want nil (cleared)", tk.ClaimedAt)
				}
			}
		})
	}
}

func TestCheckClaimBlocked(t *testing.T) {
	const defaultTimeout = time.Hour

	pastTime := time.Now().Add(-2 * time.Hour)
	recentTime := time.Now().Add(-10 * time.Minute)

	tests := []struct {
		name     string
		task     Task
		claimant string
		timeout  time.Duration
	}{
		{
			name:     "active claim by different agent blocks",
			task:     Task{ID: 4, ClaimedBy: "agent-1", ClaimedAt: &recentTime},
			claimant: "agent-2",
			timeout:  defaultTimeout,
		},
		{
			name:     "ClaimedBy set but ClaimedAt nil blocks",
			task:     Task{ID: 6, ClaimedBy: "agent-1", ClaimedAt: nil},
			claimant: "agent-2",
			timeout:  defaultTimeout,
		},
		{
			name:     "timeout zero means no expiry",
			task:     Task{ID: 7, ClaimedBy: "agent-1", ClaimedAt: &pastTime},
			claimant: "agent-2",
			timeout:  0,
		},
		{
			name:     "empty claimant on claimed task blocks",
			task:     Task{ID: 9, ClaimedBy: "agent-1", ClaimedAt: &recentTime},
			claimant: "",
			timeout:  defaultTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := tt.task
			if tt.task.ClaimedAt != nil {
				cp := *tt.task.ClaimedAt
				tk.ClaimedAt = &cp
			}

			err := CheckClaim(&tk, tt.claimant, false, tt.timeout)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var cliErr *clierr.Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("expected clierr.Error, got %T: %v", err, err)
			}
			if cliErr.Code != clierr.TaskClaimed {
				t.Errorf("code = %q, want %q", cliErr.Code, clierr.TaskClaimed)
			}
		})
	}
}

func TestCheckClaimErrorDetails(t *testing.T) {
	const remainingUnknown = "unknown"

	recentTime := time.Now().Add(-10 * time.Minute)
	tk := &Task{ID: 42, ClaimedBy: "agent-x", ClaimedAt: &recentTime}

	err := CheckClaim(tk, "other-agent", false, time.Hour)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}

	if cliErr.Details["id"] != 42 {
		t.Errorf("details[id] = %v, want 42", cliErr.Details["id"])
	}
	if cliErr.Details["claimed_by"] != "agent-x" {
		t.Errorf("details[claimed_by] = %v, want %q", cliErr.Details["claimed_by"], "agent-x")
	}
	remaining, ok := cliErr.Details["remaining"].(string)
	if !ok {
		t.Fatalf("details[remaining] type = %T, want string", cliErr.Details["remaining"])
	}
	if remaining == "" || remaining == remainingUnknown {
		t.Errorf("details[remaining] = %q, want a non-empty duration string", remaining)
	}
}

func TestCheckClaimRemainingUnknown(t *testing.T) {
	const remainingUnknown = "unknown"

	// When ClaimedAt is nil and timeout > 0, remaining should be "unknown".
	tk := &Task{ID: 1, ClaimedBy: "agent-1", ClaimedAt: nil}

	err := CheckClaim(tk, "other", false, time.Hour)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}

	if cliErr.Details["remaining"] != remainingUnknown {
		t.Errorf("details[remaining] = %v, want %q", cliErr.Details["remaining"], remainingUnknown)
	}
}
