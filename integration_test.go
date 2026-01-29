package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullWorkflow tests the complete analysis workflow
// from parsing through reporting with all output formats
func TestIntegration_FullWorkflow(t *testing.T) {
	// Use testdata/sample directory which has sample Go files
	projectPath := "testdata/sample"

	// Verify test directory exists
	_, err := os.Stat(projectPath)
	require.NoError(t, err, "testdata/sample directory should exist")

	t.Run("console output format", func(t *testing.T) {
		cfg := config.Default()
		cfg.Output.Format = "console"
		cfg.Output.Verbose = true

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		assert.NoError(t, err, "should complete analysis successfully")
	})

	t.Run("markdown output format", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "report.md")

		cfg := config.Default()
		cfg.Output.Format = "markdown"
		cfg.Output.OutputFile = tempFile

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		require.NoError(t, err, "should complete analysis successfully")

		// Verify markdown file was created
		data, err := os.ReadFile(tempFile)
		require.NoError(t, err, "should read markdown file")
		assert.Contains(t, string(data), "# Code Review Assistant - Analysis Report")
		assert.Contains(t, string(data), "## Summary")
		assert.Contains(t, string(data), "## Issues Found")
	})

	t.Run("json output format", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "report.json")

		cfg := config.Default()
		cfg.Output.Format = "json"
		cfg.Output.OutputFile = tempFile
		cfg.Output.JSONPretty = true

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		require.NoError(t, err, "should complete analysis successfully")

		// Verify JSON file was created and is valid
		data, err := os.ReadFile(tempFile)
		require.NoError(t, err, "should read JSON file")

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err, "should be valid JSON")

		assert.Contains(t, result, "timestamp")
		assert.Contains(t, result, "version")
		assert.Contains(t, result, "result")
	})
}

// TestIntegration_StorageWorkflow tests the storage and comparison workflow
func TestIntegration_StorageWorkflow(t *testing.T) {
	projectPath := "testdata/sample"

	t.Run("file storage backend", func(t *testing.T) {
		tempDir := t.TempDir()

		// First analysis - save report
		cfg := config.Default()
		cfg.Storage.Enabled = true
		cfg.Storage.Backend = "file"
		cfg.Storage.Path = tempDir
		cfg.Storage.AutoSave = true
		cfg.Comparison.Enabled = false // Don't compare on first run

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		require.NoError(t, err, "first analysis should succeed")

		// Verify report was saved
		historyDir := filepath.Join(tempDir, "history")
		entries, err := os.ReadDir(historyDir)
		require.NoError(t, err, "should read history directory")
		assert.Greater(t, len(entries), 0, "should have at least one project directory")

		// Second analysis - compare with previous
		cfg2 := config.Default()
		cfg2.Storage.Enabled = true
		cfg2.Storage.Backend = "file"
		cfg2.Storage.Path = tempDir
		cfg2.Storage.AutoSave = true
		cfg2.Comparison.Enabled = true
		cfg2.Comparison.AutoCompare = true

		orch2, err := orchestrator.New(cfg2, projectPath)
		require.NoError(t, err, "should create orchestrator for second run")

		err = orch2.Run(projectPath)
		assert.NoError(t, err, "second analysis with comparison should succeed")
	})

	t.Run("sqlite storage backend", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		// First analysis - save report
		cfg := config.Default()
		cfg.Storage.Enabled = true
		cfg.Storage.Backend = "sqlite"
		cfg.Storage.Path = dbPath
		cfg.Storage.AutoSave = true
		cfg.Comparison.Enabled = false

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		require.NoError(t, err, "first analysis should succeed")

		// Verify database was created
		_, err = os.Stat(dbPath)
		assert.NoError(t, err, "database file should exist")

		// Second analysis - compare with previous
		cfg2 := config.Default()
		cfg2.Storage.Enabled = true
		cfg2.Storage.Backend = "sqlite"
		cfg2.Storage.Path = dbPath
		cfg2.Storage.AutoSave = true
		cfg2.Comparison.Enabled = true
		cfg2.Comparison.AutoCompare = true

		orch2, err := orchestrator.New(cfg2, projectPath)
		require.NoError(t, err, "should create orchestrator for second run")

		err = orch2.Run(projectPath)
		assert.NoError(t, err, "second analysis with comparison should succeed")
	})
}

