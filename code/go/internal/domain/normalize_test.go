package domain

import "testing"

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ABC-123", "ABC-123"},
		{"ABC-123/foo bar", "ABC-123_foo_bar"},
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"special!@#$chars", "special____chars"},
		{"dots.and-dashes", "dots.and-dashes"},
		{"UPPER_lower.123", "UPPER_lower.123"},
		{"слово", "_____"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeState(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"In Progress", "in progress"},
		{"  In Progress  ", "in progress"},
		{"TODO", "todo"},
		{"closed", "closed"},
		{"  ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeState(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeStates(t *testing.T) {
	states := []string{"Closed", "Cancelled", "  Done  "}
	m := NormalizeStates(states)

	if !m["closed"] {
		t.Error("expected 'closed' to be in normalized states")
	}
	if !m["cancelled"] {
		t.Error("expected 'cancelled' to be in normalized states")
	}
	if !m["done"] {
		t.Error("expected 'done' to be in normalized states")
	}
	if m["open"] {
		t.Error("did not expect 'open' to be in normalized states")
	}
}
