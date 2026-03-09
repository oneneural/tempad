package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oneneural/tempad/internal/config"
)

var setupFlags struct {
	apiKey   string
	identity string
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for first-time configuration",
	Long: `Walks through all configuration fields interactively and writes
~/.tempad/config.yaml on completion.

Re-runnable: if a config already exists, current values are pre-filled
and only changed fields are overwritten.

For CI/scripting, pass flags directly:
  tempad setup --api-key=lin_api_xxx --identity=you@example.com`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().StringVar(&setupFlags.apiKey, "api-key", "", "Linear API key (non-interactive)")
	setupCmd.Flags().StringVar(&setupFlags.identity, "identity", "", "Linear email identity (non-interactive)")
}

func runSetup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	configPath := config.DefaultUserConfigPath()

	// Load existing config for pre-fill.
	existing, _ := config.LoadUserConfig("")

	interactive := setupFlags.apiKey == "" && setupFlags.identity == ""

	if interactive {
		fmt.Println("TEMPAD Setup Wizard")
		fmt.Println(strings.Repeat("─", 40))
		fmt.Println()
	}

	// --- Step 1: Linear API Key ---
	apiKey := setupFlags.apiKey
	if apiKey == "" {
		apiKey = promptWithDefault(scanner, "Linear API key", existing.Tracker.APIKey,
			"Get yours at https://linear.app/settings/api")
	}

	if interactive {
		fmt.Print("  Validating API key... ")
	}
	if err := config.ValidateAPIKey(ctx, resolveForValidation(apiKey)); err != nil {
		if interactive {
			fmt.Println("✗")
		}
		return fmt.Errorf("API key validation failed: %w", err)
	}
	if interactive {
		fmt.Println("✓")
		fmt.Println()
	}

	// --- Step 2: Tracker Identity ---
	identity := setupFlags.identity
	if identity == "" {
		identity = promptWithDefault(scanner, "Linear email", existing.Tracker.Identity,
			"The email associated with your Linear account")
	}

	if interactive {
		fmt.Print("  Validating identity... ")
	}
	if err := config.ValidateIdentity(ctx, resolveForValidation(apiKey), identity); err != nil {
		if interactive {
			fmt.Println("✗")
		}
		return fmt.Errorf("identity validation failed: %w", err)
	}
	if interactive {
		fmt.Println("✓")
		fmt.Println()
	}

	// --- Step 3: IDE Selection ---
	ideCommand := existing.IDE.Command
	if interactive {
		ideCommand = promptIDESelection(scanner, existing.IDE.Command)
		fmt.Println()
	}

	// --- Step 4: Agent Command (optional) ---
	agentCommand := existing.Agent.Command
	if interactive {
		agentCommand = promptAgentSelection(scanner, existing.Agent.Command)
		fmt.Println()
	}

	// --- Step 5: Workspace Root ---
	defaultRoot := config.DefaultWorkspaceRoot()
	if existing.Tracker.Identity != "" {
		// Keep whatever the user had if it was a re-run; workspace root is
		// in WORKFLOW.md or defaults, not typically in user config.
	}
	workspaceRoot := defaultRoot
	if interactive {
		workspaceRoot = promptWithDefault(scanner, "Workspace root", "", defaultRoot)
		if workspaceRoot == "" {
			workspaceRoot = defaultRoot
		}

		// Ensure directory exists.
		if err := config.EnsureDirectory(workspaceRoot); err != nil {
			return fmt.Errorf("failed to create workspace directory: %w", err)
		}
		fmt.Printf("  Workspace directory ready: %s\n", config.ExpandHome(workspaceRoot))
		fmt.Println()
	}

	// --- Step 6: Review & Confirm ---
	result := &config.UserConfig{
		Tracker: config.UserTrackerConfig{
			Identity: identity,
			APIKey:   apiKey,
		},
		IDE: config.UserIDEConfig{
			Command: ideCommand,
			Args:    existing.IDE.Args,
		},
		Agent: config.UserAgentConfig{
			Command:        agentCommand,
			Args:           existing.Agent.Args,
			PromptDelivery: existing.Agent.PromptDelivery,
		},
		Display: config.UserDisplayConfig{
			Theme: orDefault(existing.Display.Theme, "auto"),
		},
		Logging: config.UserLoggingConfig{
			Level: orDefault(existing.Logging.Level, "info"),
			File:  existing.Logging.File,
		},
		Notifications: existing.Notifications,
	}

	if interactive {
		fmt.Println("Configuration Summary")
		fmt.Println(strings.Repeat("─", 40))
		fmt.Printf("  API Key:        %s\n", maskAPIKey(apiKey))
		fmt.Printf("  Identity:       %s\n", identity)
		fmt.Printf("  IDE:            %s\n", orDefault(ideCommand, "(not set)"))
		fmt.Printf("  Agent:          %s\n", orDefault(agentCommand, "(not set)"))
		fmt.Printf("  Workspace Root: %s\n", workspaceRoot)
		fmt.Println()

		if !promptConfirm(scanner, fmt.Sprintf("Write config to %s?", configPath)) {
			fmt.Println("Setup cancelled.")
			return nil
		}
	}

	// Write config.
	if err := config.WriteUserConfig(configPath, result); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Ensure ~/.tempad/ directory structure exists.
	tempadDir := config.ExpandHome("~/.tempad")
	for _, sub := range []string{"workspaces", "logs"} {
		if err := config.EnsureDirectory(tempadDir + "/" + sub); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", sub, err)
		}
	}

	fmt.Printf("Config written to %s\n", configPath)
	if interactive {
		fmt.Println("Run `tempad` to start using TEMPAD.")
	}
	return nil
}

