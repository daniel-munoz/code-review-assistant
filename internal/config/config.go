package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/daniel-munoz/code-review-assistant/internal/constants"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration.
//
// Configuration can be loaded from multiple sources with the following priority
// (highest to lowest):
//  1. CLI flags (set via Merge())
//  2. Environment variables (prefixed with CRA_)
//  3. Configuration file (YAML format)
//  4. Default values (from Default())
//
// This layered approach allows flexible configuration for different environments
// while maintaining sensible defaults.
type Config struct {
	Language   string           `mapstructure:"language"` // Language to analyze: "auto", "go", "python", etc.
	Analysis   AnalysisConfig   `mapstructure:"analysis"`
	Output     OutputConfig     `mapstructure:"output"`
	Storage    StorageConfig    `mapstructure:"storage"`    // Phase 3: Persistent storage
	Comparison ComparisonConfig `mapstructure:"comparison"` // Phase 3: Historical comparison
}

// AnalysisConfig contains all settings that control code analysis behavior.
//
// This includes:
//   - File and function size thresholds
//   - Complexity thresholds
//   - Anti-pattern detection settings
//   - Test coverage configuration
//   - Dependency analysis settings
//
// All threshold values can be overridden via CLI flags, environment variables,
// or configuration files.
type AnalysisConfig struct {
	ExcludePatterns       []string `mapstructure:"exclude_patterns"`
	LargeFileThreshold    int      `mapstructure:"large_file_threshold"`
	LongFunctionThreshold int      `mapstructure:"long_function_threshold"`
	MinCommentRatio       float64  `mapstructure:"min_comment_ratio"`
	ComplexityThreshold   int      `mapstructure:"complexity_threshold"`

	// Anti-pattern detection
	MaxParameters           int  `mapstructure:"max_parameters"`
	MaxNestingDepth         int  `mapstructure:"max_nesting_depth"`
	MaxReturnStatements     int  `mapstructure:"max_return_statements"`
	DetectMagicNumbers      bool `mapstructure:"detect_magic_numbers"`
	DetectDuplicateErrors   bool `mapstructure:"detect_duplicate_errors"`
	DetectNonNullAssertions bool `mapstructure:"detect_non_null_assertions"` // Kotlin: flag !! usage
	DetectRunBlocking       bool `mapstructure:"detect_run_blocking"`        // Kotlin: flag runBlocking outside main

	// Test coverage
	EnableCoverage       bool    `mapstructure:"enable_coverage"`
	MinCoverageThreshold float64 `mapstructure:"min_coverage_threshold"`
	CoverageTimeout      int     `mapstructure:"coverage_timeout_seconds"`

	// Dependency analysis
	MaxImports              int  `mapstructure:"max_imports"`
	MaxExternalDependencies int  `mapstructure:"max_external_dependencies"`
	DetectCircularDeps      bool `mapstructure:"detect_circular_deps"`

	// Parallel processing
	Workers int `mapstructure:"workers"` // 0=auto (runtime.NumCPU), 1=sequential, N=parallel
}

// OutputConfig contains settings that control report formatting and output.
//
// Format determines the output style: console, markdown, json.
// Verbose enables detailed per-file and per-package reporting.
// OutputFile specifies a file path for output (empty means stdout).
// JSONPretty controls JSON formatting (pretty vs compact).
// QuietMode disables live status reporting.
// ShowStatus forces status reporting even when output is piped.
type OutputConfig struct {
	Format     string `mapstructure:"format"`
	Verbose    bool   `mapstructure:"verbose"`
	OutputFile string `mapstructure:"output_file"` // Phase 3: File output path
	JSONPretty bool   `mapstructure:"json_pretty"` // Phase 3: Pretty-print JSON
	QuietMode  bool   `mapstructure:"quiet"`       // Disable live status reporting
	ShowStatus bool   `mapstructure:"show_status"` // Force enable status reporting
}

// StorageConfig contains settings for persistent storage of analysis reports.
//
// Phase 3: Enables saving reports for historical tracking and comparison.
type StorageConfig struct {
	Enabled     bool   `mapstructure:"enabled"`      // Enable storage
	Backend     string `mapstructure:"backend"`      // "file" or "sqlite"
	Path        string `mapstructure:"path"`         // Custom storage path (empty = default)
	AutoSave    bool   `mapstructure:"auto_save"`    // Automatically save after analysis
	ProjectMode bool   `mapstructure:"project_mode"` // Use ./.cra instead of ~/.cra
}

// ComparisonConfig contains settings for historical comparison.
//
// Phase 3: Enables comparison with previous analysis runs.
type ComparisonConfig struct {
	Enabled         bool    `mapstructure:"enabled"`          // Enable comparison
	AutoCompare     bool    `mapstructure:"auto_compare"`     // Auto-compare with latest
	StableThreshold float64 `mapstructure:"stable_threshold"` // % change for "stable" (default: 5.0)
}

