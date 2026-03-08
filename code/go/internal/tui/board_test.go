package tui

import (
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/stretchr/testify/assert"
)

func intPtr(v int) *int { return &v }

func TestSortIssues(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	issues := []domain.Issue{
		{Identifier: "C", Priority: intPtr(3), CreatedAt: &now},
		{Identifier: "A", Priority: intPtr(1), CreatedAt: &now},
		{Identifier: "B", Priority: intPtr(1), CreatedAt: &earlier},
		{Identifier: "D", Priority: nil, CreatedAt: &now},       // nil priority → last
		{Identifier: "E", Priority: intPtr(2), CreatedAt: &later},
	}

	sortIssues(issues)

	expected := []string{"B", "A", "E", "C", "D"}
	var got []string
	for _, i := range issues {
		got = append(got, i.Identifier)
	}
	assert.Equal(t, expected, got, "sort: priority asc (nil last) → created_at oldest → identifier")
}

func TestSortIssues_SamePrioritySameTime(t *testing.T) {
	now := time.Now()
	issues := []domain.Issue{
		{Identifier: "ZZZ-3", Priority: intPtr(2), CreatedAt: &now},
		{Identifier: "AAA-1", Priority: intPtr(2), CreatedAt: &now},
		{Identifier: "MMM-2", Priority: intPtr(2), CreatedAt: &now},
	}

	sortIssues(issues)

	expected := []string{"AAA-1", "MMM-2", "ZZZ-3"}
	var got []string
	for _, i := range issues {
		got = append(got, i.Identifier)
	}
	assert.Equal(t, expected, got, "same priority+time → identifier tiebreaker")
}

func TestIsBlocked(t *testing.T) {
	terminal := map[string]bool{"done": true, "closed": true}

	// Todo with non-terminal blocker → blocked
	issue := domain.Issue{
		State:     "Todo",
		BlockedBy: []domain.BlockerRef{{ID: "1", State: "In Progress"}},
	}
	assert.True(t, isBlocked(issue, terminal))

	// Todo with terminal blocker → not blocked
	issue.BlockedBy = []domain.BlockerRef{{ID: "1", State: "Done"}}
	assert.False(t, isBlocked(issue, terminal))

	// Non-Todo state → never blocked
	issue.State = "In Progress"
	issue.BlockedBy = []domain.BlockerRef{{ID: "1", State: "In Progress"}}
	assert.False(t, isBlocked(issue, terminal))

	// Todo with no blockers → not blocked
	issue.State = "Todo"
	issue.BlockedBy = nil
	assert.False(t, isBlocked(issue, terminal))
}

func TestViewBoard_EmptyState(t *testing.T) {
	m := Model{
		cfg: &defaultTestConfig,
	}

	view := m.viewBoard()
	assert.Contains(t, view, "TEMPAD")
	assert.Contains(t, view, "No available tasks")
	assert.Contains(t, view, "No active tasks")
}

var defaultTestConfig = testConfig()

func testConfig() config.ServiceConfig {
	return config.ServiceConfig{
		TerminalStates: []string{"Done", "Closed", "Cancelled"},
		PollIntervalMs: 30000,
	}
}
