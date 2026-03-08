package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig represents personal preferences from ~/.tempad/config.yaml.
// See Spec Section 4.1.3, Section 7.
type UserConfig struct {
	Tracker  UserTrackerConfig  `yaml:"tracker"`
	IDE      UserIDEConfig      `yaml:"ide"`
	Agent    UserAgentConfig    `yaml:"agent"`
	Display  UserDisplayConfig  `yaml:"display"`
	Logging  UserLoggingConfig  `yaml:"logging"`
}

// UserTrackerConfig holds tracker-related personal preferences.
type UserTrackerConfig struct {
	Identity string `yaml:"identity"` // email or user ID
	APIKey   string `yaml:"api_key"`  // literal or $VAR
}

// UserIDEConfig holds IDE preferences.
type UserIDEConfig struct {
	Command string `yaml:"command"` // default: "code"
	Args    string `yaml:"args"`    // extra args
}

// UserAgentConfig holds agent preferences.
type UserAgentConfig struct {
	Command string `yaml:"command"` // default agent for daemon mode
	Args    string `yaml:"args"`    // extra args
}

// UserDisplayConfig holds display preferences.
type UserDisplayConfig struct {
	Theme string `yaml:"theme"` // "auto", "dark", "light"
}

// UserLoggingConfig holds logging preferences.
type UserLoggingConfig struct {
	Level string `yaml:"level"` // "debug", "info", "warn", "error"
	File  string `yaml:"file"`  // log file path
}

// DefaultUserConfigPath returns the default path to the user config file.
func DefaultUserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".tempad", "config.yaml")
	}
	return filepath.Join(home, ".tempad", "config.yaml")
}

// LoadUserConfig loads and parses the user configuration file.
// See Spec Section 7.1, 7.2, 7.3.
//
// If the file doesn't exist, returns an empty config without error
// (the file will be created by `tempad init`).
//
// $VAR_NAME values are preserved as-is at this stage — resolution
// happens during the merge step.
func LoadUserConfig(path string) (*UserConfig, error) {
	if path == "" {
		path = DefaultUserConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserConfig{}, nil
		}
		return nil, fmt.Errorf("cannot read user config %s: %w", path, err)
	}

	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("user_config_parse_error: failed to parse %s: %w", path, err)
	}

	return &cfg, nil
}

// DefaultUserConfigTemplate returns the commented YAML template written
// by `tempad init`. Matches Spec Appendix B / Section 7.2.
func DefaultUserConfigTemplate() string {
	return `# TEMPAD User Configuration
# This file stores personal preferences. Team settings live in WORKFLOW.md.

# Tracker identity — who you are in Linear
tracker:
  identity: ""              # your Linear email or user ID (required)
  api_key: "$LINEAR_API_KEY" # override repo-level key if needed

# IDE preferences (TUI mode)
ide:
  command: "code"            # code, cursor, zed, idea, webstorm, etc.
  # args: "--new-window"     # extra arguments (optional)

# Default agent for daemon mode
agent:
  command: ""                # e.g. "claude-code --auto", "codex", "opencode"
  # args: null               # extra arguments (optional)

# Display preferences
display:
  theme: "auto"              # auto, dark, light

# Logging
logging:
  level: "info"              # debug, info, warn, error
  # file: "~/.tempad/logs/tempad.log"
`
}
