package reporter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTMLReporter(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	reporter := NewHTMLReporter(cfg)
	assert.NotNil(t, reporter)
	assert.Equal(t, cfg, reporter.config)
	assert.NotNil(t, reporter.output) // Should default to stdout
	assert.Nil(t, reporter.storage)   // No storage by default
}

func TestHTMLReporter_WithStorage(t *testing.T) {
	cfg := &config.OutputConfig{
		Format: "html",
	}

	reporter := NewHTMLReporter(cfg)
	assert.Nil(t, reporter.storage)

	// Note: We can't easily create a real storage without significant setup,
	// so we just verify the method exists and returns the reporter
	result := reporter.WithStorage(nil)
	assert.Equal(t, reporter, result) // Should return self for chaining
}

func TestHTMLReporter_Report_MinimalResult(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     10,
		TotalLines:     1000,
		TotalCodeLines: 700,
		TotalFunctions: 50,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 14.0,
			FunctionLengthP95:     30,
			CommentRatio:          0.2,
			AverageComplexity:     3.5,
			ComplexityP95:         8,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewHTMLReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify HTML structure
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "<html")
	assert.Contains(t, output, "</html>")
	assert.Contains(t, output, "<head>")
	assert.Contains(t, output, "</head>")
	assert.Contains(t, output, "<body>")
	assert.Contains(t, output, "</body>")

	// Verify favicon
	assert.Contains(t, output, `<link rel="icon"`)

	// Verify title
	assert.Contains(t, output, "<title>Code Review Dashboard - project</title>")

	// Verify project info
	assert.Contains(t, output, "/test/project")
	assert.Contains(t, output, "Code Review Dashboard")

	// Verify summary metrics
	assert.Contains(t, output, "Total Files")
	assert.Contains(t, output, "10")
	assert.Contains(t, output, "Total Lines")
	assert.Contains(t, output, "1,000")

	// Verify aggregate metrics
	assert.Contains(t, output, "Avg Complexity")
	assert.Contains(t, output, "3.5")
}

func TestHTMLReporter_Report_WithIssues(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	issues := []*analyzer.Issue{
		{
			Severity:  "error",
			Type:      "high_complexity",
			Message:   "Function has high cyclomatic complexity",
			File:      "internal/test.go",
			Line:      42,
			Function:  "TestFunction",
			Value:     15,
			Threshold: 10,
		},
		{
			Severity:  "warning",
			Type:      "large_file",
			Message:   "File exceeds maximum line count",
			File:      "internal/large.go",
			Line:      1,
			Value:     600,
			Threshold: 500,
		},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     1200,
		TotalCodeLines: 900,
		TotalFunctions: 30,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 15.0,
			FunctionLengthP95:     35,
			CommentRatio:          0.15,
			AverageComplexity:     5.0,
			ComplexityP95:         12,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: issues,
	}

	reporter := NewHTMLReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify issues section exists
	assert.Contains(t, output, "Issues Found")
	assert.Contains(t, output, "(2)")

	// Verify issue details
	assert.Contains(t, output, "Function has high cyclomatic complexity")
	assert.Contains(t, output, "internal/test.go")
	assert.Contains(t, output, "TestFunction")
	assert.Contains(t, output, "File exceeds maximum line count")
	assert.Contains(t, output, "internal/large.go")

	// Verify severity counts
	assert.Contains(t, output, "Errors")
	assert.Contains(t, output, "Warnings")

	// Verify severity icons/classes
	assert.Contains(t, output, "severity-error")
	assert.Contains(t, output, "severity-warning")
}

func TestHTMLReporter_Report_WithCoverage(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	coverage := &analyzer.CoverageReport{
		AverageCoverage:  75.5,
		LowCoverageCount: 1,
		Packages: []*analyzer.PackageCoverage{
			{PackagePath: "internal/pkg1", Coverage: 85.0},
			{PackagePath: "internal/pkg2", Coverage: 45.0},
		},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     1000,
		TotalCodeLines: 700,
		TotalFunctions: 30,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 15.0,
			FunctionLengthP95:     35,
			CommentRatio:          0.2,
			AverageComplexity:     4.0,
			ComplexityP95:         10,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:    []*analyzer.FileAnalysis{},
		Issues:   []*analyzer.Issue{},
		Coverage: coverage,
	}

	reporter := NewHTMLReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify coverage section
	assert.Contains(t, output, "Test Coverage")
	assert.Contains(t, output, "75.5")
	assert.Contains(t, output, "internal/pkg1")
	assert.Contains(t, output, "internal/pkg2")
	// Coverage values appear as integers in JSON: [85,45]
	assert.Contains(t, output, "[85,45]")
}

