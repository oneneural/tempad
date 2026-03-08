package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUserConfig_FullExample(t *testing.T) {
	content := `
tracker:
  identity: "user@example.com"
  api_key: "$LINEAR_API_KEY"

ide:
  command: "cursor"
  args: "--new-window"

agent:
  command: "claude-code --auto"

display:
  theme: "dark"

logging:
  level: "debug"
  file: "~/.tempad/logs/tempad.log"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadUserConfig(path)
	if err != nil {
		t.Fatalf("LoadUserConfig() error: %v", err)
	}

	if cfg.Tracker.Identity != "user@example.com" {
		t.Errorf("Tracker.Identity = %q, want %q", cfg.Tracker.Identity, "user@example.com")
	}
	if cfg.Tracker.APIKey != "$LINEAR_API_KEY" {
		t.Errorf("Tracker.APIKey = %q, want %q", cfg.Tracker.APIKey, "$LINEAR_API_KEY")
	}
	if cfg.IDE.Command != "cursor" {
		t.Errorf("IDE.Command = %q, want %q", cfg.IDE.Command, "cursor")
	}
	if cfg.IDE.Args != "--new-window" {
		t.Errorf("IDE.Args = %q, want %q", cfg.IDE.Args, "--new-window")
	}
	if cfg.Agent.Command != "claude-code --auto" {
		t.Errorf("Agent.Command = %q, want %q", cfg.Agent.Command, "claude-code --auto")
	}
	if cfg.Display.Theme != "dark" {
		t.Errorf("Display.Theme = %q, want %q", cfg.Display.Theme, "dark")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
}

func TestLoadUserConfig_MissingFile(t *testing.T) {
	cfg, err := LoadUserConfig("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Tracker.Identity != "" {
		t.Errorf("expected empty identity, got %q", cfg.Tracker.Identity)
	}
}

func TestLoadUserConfig_MalformedYAML(t *testing.T) {
	content := `this: is: not: valid: yaml: [}`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadUserConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadUserConfig_VarPreserved(t *testing.T) {
	content := `
tracker:
  api_key: "$MY_CUSTOM_KEY"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadUserConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// $VAR values should be preserved as-is at this stage.
	if cfg.Tracker.APIKey != "$MY_CUSTOM_KEY" {
		t.Errorf("APIKey = %q, want %q", cfg.Tracker.APIKey, "$MY_CUSTOM_KEY")
	}
}
