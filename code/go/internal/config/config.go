package config

import (
	"os"
	"path/filepath"
)

// ServiceConfig is the fully merged, typed runtime configuration.
// See Spec Section 4.1.4, Section 8.4, Architecture doc Section 7.2.
type ServiceConfig struct {
	// Tracker
	TrackerKind        string `json:"tracker_kind"`
	TrackerEndpoint    string `json:"tracker_endpoint"`
	TrackerAPIKey      string `json:"tracker_api_key"`
	TrackerProjectSlug string `json:"tracker_project_slug"`
	TrackerIdentity    string `json:"tracker_identity"`
	ActiveStates       []string `json:"active_states"`
	TerminalStates     []string `json:"terminal_states"`

	// Polling
	PollIntervalMs int `json:"poll_interval_ms"`

	// Workspace
	WorkspaceRoot string `json:"workspace_root"`

	// Hooks
	AfterCreateHook  string `json:"after_create_hook"`
	BeforeRunHook    string `json:"before_run_hook"`
	AfterRunHook     string `json:"after_run_hook"`
	BeforeRemoveHook string `json:"before_remove_hook"`
	HookTimeoutMs    int    `json:"hook_timeout_ms"`

	// Agent (daemon mode)
	AgentCommand         string         `json:"agent_command"`
	AgentArgs            string         `json:"agent_args"`
	PromptDelivery       string         `json:"prompt_delivery"`
	MaxConcurrent        int            `json:"max_concurrent"`
	MaxConcurrentByState map[string]int `json:"max_concurrent_by_state"`
	MaxTurns             int            `json:"max_turns"`
	MaxRetries           int            `json:"max_retries"`
	MaxRetryBackoffMs    int            `json:"max_retry_backoff_ms"`
	TurnTimeoutMs        int            `json:"turn_timeout_ms"`
	StallTimeoutMs       int            `json:"stall_timeout_ms"`
	ReadTimeoutMs        int            `json:"read_timeout_ms"`

	// IDE (TUI mode)
	IDECommand string `json:"ide_command"`
	IDEArgs    string `json:"ide_args"`

	// Display
	Theme string `json:"theme"`

	// Logging
	LogLevel string `json:"log_level"`
	LogFile  string `json:"log_file"`

	// Server (optional extension)
	ServerPort int `json:"server_port"`

	// Notifications
	NotificationsEnabled bool     `json:"notifications_enabled"`
	NotificationEvents   []string `json:"notification_events"`

	// DryRun skips agent launch in daemon mode; logs the command instead.
	DryRun bool `json:"dry_run"`

	// Internal: path to the workflow file (for hot reload).
	WorkflowPath string `json:"-"`

	// Internal: prompt template body from WORKFLOW.md (for IDE agent context).
	PromptTemplate string `json:"-"`
}

// CLIFlags holds command-line overrides. These have highest precedence.
type CLIFlags struct {
	Daemon       bool
	DryRun       bool
	WorkflowPath string
	Identity     string
	Agent        string
	IDE          string
	Port         int
	LogLevel     string
}

// Defaults returns a ServiceConfig with all built-in defaults applied.
// See Spec Section 8.4.
func Defaults() *ServiceConfig {
	return &ServiceConfig{
		TrackerEndpoint: "https://api.linear.app/graphql",
		ActiveStates:    []string{"Todo", "In Progress"},
		TerminalStates:  []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"},
		PollIntervalMs:  30000,
		WorkspaceRoot:   filepath.Join(os.TempDir(), "tempad_workspaces"),
		HookTimeoutMs:   60000,
		PromptDelivery:  "file",
		MaxConcurrent:   5,
		MaxTurns:        20,
		MaxRetries:      10,
		MaxRetryBackoffMs: 300000,
		TurnTimeoutMs:     3600000,
		StallTimeoutMs:    300000,
		ReadTimeoutMs:     5000,
		IDECommand:        "code",
		Theme:             "auto",
		LogLevel:          "info",
		LogFile:           ExpandHome("~/.tempad/logs/tempad.log"),
		MaxConcurrentByState: make(map[string]int),
		NotificationsEnabled: true,
	}
}
