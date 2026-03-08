package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/workspace"
)

var cleanCmd = &cobra.Command{
	Use:   "clean [identifier]",
	Short: "Clean up workspaces for terminal issues",
	Long: `Without arguments: removes workspaces for all issues in terminal states.
With an identifier: removes the workspace for that specific issue.

Requires a valid WORKFLOW.md for workspace root configuration.
Without an identifier, also requires tracker configuration to query terminal issues.`,
	RunE: runClean,
}

func runClean(cmd *cobra.Command, args []string) error {
	// Load config to get workspace root.
	workflowPath := cliFlags.WorkflowPath
	workflow, err := config.LoadWorkflow(workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading workflow: %v\n", err)
		os.Exit(1)
	}

	userCfg, err := config.LoadUserConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading user config: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Merge(&cliFlags, userCfg, workflow)

	// Determine workspace root.
	wsRoot := cfg.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = filepath.Join(".", ".tempad-workspaces")
	}

	mgr, err := workspace.NewManager(wsRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing workspace manager: %v\n", err)
		os.Exit(1)
	}

	if len(args) > 0 {
		// Clean specific workspace.
		identifier := args[0]
		if err := mgr.CleanForIssue(identifier); err != nil {
			fmt.Fprintf(os.Stderr, "Error cleaning workspace for %s: %v\n", identifier, err)
			os.Exit(1)
		}
		fmt.Printf("Cleaned workspace for %s\n", identifier)
		return nil
	}

	// Clean all terminal workspaces — requires tracker client.
	fmt.Println("Cleaning all terminal workspaces requires a tracker connection.")
	fmt.Println("Use 'tempad clean <identifier>' to clean a specific workspace.")
	fmt.Printf("Workspace root: %s\n", mgr.Root())
	return nil
}
