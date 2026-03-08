// Package main is the entry point for the TEMPAD CLI.
// TEMPAD (Temporal Execution & Management Poll-Agent Dispatcher) bridges
// issue trackers and developer workflows.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/orchestrator"
	"github.com/oneneural/tempad/internal/tracker/linear"
	"github.com/oneneural/tempad/internal/tui"
	"github.com/oneneural/tempad/internal/workspace"
)

// Version is set at build time via ldflags.
var Version = "dev"

// cliFlags holds the parsed CLI flags.
var cliFlags config.CLIFlags

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "tempad",
	Short: "T.E.M.P.A.D. — Temporal Execution & Management Poll-Agent Dispatcher",
	Long: `T.E.M.P.A.D. bridges issue trackers and developer workflows.

In TUI mode (default), it shows a live task board from your issue tracker,
lets you pick tasks, and opens your IDE.

In daemon mode (--daemon), it auto-dispatches coding agents headlessly.`,
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cliFlags.Daemon {
			return runDaemon()
		}
		return runTUI()
	},
}

func runTUI() error {
	// Load and merge config.
	cfg, _, err := config.Load(&cliFlags)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Validate for TUI mode.
	if err := config.ValidateForStartup(cfg, "tui"); err != nil {
		return err
	}

	// Create tracker client.
	client := linear.NewLinearClient(linear.Config{
		Endpoint:       cfg.TrackerEndpoint,
		APIKey:         cfg.TrackerAPIKey,
		ProjectSlug:    cfg.TrackerProjectSlug,
		Identity:       cfg.TrackerIdentity,
		ActiveStates:   cfg.ActiveStates,
		TerminalStates: cfg.TerminalStates,
	})

	// Create workspace manager.
	ws, err := workspace.NewManager(cfg.WorkspaceRoot)
	if err != nil {
		return fmt.Errorf("workspace manager: %w", err)
	}

	// Startup terminal workspace cleanup.
	ctx := context.Background()
	if terminalIssues, fetchErr := client.FetchIssuesByStates(ctx, cfg.TerminalStates); fetchErr == nil {
		if cleaned, cleanErr := ws.CleanTerminal(terminalIssues); cleanErr == nil && cleaned > 0 {
			fmt.Fprintf(os.Stderr, "Cleaned %d terminal workspaces\n", cleaned)
		}
	}

	// Create and run Bubble Tea program.
	model := tui.NewModel(ctx, cfg, client, ws, nil)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func runDaemon() error {
	// Load and merge config.
	cfg, _, err := config.Load(&cliFlags)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Validate for daemon mode (requires agent.command).
	if err := config.ValidateForStartup(cfg, "daemon"); err != nil {
		return err
	}

	// Set up structured logger.
	logLevel := parseLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Create tracker client.
	client := linear.NewLinearClient(linear.Config{
		Endpoint:       cfg.TrackerEndpoint,
		APIKey:         cfg.TrackerAPIKey,
		ProjectSlug:    cfg.TrackerProjectSlug,
		Identity:       cfg.TrackerIdentity,
		ActiveStates:   cfg.ActiveStates,
		TerminalStates: cfg.TerminalStates,
	})

	// Create workspace manager.
	ws, err := workspace.NewManager(cfg.WorkspaceRoot)
	if err != nil {
		return fmt.Errorf("workspace manager: %w", err)
	}

	// Startup terminal workspace cleanup.
	ctx := context.Background()
	if terminalIssues, fetchErr := client.FetchIssuesByStates(ctx, cfg.TerminalStates); fetchErr == nil {
		if cleaned, cleanErr := ws.CleanTerminal(terminalIssues); cleanErr == nil && cleaned > 0 {
			logger.Info("cleaned terminal workspaces", "count", cleaned)
		}
	}

	// Set up signal handling: SIGINT/SIGTERM → cancel context.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create and run orchestrator.
	orch := orchestrator.New(cfg, client, ws, logger)

	logger.Info("daemon starting",
		"poll_interval_ms", cfg.PollIntervalMs,
		"max_concurrent", cfg.MaxConcurrent,
		"agent_command", cfg.AgentCommand,
		"project", cfg.TrackerProjectSlug,
	)

	return orch.Run(ctx)
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func init() {
	// Global flags.
	rootCmd.PersistentFlags().BoolVar(&cliFlags.Daemon, "daemon", false,
		"Run in headless daemon mode")
	rootCmd.PersistentFlags().StringVar(&cliFlags.WorkflowPath, "workflow", "",
		"Path to WORKFLOW.md (default: ./WORKFLOW.md)")
	rootCmd.PersistentFlags().StringVar(&cliFlags.Identity, "identity", "",
		"Override tracker identity")
	rootCmd.PersistentFlags().StringVar(&cliFlags.Agent, "agent", "",
		"Override agent command")
	rootCmd.PersistentFlags().StringVar(&cliFlags.IDE, "ide", "",
		"Override IDE command")
	rootCmd.PersistentFlags().IntVar(&cliFlags.Port, "port", 0,
		"HTTP server port (0 = disabled)")
	rootCmd.PersistentFlags().StringVar(&cliFlags.LogLevel, "log-level", "",
		"Log level (debug, info, warn, error)")

	// Subcommands.
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(cleanCmd)
}
