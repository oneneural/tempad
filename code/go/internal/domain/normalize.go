package domain

import (
	"regexp"
	"strings"
)

// unsafeCharsRe matches any character not in [A-Za-z0-9._-].
var unsafeCharsRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// SanitizeIdentifier derives a workspace-safe key from an issue identifier
// by replacing any character not in [A-Za-z0-9._-] with '_'.
// See Spec Section 4.2.
func SanitizeIdentifier(identifier string) string {
	return unsafeCharsRe.ReplaceAllString(identifier, "_")
}

// NormalizeState normalizes a tracker state name by trimming whitespace
// and converting to lowercase. All state comparisons should use this
// function. See Spec Section 4.2.
func NormalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

// NormalizeStates normalizes a slice of state names and returns a
// lookup map for O(1) membership checks.
func NormalizeStates(states []string) map[string]bool {
	m := make(map[string]bool, len(states))
	for _, s := range states {
		m[NormalizeState(s)] = true
	}
	return m
}