// Default returns a Config with sensible default values for all settings.
//
// These defaults are based on industry best practices and common code quality
// standards:
//   - Large file threshold: 500 lines
//   - Long function threshold: 50 lines
//   - Cyclomatic complexity threshold: 10
//   - Minimum comment ratio: 15%
//   - Test coverage threshold: 50%
//
// All anti-pattern detectors are enabled by default. Files matching common
// patterns (vendor/, testdata/, *_test.go, *.pb.go) are excluded from analysis.
func Default() *Config {
	return &Config{
		Language: "auto", // Auto-detect language by default
		Analysis: AnalysisConfig{
			ExcludePatterns:       constants.DefaultExcludePatterns,
			LargeFileThreshold:    constants.DefaultLargeFileThreshold,
			LongFunctionThreshold: constants.DefaultLongFunctionThreshold,
			MinCommentRatio:       constants.DefaultMinCommentRatio,
			ComplexityThreshold:   constants.DefaultComplexityThreshold,

			// Anti-pattern detection defaults
			MaxParameters:           constants.DefaultMaxParameters,
			MaxNestingDepth:         constants.DefaultMaxNestingDepth,
			MaxReturnStatements:     constants.DefaultMaxReturnStatements,
			DetectMagicNumbers:      constants.DefaultDetectMagicNumbers,
			DetectDuplicateErrors:   constants.DefaultDetectDuplicateErrors,
			DetectNonNullAssertions: constants.DefaultDetectNonNullAssertions,
			DetectRunBlocking:       constants.DefaultDetectRunBlocking,

			// Test coverage defaults
			EnableCoverage:       constants.DefaultEnableCoverage,
			MinCoverageThreshold: constants.DefaultMinCoverageThreshold,
			CoverageTimeout:      constants.DefaultCoverageTimeout,

			// Dependency analysis defaults
			MaxImports:              constants.DefaultMaxImports,
			MaxExternalDependencies: constants.DefaultMaxExternalDependencies,
			DetectCircularDeps:      constants.DefaultDetectCircularDeps,

			// Parallel processing defaults
			Workers: 0, // Auto-detect CPU count
		},
		Output: OutputConfig{
			Format:     "console",
			Verbose:    false,
			OutputFile: "",    // Phase 3: Default to stdout
			JSONPretty: true,  // Phase 3: Pretty-print by default
			QuietMode:  false, // Enable status by default
			ShowStatus: false, // Auto-detect TTY by default
		},
		Storage: StorageConfig{
			Enabled:     false, // Phase 3: Opt-in
			Backend:     "file",
			Path:        "", // Use default (~/.cra or ./.cra)
			AutoSave:    false,
			ProjectMode: false, // Use ~/.cra by default
		},
		Comparison: ComparisonConfig{
			Enabled:         false, // Phase 3: Opt-in
			AutoCompare:     false,
			StableThreshold: 5.0, // 5% change threshold
		},
	}
}

