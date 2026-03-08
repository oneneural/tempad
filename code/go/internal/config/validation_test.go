package config

import (
	"strings"
	"testing"
)

func TestValidateForStartup_Valid(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "lin_api_xxx",
		TrackerProjectSlug: "my-project",
		TrackerIdentity:    "user@example.com",
		AgentCommand:       "claude-code --auto",
	}

	if err := ValidateForStartup(cfg, "daemon"); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidateForStartup_MissingTrackerKind(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerAPIKey:      "key",
		TrackerProjectSlug: "slug",
		TrackerIdentity:    "user",
	}

	err := ValidateForStartup(cfg, "tui")
	if err == nil {
		t.Fatal("expected error for missing tracker.kind")
	}
	if !strings.Contains(err.Error(), "tracker.kind") {
		t.Errorf("error should mention tracker.kind: %v", err)
	}
}

func TestValidateForStartup_UnsupportedTrackerKind(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:        "jira",
		TrackerAPIKey:      "key",
		TrackerProjectSlug: "slug",
		TrackerIdentity:    "user",
	}

	err := ValidateForStartup(cfg, "tui")
	if err == nil {
		t.Fatal("expected error for unsupported tracker.kind")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention unsupported: %v", err)
	}
}

func TestValidateForStartup_MissingAPIKey(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:        "linear",
		TrackerProjectSlug: "slug",
		TrackerIdentity:    "user",
	}

	err := ValidateForStartup(cfg, "tui")
	if err == nil {
		t.Fatal("expected error for missing api_key")
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Errorf("error should mention api_key: %v", err)
	}
}

func TestValidateForStartup_MissingProjectSlug(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:     "linear",
		TrackerAPIKey:   "key",
		TrackerIdentity: "user",
	}

	err := ValidateForStartup(cfg, "tui")
	if err == nil {
		t.Fatal("expected error for missing project_slug")
	}
	if !strings.Contains(err.Error(), "project_slug") {
		t.Errorf("error should mention project_slug: %v", err)
	}
}

func TestValidateForStartup_MissingIdentity(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "key",
		TrackerProjectSlug: "slug",
	}

	err := ValidateForStartup(cfg, "tui")
	if err == nil {
		t.Fatal("expected error for missing identity")
	}
	if !strings.Contains(err.Error(), "identity") {
		t.Errorf("error should mention identity: %v", err)
	}
}

func TestValidateForStartup_MissingAgentCommand_DaemonMode(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "key",
		TrackerProjectSlug: "slug",
		TrackerIdentity:    "user",
	}

	err := ValidateForStartup(cfg, "daemon")
	if err == nil {
		t.Fatal("expected error for missing agent.command in daemon mode")
	}
	if !strings.Contains(err.Error(), "agent.command") {
		t.Errorf("error should mention agent.command: %v", err)
	}
}

func TestValidateForStartup_MissingAgentCommand_TUIMode(t *testing.T) {
	cfg := &ServiceConfig{
		TrackerKind:        "linear",
		TrackerAPIKey:      "key",
		TrackerProjectSlug: "slug",
		TrackerIdentity:    "user",
	}

	// TUI mode does not require agent.command.
	err := ValidateForStartup(cfg, "tui")
	if err != nil {
		t.Errorf("TUI mode should not require agent.command, got: %v", err)
	}
}

func TestValidateForStartup_MultipleErrors(t *testing.T) {
	cfg := &ServiceConfig{}

	err := ValidateForStartup(cfg, "daemon")
	if err == nil {
		t.Fatal("expected multiple errors")
	}

	vErrs, ok := err.(*ValidationErrors)
	if !ok {
		t.Fatalf("expected *ValidationErrors, got %T", err)
	}
	// Should have at least: tracker.kind, api_key, identity, agent.command
	if len(vErrs.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(vErrs.Errors), err)
	}
}
