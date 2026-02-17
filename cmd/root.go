package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version    = "dev"
	commitHash = "none"
	buildDate  = "unknown"
)

// SetVersionInfo sets the build version info from ldflags.
func SetVersionInfo(v, c, d string) {
	version = v
	commitHash = c
	buildDate = d
}

var rootCmd = &cobra.Command{
	Use:   "codetok",
	Short: "Track token usage across coding CLI tools",
	Long: `codetok aggregates and visualizes token usage from multiple
AI coding CLI tools including Claude Code, OpenCode, Codex CLI,
Kimi CLI, and Cursor.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("codetok %s (commit: %s, built: %s)\n", version, commitHash, buildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