// promptWithDefault prompts for input with an optional default value.
func promptWithDefault(scanner *bufio.Scanner, label, currentValue, hint string) string {
	if hint != "" {
		fmt.Printf("  %s\n", hint)
	}
	if currentValue != "" {
		fmt.Printf("  %s [%s]: ", label, maskIfKey(label, currentValue))
	} else {
		fmt.Printf("  %s: ", label)
	}

	if !scanner.Scan() {
		return currentValue
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return currentValue
	}
	return input
}

// promptIDESelection presents a list of known IDEs and validates the choice.
func promptIDESelection(scanner *bufio.Scanner, current string) string {
	fmt.Println("  IDE selection:")
	for i, ide := range config.KnownIDEs {
		avail := "  "
		if _, err := config.LookupBinary(ide.Command); err == nil {
			avail = "✓ "
		}
		marker := "  "
		if ide.Command == current {
			marker = "* "
		}
		fmt.Printf("    %s%s%d) %s (%s)\n", marker, avail, i+1, ide.Name, ide.Command)
	}
	fmt.Printf("    %d) Other (enter custom command)\n", len(config.KnownIDEs)+1)

	defaultHint := ""
	if current != "" {
		defaultHint = fmt.Sprintf(" [%s]", current)
	}
	fmt.Printf("  Choose IDE%s: ", defaultHint)

	if !scanner.Scan() {
		return current
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return current
	}

	// Try as number.
	if n, err := strconv.Atoi(input); err == nil {
		if n >= 1 && n <= len(config.KnownIDEs) {
			chosen := config.KnownIDEs[n-1].Command
			if _, err := config.LookupBinary(chosen); err != nil {
				fmt.Printf("  Warning: %s\n", err)
			} else {
				fmt.Printf("  Found: %s\n", chosen)
			}
			return chosen
		}
		if n == len(config.KnownIDEs)+1 {
			fmt.Print("  Custom IDE command: ")
			if scanner.Scan() {
				custom := strings.TrimSpace(scanner.Text())
				if custom != "" {
					if _, err := config.LookupBinary(custom); err != nil {
						fmt.Printf("  Warning: %s\n", err)
					} else {
						fmt.Printf("  Found: %s\n", custom)
					}
					return custom
				}
			}
			return current
		}
	}

	// Treat as direct command name.
	if _, err := config.LookupBinary(input); err != nil {
		fmt.Printf("  Warning: %s\n", err)
	} else {
		fmt.Printf("  Found: %s\n", input)
	}
	return input
}

// promptAgentSelection presents a list of known agents.
func promptAgentSelection(scanner *bufio.Scanner, current string) string {
	fmt.Println("  Agent command (optional, for daemon mode):")
	for i, agent := range config.KnownAgents {
		avail := "  "
		if _, err := config.LookupBinary(agent.Command); err == nil {
			avail = "✓ "
		}
		marker := "  "
		if agent.Command == current {
			marker = "* "
		}
		fmt.Printf("    %s%s%d) %s (%s)\n", marker, avail, i+1, agent.Name, agent.Command)
	}
	fmt.Printf("    %d) Other (enter custom command)\n", len(config.KnownAgents)+1)
	fmt.Printf("    %d) Skip (no agent)\n", len(config.KnownAgents)+2)

	defaultHint := ""
	if current != "" {
		defaultHint = fmt.Sprintf(" [%s]", current)
	}
	fmt.Printf("  Choose agent%s: ", defaultHint)

	if !scanner.Scan() {
		return current
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return current
	}

	if n, err := strconv.Atoi(input); err == nil {
		if n >= 1 && n <= len(config.KnownAgents) {
			chosen := config.KnownAgents[n-1].Command
			if _, err := config.LookupBinary(chosen); err != nil {
				fmt.Printf("  Warning: %s\n", err)
			} else {
				fmt.Printf("  Found: %s\n", chosen)
			}
			return chosen
		}
		if n == len(config.KnownAgents)+1 {
			fmt.Print("  Custom agent command: ")
			if scanner.Scan() {
				custom := strings.TrimSpace(scanner.Text())
				if custom != "" {
					if _, err := config.LookupBinary(custom); err != nil {
						fmt.Printf("  Warning: %s\n", err)
					} else {
						fmt.Printf("  Found: %s\n", custom)
					}
					return custom
				}
			}
			return current
		}
		if n == len(config.KnownAgents)+2 {
			return ""
		}
	}

	// Treat as direct command name.
	if input != "" {
		if _, err := config.LookupBinary(input); err != nil {
			fmt.Printf("  Warning: %s\n", err)
		} else {
			fmt.Printf("  Found: %s\n", input)
		}
	}
	return input
}

// promptConfirm asks a yes/no question with yes as default.
func promptConfirm(scanner *bufio.Scanner, question string) bool {
	fmt.Printf("  %s [Y/n]: ", question)
	if !scanner.Scan() {
		return true
	}
	input := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return input == "" || input == "y" || input == "yes"
}

// resolveForValidation resolves $VAR references for validation purposes.
func resolveForValidation(value string) string {
	return config.ResolveEnvVar(value)
}

// maskAPIKey masks all but the first 8 characters of an API key.
func maskAPIKey(key string) string {
	if strings.HasPrefix(key, "$") {
		return key // env var reference, show as-is
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:8] + strings.Repeat("*", len(key)-8)
}

// maskIfKey masks the value if the label suggests it's a sensitive field.
func maskIfKey(label, value string) string {
	lower := strings.ToLower(label)
	if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
		return maskAPIKey(value)
	}
	return value
}

// orDefault returns value if non-empty, otherwise returns fallback.
func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