func TestHTMLReporter_Report_WithCharts(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath: "/test/project",
		Files: []*analyzer.FileAnalysis{
			{
				Path: "internal/test.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 100,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 5},
						{Complexity: 10},
					},
				},
			},
		},
		TotalFiles:     1,
		TotalLines:     100,
		TotalCodeLines: 70,
		TotalFunctions: 2,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 10.0,
			FunctionLengthP95:     15,
			CommentRatio:          0.2,
			AverageComplexity:     7.5,
			ComplexityP95:         10,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Issues: []*analyzer.Issue{
			{Type: "high_complexity", Severity: "warning"},
		},
	}

	reporter := NewHTMLReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify Chart.js is included
	assert.Contains(t, output, "chart.js")

	// Verify chart data is embedded
	assert.Contains(t, output, "chartData")
	assert.Contains(t, output, "complexityDist")

	// Verify chart containers exist
	assert.Contains(t, output, "complexityChart")
	assert.Contains(t, output, "issuesChart")

	// Verify visualizations section
	assert.Contains(t, output, "Visualizations")
	assert.Contains(t, output, "Complexity Distribution")
	assert.Contains(t, output, "Issues by Type")
	assert.Contains(t, output, "Complexity Heatmap")
}

func TestHTMLReporter_Report_WithDependencies(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	dependencies := &analyzer.DependencyReport{
		TotalPackages:     5,
		HighImportCount:   2,
		HighExternalCount: 1,
		Packages: []*analyzer.PackageDependencies{
			{
				PackageName:     "pkg/handler",
				TotalImports:    10,
				InternalImports: []string{"pkg/models"},
				ExternalImports: []string{"github.com/gorilla/mux"},
			},
		},
		CircularDependencies: []*analyzer.CircularDependency{
			{Cycle: []string{"pkg/a", "pkg/b", "pkg/a"}},
		},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     1000,
		TotalCodeLines: 700,
		TotalFunctions: 30,
		Files: []*analyzer.FileAnalysis{
			{
				Path: "test.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 50,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 3},
					},
				},
			},
		},
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 15.0,
			FunctionLengthP95:     35,
			CommentRatio:          0.2,
			AverageComplexity:     4.0,
			ComplexityP95:         10,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Issues:       []*analyzer.Issue{},
		Dependencies: dependencies,
	}

	reporter := NewHTMLReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify dependencies section
	assert.Contains(t, output, "Dependencies")
	assert.Contains(t, output, "Total Packages")
	assert.Contains(t, output, "Circular Dependencies")

	// Verify vis-network.js for dependency graph
	assert.Contains(t, output, "vis-network")
	assert.Contains(t, output, "dependencyGraph")
	assert.Contains(t, output, "Dependency Graph")
}

func TestHTMLReporter_Report_ToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "reports", "test.html")

	cfg := &config.OutputConfig{
		Format:     "html",
		Verbose:    false,
		OutputFile: outputPath,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     10,
		TotalLines:     1000,
		TotalCodeLines: 700,
		TotalFunctions: 50,
		Files: []*analyzer.FileAnalysis{
			{
				Path: "test.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 50,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 3},
					},
				},
			},
		},
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 14.0,
			FunctionLengthP95:     30,
			CommentRatio:          0.2,
			AverageComplexity:     3.5,
			ComplexityP95:         8,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewHTMLReporter(cfg)
	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, outputPath)

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	output := string(content)
	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "/test/project")
	assert.Contains(t, output, "Total Files")
}

