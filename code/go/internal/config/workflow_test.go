package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkflow_FullExample(t *testing.T) {
	content := `---
tracker:
  kind: linear
  api_key: "$LINEAR_API_KEY"
  project_slug: my-project
  active_states:
    - Todo
    - In Progress

polling:
  interval_ms: 30000

workspace:
  root: ~/workspaces

hooks:
  after_create: |
    git clone https://github.com/org/repo.git .
    npm install
  before_run: |
    git pull origin main

agent:
  command: "claude-code --auto"
  max_concurrent: 3
---

# Workflow Prompt

You are working on issue {{ issue.identifier }}: {{ issue.title }}

{{ issue.description }}
`

	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	wf, err := LoadWorkflow(path)
	if err != nil {
		t.Fatalf("LoadWorkflow() error: %v", err)
	}

	// Check config map.
	kind, ok := getNestedString(wf.Config, "tracker.kind")
	if !ok || kind != "linear" {
		t.Errorf("tracker.kind = %q, want %q", kind, "linear")
	}

	slug, ok := getNestedString(wf.Config, "tracker.project_slug")
	if !ok || slug != "my-project" {
		t.Errorf("tracker.project_slug = %q, want %q", slug, "my-project")
	}

	// Check prompt body.
	if wf.PromptTemplate == "" {
		t.Error("PromptTemplate is empty")
	}
	if !contains(wf.PromptTemplate, "{{ issue.identifier }}") {
		t.Error("PromptTemplate missing template variable")
	}
}

func TestLoadWorkflow_MissingFile(t *testing.T) {
	_, err := LoadWorkflow("/nonexistent/WORKFLOW.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	wfErr, ok := err.(*WorkflowError)
	if !ok {
		t.Fatalf("expected *WorkflowError, got %T", err)
	}
	if wfErr.Kind != "missing_workflow_file" {
		t.Errorf("error kind = %q, want %q", wfErr.Kind, "missing_workflow_file")
	}
}

func TestLoadWorkflow_NoFrontMatter(t *testing.T) {
	content := `# Just a prompt

Work on {{ issue.identifier }}.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	wf, err := LoadWorkflow(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(wf.Config) != 0 {
		t.Errorf("Config should be empty, got %v", wf.Config)
	}
	if !contains(wf.PromptTemplate, "{{ issue.identifier }}") {
		t.Error("PromptTemplate should contain the full file content")
	}
}

func TestLoadWorkflow_FrontMatterNotAMap(t *testing.T) {
	content := `---
- item1
- item2
---

Body here.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadWorkflow(path)
	if err == nil {
		t.Fatal("expected error for non-map front matter")
	}
	wfErr, ok := err.(*WorkflowError)
	if !ok {
		t.Fatalf("expected *WorkflowError, got %T", err)
	}
	if wfErr.Kind != "workflow_front_matter_not_a_map" {
		t.Errorf("error kind = %q, want %q", wfErr.Kind, "workflow_front_matter_not_a_map")
	}
}

func TestLoadWorkflow_EmptyFrontMatter(t *testing.T) {
	content := `---
---

Body after empty front matter.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	wf, err := LoadWorkflow(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Config) != 0 {
		t.Errorf("Config should be empty for empty front matter, got %v", wf.Config)
	}
	if wf.PromptTemplate != "Body after empty front matter." {
		t.Errorf("PromptTemplate = %q, want %q", wf.PromptTemplate, "Body after empty front matter.")
	}
}

func TestLoadWorkflow_UnknownKeysIgnored(t *testing.T) {
	content := `---
tracker:
  kind: linear
future_key: future_value
another_unknown:
  nested: true
---

Prompt body.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	wf, err := LoadWorkflow(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	kind, ok := getNestedString(wf.Config, "tracker.kind")
	if !ok || kind != "linear" {
		t.Errorf("tracker.kind = %q, want %q", kind, "linear")
	}

	// Unknown keys should be present (not rejected).
	if _, exists := wf.Config["future_key"]; !exists {
		t.Error("expected unknown key 'future_key' to be preserved")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
