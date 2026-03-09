package orchestrator

import (
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/stretchr/testify/assert"
)

func intPtr(v int) *int { return &v }

func TestSelectCandidates_Filtering(t *testing.T) {
	cfg := &config.ServiceConfig{
		TerminalStates:  []string{"Done", "Closed"},
		TrackerIdentity: "me@example.com",
		MaxConcurrent:   5,
	}
	o := New(cfg, nil, nil, testLogger(), nil)

	now := time.Now()
	issues := []domain.Issue{
		// Eligible: unassigned, has all fields.
		{ID: "1", Identifier: "P-1", Title: "Task 1", State: "Todo", Priority: intPtr(1), CreatedAt: &now},
		// Missing fields — filtered out.
		{ID: "2", Identifier: "", Title: "Task 2", State: "Todo"},
		// Assigned to someone else — filtered out.
		{ID: "3", Identifier: "P-3", Title: "Task 3", State: "Todo", Assignee: "other@example.com", CreatedAt: &now},
		// Assigned to me — eligible (resumption).
		{ID: "4", Identifier: "P-4", Title: "Task 4", State: "In Progress", Assignee: "me@example.com", Priority: intPtr(2), CreatedAt: &now},
		// Blocked Todo — filtered out.
		{ID: "5", Identifier: "P-5", Title: "Task 5", State: "Todo",
			BlockedBy: []domain.BlockerRef{{ID: "b1", State: "In Progress"}}, CreatedAt: &now},
	}

	// Mark one as claimed.
	o.state.Claimed["1"] = true

	candidates := o.selectCandidates(issues)

	var ids []string
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}
	// Only "4" is eligible (1 is claimed, 2 missing fields, 3 assigned to other, 5 blocked).
	assert.Equal(t, []string{"4"}, ids)
}

func TestSelectCandidates_Sorting(t *testing.T) {
	cfg := &config.ServiceConfig{
		TerminalStates: []string{"Done"},
		MaxConcurrent:  5,
	}
	o := New(cfg, nil, nil, testLogger(), nil)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	issues := []domain.Issue{
		{ID: "a", Identifier: "C", Title: "t", State: "Todo", Priority: intPtr(3), CreatedAt: &now},
		{ID: "b", Identifier: "A", Title: "t", State: "Todo", Priority: intPtr(1), CreatedAt: &now},
		{ID: "c", Identifier: "B", Title: "t", State: "Todo", Priority: intPtr(1), CreatedAt: &earlier},
		{ID: "d", Identifier: "D", Title: "t", State: "Todo", Priority: nil, CreatedAt: &now},
	}

	candidates := o.selectCandidates(issues)

	var ids []string
	for _, c := range candidates {
		ids = append(ids, c.Identifier)
	}
	// B (P1, earlier) → A (P1, later) → C (P3) → D (nil priority, last)
	assert.Equal(t, []string{"B", "A", "C", "D"}, ids)
}

func TestSelectCandidates_RetrySkipped(t *testing.T) {
	cfg := &config.ServiceConfig{
		TerminalStates: []string{"Done"},
		MaxConcurrent:  5,
	}
	o := New(cfg, nil, nil, testLogger(), nil)

	now := time.Now()
	issues := []domain.Issue{
		{ID: "1", Identifier: "P-1", Title: "t", State: "Todo", CreatedAt: &now},
		{ID: "2", Identifier: "P-2", Title: "t", State: "Todo", CreatedAt: &now},
	}

	// Mark one as in retry.
	o.state.RetryAttempts["1"] = &domain.RetryEntry{IssueID: "1"}

	candidates := o.selectCandidates(issues)
	assert.Len(t, candidates, 1)
	assert.Equal(t, "2", candidates[0].ID)
}

func TestHasRequiredFields(t *testing.T) {
	assert.True(t, hasRequiredFields(domain.Issue{ID: "1", Identifier: "P-1", Title: "t", State: "s"}))
	assert.False(t, hasRequiredFields(domain.Issue{ID: "", Identifier: "P-1", Title: "t", State: "s"}))
	assert.False(t, hasRequiredFields(domain.Issue{ID: "1", Identifier: "", Title: "t", State: "s"}))
	assert.False(t, hasRequiredFields(domain.Issue{ID: "1", Identifier: "P-1", Title: "", State: "s"}))
	assert.False(t, hasRequiredFields(domain.Issue{ID: "1", Identifier: "P-1", Title: "t", State: ""}))
}

func TestStateSlotAvailable(t *testing.T) {
	cfg := &config.ServiceConfig{
		MaxConcurrent: 5,
		MaxConcurrentByState: map[string]int{
			"todo":        2,
			"in progress": 1,
		},
	}
	o := New(cfg, nil, nil, testLogger(), nil)

	// No running issues — should be available.
	assert.True(t, o.stateSlotAvailable("Todo"))
	assert.True(t, o.stateSlotAvailable("In Progress"))

	// Add a running issue in "todo" state.
	o.state.Running["1"] = &domain.RunAttempt{IssueID: "1", Status: "todo"}

	// Still under limit.
	assert.True(t, o.stateSlotAvailable("Todo"))

	// Add another.
	o.state.Running["2"] = &domain.RunAttempt{IssueID: "2", Status: "todo"}

	// At limit.
	assert.False(t, o.stateSlotAvailable("Todo"))

	// State without per-state limit — always available (falls back to global).
	assert.True(t, o.stateSlotAvailable("Review"))
}

func TestStateSlotAvailable_NoConfig(t *testing.T) {
	cfg := &config.ServiceConfig{MaxConcurrent: 5}
	o := New(cfg, nil, nil, testLogger(), nil)

	// No per-state config — always available.
	assert.True(t, o.stateSlotAvailable("Todo"))
}

func TestStateSlotAvailable_InvalidLimit(t *testing.T) {
	cfg := &config.ServiceConfig{
		MaxConcurrent:        5,
		MaxConcurrentByState: map[string]int{"todo": -1},
	}
	o := New(cfg, nil, nil, testLogger(), nil)

	// Invalid limit ignored — falls back to global.
	assert.True(t, o.stateSlotAvailable("Todo"))
}
