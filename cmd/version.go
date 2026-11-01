package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information - set via ldflags during build
	Version   = "dev"
	CommitSHA = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, commit SHA, and build date of the Senator CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Senator CLI\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Commit:     %s\n", CommitSHA)
		fmt.Printf("  Built:      %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
