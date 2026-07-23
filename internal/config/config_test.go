package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	t.Run("has sensible defaults", func(t *testing.T) {
		assert.NotNil(t, cfg, "config should not be nil")
		assert.NotNil(t, cfg.Analysis, "analysis config should not be nil")
		assert.NotNil(t, cfg.Output, "output config should not be nil")

		// Phase 1 defaults
		assert.Equal(t, 500, cfg.Analysis.LargeFileThreshold, "default large file threshold")
		assert.Equal(t, 50, cfg.Analysis.LongFunctionThreshold, "default long function threshold")
		assert.Equal(t, 0.15, cfg.Analysis.MinCommentRatio, "default min comment ratio")

		// Phase 2.1: Complexity
		assert.Equal(t, 10, cfg.Analysis.ComplexityThreshold, "default complexity threshold")

		// Phase 2.2: Anti-patterns
		assert.Equal(t, 5, cfg.Analysis.MaxParameters, "default max parameters")
		assert.Equal(t, 4, cfg.Analysis.MaxNestingDepth, "default max nesting depth")
		assert.Equal(t, 3, cfg.Analysis.MaxReturnStatements, "default max return statements")
		assert.True(t, cfg.Analysis.DetectMagicNumbers, "default detect magic numbers")
		assert.True(t, cfg.Analysis.DetectDuplicateErrors, "default detect duplicate errors")

		// Phase 2.3: Coverage
		assert.True(t, cfg.Analysis.EnableCoverage, "default enable coverage")
		assert.Equal(t, 50.0, cfg.Analysis.MinCoverageThreshold, "default min coverage threshold")
		assert.Equal(t, 30, cfg.Analysis.CoverageTimeout, "default coverage timeout")

		// Phase 2.4: Dependencies
		assert.Equal(t, 10, cfg.Analysis.MaxImports, "default max imports")
		assert.Equal(t, 10, cfg.Analysis.MaxExternalDependencies, "default max external dependencies")
		assert.True(t, cfg.Analysis.DetectCircularDeps, "default detect circular deps")

		// Output
		assert.Equal(t, "console", cfg.Output.Format, "default output format")
		assert.False(t, cfg.Output.Verbose, "default verbose")
	})

	t.Run("has default exclude patterns", func(t *testing.T) {
		assert.Len(t, cfg.Analysis.ExcludePatterns, 4, "should have 4 default exclude patterns")
		assert.Contains(t, cfg.Analysis.ExcludePatterns, "vendor/**")
		assert.Contains(t, cfg.Analysis.ExcludePatterns, "**/*_test.go")
		assert.Contains(t, cfg.Analysis.ExcludePatterns, "**/testdata/**")
		assert.Contains(t, cfg.Analysis.ExcludePatterns, "**/*.pb.go")
	})
}

func TestLoadConfig_NoConfigFile(t *testing.T) {
	t.Run("explicit non-existent path returns error", func(t *testing.T) {
		// Load config with explicit non-existent path
		cfg, err := LoadConfig("nonexistent-config.yaml")

		// When explicit path is provided but doesn't exist, viper returns an error
		assert.Error(t, err, "should error when explicit config file not found")
		assert.Nil(t, cfg, "should return nil config on error")
	})

	t.Run("no path uses defaults when no config found", func(t *testing.T) {
		// Load config with empty path (searches standard locations)
		// This won't find a config file in the test directory, but should use defaults
		cfg, err := LoadConfig("")

		require.NoError(t, err, "should not error when searching standard locations")
		require.NotNil(t, cfg, "should return config with defaults")

		// Verify defaults are loaded
		assert.Equal(t, 500, cfg.Analysis.LargeFileThreshold, "should use default large file threshold")
		assert.Equal(t, "console", cfg.Output.Format, "should use default format")
	})
}