// LoadConfig loads configuration from multiple sources and merges them.
//
// Configuration is loaded in the following order (later sources override earlier):
//  1. Default values (from Default())
//  2. Configuration file (YAML format)
//  3. Environment variables (prefixed with CRA_)
//
// Configuration File Search:
// If configPath is provided, only that file is loaded. Otherwise, the loader
// searches for "config.yaml" in:
//   - Current directory
//   - User's home directory (~/.cra/)
//
// Environment Variables:
// All settings can be overridden via environment variables with the CRA_ prefix.
// For example: CRA_ANALYSIS_COMPLEXITY_THRESHOLD=15
//
// Returns the merged configuration or an error if the config file exists but
// cannot be read. Missing config files are acceptable and will use defaults.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	defaults := Default()
	v.SetDefault("language", defaults.Language)
	v.SetDefault("analysis.exclude_patterns", defaults.Analysis.ExcludePatterns)
	v.SetDefault("analysis.large_file_threshold", defaults.Analysis.LargeFileThreshold)
	v.SetDefault("analysis.long_function_threshold", defaults.Analysis.LongFunctionThreshold)
	v.SetDefault("analysis.min_comment_ratio", defaults.Analysis.MinCommentRatio)
	v.SetDefault("analysis.complexity_threshold", defaults.Analysis.ComplexityThreshold)
	v.SetDefault("analysis.max_parameters", defaults.Analysis.MaxParameters)
	v.SetDefault("analysis.max_nesting_depth", defaults.Analysis.MaxNestingDepth)
	v.SetDefault("analysis.max_return_statements", defaults.Analysis.MaxReturnStatements)
	v.SetDefault("analysis.detect_magic_numbers", defaults.Analysis.DetectMagicNumbers)
	v.SetDefault("analysis.detect_duplicate_errors", defaults.Analysis.DetectDuplicateErrors)
	v.SetDefault("analysis.detect_non_null_assertions", defaults.Analysis.DetectNonNullAssertions)
	v.SetDefault("analysis.detect_run_blocking", defaults.Analysis.DetectRunBlocking)
	v.SetDefault("analysis.enable_coverage", defaults.Analysis.EnableCoverage)
	v.SetDefault("analysis.min_coverage_threshold", defaults.Analysis.MinCoverageThreshold)
	v.SetDefault("analysis.coverage_timeout_seconds", defaults.Analysis.CoverageTimeout)
	v.SetDefault("analysis.max_imports", defaults.Analysis.MaxImports)
	v.SetDefault("analysis.max_external_dependencies", defaults.Analysis.MaxExternalDependencies)
	v.SetDefault("analysis.detect_circular_deps", defaults.Analysis.DetectCircularDeps)
	v.SetDefault("analysis.workers", defaults.Analysis.Workers)
	v.SetDefault("output.format", defaults.Output.Format)
	v.SetDefault("output.verbose", defaults.Output.Verbose)
	v.SetDefault("output.output_file", defaults.Output.OutputFile)
	v.SetDefault("output.json_pretty", defaults.Output.JSONPretty)
	v.SetDefault("output.quiet", defaults.Output.QuietMode)
	v.SetDefault("output.show_status", defaults.Output.ShowStatus)
	v.SetDefault("storage.enabled", defaults.Storage.Enabled)
	v.SetDefault("storage.backend", defaults.Storage.Backend)
	v.SetDefault("storage.path", defaults.Storage.Path)
	v.SetDefault("storage.auto_save", defaults.Storage.AutoSave)
	v.SetDefault("storage.project_mode", defaults.Storage.ProjectMode)
	v.SetDefault("comparison.enabled", defaults.Comparison.Enabled)
	v.SetDefault("comparison.auto_compare", defaults.Comparison.AutoCompare)
	v.SetDefault("comparison.stable_threshold", defaults.Comparison.StableThreshold)

	// Environment variables
	v.SetEnvPrefix("CRA") // Code Review Assistant
	v.AutomaticEnv()

	// Config file
	if configPath != "" {
		// Explicit config path provided
		v.SetConfigFile(configPath)
	} else {
		// Search for config in standard locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Current directory
		v.AddConfigPath(".")

		// User config directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".cra"))
		}
	}

	// Read config file (optional - don't fail if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file found but error reading it
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found - use defaults, this is OK
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}

// Merge merges CLI flag overrides into the configuration.
//
// This method applies command-line flag overrides with the highest priority,
// allowing users to override config file and environment settings on a
// per-invocation basis.
//
// The overrides map should contain flag names as keys and their values.
// Only non-zero/non-empty values are applied to avoid overwriting valid
// config with flag defaults.
//
// Example:
//
//	overrides := map[string]interface{}{
//	    "complexity_threshold": 15,
//	    "verbose": true,
//	}
//	config.Merge(overrides)
func (c *Config) Merge(overrides map[string]interface{}) {
	c.mergeLanguageSettings(overrides)
	c.mergeAnalysisThresholds(overrides)
	c.mergeAntiPatternSettings(overrides)
	c.mergeCoverageSettings(overrides)
	c.mergeDependencySettings(overrides)
	c.mergeWorkerSettings(overrides)
	c.mergeOutputSettings(overrides)
	c.mergeStorageSettings(overrides)
	c.mergeComparisonSettings(overrides)
	c.mergeExcludePatterns(overrides)
}

// mergeLanguageSettings merges language settings
func (c *Config) mergeLanguageSettings(overrides map[string]interface{}) {
	mergeStringIfNonEmpty(&c.Language, overrides, "language")
}

// mergeAnalysisThresholds merges Phase 1 analysis threshold overrides
func (c *Config) mergeAnalysisThresholds(overrides map[string]interface{}) {
	mergeIntIfPositive(&c.Analysis.LargeFileThreshold, overrides, "large_file_threshold")
	mergeIntIfPositive(&c.Analysis.LongFunctionThreshold, overrides, "long_function_threshold")
	mergeIntIfPositive(&c.Analysis.ComplexityThreshold, overrides, "complexity_threshold")
}

