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
	targetPath              string
	languageFlag            string // Language to analyze
	excludePatterns         []string
	largeFileThreshold      int
	longFunctionThreshold   int
	complexityThreshold     int
	outputFormat            string
	outputFile              string // Phase 3: Output file path
	jsonPretty              bool   // Phase 3: Pretty-print JSON
	enableCoverage          bool
	minCoverageThreshold    float64
	coverageTimeout         int
	maxImports              int
	maxExternalDependencies int
	detectCircularDeps      bool

	// Phase 3: Storage and comparison flags
	saveReport     bool
	compareReport  bool
	storageBackend string
	storagePath    string

	// Live status flags
	quietMode  bool
	showStatus bool

	// Parallel processing flags
	workerCount int
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze a codebase for code quality issues",
	Long: `Analyze a codebase and generate a report with code quality insights.

The tool will parse source files in the specified directory, calculate various
metrics, and identify potential issues such as large files, long functions,
and low test coverage.

Language is auto-detected by default. Currently supports Go, Python,
JavaScript/TypeScript, and Kotlin.

Example usage:
  code-review-assistant analyze .
  code-review-assistant analyze /path/to/project
  code-review-assistant analyze . --language javascript
  code-review-assistant analyze . --language kotlin
  code-review-assistant analyze . --large-file-threshold 1000
  code-review-assistant analyze . --exclude "generated/**" --verbose`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	// Language flag
	analyzeCmd.Flags().StringVarP(&languageFlag, "language", "l", "auto", "language to analyze (auto, go, python, javascript, kotlin)")

	// Local flags specific to analyze command
	analyzeCmd.Flags().StringSliceVar(&excludePatterns, "exclude", []string{}, "additional exclude patterns (can be repeated)")
	analyzeCmd.Flags().IntVar(&largeFileThreshold, "large-file-threshold", 0, "override large file threshold (lines)")
	analyzeCmd.Flags().IntVar(&longFunctionThreshold, "long-function-threshold", 0, "override long function threshold (lines)")
	analyzeCmd.Flags().IntVar(&complexityThreshold, "complexity-threshold", 0, "override cyclomatic complexity threshold")
	analyzeCmd.Flags().StringVarP(&outputFormat, "format", "f", "", "output format (console, markdown, json)")
	analyzeCmd.Flags().StringVarP(&outputFile, "output-file", "o", "", "write output to file (default: stdout)")
	analyzeCmd.Flags().BoolVar(&jsonPretty, "json-pretty", true, "pretty-print JSON output")
	analyzeCmd.Flags().BoolVar(&enableCoverage, "enable-coverage", true, "enable test coverage analysis")
	analyzeCmd.Flags().Float64Var(&minCoverageThreshold, "min-coverage-threshold", 0, "minimum coverage threshold percentage (0-100)")
	analyzeCmd.Flags().IntVar(&coverageTimeout, "coverage-timeout", 0, "timeout for test execution per package (seconds)")
	analyzeCmd.Flags().IntVar(&maxImports, "max-imports", 0, "maximum imports per package before flagging")
	analyzeCmd.Flags().IntVar(&maxExternalDependencies, "max-external-dependencies", 0, "maximum external dependencies per package")
	analyzeCmd.Flags().BoolVar(&detectCircularDeps, "detect-circular-deps", true, "detect circular dependencies between packages")

	// Phase 3: Storage and comparison flags
	analyzeCmd.Flags().BoolVar(&saveReport, "save-report", false, "save report to storage for historical tracking")
	analyzeCmd.Flags().BoolVar(&compareReport, "compare", false, "compare with previous report from storage")
	analyzeCmd.Flags().StringVar(&storageBackend, "storage-backend", "file", "storage backend: file or sqlite")
	analyzeCmd.Flags().StringVar(&storagePath, "storage-path", "", "custom storage path (default: ~/.cra)")

	// Live status flags
	analyzeCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "disable live status reporting")
	analyzeCmd.Flags().BoolVar(&showStatus, "show-status", false, "force enable status reporting (even when piped)")

	// Parallel processing flags
	analyzeCmd.Flags().IntVar(&workerCount, "workers", -1, "number of parallel workers (-1=use config, 0=auto, 1=sequential)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Determine and validate target path
	targetPath = getTargetPath(args)
	if err := validatePath(targetPath); err != nil {
		return err
	}

	// Load and configure
	cfg, err := loadAndMergeConfig(cmd)
	if err != nil {
		return err
	}

	// Execute analysis
	return executeAnalysis(cfg, targetPath)
}