func TestLoadConfig_ValidConfigFile(t *testing.T) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configContent := `analysis:
  exclude_patterns:
    - "custom/**"
  large_file_threshold: 1000
  long_function_threshold: 100
  min_comment_ratio: 0.20
  complexity_threshold: 15
  max_parameters: 7
  max_nesting_depth: 5
  max_return_statements: 4
  detect_magic_numbers: false
  detect_duplicate_errors: false
  detect_non_null_assertions: false
  detect_run_blocking: false
  enable_coverage: false
  min_coverage_threshold: 80.0
  coverage_timeout_seconds: 60
  max_imports: 20
  max_external_dependencies: 15
  detect_circular_deps: false

output:
  format: "json"
  verbose: true
`
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err, "should load config without error")
	require.NotNil(t, cfg, "config should not be nil")

	// Verify all custom values are loaded
	t.Run("loads Phase 1 settings", func(t *testing.T) {
		assert.Equal(t, 1000, cfg.Analysis.LargeFileThreshold)
		assert.Equal(t, 100, cfg.Analysis.LongFunctionThreshold)
		assert.Equal(t, 0.20, cfg.Analysis.MinCommentRatio)
	})

	t.Run("loads Phase 2.1 settings", func(t *testing.T) {
		assert.Equal(t, 15, cfg.Analysis.ComplexityThreshold)
	})

	t.Run("loads Phase 2.2 settings", func(t *testing.T) {
		assert.Equal(t, 7, cfg.Analysis.MaxParameters)
		assert.Equal(t, 5, cfg.Analysis.MaxNestingDepth)
		assert.Equal(t, 4, cfg.Analysis.MaxReturnStatements)
		assert.False(t, cfg.Analysis.DetectMagicNumbers)
		assert.False(t, cfg.Analysis.DetectDuplicateErrors)
		assert.False(t, cfg.Analysis.DetectNonNullAssertions)
		assert.False(t, cfg.Analysis.DetectRunBlocking)
	})

	t.Run("loads Phase 2.3 settings", func(t *testing.T) {
		assert.False(t, cfg.Analysis.EnableCoverage)
		assert.Equal(t, 80.0, cfg.Analysis.MinCoverageThreshold)
		assert.Equal(t, 60, cfg.Analysis.CoverageTimeout)
	})

	t.Run("loads Phase 2.4 settings", func(t *testing.T) {
		assert.Equal(t, 20, cfg.Analysis.MaxImports)
		assert.Equal(t, 15, cfg.Analysis.MaxExternalDependencies)
		assert.False(t, cfg.Analysis.DetectCircularDeps)
	})

	t.Run("loads output settings", func(t *testing.T) {
		assert.Equal(t, "json", cfg.Output.Format)
		assert.True(t, cfg.Output.Verbose)
	})

	t.Run("loads custom exclude patterns", func(t *testing.T) {
		assert.Contains(t, cfg.Analysis.ExcludePatterns, "custom/**")
	})
}