// mergeAntiPatternSettings merges Phase 2.2 anti-pattern detection settings
func (c *Config) mergeAntiPatternSettings(overrides map[string]interface{}) {
	mergeIntIfPositive(&c.Analysis.MaxParameters, overrides, "max_parameters")
	mergeIntIfPositive(&c.Analysis.MaxNestingDepth, overrides, "max_nesting_depth")
	mergeIntIfPositive(&c.Analysis.MaxReturnStatements, overrides, "max_return_statements")
	mergeBool(&c.Analysis.DetectMagicNumbers, overrides, "detect_magic_numbers")
	mergeBool(&c.Analysis.DetectDuplicateErrors, overrides, "detect_duplicate_errors")
	mergeBool(&c.Analysis.DetectNonNullAssertions, overrides, "detect_non_null_assertions")
	mergeBool(&c.Analysis.DetectRunBlocking, overrides, "detect_run_blocking")
}

// mergeCoverageSettings merges Phase 2.3 test coverage settings
func (c *Config) mergeCoverageSettings(overrides map[string]interface{}) {
	mergeBool(&c.Analysis.EnableCoverage, overrides, "enable_coverage")
	mergeFloatIfPositive(&c.Analysis.MinCoverageThreshold, overrides, "min_coverage_threshold")
	mergeIntIfPositive(&c.Analysis.CoverageTimeout, overrides, "coverage_timeout")
}

// mergeDependencySettings merges Phase 2.4 dependency analysis settings
func (c *Config) mergeDependencySettings(overrides map[string]interface{}) {
	mergeIntIfPositive(&c.Analysis.MaxImports, overrides, "max_imports")
	mergeIntIfPositive(&c.Analysis.MaxExternalDependencies, overrides, "max_external_dependencies")
	mergeBool(&c.Analysis.DetectCircularDeps, overrides, "detect_circular_deps")
}

// mergeWorkerSettings merges parallel processing settings
func (c *Config) mergeWorkerSettings(overrides map[string]interface{}) {
	if val, ok := overrides["workers"].(int); ok && val >= 0 {
		c.Analysis.Workers = val
	}
}

// mergeOutputSettings merges output configuration settings
func (c *Config) mergeOutputSettings(overrides map[string]interface{}) {
	mergeStringIfNonEmpty(&c.Output.Format, overrides, "format")
	mergeBool(&c.Output.Verbose, overrides, "verbose")
	mergeStringIfNonEmpty(&c.Output.OutputFile, overrides, "output_file")
	mergeBool(&c.Output.JSONPretty, overrides, "json_pretty")
	mergeBool(&c.Output.QuietMode, overrides, "quiet")
	mergeBool(&c.Output.ShowStatus, overrides, "show_status")
}

// mergeStorageSettings merges Phase 3 storage settings
func (c *Config) mergeStorageSettings(overrides map[string]interface{}) {
	mergeBool(&c.Storage.Enabled, overrides, "storage_enabled")
	mergeStringIfNonEmpty(&c.Storage.Backend, overrides, "storage_backend")
	mergeStringIfNonEmpty(&c.Storage.Path, overrides, "storage_path")
	mergeBool(&c.Storage.AutoSave, overrides, "auto_save")
	mergeBool(&c.Storage.ProjectMode, overrides, "project_mode")
}

// mergeComparisonSettings merges Phase 3 comparison settings
func (c *Config) mergeComparisonSettings(overrides map[string]interface{}) {
	mergeBool(&c.Comparison.Enabled, overrides, "comparison_enabled")
	mergeBool(&c.Comparison.AutoCompare, overrides, "auto_compare")
	mergeFloatIfPositive(&c.Comparison.StableThreshold, overrides, "stable_threshold")
}

// mergeExcludePatterns appends additional exclude patterns if provided
func (c *Config) mergeExcludePatterns(overrides map[string]interface{}) {
	if val, ok := overrides["exclude"].([]string); ok && len(val) > 0 {
		c.Analysis.ExcludePatterns = append(c.Analysis.ExcludePatterns, val...)
	}
}

// mergeIntIfPositive updates target with override value if it exists and is positive
func mergeIntIfPositive(target *int, overrides map[string]interface{}, key string) {
	if val, ok := overrides[key].(int); ok && val > 0 {
		*target = val
	}
}

// mergeBool updates target with override value if it exists
func mergeBool(target *bool, overrides map[string]interface{}, key string) {
	if val, ok := overrides[key].(bool); ok {
		*target = val
	}
}

// mergeFloatIfPositive updates target with override value if it exists and is positive
func mergeFloatIfPositive(target *float64, overrides map[string]interface{}, key string) {
	if val, ok := overrides[key].(float64); ok && val > 0 {
		*target = val
	}
}

// mergeStringIfNonEmpty updates target with override value if it exists and is non-empty
func mergeStringIfNonEmpty(target *string, overrides map[string]interface{}, key string) {
	if val, ok := overrides[key].(string); ok && val != "" {
		*target = val
	}
}
