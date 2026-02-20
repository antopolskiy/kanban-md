package task

import (
	"errors"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/clierr"
)

// --- ValidateClass tests ---

func TestValidateClass_Valid(t *testing.T) {
	err := ValidateClass("standard", []string{"expedite", "standard", "intangible"})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateClass_FirstElement(t *testing.T) {
	err := ValidateClass("expedite", []string{"expedite", "standard", "intangible"})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateClass_LastElement(t *testing.T) {
	err := ValidateClass("intangible", []string{"expedite", "standard", "intangible"})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateClass_Invalid(t *testing.T) {
	err := ValidateClass("nonexistent", []string{"expedite", "standard", "intangible"})
	if err == nil {
		t.Fatal("expected error for invalid class")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.InvalidClass {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.InvalidClass)
	}
	if cliErr.Details["class"] != "nonexistent" {
		t.Errorf("details[class] = %v, want %q", cliErr.Details["class"], "nonexistent")
	}
	allowed, ok := cliErr.Details["allowed"].([]string)
	if !ok {
		t.Fatalf("details[allowed] type = %T, want []string", cliErr.Details["allowed"])
	}
	if len(allowed) != 3 {
		t.Errorf("details[allowed] len = %d, want 3", len(allowed))
	}
}

func TestValidateClass_EmptyAllowed(t *testing.T) {
	err := ValidateClass("standard", nil)
	if err == nil {
		t.Fatal("expected error when allowed list is empty")
	}
}

func TestValidateClass_EmptyClass(t *testing.T) {
	err := ValidateClass("", []string{"standard"})
	if err == nil {
		t.Fatal("expected error for empty class name")
	}
}

// --- ValidateClassWIPExceeded tests ---

func TestValidateClassWIPExceeded_Details(t *testing.T) {
	err := ValidateClassWIPExceeded("expedite", 1, 1)
	if err.Code != clierr.ClassWIPExceeded {
		t.Errorf("code = %q, want %q", err.Code, clierr.ClassWIPExceeded)
	}
	if err.Details["class"] != "expedite" {
		t.Errorf("details[class] = %v, want %q", err.Details["class"], "expedite")
	}
	if err.Details["limit"] != 1 {
		t.Errorf("details[limit] = %v, want 1", err.Details["limit"])
	}
	if err.Details["current"] != 1 {
		t.Errorf("details[current] = %v, want 1", err.Details["current"])
	}
}

func TestValidateClassWIPExceeded_MessageFormat(t *testing.T) {
	err := ValidateClassWIPExceeded("standard", 5, 5)
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	// Check the message contains the class name and counts.
	if !containsStr(msg, "standard") {
		t.Errorf("message %q should contain class name", msg)
	}
	if !containsStr(msg, "5/5") {
		t.Errorf("message %q should contain current/limit", msg)
	}
}

// --- ValidateTaskClaimed tests ---

func TestValidateTaskClaimed_Details(t *testing.T) {
	err := ValidateTaskClaimed(42, "agent-x", "50m0s")
	if err.Code != clierr.TaskClaimed {
		t.Errorf("code = %q, want %q", err.Code, clierr.TaskClaimed)
	}
	if err.Details["id"] != 42 {
		t.Errorf("details[id] = %v, want 42", err.Details["id"])
	}
	if err.Details["claimed_by"] != "agent-x" {
		t.Errorf("details[claimed_by] = %v, want %q", err.Details["claimed_by"], "agent-x")
	}
	if err.Details["remaining"] != "50m0s" {
		t.Errorf("details[remaining] = %v, want %q", err.Details["remaining"], "50m0s")
	}
}

func TestValidateTaskClaimed_MessageFormat(t *testing.T) {
	err := ValidateTaskClaimed(7, "bot-1", "30m0s")
	msg := err.Error()
	if !containsStr(msg, "bot-1") {
		t.Errorf("message %q should contain claimant", msg)
	}
	if !containsStr(msg, "--claim bot-1") {
		t.Errorf("message %q should suggest '--claim bot-1' to re-claim", msg)
	}
}

// --- ValidateDependencyIDs tests ---

func TestValidateDependencyIDs_Valid(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "First task", "backlog")
	createTestTask(t, dir, 2, "Second task", "todo")

	err := ValidateDependencyIDs(dir, 3, []int{1, 2})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateDependencyIDs_SelfReference(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "First task", "backlog")

	err := ValidateDependencyIDs(dir, 1, []int{1})
	if err == nil {
		t.Fatal("expected error for self-reference")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.SelfReference {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.SelfReference)
	}
}

func TestValidateDependencyIDs_NotFound(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "First task", "backlog")

	err := ValidateDependencyIDs(dir, 1, []int{99})
	if err == nil {
		t.Fatal("expected error for non-existent dependency")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.DependencyNotFound {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.DependencyNotFound)
	}
}

func TestValidateDependencyIDs_Empty(t *testing.T) {
	dir := t.TempDir()

	err := ValidateDependencyIDs(dir, 1, nil)
	if err != nil {
		t.Errorf("expected nil for empty deps, got %v", err)
	}
}

func TestValidateDependencyIDs_SelfReferenceCheckedFirst(t *testing.T) {
	// Self-reference should be detected before checking if the task file exists.
	dir := t.TempDir()

	err := ValidateDependencyIDs(dir, 5, []int{5})
	if err == nil {
		t.Fatal("expected error for self-reference")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.SelfReference {
		t.Errorf("code = %q, want %q (self-ref checked before file lookup)", cliErr.Code, clierr.SelfReference)
	}
}

func TestValidateDependencyIDs_MultipleWithOneBad(t *testing.T) {
	dir := t.TempDir()
	createTestTask(t, dir, 1, "First task", "backlog")
	createTestTask(t, dir, 2, "Second task", "todo")

	// First dep is valid, second is not found.
	err := ValidateDependencyIDs(dir, 5, []int{1, 99})
	if err == nil {
		t.Fatal("expected error for non-existent dependency")
	}
	var cliErr *clierr.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected clierr.Error, got %T", err)
	}
	if cliErr.Code != clierr.DependencyNotFound {
		t.Errorf("code = %q, want %q", cliErr.Code, clierr.DependencyNotFound)
	}
}

// containsStr is a simple substring check helper for test assertions.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstr(s, sub))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
