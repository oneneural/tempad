// Package domain defines the core data structures used throughout TEMPAD.
// This package has zero dependencies on infrastructure packages — it is
// imported by every other package and must remain a leaf node in the
// dependency graph.
package domain

import "time"

// Issue is the normalized issue record used by orchestration, TUI display,
// prompt rendering, and observability. See Spec Section 4.1.1.
type Issue struct {
	// ID is the stable tracker-internal identifier (e.g. Linear UUID).
	ID string `json:"id"`

	// Identifier is the human-readable ticket key (e.g. "ABC-123").
	Identifier string `json:"identifier"`

	// Title is the issue title.
	Title string `json:"title"`

	// Description is the full issue body (may be empty).
	Description string `json:"description"`

	// Priority is an integer where lower numbers are higher priority.
	// nil means no priority set.
	Priority *int `json:"priority"`

	// State is the current tracker state name (e.g. "In Progress").
	State string `json:"state"`

	// Assignee is the tracker user ID or email of the current assignee.
	// Empty string means unassigned.
	Assignee string `json:"assignee"`

	// BranchName is tracker-provided branch metadata if available.
	BranchName string `json:"branch_name"`

	// URL is the web link to the issue in the tracker.
	URL string `json:"url"`

	// Labels are normalized to lowercase.
	Labels []string `json:"labels"`

	// BlockedBy is the list of blocking issue references.
	BlockedBy []BlockerRef `json:"blocked_by"`

	// CreatedAt is the issue creation timestamp.
	CreatedAt *time.Time `json:"created_at"`

	// UpdatedAt is the last modification timestamp.
	UpdatedAt *time.Time `json:"updated_at"`
}

// BlockerRef represents a reference to a blocking issue.
// See Spec Section 4.1.1 (blocked_by list).
type BlockerRef struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	State      string `json:"state"`
}

// HasNonTerminalBlockers returns true if the issue has any blockers whose
// state is not in the given set of terminal states. States are compared
// after normalization (trim + lowercase).
func (i *Issue) HasNonTerminalBlockers(terminalStates map[string]bool) bool {
	for _, b := range i.BlockedBy {
		normalized := NormalizeState(b.State)
		if !terminalStates[normalized] {
			return true
		}
	}
	return false
}
