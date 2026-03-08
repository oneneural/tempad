package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeIdentifier_Extended(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with slashes", "PROJ/123", "PROJ_123"},
		{"with dots", "PROJ.123", "PROJ.123"},
		{"with underscores", "PROJ_123", "PROJ_123"},
		{"unicode accented", "PROJ-123-ñ", "PROJ-123-_"},
		{"special chars", "a@b#c$d", "a_b_c_d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SanitizeIdentifier(tt.input))
		})
	}
}

func TestHasNonTerminalBlockers(t *testing.T) {
	terminal := NormalizeStates([]string{"Done", "Closed"})

	tests := []struct {
		name     string
		blockers []BlockerRef
		want     bool
	}{
		{"no blockers", nil, false},
		{"all terminal", []BlockerRef{
			{ID: "1", State: "Done"},
			{ID: "2", State: "Closed"},
		}, false},
		{"one non-terminal", []BlockerRef{
			{ID: "1", State: "In Progress"},
		}, true},
		{"mixed", []BlockerRef{
			{ID: "1", State: "Done"},
			{ID: "2", State: "In Progress"},
		}, true},
		{"case insensitive", []BlockerRef{
			{ID: "1", State: "DONE"},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &Issue{BlockedBy: tt.blockers}
			assert.Equal(t, tt.want, issue.HasNonTerminalBlockers(terminal))
		})
	}
}

func TestOrchestratorState(t *testing.T) {
	t.Run("NewOrchestratorState", func(t *testing.T) {
		s := NewOrchestratorState(1000, 5)
		assert.Equal(t, 1000, s.PollIntervalMs)
		assert.Equal(t, 5, s.MaxConcurrentAgents)
		assert.NotNil(t, s.Running)
		assert.NotNil(t, s.Claimed)
		assert.NotNil(t, s.RetryAttempts)
		assert.NotNil(t, s.Completed)
	})

	t.Run("RunningCount", func(t *testing.T) {
		s := NewOrchestratorState(1000, 5)
		assert.Equal(t, 0, s.RunningCount())
		s.Running["1"] = &RunAttempt{IssueID: "1"}
		assert.Equal(t, 1, s.RunningCount())
	})

	t.Run("AvailableSlots", func(t *testing.T) {
		s := NewOrchestratorState(1000, 3)
		assert.Equal(t, 3, s.AvailableSlots())
		s.Running["1"] = &RunAttempt{IssueID: "1"}
		assert.Equal(t, 2, s.AvailableSlots())
		s.Running["2"] = &RunAttempt{IssueID: "2"}
		s.Running["3"] = &RunAttempt{IssueID: "3"}
		assert.Equal(t, 0, s.AvailableSlots())
		s.Running["4"] = &RunAttempt{IssueID: "4"} // Over limit.
		assert.Equal(t, 0, s.AvailableSlots())
	})

	t.Run("IsClaimedOrRunning", func(t *testing.T) {
		s := NewOrchestratorState(1000, 5)
		assert.False(t, s.IsClaimedOrRunning("1"))
		s.Claimed["1"] = true
		assert.True(t, s.IsClaimedOrRunning("1"))
	})

	t.Run("Snapshot", func(t *testing.T) {
		s := NewOrchestratorState(1000, 5)
		state, unlock := s.Snapshot()
		defer unlock()
		assert.Equal(t, 1000, state.PollIntervalMs)
	})
}