// TestIntegration_ConfigurationMerge tests configuration precedence
func TestIntegration_ConfigurationMerge(t *testing.T) {
	projectPath := "testdata/sample"

	t.Run("custom thresholds", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LargeFileThreshold = 100
		cfg.Analysis.LongFunctionThreshold = 20
		cfg.Analysis.ComplexityThreshold = 5
		cfg.Analysis.MaxParameters = 3

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		assert.NoError(t, err, "should complete with custom thresholds")
	})

	t.Run("exclude patterns", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.ExcludePatterns = []string{
			"**/*_test.go",
			"**/testdata/**",
		}

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		assert.NoError(t, err, "should complete with exclude patterns")
	})
}

// TestIntegration_ErrorHandling tests error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	projectPath := "testdata/sample"

	t.Run("nonexistent directory", func(t *testing.T) {
		cfg := config.Default()
		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run("/nonexistent/path")
		assert.Error(t, err, "should return error for nonexistent path")
	})

	t.Run("invalid output file path", func(t *testing.T) {
		cfg := config.Default()
		cfg.Output.Format = "markdown"
		cfg.Output.OutputFile = "/nonexistent/directory/report.md"

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run("testdata/sample")
		assert.Error(t, err, "should return error for invalid output path")
	})

	t.Run("comparison without storage", func(t *testing.T) {
		cfg := config.Default()
		cfg.Storage.Enabled = false
		cfg.Comparison.Enabled = true // Try to compare without storage

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run("testdata/sample")
		// Should complete but skip comparison (logged warning)
		assert.NoError(t, err, "should complete but skip comparison")
	})
}

// TestIntegration_CoverageAnalysis tests coverage integration
func TestIntegration_CoverageAnalysis(t *testing.T) {
	projectPath := "testdata/sample"

	t.Run("coverage enabled", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.EnableCoverage = true
		cfg.Analysis.MinCoverageThreshold = 50.0

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		// May succeed or fail depending on test coverage, but should not crash
		if err != nil {
			// If error, it should be a meaningful error message
			assert.NotEmpty(t, err.Error())
		}
	})

	t.Run("coverage disabled", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.EnableCoverage = false

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		assert.NoError(t, err, "should complete successfully")
	})
}

// TestIntegration_DependencyAnalysis tests dependency analysis features
func TestIntegration_DependencyAnalysis(t *testing.T) {
	projectPath := "testdata/sample"

	t.Run("circular dependency detection", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.DetectCircularDeps = true

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		assert.NoError(t, err, "should complete dependency analysis")
	})

	t.Run("import limits", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.MaxImports = 5
		cfg.Analysis.MaxExternalDependencies = 3

		orch, err := orchestrator.New(cfg, projectPath)
		require.NoError(t, err, "should create orchestrator")

		err = orch.Run(projectPath)
		assert.NoError(t, err, "should complete with import limits")
	})
}

// TestIntegration_AllFormatsWithComparison tests all output formats with comparison
func TestIntegration_AllFormatsWithComparison(t *testing.T) {
	projectPath := "testdata/sample"
	tempDir := t.TempDir()

	// First run - save report
	cfg1 := config.Default()
	cfg1.Storage.Enabled = true
	cfg1.Storage.Backend = "file"
	cfg1.Storage.Path = tempDir
	cfg1.Storage.AutoSave = true

	orch1, err := orchestrator.New(cfg1, projectPath)
	require.NoError(t, err, "should create orchestrator")

	err = orch1.Run(projectPath)
	require.NoError(t, err, "first analysis should succeed")

	// Test each format with comparison
	formats := []string{"console", "markdown", "json"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			outputFile := ""
			if format != "console" {
				outputFile = filepath.Join(t.TempDir(), "report."+format)
			}

			cfg := config.Default()
			cfg.Output.Format = format
			cfg.Output.OutputFile = outputFile
			cfg.Storage.Enabled = true
			cfg.Storage.Backend = "file"
			cfg.Storage.Path = tempDir
			cfg.Storage.AutoSave = true
			cfg.Comparison.Enabled = true
			cfg.Comparison.AutoCompare = true

			orch, err := orchestrator.New(cfg, projectPath)
			require.NoError(t, err, "should create orchestrator")

			err = orch.Run(projectPath)
			assert.NoError(t, err, "should complete with "+format+" format and comparison")

			// Verify output file if not console
			if format != "console" {
				_, err := os.Stat(outputFile)
				assert.NoError(t, err, "output file should exist for "+format)
			}
		})
	}
}
