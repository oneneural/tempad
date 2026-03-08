package config

import (
	"os"
	"testing"
)

func TestMerge_DefaultsApplied(t *testing.T) {
	cfg := Merge(nil, nil, nil)

	if cfg.TrackerEndpoint != "https://api.linear.app/graphql" {
		t.Errorf("TrackerEndpoint = %q, want default", cfg.TrackerEndpoint)
	}
	if cfg.PollIntervalMs != 30000 {
		t.Errorf("PollIntervalMs = %d, want 30000", cfg.PollIntervalMs)
	}
	if cfg.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %d, want 5", cfg.MaxConcurrent)
	}
	if cfg.MaxTurns != 20 {
		t.Errorf("MaxTurns = %d, want 20", cfg.MaxTurns)
	}
	if cfg.MaxRetries != 10 {
		t.Errorf("MaxRetries = %d, want 10", cfg.MaxRetries)
	}
	if cfg.PromptDelivery != "file" {
		t.Errorf("PromptDelivery = %q, want %q", cfg.PromptDelivery, "file")
	}
	if cfg.IDECommand != "code" {
		t.Errorf("IDECommand = %q, want %q", cfg.IDECommand, "code")
	}
	if cfg.Theme != "auto" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "auto")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if len(cfg.ActiveStates) != 2 {
		t.Errorf("ActiveStates length = %d, want 2", len(cfg.ActiveStates))
	}
	if len(cfg.TerminalStates) != 5 {
		t.Errorf("TerminalStates length = %d, want 5", len(cfg.TerminalStates))
	}
}

func TestMerge_CLIOverridesUserConfig(t *testing.T) {
	user := &UserConfig{}
	user.Tracker.Identity = "user@example.com"

	cli := &CLIFlags{
		Identity: "cli@example.com",
	}

	cfg := Merge(cli, user, nil)
	if cfg.TrackerIdentity != "cli@example.com" {
		t.Errorf("TrackerIdentity = %q, want CLI override %q",
			cfg.TrackerIdentity, "cli@example.com")
	}
}

func TestMerge_UserOverridesRepoForPersonalFields(t *testing.T) {
	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"agent": map[string]any{
				"command": "repo-agent",
			},
		},
	}
	user := &UserConfig{}
	user.Agent.Command = "user-agent"

	cfg := Merge(nil, user, workflow)
	if cfg.AgentCommand != "user-agent" {
		t.Errorf("AgentCommand = %q, want user override %q",
			cfg.AgentCommand, "user-agent")
	}
}

func TestMerge_RepoWinsForTeamSettings(t *testing.T) {
	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"hooks": map[string]any{
				"after_create": "git clone repo .",
			},
			"agent": map[string]any{
				"max_concurrent": 3,
			},
		},
	}
	// User config cannot override hooks or concurrency.
	user := &UserConfig{}

	cfg := Merge(nil, user, workflow)
	if cfg.AfterCreateHook != "git clone repo ." {
		t.Errorf("AfterCreateHook = %q, want repo value", cfg.AfterCreateHook)
	}
	if cfg.MaxConcurrent != 3 {
		t.Errorf("MaxConcurrent = %d, want repo value 3", cfg.MaxConcurrent)
	}
}

func TestMerge_EnvVarResolution(t *testing.T) {
	os.Setenv("TEMPAD_TEST_API_KEY", "resolved_key")
	defer os.Unsetenv("TEMPAD_TEST_API_KEY")

	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"tracker": map[string]any{
				"api_key": "$TEMPAD_TEST_API_KEY",
			},
		},
	}

	cfg := Merge(nil, nil, workflow)
	if cfg.TrackerAPIKey != "resolved_key" {
		t.Errorf("TrackerAPIKey = %q, want %q", cfg.TrackerAPIKey, "resolved_key")
	}
}

func TestMerge_EmptyEnvVarResolvesToEmpty(t *testing.T) {
	os.Unsetenv("TEMPAD_NONEXISTENT_KEY")

	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"tracker": map[string]any{
				"api_key": "$TEMPAD_NONEXISTENT_KEY",
			},
		},
	}

	cfg := Merge(nil, nil, workflow)
	if cfg.TrackerAPIKey != "" {
		t.Errorf("TrackerAPIKey = %q, want empty for unset env var", cfg.TrackerAPIKey)
	}
}

func TestMerge_CommaSeparatedStates(t *testing.T) {
	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"tracker": map[string]any{
				"active_states": "Todo, In Progress, Review",
			},
		},
	}

	cfg := Merge(nil, nil, workflow)
	if len(cfg.ActiveStates) != 3 {
		t.Errorf("ActiveStates length = %d, want 3", len(cfg.ActiveStates))
	}
	if cfg.ActiveStates[0] != "Todo" {
		t.Errorf("ActiveStates[0] = %q, want %q", cfg.ActiveStates[0], "Todo")
	}
	if cfg.ActiveStates[1] != "In Progress" {
		t.Errorf("ActiveStates[1] = %q, want %q", cfg.ActiveStates[1], "In Progress")
	}
}

func TestMerge_WorkflowOverridesDefaults(t *testing.T) {
	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"polling": map[string]any{
				"interval_ms": 15000,
			},
		},
	}

	cfg := Merge(nil, nil, workflow)
	if cfg.PollIntervalMs != 15000 {
		t.Errorf("PollIntervalMs = %d, want 15000", cfg.PollIntervalMs)
	}
}

func TestMerge_StringIntegerParsing(t *testing.T) {
	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"polling": map[string]any{
				"interval_ms": "20000",
			},
		},
	}

	cfg := Merge(nil, nil, workflow)
	if cfg.PollIntervalMs != 20000 {
		t.Errorf("PollIntervalMs = %d, want 20000", cfg.PollIntervalMs)
	}
}

func TestMerge_MaxConcurrentByState(t *testing.T) {
	workflow := &WorkflowDefinition{
		Config: map[string]any{
			"agent": map[string]any{
				"max_concurrent_by_state": map[string]any{
					"In Progress": 2,
					"Todo":        3,
				},
			},
		},
	}

	cfg := Merge(nil, nil, workflow)
	if cfg.MaxConcurrentByState["In Progress"] != 2 {
		t.Errorf("MaxConcurrentByState[In Progress] = %d, want 2",
			cfg.MaxConcurrentByState["In Progress"])
	}
	if cfg.MaxConcurrentByState["Todo"] != 3 {
		t.Errorf("MaxConcurrentByState[Todo] = %d, want 3",
			cfg.MaxConcurrentByState["Todo"])
	}
}
