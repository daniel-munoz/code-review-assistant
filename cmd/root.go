package cmd

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "code-review-assistant",
	Short: "Analyze Go codebases for code quality insights",
	Long: `Code Review Assistant is a CLI tool that analyzes Go codebases
to provide actionable insights about code quality, complexity, and maintainability.

It helps developers identify technical debt, track metrics, and maintain
high code quality standards.`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.yaml or ~/.cra/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output with detailed per-file metrics")
}

// GetConfigFile returns the configured config file path
func GetConfigFile() string {
	return cfgFile
}

// SetConfigFile sets the config file path (primarily for testing)
func SetConfigFile(path string) {
	cfgFile = path
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}
