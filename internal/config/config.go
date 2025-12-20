package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Analysis AnalysisConfig `mapstructure:"analysis"`
	Output   OutputConfig   `mapstructure:"output"`
}

// AnalysisConfig contains settings for code analysis
type AnalysisConfig struct {
	ExcludePatterns        []string `mapstructure:"exclude_patterns"`
	LargeFileThreshold     int      `mapstructure:"large_file_threshold"`
	LongFunctionThreshold  int      `mapstructure:"long_function_threshold"`
	MinCommentRatio        float64  `mapstructure:"min_comment_ratio"`
	ComplexityThreshold    int      `mapstructure:"complexity_threshold"`

	// Anti-pattern detection
	MaxParameters          int      `mapstructure:"max_parameters"`
	MaxNestingDepth        int      `mapstructure:"max_nesting_depth"`
	MaxReturnStatements    int      `mapstructure:"max_return_statements"`
	DetectMagicNumbers     bool     `mapstructure:"detect_magic_numbers"`
	DetectDuplicateErrors  bool     `mapstructure:"detect_duplicate_errors"`

	// Test coverage
	EnableCoverage        bool    `mapstructure:"enable_coverage"`
	MinCoverageThreshold  float64 `mapstructure:"min_coverage_threshold"`
	CoverageTimeout       int     `mapstructure:"coverage_timeout_seconds"`

	// Dependency analysis
	MaxImports              int  `mapstructure:"max_imports"`
	MaxExternalDependencies int  `mapstructure:"max_external_dependencies"`
	DetectCircularDeps      bool `mapstructure:"detect_circular_deps"`
}

// OutputConfig contains settings for report output
type OutputConfig struct {
	Format  string `mapstructure:"format"`
	Verbose bool   `mapstructure:"verbose"`
}

// Default returns a Config with sensible default values
func Default() *Config {
	return &Config{
		Analysis: AnalysisConfig{
			ExcludePatterns: []string{
				"vendor/**",
				"**/*_test.go",
				"**/testdata/**",
				"**/*.pb.go",
			},
			LargeFileThreshold:    500,
			LongFunctionThreshold: 50,
			MinCommentRatio:       0.15,
			ComplexityThreshold:   10,

			// Anti-pattern detection defaults
			MaxParameters:         5,
			MaxNestingDepth:       4,
			MaxReturnStatements:   3,
			DetectMagicNumbers:    true,
			DetectDuplicateErrors: true,

			// Test coverage defaults
			EnableCoverage:       true,
			MinCoverageThreshold: 50.0, // 50%
			CoverageTimeout:      30,   // 30 seconds per package

			// Dependency analysis defaults
			MaxImports:              10,
			MaxExternalDependencies: 10,
			DetectCircularDeps:      true,
		},
		Output: OutputConfig{
			Format:  "console",
			Verbose: false,
		},
	}
}

// LoadConfig loads configuration from file, environment, and defaults
// Priority (highest to lowest): CLI flags > Environment variables > Config file > Defaults
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	defaults := Default()
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
	v.SetDefault("analysis.enable_coverage", defaults.Analysis.EnableCoverage)
	v.SetDefault("analysis.min_coverage_threshold", defaults.Analysis.MinCoverageThreshold)
	v.SetDefault("analysis.coverage_timeout_seconds", defaults.Analysis.CoverageTimeout)
	v.SetDefault("analysis.max_imports", defaults.Analysis.MaxImports)
	v.SetDefault("analysis.max_external_dependencies", defaults.Analysis.MaxExternalDependencies)
	v.SetDefault("analysis.detect_circular_deps", defaults.Analysis.DetectCircularDeps)
	v.SetDefault("output.format", defaults.Output.Format)
	v.SetDefault("output.verbose", defaults.Output.Verbose)

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

// Merge merges CLI flag overrides into the config
func (c *Config) Merge(overrides map[string]interface{}) {
	c.mergeAnalysisThresholds(overrides)
	c.mergeAntiPatternSettings(overrides)
	c.mergeCoverageSettings(overrides)
	c.mergeDependencySettings(overrides)
	c.mergeOutputSettings(overrides)
	c.mergeExcludePatterns(overrides)
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

// mergeOutputSettings merges output configuration settings
func (c *Config) mergeOutputSettings(overrides map[string]interface{}) {
	mergeStringIfNonEmpty(&c.Output.Format, overrides, "format")
	mergeBool(&c.Output.Verbose, overrides, "verbose")
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
