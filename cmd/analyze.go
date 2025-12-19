package cmd

import (
	"fmt"
	"os"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/orchestrator"
	"github.com/spf13/cobra"
)

var (
	// Flags for analyze command
	targetPath            string
	excludePatterns       []string
	largeFileThreshold    int
	longFunctionThreshold int
	outputFormat          string
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze a Go codebase",
	Long: `Analyze a Go codebase and generate a report with code quality insights.

The tool will parse all Go files in the specified directory, calculate various
metrics, and identify potential issues such as large files, long functions,
and low test coverage.

Example usage:
  code-review-assistant analyze .
  code-review-assistant analyze /path/to/project
  code-review-assistant analyze . --large-file-threshold 1000
  code-review-assistant analyze . --exclude "generated/**" --verbose`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	// Local flags specific to analyze command
	analyzeCmd.Flags().StringSliceVar(&excludePatterns, "exclude", []string{}, "additional exclude patterns (can be repeated)")
	analyzeCmd.Flags().IntVar(&largeFileThreshold, "large-file-threshold", 0, "override large file threshold (lines)")
	analyzeCmd.Flags().IntVar(&longFunctionThreshold, "long-function-threshold", 0, "override long function threshold (lines)")
	analyzeCmd.Flags().StringVarP(&outputFormat, "format", "f", "", "output format (console, json, markdown)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Determine target path
	targetPath = "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	// Verify target path exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", targetPath)
	}

	// Load configuration
	cfg, err := config.LoadConfig(GetConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply CLI flag overrides
	overrides := make(map[string]interface{})
	if largeFileThreshold > 0 {
		overrides["large_file_threshold"] = largeFileThreshold
	}
	if longFunctionThreshold > 0 {
		overrides["long_function_threshold"] = longFunctionThreshold
	}
	if outputFormat != "" {
		overrides["format"] = outputFormat
	}
	if IsVerbose() {
		overrides["verbose"] = true
	}
	if len(excludePatterns) > 0 {
		overrides["exclude"] = excludePatterns
	}
	cfg.Merge(overrides)

	// Create and run orchestrator
	orch, err := orchestrator.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	if err := orch.Run(targetPath); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	return nil
}
