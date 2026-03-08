package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/oneneural/tempad/internal/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Loads WORKFLOW.md and user config, merges them, and validates the result.
Prints "Configuration valid" on success, or detailed errors on failure.
Exit code 0 = valid, 1 = invalid.`,
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Determine mode for validation purposes.
	mode := "tui"
	if cliFlags.Daemon {
		mode = "daemon"
	}

	// Load workflow.
	workflowPath := cliFlags.WorkflowPath
	workflow, err := config.LoadWorkflow(workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading workflow: %v\n", err)
		os.Exit(1)
	}

	// Load user config.
	userCfg, err := config.LoadUserConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading user config: %v\n", err)
		os.Exit(1)
	}

	// Merge.
	cfg := config.Merge(&cliFlags, userCfg, workflow)

	// Validate.
	if err := config.ValidateForStartup(cfg, mode); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration invalid:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration valid")

	// Print summary.
	fmt.Printf("\n  Tracker:     %s\n", cfg.TrackerKind)
	fmt.Printf("  Project:     %s\n", cfg.TrackerProjectSlug)
	fmt.Printf("  Identity:    %s\n", cfg.TrackerIdentity)
	fmt.Printf("  Mode:        %s\n", mode)
	if mode == "daemon" {
		fmt.Printf("  Agent:       %s\n", cfg.AgentCommand)
		fmt.Printf("  Concurrency: %d\n", cfg.MaxConcurrent)
	} else {
		fmt.Printf("  IDE:         %s\n", cfg.IDECommand)
	}
	fmt.Printf("  Poll:        %dms\n", cfg.PollIntervalMs)

	if workflow.PromptTemplate != "" {
		lines := countLines(workflow.PromptTemplate)
		fmt.Printf("  Prompt:      %d lines\n", lines)
	}

	return nil
}

func countLines(s string) int {
	n := 1
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}