func TestLoadConfig_InvalidConfigFile(t *testing.T) {
	// Create a temporary invalid config file
	tempDir, err := os.MkdirTemp("", "config-test-invalid-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	invalidContent := `analysis:
  large_file_threshold: "not a number"  # Invalid: should be int
  min_comment_ratio: true  # Invalid: should be float
`
	configPath := filepath.Join(tempDir, "invalid-config.yaml")
	err = os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Try to load the invalid config
	_, err = LoadConfig(configPath)
	assert.Error(t, err, "should return error for invalid config")
	assert.Contains(t, err.Error(), "error", "error message should indicate parsing error")
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	// Create a config file with only some values
	tempDir, err := os.MkdirTemp("", "config-test-partial-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	partialContent := `analysis:
  large_file_threshold: 750
  complexity_threshold: 12
`
	configPath := filepath.Join(tempDir, "partial-config.yaml")
	err = os.WriteFile(configPath, []byte(partialContent), 0644)
	require.NoError(t, err)

	// Load the partial config
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err, "should load partial config without error")

	// Verify specified values are loaded
	assert.Equal(t, 750, cfg.Analysis.LargeFileThreshold, "should use config value")
	assert.Equal(t, 12, cfg.Analysis.ComplexityThreshold, "should use config value")

	// Verify unspecified values use defaults
	assert.Equal(t, 50, cfg.Analysis.LongFunctionThreshold, "should use default for unspecified")
	assert.Equal(t, "console", cfg.Output.Format, "should use default for unspecified")
	assert.Equal(t, 5, cfg.Analysis.MaxParameters, "should use default for unspecified")
}

func TestMerge(t *testing.T) {
	testCases := []struct {
		name           string
		overrides      map[string]interface{}
		validateConfig func(t *testing.T, cfg *Config)
	}{
		{
			name: "merge Phase 1 overrides",
			overrides: map[string]interface{}{
				"large_file_threshold":    1000,
				"long_function_threshold": 75,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1000, cfg.Analysis.LargeFileThreshold)
				assert.Equal(t, 75, cfg.Analysis.LongFunctionThreshold)
				// Other values should remain default
				assert.Equal(t, 10, cfg.Analysis.ComplexityThreshold)
			},
		},
		{
			name: "merge Phase 2.1 complexity override",
			overrides: map[string]interface{}{
				"complexity_threshold": 15,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 15, cfg.Analysis.ComplexityThreshold)
				// Other values should remain default
				assert.Equal(t, 500, cfg.Analysis.LargeFileThreshold)
			},
		},
		{
			name: "merge Phase 2.2 anti-pattern overrides",
			overrides: map[string]interface{}{
				"max_parameters":             7,
				"max_nesting_depth":          5,
				"max_return_statements":      4,
				"detect_magic_numbers":       false,
				"detect_duplicate_errors":    false,
				"detect_non_null_assertions": false,
				"detect_run_blocking":        false,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 7, cfg.Analysis.MaxParameters)
				assert.Equal(t, 5, cfg.Analysis.MaxNestingDepth)
				assert.Equal(t, 4, cfg.Analysis.MaxReturnStatements)
				assert.False(t, cfg.Analysis.DetectMagicNumbers)
				assert.False(t, cfg.Analysis.DetectDuplicateErrors)
				assert.False(t, cfg.Analysis.DetectNonNullAssertions)
				assert.False(t, cfg.Analysis.DetectRunBlocking)
			},
		},
		{
			name: "merge Phase 2.3 coverage overrides",
			overrides: map[string]interface{}{
				"enable_coverage":        false,
				"min_coverage_threshold": 80.5,
				"coverage_timeout":       60,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.Analysis.EnableCoverage)
				assert.Equal(t, 80.5, cfg.Analysis.MinCoverageThreshold)
				assert.Equal(t, 60, cfg.Analysis.CoverageTimeout)
			},
		},
		{
			name: "merge Phase 2.4 dependency overrides",
			overrides: map[string]interface{}{
				"max_imports":               20,
				"max_external_dependencies": 15,
				"detect_circular_deps":      false,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 20, cfg.Analysis.MaxImports)
				assert.Equal(t, 15, cfg.Analysis.MaxExternalDependencies)
				assert.False(t, cfg.Analysis.DetectCircularDeps)
			},
		},
		{
			name: "merge output overrides",
			overrides: map[string]interface{}{
				"format":  "json",
				"verbose": true,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "json", cfg.Output.Format)
				assert.True(t, cfg.Output.Verbose)
			},
		},
		{
			name: "merge exclude patterns",
			overrides: map[string]interface{}{
				"exclude": []string{"custom/**", "generated/**"},
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Contains(t, cfg.Analysis.ExcludePatterns, "custom/**")
				assert.Contains(t, cfg.Analysis.ExcludePatterns, "generated/**")
				// Original patterns should still be there
				assert.Contains(t, cfg.Analysis.ExcludePatterns, "vendor/**")
			},
		},
		{
			name: "merge all overrides at once",
			overrides: map[string]interface{}{
				"large_file_threshold":    1000,
				"long_function_threshold": 75,
				"complexity_threshold":    15,
				"max_parameters":          7,
				"enable_coverage":         false,
				"min_coverage_threshold":  80.0,
				"max_imports":             20,
				"detect_circular_deps":    false,
				"format":                  "json",
				"verbose":                 true,
				"exclude":                 []string{"custom/**"},
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1000, cfg.Analysis.LargeFileThreshold)
				assert.Equal(t, 75, cfg.Analysis.LongFunctionThreshold)
				assert.Equal(t, 15, cfg.Analysis.ComplexityThreshold)
				assert.Equal(t, 7, cfg.Analysis.MaxParameters)
				assert.False(t, cfg.Analysis.EnableCoverage)
				assert.Equal(t, 80.0, cfg.Analysis.MinCoverageThreshold)
				assert.Equal(t, 20, cfg.Analysis.MaxImports)
				assert.False(t, cfg.Analysis.DetectCircularDeps)
				assert.Equal(t, "json", cfg.Output.Format)
				assert.True(t, cfg.Output.Verbose)
				assert.Contains(t, cfg.Analysis.ExcludePatterns, "custom/**")
			},
		},
		{
			name: "zero values are not merged for int fields",
			overrides: map[string]interface{}{
				"large_file_threshold": 0,
				"complexity_threshold": 0,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				// Zero values should not override defaults
				assert.Equal(t, 500, cfg.Analysis.LargeFileThreshold)
				assert.Equal(t, 10, cfg.Analysis.ComplexityThreshold)
			},
		},
		{
			name: "boolean overrides work with false values",
			overrides: map[string]interface{}{
				"enable_coverage":      false,
				"detect_circular_deps": false,
				"detect_magic_numbers": false,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				// False values should be applied (not treated as zero values)
				assert.False(t, cfg.Analysis.EnableCoverage)
				assert.False(t, cfg.Analysis.DetectCircularDeps)
				assert.False(t, cfg.Analysis.DetectMagicNumbers)
			},
		},
		{
			name:      "empty overrides map doesn't change config",
			overrides: map[string]interface{}{},
			validateConfig: func(t *testing.T, cfg *Config) {
				// All values should remain default
				assert.Equal(t, 500, cfg.Analysis.LargeFileThreshold)
				assert.Equal(t, 50, cfg.Analysis.LongFunctionThreshold)
				assert.Equal(t, 10, cfg.Analysis.ComplexityThreshold)
				assert.Equal(t, "console", cfg.Output.Format)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Start with default config
			cfg := Default()

			// Apply overrides
			cfg.Merge(tc.overrides)

			// Validate the result
			tc.validateConfig(t, cfg)
		})
	}
}

func TestMerge_TypeSafety(t *testing.T) {
	t.Run("wrong type in overrides is ignored", func(t *testing.T) {
		cfg := Default()

		// Try to override with wrong types
		overrides := map[string]interface{}{
			"large_file_threshold": "not a number", // Should be int
			"format":               123,            // Should be string
			"verbose":              "true",         // Should be bool
		}

		cfg.Merge(overrides)

		// Original values should be unchanged
		assert.Equal(t, 500, cfg.Analysis.LargeFileThreshold)
		assert.Equal(t, "console", cfg.Output.Format)
		assert.False(t, cfg.Output.Verbose)
	})
}
