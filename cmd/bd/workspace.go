package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// workspaceCmd represents the workspace command
var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Generate VS Code workspace for all worktrees",
	Long: `Generate a VS Code workspace file that includes all Git worktrees as folders.
This creates a multi-root workspace where each worktree appears as a separate
folder with its branch name in the title for easy identification.`,
	Run: func(cmd *cobra.Command, args []string) {
		scriptPath := filepath.Join(getScriptDir(), "generate-workspace.sh")

		// Check if script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: workspace generator script not found at %s\n", scriptPath)
			os.Exit(1)
		}

		// Execute the workspace generator
		execCmd := exec.Command(scriptPath)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to generate workspace: %v\n", err)
			os.Exit(1)
		}
	},
}

func getScriptDir() string {
	// Try to find the script in several common locations
	possiblePaths := []string{
		filepath.Join(".", "scripts"),                            // Running from repo root
		filepath.Join("..", "scripts"),                           // Running from bin/
		filepath.Join(filepath.Dir(os.Args[0]), "..", "scripts"), // Relative to executable
		filepath.Join(os.Getenv("HOME"), "gt", "scripts"),        // Home directory
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, "generate-workspace.sh")); err == nil {
			return path
		}
	}

	// Fallback to current directory
	return "scripts"
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
}
