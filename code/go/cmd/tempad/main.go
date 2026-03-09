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
	"github.com/oneneural/tempad/internal/logging"
	"github.com/oneneural/tempad/internal/notify"
	"github.com/oneneural/tempad/internal/orchestrator"
	"github.com/oneneural/tempad/internal/server"
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

In daemon mode (--daemon), it shows the TUI board with an embedded orchestrator
that auto-dispatches coding agents, with streaming log output.

In headless mode (--headless), it runs the orchestrator without TUI for CI/servers.`,
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cliFlags.Headless {
			return runHeadless()
		}
		if cliFlags.Daemon {
			return runTUIWithOrchestrator()
		}
		if cliFlags.DryRun {
			fmt.Fprintf(os.Stderr, "Warning: --dry-run only applies to daemon/headless mode\n")
		}
		if cliFlags.Port > 0 {
			fmt.Fprintf(os.Stderr, "Warning: --port is only used in daemon/headless mode\n")
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

	// Resolve tracker identity for claim operations.
	ctx := context.Background()
	if err := client.ResolveIdentity(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to resolve tracker identity: %v\n", err)
	}

	// Startup terminal workspace cleanup.
	if terminalIssues, fetchErr := client.FetchIssuesByStates(ctx, cfg.TerminalStates); fetchErr == nil {
		if cleaned, cleanErr := ws.CleanTerminal(terminalIssues); cleanErr == nil && cleaned > 0 {
			fmt.Fprintf(os.Stderr, "Cleaned %d terminal workspaces\n", cleaned)
		}
	}

	// Create notifier.
	notifier := notify.New(notify.Config{
		Enabled: cfg.NotificationsEnabled,
		Events:  cfg.NotificationEvents,
	}, slog.Default())

	// Create and run Bubble Tea program.
	model := tui.NewModel(ctx, cfg, client, ws, nil, notifier)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func runTUIWithOrchestrator() error {
	// Load and merge config.
	cfg, _, err := config.Load(&cliFlags)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// Validate for daemon mode (requires agent.command).
	if err := config.ValidateForStartup(cfg, "daemon"); err != nil {
		return err
	}

	// Set up structured logger (file-based for daemon, keeps TUI clean).
	logger := logging.Setup(logging.Config{
		Level: cfg.LogLevel,
		File:  cfg.LogFile,
		Mode:  "daemon",
	})

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

	// Resolve tracker identity.
	ctx := context.Background()
	if err := client.ResolveIdentity(ctx); err != nil {
		logger.Warn("failed to resolve tracker identity", "error", err)
	}

	// Startup terminal workspace cleanup.
	if terminalIssues, fetchErr := client.FetchIssuesByStates(ctx, cfg.TerminalStates); fetchErr == nil {
		if cleaned, cleanErr := ws.CleanTerminal(terminalIssues); cleanErr == nil && cleaned > 0 {
			logger.Info("cleaned terminal workspaces", "count", cleaned)
		}
	}

	// Set up signal handling.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create notifier.
	notifier := notify.New(notify.Config{
		Enabled: cfg.NotificationsEnabled,
		Events:  cfg.NotificationEvents,
	}, logger)

	// Create orchestrator.
	orch := orchestrator.New(cfg, client, ws, logger, notifier)

	// Start HTTP server if --port is specified.
	if cfg.ServerPort > 0 {
		srv, err := server.New(cfg.ServerPort, orch, logger)
		if err != nil {
			return fmt.Errorf("HTTP server: %w", err)
		}
		go srv.Serve(ctx)
		logger.Info("HTTP server started", "addr", srv.Addr())
	}

	// Start orchestrator in background goroutine.
	go func() {
		if runErr := orch.Run(ctx); runErr != nil {
			logger.Error("orchestrator error", "error", runErr)
		}
	}()

	// Create TUI model with embedded orchestrator.
	model := tui.NewModelWithOrchestrator(ctx, cfg, client, ws, nil, notifier, orch)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Stop orchestrator when TUI exits.
	stop()
	return nil
}

func runHeadless() error {
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
	logger := logging.Setup(logging.Config{
		Level: cfg.LogLevel,
		File:  cfg.LogFile,
		Mode:  "daemon",
	})

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

	// Resolve tracker identity for "assigned to me" queries.
	ctx := context.Background()
	if err := client.ResolveIdentity(ctx); err != nil {
		logger.Warn("failed to resolve tracker identity, assigned-to-me queries disabled", "error", err)
	}

	// Startup terminal workspace cleanup.
	if terminalIssues, fetchErr := client.FetchIssuesByStates(ctx, cfg.TerminalStates); fetchErr == nil {
		if cleaned, cleanErr := ws.CleanTerminal(terminalIssues); cleanErr == nil && cleaned > 0 {
			logger.Info("cleaned terminal workspaces", "count", cleaned)
		}
	}

	// Set up signal handling: SIGINT/SIGTERM → cancel context.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create notifier.
	notifier := notify.New(notify.Config{
		Enabled: cfg.NotificationsEnabled,
		Events:  cfg.NotificationEvents,
	}, logger)

	// Create orchestrator.
	orch := orchestrator.New(cfg, client, ws, logger, notifier)

	// Start HTTP server if --port is specified.
	if cfg.ServerPort > 0 {
		srv, err := server.New(cfg.ServerPort, orch, logger)
		if err != nil {
			return fmt.Errorf("HTTP server: %w", err)
		}
		go srv.Serve(ctx)
		logger.Info("HTTP server started", "addr", srv.Addr())
	}

	logger.Info("headless daemon starting",
		"poll_interval_ms", cfg.PollIntervalMs,
		"max_concurrent", cfg.MaxConcurrent,
		"agent_command", cfg.AgentCommand,
		"project", cfg.TrackerProjectSlug,
		"dry_run", cfg.DryRun,
	)

	return orch.Run(ctx)
}

func init() {
	// Global flags.
	rootCmd.PersistentFlags().BoolVar(&cliFlags.Daemon, "daemon", false,
		"Run TUI with embedded orchestrator (auto-dispatch agents)")
	rootCmd.PersistentFlags().BoolVar(&cliFlags.Headless, "headless", false,
		"Run headless daemon mode for CI/servers (no TUI)")
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
	rootCmd.PersistentFlags().BoolVar(&cliFlags.DryRun, "dry-run", false,
		"Run full pipeline but skip agent launch (daemon/headless mode only)")

	// Subcommands.
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(setupCmd)
}
