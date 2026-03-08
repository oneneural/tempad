package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/oneneural/tempad/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default user configuration",
	Long: `Creates ~/.tempad/config.yaml with commented default values.
If the file already exists, it is not overwritten.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := config.DefaultUserConfigPath()
	configDir := filepath.Dir(configPath)

	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", configDir, err)
	}

	// Check if the file already exists.
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists: %s\n", configPath)
		fmt.Println("To reset, delete the file and run `tempad init` again.")
		return nil
	}

	// Write the default config template.
	template := config.DefaultUserConfigTemplate()
	if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created default config: %s\n", configPath)
	fmt.Println("Edit this file to set your tracker identity and preferences.")
	return nil
}