// getTargetPath determines the target path from command arguments
func getTargetPath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

// validatePath verifies that the target path exists
func validatePath(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}
	return nil
}

// loadAndMergeConfig loads configuration and applies CLI flag overrides
func loadAndMergeConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg, err := config.LoadConfig(GetConfigFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	overrides := buildOverridesMap(cmd)
	cfg.Merge(overrides)
	return cfg, nil
}

// buildOverridesMap constructs the overrides map from CLI flags
func buildOverridesMap(cmd *cobra.Command) map[string]interface{} {
	overrides := make(map[string]interface{})

	addLanguageOverrides(overrides, cmd)
	addAnalysisOverrides(overrides, cmd)
	addCoverageOverrides(overrides, cmd)
	addDependencyOverrides(overrides, cmd)
	addOutputOverrides(overrides)
	addStorageOverrides(overrides, cmd)

	return overrides
}

// addLanguageOverrides adds language setting overrides
func addLanguageOverrides(overrides map[string]interface{}, cmd *cobra.Command) {
	if cmd.Flags().Changed("language") {
		overrides["language"] = languageFlag
	}
}

// addAnalysisOverrides adds Phase 1 analysis threshold overrides
func addAnalysisOverrides(overrides map[string]interface{}, cmd *cobra.Command) {
	if largeFileThreshold > 0 {
		overrides["large_file_threshold"] = largeFileThreshold
	}
	if longFunctionThreshold > 0 {
		overrides["long_function_threshold"] = longFunctionThreshold
	}
	if complexityThreshold > 0 {
		overrides["complexity_threshold"] = complexityThreshold
	}
	if len(excludePatterns) > 0 {
		overrides["exclude"] = excludePatterns
	}
	if cmd.Flags().Changed("workers") {
		overrides["workers"] = workerCount
	}
}

// addCoverageOverrides adds Phase 2.3 coverage setting overrides
func addCoverageOverrides(overrides map[string]interface{}, cmd *cobra.Command) {
	if cmd.Flags().Changed("enable-coverage") {
		overrides["enable_coverage"] = enableCoverage
	}
	if minCoverageThreshold > 0 {
		overrides["min_coverage_threshold"] = minCoverageThreshold
	}
	if coverageTimeout > 0 {
		overrides["coverage_timeout"] = coverageTimeout
	}
}

// addDependencyOverrides adds Phase 2.4 dependency setting overrides
func addDependencyOverrides(overrides map[string]interface{}, cmd *cobra.Command) {
	if maxImports > 0 {
		overrides["max_imports"] = maxImports
	}
	if maxExternalDependencies > 0 {
		overrides["max_external_dependencies"] = maxExternalDependencies
	}
	if cmd.Flags().Changed("detect-circular-deps") {
		overrides["detect_circular_deps"] = detectCircularDeps
	}
}

// addOutputOverrides adds output configuration overrides
func addOutputOverrides(overrides map[string]interface{}) {
	if outputFormat != "" {
		overrides["format"] = outputFormat
	}
	if outputFile != "" {
		overrides["output_file"] = outputFile
	}
	// Only add json_pretty if explicitly set by user
	if outputFormat == "json" {
		overrides["json_pretty"] = jsonPretty
	}
	if IsVerbose() {
		overrides["verbose"] = true
	}
	if quietMode {
		overrides["quiet"] = true
	}
	if showStatus {
		overrides["show_status"] = true
	}
}

// addStorageOverrides adds Phase 3 storage and comparison setting overrides
func addStorageOverrides(overrides map[string]interface{}, cmd *cobra.Command) {
	// Enable storage if save-report or compare flags are set
	if cmd.Flags().Changed("save-report") || cmd.Flags().Changed("compare") {
		overrides["storage_enabled"] = saveReport || compareReport
	}

	// Enable comparison if --compare flag is set
	if cmd.Flags().Changed("compare") {
		overrides["comparison_enabled"] = compareReport
	}

	if storageBackend != "" {
		overrides["storage_backend"] = storageBackend
	}

	if storagePath != "" {
		overrides["storage_path"] = storagePath
	}
}

// executeAnalysis creates the orchestrator and runs the analysis
func executeAnalysis(cfg *config.Config, targetPath string) error {
	orch, err := orchestrator.New(cfg, targetPath)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}
	defer orch.Close() // Phase 3: Close storage resources

	if err := orch.Run(targetPath); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	return nil
}
