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
