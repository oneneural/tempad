package tui

import (
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestViewDetail_AllFields(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	p := 2
	m := Model{
		cfg: &defaultTestConfig,
		detailIssue: &domain.Issue{
			ID:          "id-1",
			Identifier:  "PROJ-42",
			Title:       "Fix the login bug",
			State:       "In Progress",
			Priority:    &p,
			Assignee:    "dev@example.com",
			Labels:      []string{"bug", "auth"},
			URL:         "https://linear.app/proj/PROJ-42",
			BranchName:  "fix/login-bug",
			Description: "The login form fails on mobile devices.",
			CreatedAt:   &now,
			UpdatedAt:   &now,
			BlockedBy: []domain.BlockerRef{
				{Identifier: "PROJ-40", State: "In Progress"},
			},
		},
		width: 80,
	}

	view := m.viewDetail()

	assert.Contains(t, view, "PROJ-42")
	assert.Contains(t, view, "Fix the login bug")
	assert.Contains(t, view, "In Progress")
	assert.Contains(t, view, "High (P2)")
	assert.Contains(t, view, "dev@example.com")
	assert.Contains(t, view, "bug, auth")
	assert.Contains(t, view, "https://linear.app/proj/PROJ-42")
	assert.Contains(t, view, "fix/login-bug")
	assert.Contains(t, view, "2026-03-08 12:00")
	assert.Contains(t, view, "PROJ-40")
	assert.Contains(t, view, "login form fails")
	assert.Contains(t, view, "Esc/Backspace")
}

func TestViewDetail_NilIssue(t *testing.T) {
	m := Model{cfg: &defaultTestConfig}
	view := m.viewDetail()
	assert.Contains(t, view, "No issue selected")
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{
			name:     "short line",
			text:     "hello world",
			width:    20,
			expected: "hello world",
		},
		{
			name:     "long line wraps",
			text:     "the quick brown fox jumps over the lazy dog",
			width:    20,
			expected: "the quick brown fox\njumps over the lazy\ndog",
		},
		{
			name:     "preserves newlines",
			text:     "line one\nline two",
			width:    80,
			expected: "line one\nline two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrap(tt.text, tt.width)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormatPriority(t *testing.T) {
	assert.Equal(t, "None", formatPriority(nil))
	p1 := 1
	assert.Equal(t, "Urgent (P1)", formatPriority(&p1))
	p4 := 4
	assert.Equal(t, "Low (P4)", formatPriority(&p4))
}
