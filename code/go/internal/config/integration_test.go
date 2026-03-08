package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_FullConfigPipeline exercises the full config loading,
// merging, and validation pipeline end-to-end.
func TestIntegration_FullConfigPipeline(t *testing.T) {
	// Set up environment.
	os.Setenv("TEMPAD_TEST_LINEAR_KEY", "lin_api_test_key_123")
	defer os.Unsetenv("TEMPAD_TEST_LINEAR_KEY")

	// Create a temp WORKFLOW.md.
	dir := t.TempDir()
	workflowContent := `---
tracker:
  kind: linear
  api_key: "$TEMPAD_TEST_LINEAR_KEY"
  project_slug: my-project
  active_states:
    - Todo
    - In Progress
    - Review

polling:
  interval_ms: 15000

workspace:
  root: /tmp/tempad_test_workspaces

hooks:
  after_create: |
    echo "workspace created"
  before_run: |
    echo "before run"

agent:
  command: "echo 'agent placeholder'"
  max_concurrent: 3
  max_retries: 5
  prompt_delivery: file
---

# Issue: {{ issue.identifier }}

## {{ issue.title }}

{{ issue.description }}

Priority: {{ issue.priority | default: "None" }}
State: {{ issue.state }}

{% if issue.labels.size > 0 %}
Labels: {% for label in issue.labels %}{{ label }}{% unless forloop.last %}, {% endunless %}{% endfor %}
{% endif %}

{% if attempt %}
This is retry attempt {{ attempt }}.
{% endif %}
`
	workflowPath := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 1: Load workflow.
	workflow, err := LoadWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("LoadWorkflow() error: %v", err)
	}

	if workflow.PromptTemplate == "" {
		t.Error("expected non-empty prompt template")
	}

	// Verify tracker config extracted.
	kind, ok := getNestedString(workflow.Config, "tracker.kind")
	if !ok || kind != "linear" {
		t.Errorf("tracker.kind = %q, want %q", kind, "linear")
	}

	// Step 2: Load user config (empty — uses defaults).
	userCfg := &UserConfig{}
	userCfg.Tracker.Identity = "testdev@example.com"
	userCfg.IDE.Command = "cursor"
	userCfg.Display.Theme = "dark"

	// Step 3: Merge with CLI flags.
	cli := &CLIFlags{
		WorkflowPath: workflowPath,
		LogLevel:     "debug",
	}

	cfg := Merge(cli, userCfg, workflow)

	// Step 4: Verify merge results.

	// Tracker settings (from workflow).
	if cfg.TrackerKind != "linear" {
		t.Errorf("TrackerKind = %q, want %q", cfg.TrackerKind, "linear")
	}
	if cfg.TrackerAPIKey != "lin_api_test_key_123" {
		t.Errorf("TrackerAPIKey = %q, want resolved key", cfg.TrackerAPIKey)
	}
	if cfg.TrackerProjectSlug != "my-project" {
		t.Errorf("TrackerProjectSlug = %q, want %q", cfg.TrackerProjectSlug, "my-project")
	}

	// Active states (from workflow, overriding defaults).
	if len(cfg.ActiveStates) != 3 {
		t.Errorf("ActiveStates len = %d, want 3", len(cfg.ActiveStates))
	}

	// Terminal states (default, not overridden).
	if len(cfg.TerminalStates) != 5 {
		t.Errorf("TerminalStates len = %d, want 5 (defaults)", len(cfg.TerminalStates))
	}

	// Polling (from workflow).
	if cfg.PollIntervalMs != 15000 {
		t.Errorf("PollIntervalMs = %d, want 15000", cfg.PollIntervalMs)
	}

	// Hooks (from workflow, not overridable).
	if !strings.Contains(cfg.AfterCreateHook, "workspace created") {
		t.Error("AfterCreateHook not set from workflow")
	}

	// Agent (from workflow).
	if cfg.MaxConcurrent != 3 {
		t.Errorf("MaxConcurrent = %d, want 3", cfg.MaxConcurrent)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}

	// Personal preferences (from user config).
	if cfg.TrackerIdentity != "testdev@example.com" {
		t.Errorf("TrackerIdentity = %q, want user value", cfg.TrackerIdentity)
	}
	if cfg.IDECommand != "cursor" {
		t.Errorf("IDECommand = %q, want user value %q", cfg.IDECommand, "cursor")
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want user value %q", cfg.Theme, "dark")
	}

	// CLI override.
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want CLI value %q", cfg.LogLevel, "debug")
	}

	// Step 5: Validate for TUI mode.
	if err := ValidateForStartup(cfg, "tui"); err != nil {
		t.Errorf("TUI mode validation failed: %v", err)
	}

	// Step 6: Validate for daemon mode.
	if err := ValidateForStartup(cfg, "daemon"); err != nil {
		t.Errorf("Daemon mode validation failed: %v", err)
	}
}

// TestIntegration_ValidationFailure tests the pipeline with invalid config.
func TestIntegration_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	workflowContent := `---
tracker:
  kind: linear
---

Just a prompt.
`
	workflowPath := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	workflow, err := LoadWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("LoadWorkflow() error: %v", err)
	}

	// No user config, no CLI flags — should fail validation.
	cfg := Merge(nil, nil, workflow)

	err = ValidateForStartup(cfg, "daemon")
	if err == nil {
		t.Fatal("expected validation failure")
	}

	// Should mention multiple missing fields.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "api_key") {
		t.Error("error should mention api_key")
	}
	if !strings.Contains(errMsg, "identity") {
		t.Error("error should mention identity")
	}
}

// TestIntegration_MissingWorkflow tests loading when WORKFLOW.md doesn't exist.
func TestIntegration_MissingWorkflow(t *testing.T) {
	_, err := LoadWorkflow("/nonexistent/path/WORKFLOW.md")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}

	wfErr, ok := err.(*WorkflowError)
	if !ok {
		t.Fatalf("expected *WorkflowError, got %T", err)
	}
	if wfErr.Kind != "missing_workflow_file" {
		t.Errorf("error kind = %q, want %q", wfErr.Kind, "missing_workflow_file")
	}
}
