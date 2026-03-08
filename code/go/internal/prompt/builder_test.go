package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/domain"
)

func newTestIssue() domain.Issue {
	priority := 1
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	return domain.Issue{
		ID:          "abc-uuid-123",
		Identifier:  "ABC-123",
		Title:       "Fix the login bug",
		Description: "Users can't log in when using OAuth.",
		Priority:    &priority,
		State:       "In Progress",
		Assignee:    "dev@example.com",
		BranchName:  "abc-123-fix-login",
		URL:         "https://linear.app/team/issue/ABC-123",
		Labels:      []string{"bug", "auth", "p1"},
		BlockedBy: []domain.BlockerRef{
			{ID: "xyz-uuid", Identifier: "XYZ-001", State: "Todo"},
		},
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

func TestRender_BasicTemplate(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	tmpl := "Work on {{ issue.identifier }}: {{ issue.title }}"
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Work on ABC-123: Fix the login bug"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRender_AllFields(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	tmpl := `ID: {{ issue.id }}
Identifier: {{ issue.identifier }}
Title: {{ issue.title }}
State: {{ issue.state }}
Priority: {{ issue.priority }}
Assignee: {{ issue.assignee }}
Branch: {{ issue.branch_name }}
URL: {{ issue.url }}
Description: {{ issue.description }}`

	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "abc-uuid-123") {
		t.Error("missing id")
	}
	if !strings.Contains(result, "ABC-123") {
		t.Error("missing identifier")
	}
	if !strings.Contains(result, "Fix the login bug") {
		t.Error("missing title")
	}
	if !strings.Contains(result, "In Progress") {
		t.Error("missing state")
	}
}

func TestRender_LabelsIteration(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	tmpl := `{% for label in issue.labels %}{{ label }} {% endfor %}`
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "bug") {
		t.Error("missing label 'bug'")
	}
	if !strings.Contains(result, "auth") {
		t.Error("missing label 'auth'")
	}
	if !strings.Contains(result, "p1") {
		t.Error("missing label 'p1'")
	}
}

func TestRender_BlockedByIteration(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	tmpl := `{% for blocker in issue.blocked_by %}{{ blocker.identifier }} ({{ blocker.state }}) {% endfor %}`
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "XYZ-001") {
		t.Error("missing blocker identifier")
	}
	if !strings.Contains(result, "Todo") {
		t.Error("missing blocker state")
	}
}

func TestRender_AttemptNil(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	tmpl := `{% if attempt %}Retry attempt {{ attempt }}{% else %}First run{% endif %}`
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "First run") {
		t.Errorf("expected 'First run', got %q", result)
	}
}

func TestRender_AttemptSet(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()
	attempt := 3

	tmpl := `{% if attempt %}Retry attempt {{ attempt }}{% else %}First run{% endif %}`
	result, err := b.Render(tmpl, issue, &attempt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Retry attempt 3") {
		t.Errorf("expected 'Retry attempt 3', got %q", result)
	}
}

func TestRender_DefaultFilter(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()
	issue.Priority = nil

	tmpl := `Priority: {{ issue.priority | default: "None" }}`
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "None") {
		t.Errorf("expected default filter to produce 'None', got %q", result)
	}
}

func TestRender_EmptyTemplate(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	// Empty template should use the default prompt.
	result, err := b.Render("", issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "ABC-123") {
		t.Errorf("expected default prompt with identifier, got %q", result)
	}
}

func TestRender_NoPriority(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()
	issue.Priority = nil

	tmpl := `{% if issue.priority %}P{{ issue.priority }}{% else %}No priority{% endif %}`
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No priority") {
		t.Errorf("expected 'No priority', got %q", result)
	}
}

func TestRender_Timestamps(t *testing.T) {
	b := NewBuilder()
	issue := newTestIssue()

	tmpl := `Created: {{ issue.created_at }}`
	result, err := b.Render(tmpl, issue, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "2026-03-08") {
		t.Errorf("expected ISO timestamp, got %q", result)
	}
}
