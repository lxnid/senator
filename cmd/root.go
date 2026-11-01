package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "senator",
	Short: "A universal MCP host & multi-agent code review CLI",
	Long: `Senator is a CLI tool that orchestrates AI code review tasks using the
Model Context Protocol (MCP). It convenes a "Senate" of AI agents to debate
code changes before they are committed.

The tool connects to any standard MCP Server (GitHub, Postgres, Filesystem)
and uses multiple LLM models to provide comprehensive code reviews with
different perspectives (security, logic, style).`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.senator.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load .env file if it exists
	_ = godotenv.Load()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Explicitly check for local config files
		if _, err := os.Stat("./senator.yaml"); err == nil {
			viper.SetConfigFile("./senator.yaml")
		} else if _, err := os.Stat("./configs/senator.yaml"); err == nil {
			viper.SetConfigFile("./configs/senator.yaml")
		} else {
			// Fallback to home directory .senator.yaml
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".senator")
		}
	}

	// Read environment variables
	viper.SetEnvPrefix("SENATOR")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
