package reporter

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReporter(t *testing.T) {
	t.Run("creates console reporter for console format", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter, err := NewReporter(cfg)

		require.NoError(t, err, "should create console reporter without error")
		require.NotNil(t, reporter, "reporter should not be nil")
		assert.IsType(t, &ConsoleReporter{}, reporter, "should be console reporter")
	})

	t.Run("creates console reporter for empty format", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "",
			Verbose: false,
		}

		reporter, err := NewReporter(cfg)

		require.NoError(t, err, "should create console reporter for empty format")
		require.NotNil(t, reporter, "reporter should not be nil")
	})

	t.Run("returns error for unsupported format", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "xml",
			Verbose: false,
		}

		reporter, err := NewReporter(cfg)

		assert.Error(t, err, "should return error for unsupported format")
		assert.Nil(t, reporter, "reporter should be nil on error")
		assert.Contains(t, err.Error(), "unsupported output format", "error message should indicate unsupported format")
	})
}

func TestNewConsoleReporter(t *testing.T) {
	t.Run("creates console reporter with verbose off", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter := NewConsoleReporter(cfg)

		require.NotNil(t, reporter, "reporter should not be nil")
		assert.Equal(t, cfg, reporter.config, "config should be set")
		assert.False(t, reporter.config.Verbose, "verbose should be false")
	})

	t.Run("creates console reporter with verbose on", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: true,
		}

		reporter := NewConsoleReporter(cfg)

		require.NotNil(t, reporter, "reporter should not be nil")
		assert.Equal(t, cfg, reporter.config, "config should be set")
		assert.True(t, reporter.config.Verbose, "verbose should be true")
	})
}

func TestConsoleReporter_Report(t *testing.T) {
	t.Run("reports successfully with minimal result", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter := NewConsoleReporter(cfg)

		// Create minimal analysis result
		result := &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     5,
			TotalLines:     1000,
			TotalCodeLines: 800,
			TotalFunctions: 20,
			Metrics: &analyzer.AggregateMetrics{
				AverageFunctionLength: 15.5,
				FunctionLengthP95:     50,
				CommentRatio:          0.15,
				AverageComplexity:     5.2,
				ComplexityP95:         12,
			},
			Issues: []*analyzer.Issue{},
		}

		err := reporter.Report(result)

		assert.NoError(t, err, "should report without error")
	})

	t.Run("reports with issues", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter := NewConsoleReporter(cfg)

		// Create result with some issues
		result := &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     5,
			TotalLines:     1000,
			TotalCodeLines: 800,
			TotalFunctions: 20,
			Metrics: &analyzer.AggregateMetrics{
				AverageFunctionLength: 15.5,
				FunctionLengthP95:     50,
				CommentRatio:          0.15,
				AverageComplexity:     5.2,
				ComplexityP95:         12,
			},
			Issues: []*analyzer.Issue{
				{
					Type:     "large_file",
					Severity: "warning",
					Message:  "File exceeds size threshold",
					File:     "large.go",
					Line:     1,
				},
				{
					Type:     "high_complexity",
					Severity: "warning",
					Message:  "Function has high cyclomatic complexity",
					File:     "complex.go",
					Line:     42,
					Function: "ComplexFunction",
				},
			},
		}

		err := reporter.Report(result)

		assert.NoError(t, err, "should report with issues without error")
	})

	t.Run("reports with coverage data", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter := NewConsoleReporter(cfg)

		result := &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     5,
			TotalLines:     1000,
			TotalCodeLines: 800,
			TotalFunctions: 20,
			Metrics: &analyzer.AggregateMetrics{
				AverageFunctionLength: 15.5,
				FunctionLengthP95:     50,
				CommentRatio:          0.15,
			},
			Issues: []*analyzer.Issue{},
			Coverage: &analyzer.CoverageReport{
				AverageCoverage:  68.5,
				LowCoverageCount: 2,
				Packages: []*analyzer.PackageCoverage{
					{PackagePath: "pkg1", Coverage: 75.0},
					{PackagePath: "pkg2", Coverage: 62.0},
				},
			},
		}

		err := reporter.Report(result)

		assert.NoError(t, err, "should report with coverage data without error")
	})

	t.Run("reports with dependency data", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter := NewConsoleReporter(cfg)

		result := &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     5,
			TotalLines:     1000,
			TotalCodeLines: 800,
			TotalFunctions: 20,
			Metrics: &analyzer.AggregateMetrics{
				AverageFunctionLength: 15.5,
				FunctionLengthP95:     50,
				CommentRatio:          0.15,
			},
			Issues: []*analyzer.Issue{},
			Dependencies: &analyzer.DependencyReport{
				TotalPackages:     3,
				HighImportCount:   1,
				HighExternalCount: 0,
				Packages: []*analyzer.PackageDependencies{
					{
						PackageName:         "main",
						StdlibImports:       []string{"fmt", "os"},
						InternalImports:     []string{"internal/config"},
						ExternalImports:     []string{"github.com/spf13/cobra"},
						TotalImports:        4,
						ExternalImportCount: 1,
					},
				},
			},
		}

		err := reporter.Report(result)

		assert.NoError(t, err, "should report with dependency data without error")
	})

	t.Run("reports with verbose mode", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: true,
		}

		reporter := NewConsoleReporter(cfg)

		result := &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     5,
			TotalLines:     1000,
			TotalCodeLines: 800,
			TotalFunctions: 20,
			Metrics: &analyzer.AggregateMetrics{
				AverageFunctionLength: 15.5,
				FunctionLengthP95:     50,
				CommentRatio:          0.15,
			},
			Issues: []*analyzer.Issue{},
			Coverage: &analyzer.CoverageReport{
				AverageCoverage: 75.0,
				Packages: []*analyzer.PackageCoverage{
					{PackagePath: "pkg1", Coverage: 75.0},
				},
			},
		}

		err := reporter.Report(result)

		assert.NoError(t, err, "should report in verbose mode without error")
	})
}

func TestConsoleReporter_ReportFormat(t *testing.T) {
	t.Run("reports with all sections", func(t *testing.T) {
		cfg := &config.OutputConfig{
			Format:  "console",
			Verbose: false,
		}

		reporter := NewConsoleReporter(cfg)

		result := &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     2,
			TotalLines:     100,
			TotalCodeLines: 80,
			TotalFunctions: 5,
			Metrics: &analyzer.AggregateMetrics{
				AverageFunctionLength: 10.0,
				FunctionLengthP95:     20,
				CommentRatio:          0.15,
				AverageComplexity:     5.0,
				ComplexityP95:         10,
				LargestFiles: []*analyzer.FileSize{
					{Path: "large.go", Lines: 500},
				},
				MostComplexFunctions: []*analyzer.FunctionInfo{
					{File: "complex.go", Function: "DoWork", Complexity: 15, Lines: 50},
				},
			},
			Issues: []*analyzer.Issue{
				{
					Type:     "large_file",
					Severity: "warning",
					Message:  "File exceeds threshold",
					File:     "large.go",
				},
			},
		}

		err := reporter.Report(result)

		assert.NoError(t, err, "should report all sections without error")
	})
}
