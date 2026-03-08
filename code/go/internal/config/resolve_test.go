package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvVar(t *testing.T) {
	// Set a test env var.
	os.Setenv("TEMPAD_TEST_KEY", "resolved_value")
	defer os.Unsetenv("TEMPAD_TEST_KEY")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"literal value", "my_key_123", "my_key_123"},
		{"env var set", "$TEMPAD_TEST_KEY", "resolved_value"},
		{"env var unset", "$NONEXISTENT_VAR_12345", ""},
		{"empty string", "", ""},
		{"dollar only", "$", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveEnvVar(tt.input)
			if got != tt.want {
				t.Errorf("ResolveEnvVar(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tilde only", "~", home},
		{"tilde slash", "~/workspaces", filepath.Join(home, "workspaces")},
		{"tilde nested", "~/a/b/c", filepath.Join(home, "a/b/c")},
		{"absolute path", "/usr/local/bin", "/usr/local/bin"},
		{"relative path", "relative/path", "relative/path"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHome(tt.input)
			if got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