func TestHTMLReporter_TemplateHelpers(t *testing.T) {
	t.Run("formatNumber", func(t *testing.T) {
		tests := []struct {
			input    int
			expected string
		}{
			{0, "0"},
			{999, "999"},
			{1000, "1,000"},
			{1234567, "1,234,567"},
		}

		for _, tt := range tests {
			result := formatNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("formatFloat", func(t *testing.T) {
		tests := []struct {
			input    float64
			expected string
		}{
			{0.0, "0.0"},
			{3.5, "3.5"},
			{10.123, "10.1"},
			{99.99, "100.0"},
		}

		for _, tt := range tests {
			result := formatFloat(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("severityIcon", func(t *testing.T) {
		assert.Equal(t, "❌", severityIcon("error"))
		assert.Equal(t, "⚠️", severityIcon("warning"))
		assert.Equal(t, "ℹ️", severityIcon("info"))
		assert.Equal(t, "", severityIcon("unknown"))
	})

	t.Run("severityClass", func(t *testing.T) {
		assert.Equal(t, "severity-error", severityClass("error"))
		assert.Equal(t, "severity-warning", severityClass("warning"))
		assert.Equal(t, "severity-info", severityClass("info"))
		assert.Equal(t, "", severityClass("unknown"))
	})

	t.Run("add", func(t *testing.T) {
		assert.Equal(t, 5, add(2, 3))
		assert.Equal(t, 0, add(0, 0))
		assert.Equal(t, -1, add(-3, 2))
	})

	t.Run("slice", func(t *testing.T) {
		arr := []string{"a", "b", "c", "d"}
		assert.Equal(t, []string{"c", "d"}, slice(arr, 2))
		assert.Equal(t, []string{"a", "b", "c", "d"}, slice(arr, 0))
		assert.Equal(t, []string{}, slice(arr, 10))
	})
}

func TestHTMLReporter_BuildTemplateData(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: true,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     10,
		TotalLines:     1000,
		TotalCodeLines: 700,
		TotalFunctions: 50,
		Files: []*analyzer.FileAnalysis{
			{
				Path: "test.go",
				Metrics: &parser.FileMetrics{
					FilePath:     "test.go",
					TotalLines:   100,
					CodeLines:    70,
					CommentLines: 20,
					BlankLines:   10,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 5},
					},
				},
			},
		},
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 14.0,
			FunctionLengthP95:     30,
			CommentRatio:          0.2,
			AverageComplexity:     3.5,
			ComplexityP95:         8,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Issues: []*analyzer.Issue{
			{Severity: "error"},
			{Severity: "warning"},
			{Severity: "warning"},
			{Severity: "info"},
		},
	}

	reporter := NewHTMLReporter(cfg)
	data := reporter.buildTemplateData(result, nil)

	require.NotNil(t, data)
	assert.Equal(t, "/test/project", data.ProjectPath)
	assert.Equal(t, "project", data.ProjectName)
	assert.NotEmpty(t, data.Timestamp)
	assert.Equal(t, 10, data.TotalFiles)
	assert.Equal(t, 1000, data.TotalLines)
	assert.Equal(t, 700, data.TotalCodeLines)
	assert.Equal(t, 50, data.TotalFunctions)
	assert.Equal(t, 70.0, data.CodePercent)
	assert.Equal(t, 4, data.IssueCount)
	assert.Equal(t, 1, data.ErrorCount)
	assert.Equal(t, 2, data.WarningCount)
	assert.Equal(t, 1, data.InfoCount)
	assert.True(t, data.Verbose)
	assert.NotNil(t, data.ChartData)
}

func TestHTMLReporter_ValidHTML(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "html",
		Verbose: false,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     500,
		TotalCodeLines: 350,
		TotalFunctions: 20,
		Files: []*analyzer.FileAnalysis{
			{
				Path: "test.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 50,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 3},
					},
				},
			},
		},
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 10.0,
			FunctionLengthP95:     20,
			CommentRatio:          0.2,
			AverageComplexity:     3.0,
			ComplexityP95:         7,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewHTMLReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify basic HTML structure
	assert.True(t, strings.HasPrefix(output, "<!DOCTYPE html>"))
	assert.Contains(t, output, "<html")
	assert.Contains(t, output, "</html>")

	// Verify all major sections close properly
	assert.Equal(t, strings.Count(output, "<div"), strings.Count(output, "</div>"))
	assert.Equal(t, strings.Count(output, "<table"), strings.Count(output, "</table>"))
	assert.Equal(t, strings.Count(output, "<tr"), strings.Count(output, "</tr>"))
	assert.Equal(t, strings.Count(output, "<td"), strings.Count(output, "</td>"))
}
