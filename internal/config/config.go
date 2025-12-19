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
	if val, ok := overrides["large_file_threshold"].(int); ok && val > 0 {
		c.Analysis.LargeFileThreshold = val
	}
	if val, ok := overrides["long_function_threshold"].(int); ok && val > 0 {
		c.Analysis.LongFunctionThreshold = val
	}
	if val, ok := overrides["complexity_threshold"].(int); ok && val > 0 {
		c.Analysis.ComplexityThreshold = val
	}
	if val, ok := overrides["max_parameters"].(int); ok && val > 0 {
		c.Analysis.MaxParameters = val
	}
	if val, ok := overrides["max_nesting_depth"].(int); ok && val > 0 {
		c.Analysis.MaxNestingDepth = val
	}
	if val, ok := overrides["max_return_statements"].(int); ok && val > 0 {
		c.Analysis.MaxReturnStatements = val
	}
	if val, ok := overrides["detect_magic_numbers"].(bool); ok {
		c.Analysis.DetectMagicNumbers = val
	}
	if val, ok := overrides["detect_duplicate_errors"].(bool); ok {
		c.Analysis.DetectDuplicateErrors = val
	}
	if val, ok := overrides["enable_coverage"].(bool); ok {
		c.Analysis.EnableCoverage = val
	}
	if val, ok := overrides["min_coverage_threshold"].(float64); ok && val > 0 {
		c.Analysis.MinCoverageThreshold = val
	}
	if val, ok := overrides["coverage_timeout"].(int); ok && val > 0 {
		c.Analysis.CoverageTimeout = val
	}
	if val, ok := overrides["max_imports"].(int); ok && val > 0 {
		c.Analysis.MaxImports = val
	}
	if val, ok := overrides["max_external_dependencies"].(int); ok && val > 0 {
		c.Analysis.MaxExternalDependencies = val
	}
	if val, ok := overrides["detect_circular_deps"].(bool); ok {
		c.Analysis.DetectCircularDeps = val
	}
	if val, ok := overrides["format"].(string); ok && val != "" {
		c.Output.Format = val
	}
	if val, ok := overrides["verbose"].(bool); ok {
		c.Output.Verbose = val
	}
	if val, ok := overrides["exclude"].([]string); ok && len(val) > 0 {
		c.Analysis.ExcludePatterns = append(c.Analysis.ExcludePatterns, val...)
	}
}
