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

func TestNewMarkdownReporter(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	reporter := NewMarkdownReporter(cfg)
	assert.NotNil(t, reporter)
	assert.Equal(t, cfg, reporter.config)
}

func TestMarkdownReporter_Report_MinimalResult(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
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

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify header
	assert.Contains(t, output, "# Code Review Assistant - Analysis Report")
	assert.Contains(t, output, "**Project:** /test/project")

	// Verify summary table
	assert.Contains(t, output, "## Summary")
	assert.Contains(t, output, "| Total Files | 10 |")
	assert.Contains(t, output, "| Total Lines | 1,000 |")

	// Verify aggregate metrics
	assert.Contains(t, output, "## Aggregate Metrics")
	assert.Contains(t, output, "| Average Function Length | 14.0 lines |")
	assert.Contains(t, output, "| Comment Ratio | 20.0% |")

	// Verify footer
	assert.Contains(t, output, "*Analysis complete.*")
}

func TestMarkdownReporter_Report_WithIssues(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	issues := []*analyzer.Issue{
		{
			Severity:  "warning",
			Type:      "high_complexity",
			Message:   "Function has high cyclomatic complexity",
			File:      "internal/test.go",
			Line:      42,
			Function:  "TestFunction",
			Value:     15,
			Threshold: 10,
		},
		{
			Severity: "info",
			Type:     "magic_number",
			Message:  "Magic number should be replaced with a named constant: 100",
			File:     "internal/utils.go",
			Line:     10,
			Function: "Calculate",
		},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     500,
		TotalCodeLines: 350,
		TotalFunctions: 20,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 10.0,
			FunctionLengthP95:     20,
			CommentRatio:          0.15,
			AverageComplexity:     4.0,
			ComplexityP95:         10,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: issues,
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify issues section
	assert.Contains(t, output, "## Issues Found (2)")
	assert.Contains(t, output, "⚠️ **[WARNING]** Function has high cyclomatic complexity")
	assert.Contains(t, output, "`internal/test.go:42`")
	assert.Contains(t, output, "**Function:** `TestFunction`")
	assert.Contains(t, output, "**Complexity:** 15 (threshold: 10)")
	assert.Contains(t, output, "ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100")
}

func TestMarkdownReporter_Report_WithLargestFiles(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	largestFiles := []*analyzer.FileSize{
		{Path: "internal/large1.go", Lines: 600},
		{Path: "internal/large2.go", Lines: 550},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     2000,
		TotalCodeLines: 1400,
		TotalFunctions: 40,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 12.0,
			FunctionLengthP95:     25,
			CommentRatio:          0.2,
			AverageComplexity:     3.0,
			ComplexityP95:         7,
			LargestFiles:          largestFiles,
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify largest files table
	assert.Contains(t, output, "## Largest Files")
	assert.Contains(t, output, "| Rank | File | Lines |")
	assert.Contains(t, output, "| 1 | `internal/large1.go` | 600 |")
	assert.Contains(t, output, "| 2 | `internal/large2.go` | 550 |")
}

func TestMarkdownReporter_Report_WithComplexFunctions(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	complexFunctions := []*analyzer.FunctionInfo{
		{File: "internal/complex1.go", Function: "ProcessData", Complexity: 15, Lines: 80},
		{File: "internal/complex2.go", Function: "Transform", Complexity: 12, Lines: 60},
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
			CommentRatio:          0.18,
			AverageComplexity:     5.0,
			ComplexityP95:         12,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  complexFunctions,
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify most complex functions table
	assert.Contains(t, output, "## Most Complex Functions")
	assert.Contains(t, output, "| Rank | Function | Complexity | Lines | File |")
	assert.Contains(t, output, "| 1 | `ProcessData` | 15 | 80 | `internal/complex1.go` |")
	assert.Contains(t, output, "| 2 | `Transform` | 12 | 60 | `internal/complex2.go` |")
}

func TestMarkdownReporter_Report_WithCoverage(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: true, // Enable verbose to see package details
	}

	coverage := &analyzer.CoverageReport{
		AverageCoverage:  75.5,
		LowCoverageCount: 1,
		Packages: []*analyzer.PackageCoverage{
			{PackagePath: "internal/pkg1", Coverage: 85.0},
			{PackagePath: "internal/pkg2", Coverage: 45.0},
			{PackagePath: "internal/pkg3", Skipped: true},
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

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify coverage section
	assert.Contains(t, output, "## Test Coverage")
	assert.Contains(t, output, "| Average Coverage | 75.5% |")
	assert.Contains(t, output, "| Total Packages | 3 |")
	assert.Contains(t, output, "| Packages Below Threshold | 1 |")

	// Verify verbose package details
	assert.Contains(t, output, "<details>")
	assert.Contains(t, output, "Package Coverage Details")
	assert.Contains(t, output, "`internal/pkg1` | 85.0%")
	assert.Contains(t, output, "`internal/pkg2` | 45.0%")
	assert.Contains(t, output, "`internal/pkg3` | No tests")
}

func TestMarkdownReporter_Report_ToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "reports", "test.md")

	cfg := &config.OutputConfig{
		Format:     "markdown",
		Verbose:    false,
		OutputFile: outputPath,
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

	reporter := NewMarkdownReporter(cfg)
	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, outputPath)

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	output := string(content)
	assert.Contains(t, output, "# Code Review Assistant - Analysis Report")
	assert.Contains(t, output, "**Project:** /test/project")
	assert.Contains(t, output, "| Total Files | 10 |")
}

func TestMarkdownReporter_FormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMarkdownReporter_CircularDependencies(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	dependencies := &analyzer.DependencyReport{
		TotalPackages:     3,
		HighImportCount:   0,
		HighExternalCount: 0,
		Packages: []*analyzer.PackageDependencies{
			{PackageName: "pkg1", TotalImports: 5},
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
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 15.0,
			FunctionLengthP95:     35,
			CommentRatio:          0.2,
			AverageComplexity:     4.0,
			ComplexityP95:         10,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:        []*analyzer.FileAnalysis{},
		Issues:       []*analyzer.Issue{},
		Dependencies: dependencies,
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify dependencies section
	assert.Contains(t, output, "## Dependencies")
	assert.Contains(t, output, "| Circular Dependencies | 1 |")
	assert.Contains(t, output, "### ⚠️ Circular Dependencies Detected")
	assert.Contains(t, output, "pkg/a -> pkg/b -> pkg/a")
}

func TestMarkdownReporter_VerboseMode(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: true,
	}

	files := []*analyzer.FileAnalysis{
		{
			Path: "internal/test.go",
			Metrics: &parser.FileMetrics{
				FilePath:     "internal/test.go",
				TotalLines:   100,
				CodeLines:    70,
				CommentLines: 20,
				BlankLines:   10,
				Functions:    []*parser.FunctionMetrics{},
			},
			LargeFile: false,
		},
		{
			Path: "internal/large.go",
			Metrics: &parser.FileMetrics{
				FilePath:     "internal/large.go",
				TotalLines:   600,
				CodeLines:    450,
				CommentLines: 100,
				BlankLines:   50,
				Functions:    []*parser.FunctionMetrics{},
			},
			LargeFile: true,
		},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     2,
		TotalLines:     700,
		TotalCodeLines: 520,
		TotalFunctions: 10,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 15.0,
			FunctionLengthP95:     30,
			CommentRatio:          0.17,
			AverageComplexity:     3.0,
			ComplexityP95:         8,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  files,
		Issues: []*analyzer.Issue{},
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify verbose file details
	assert.Contains(t, output, "<details>")
	assert.Contains(t, output, "File Details")
	assert.Contains(t, output, "### `internal/test.go`")
	assert.Contains(t, output, "- **Lines:** 100 (Code: 70, Comments: 20, Blank: 10)")
	assert.Contains(t, output, "### `internal/large.go`")
	assert.Contains(t, output, "- ⚠️ **Large file**")
}

func TestMarkdownReporter_NoIssuesNoExtras(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     500,
		TotalCodeLines: 350,
		TotalFunctions: 20,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 10.0,
			FunctionLengthP95:     20,
			CommentRatio:          0.2,
			AverageComplexity:     3.0,
			ComplexityP95:         7,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Should not contain empty sections
	assert.NotContains(t, output, "## Issues Found")
	assert.NotContains(t, output, "## Largest Files")
	assert.NotContains(t, output, "## Most Complex Functions")
	assert.NotContains(t, output, "## Test Coverage")
	assert.NotContains(t, output, "## Dependencies")

	// Should contain basic sections
	assert.Contains(t, output, "## Summary")
	assert.Contains(t, output, "## Aggregate Metrics")
}

func TestMarkdownReporter_OutputFormatting(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:  "markdown",
		Verbose: false,
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     500,
		TotalCodeLines: 350,
		TotalFunctions: 20,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 10.5,
			FunctionLengthP95:     25,
			CommentRatio:          0.15,
			AverageComplexity:     4.2,
			ComplexityP95:         10,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewMarkdownReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify markdown formatting
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Check table formatting
		if strings.Contains(line, "|---") {
			// Should have proper table separator
			assert.True(t, strings.HasPrefix(line, "|"))
			assert.True(t, strings.HasSuffix(line, "|"))
		}

		// Check header formatting
		if strings.HasPrefix(line, "##") {
			// Next line should be empty
			if i+1 < len(lines) {
				assert.Equal(t, "", lines[i+1], "Header should be followed by empty line")
			}
		}
	}
}
