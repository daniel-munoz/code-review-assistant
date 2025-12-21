package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONReporter(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
	}

	reporter := NewJSONReporter(cfg)
	assert.NotNil(t, reporter)
	assert.Equal(t, cfg, reporter.config)
}

func TestJSONReporter_Report_PrettyPrint(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
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

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify it's valid JSON
	var parsed JSONOutput
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Verify metadata
	assert.NotEmpty(t, parsed.Timestamp)
	assert.Equal(t, "0.3.0", parsed.Version)

	// Verify timestamp is valid RFC3339
	_, err = time.Parse(time.RFC3339, parsed.Timestamp)
	assert.NoError(t, err)

	// Verify result data
	assert.Equal(t, "/test/project", parsed.Result.ProjectPath)
	assert.Equal(t, 10, parsed.Result.TotalFiles)
	assert.Equal(t, 1000, parsed.Result.TotalLines)
	assert.Equal(t, 700, parsed.Result.TotalCodeLines)
	assert.Equal(t, 50, parsed.Result.TotalFunctions)

	// Verify pretty formatting (should have indentation)
	assert.Contains(t, output, "  \"timestamp\":")
	assert.Contains(t, output, "  \"version\":")
	assert.Contains(t, output, "  \"result\":")
}

func TestJSONReporter_Report_Compact(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: false,
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
			AverageComplexity:     3.0,
			ComplexityP95:         7,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	output := buf.String()

	// Verify it's valid JSON
	var parsed JSONOutput
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	// Verify compact formatting (no indentation)
	assert.NotContains(t, output, "  \"timestamp\":")
	assert.Contains(t, output, "\"timestamp\":")
	assert.Contains(t, output, "\"version\":\"0.3.0\"")

	// Verify it's a single line (plus the newline at the end)
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 1)
}

func TestJSONReporter_Report_WithIssues(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
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

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify issues
	assert.Len(t, parsed.Result.Issues, 2)

	// First issue
	assert.Equal(t, "warning", parsed.Result.Issues[0].Severity)
	assert.Equal(t, "high_complexity", parsed.Result.Issues[0].Type)
	assert.Equal(t, "internal/test.go", parsed.Result.Issues[0].File)
	assert.Equal(t, 42, parsed.Result.Issues[0].Line)
	assert.Equal(t, "TestFunction", parsed.Result.Issues[0].Function)
	assert.Equal(t, 15, parsed.Result.Issues[0].Value)
	assert.Equal(t, 10, parsed.Result.Issues[0].Threshold)

	// Second issue
	assert.Equal(t, "info", parsed.Result.Issues[1].Severity)
	assert.Equal(t, "magic_number", parsed.Result.Issues[1].Type)
}

func TestJSONReporter_Report_WithMetrics(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
	}

	largestFiles := []*analyzer.FileSize{
		{Path: "internal/large1.go", Lines: 600},
		{Path: "internal/large2.go", Lines: 550},
	}

	complexFunctions := []*analyzer.FunctionInfo{
		{File: "internal/complex1.go", Function: "ProcessData", Complexity: 15, Lines: 80},
		{File: "internal/complex2.go", Function: "Transform", Complexity: 12, Lines: 60},
	}

	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     5,
		TotalLines:     2000,
		TotalCodeLines: 1400,
		TotalFunctions: 40,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 12.5,
			FunctionLengthP95:     35,
			CommentRatio:          0.2,
			AverageComplexity:     5.0,
			ComplexityP95:         12,
			LargestFiles:          largestFiles,
			MostComplexFunctions:  complexFunctions,
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: []*analyzer.Issue{},
	}

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify metrics
	metrics := parsed.Result.Metrics
	assert.Equal(t, 12.5, metrics.AverageFunctionLength)
	assert.Equal(t, 35, metrics.FunctionLengthP95)
	assert.Equal(t, 0.2, metrics.CommentRatio)
	assert.Equal(t, 5.0, metrics.AverageComplexity)
	assert.Equal(t, 12, metrics.ComplexityP95)

	// Verify largest files
	assert.Len(t, metrics.LargestFiles, 2)
	assert.Equal(t, "internal/large1.go", metrics.LargestFiles[0].Path)
	assert.Equal(t, 600, metrics.LargestFiles[0].Lines)

	// Verify complex functions
	assert.Len(t, metrics.MostComplexFunctions, 2)
	assert.Equal(t, "ProcessData", metrics.MostComplexFunctions[0].Function)
	assert.Equal(t, 15, metrics.MostComplexFunctions[0].Complexity)
}

func TestJSONReporter_Report_WithCoverage(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
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

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify coverage
	assert.NotNil(t, parsed.Result.Coverage)
	assert.Equal(t, 75.5, parsed.Result.Coverage.AverageCoverage)
	assert.Equal(t, 1, parsed.Result.Coverage.LowCoverageCount)
	assert.Len(t, parsed.Result.Coverage.Packages, 3)

	// Verify package coverage
	assert.Equal(t, "internal/pkg1", parsed.Result.Coverage.Packages[0].PackagePath)
	assert.Equal(t, 85.0, parsed.Result.Coverage.Packages[0].Coverage)
	assert.False(t, parsed.Result.Coverage.Packages[0].Skipped)

	assert.Equal(t, "internal/pkg3", parsed.Result.Coverage.Packages[2].PackagePath)
	assert.True(t, parsed.Result.Coverage.Packages[2].Skipped)
}

func TestJSONReporter_Report_WithDependencies(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
	}

	dependencies := &analyzer.DependencyReport{
		TotalPackages:     3,
		HighImportCount:   1,
		HighExternalCount: 0,
		Packages: []*analyzer.PackageDependencies{
			{
				PackageName:         "internal/pkg1",
				StdlibImports:       []string{"fmt", "os"},
				InternalImports:     []string{"internal/pkg2"},
				ExternalImports:     []string{"github.com/foo/bar"},
				TotalImports:        4,
				ExternalImportCount: 1,
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

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify dependencies
	deps := parsed.Result.Dependencies
	assert.NotNil(t, deps)
	assert.Equal(t, 3, deps.TotalPackages)
	assert.Equal(t, 1, deps.HighImportCount)
	assert.Equal(t, 0, deps.HighExternalCount)

	// Verify package dependencies
	assert.Len(t, deps.Packages, 1)
	pkg := deps.Packages[0]
	assert.Equal(t, "internal/pkg1", pkg.PackageName)
	assert.Equal(t, []string{"fmt", "os"}, pkg.StdlibImports)
	assert.Equal(t, []string{"internal/pkg2"}, pkg.InternalImports)
	assert.Equal(t, []string{"github.com/foo/bar"}, pkg.ExternalImports)

	// Verify circular dependencies
	assert.Len(t, deps.CircularDependencies, 1)
	assert.Equal(t, []string{"pkg/a", "pkg/b", "pkg/a"}, deps.CircularDependencies[0].Cycle)
}

func TestJSONReporter_Report_ToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "reports", "test.json")

	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
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

	reporter := NewJSONReporter(cfg)
	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, outputPath)

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(content, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "0.3.0", parsed.Version)
	assert.Equal(t, "/test/project", parsed.Result.ProjectPath)
	assert.Equal(t, 10, parsed.Result.TotalFiles)
}

func TestJSONReporter_CompleteStructure(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: true,
	}

	// Create a complete result with all fields populated
	result := &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     10,
		TotalLines:     2000,
		TotalCodeLines: 1400,
		TotalFunctions: 60,
		Metrics: &analyzer.AggregateMetrics{
			AverageFunctionLength: 15.0,
			FunctionLengthP95:     40,
			CommentRatio:          0.25,
			AverageComplexity:     5.0,
			ComplexityP95:         12,
			LargestFiles: []*analyzer.FileSize{
				{Path: "file1.go", Lines: 500},
			},
			MostComplexFunctions: []*analyzer.FunctionInfo{
				{File: "file1.go", Function: "Func1", Complexity: 15, Lines: 80},
			},
		},
		Files: []*analyzer.FileAnalysis{
			{
				Path: "test.go",
				Metrics: &parser.FileMetrics{
					FilePath:     "test.go",
					TotalLines:   100,
					CodeLines:    70,
					CommentLines: 20,
					BlankLines:   10,
					Functions:    []*parser.FunctionMetrics{},
				},
			},
		},
		Issues: []*analyzer.Issue{
			{
				Severity: "warning",
				Type:     "high_complexity",
				Message:  "Test issue",
				File:     "test.go",
				Line:     10,
			},
		},
		Coverage: &analyzer.CoverageReport{
			AverageCoverage:  80.0,
			LowCoverageCount: 0,
			Packages: []*analyzer.PackageCoverage{
				{PackagePath: "pkg1", Coverage: 80.0},
			},
		},
		Dependencies: &analyzer.DependencyReport{
			TotalPackages:        1,
			HighImportCount:      0,
			HighExternalCount:    0,
			Packages:             []*analyzer.PackageDependencies{},
			CircularDependencies: []*analyzer.CircularDependency{},
		},
	}

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify all top-level fields are present
	assert.NotEmpty(t, parsed.Timestamp)
	assert.NotEmpty(t, parsed.Version)
	assert.NotNil(t, parsed.Result)
	assert.NotNil(t, parsed.Result.Metrics)
	assert.NotEmpty(t, parsed.Result.Files)
	assert.NotEmpty(t, parsed.Result.Issues)
	assert.NotNil(t, parsed.Result.Coverage)
	assert.NotNil(t, parsed.Result.Dependencies)
}

func TestJSONReporter_IssuesSortedBySeverity(t *testing.T) {
	cfg := &config.OutputConfig{
		Format:     "json",
		JSONPretty: false,
	}

	// Create issues in mixed order
	issues := []*analyzer.Issue{
		{Severity: "info", Type: "test", Message: "Info issue"},
		{Severity: "error", Type: "test", Message: "Error issue"},
		{Severity: "warning", Type: "test", Message: "Warning issue"},
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
			AverageComplexity:     3.0,
			ComplexityP95:         7,
			LargestFiles:          []*analyzer.FileSize{},
			MostComplexFunctions:  []*analyzer.FunctionInfo{},
		},
		Files:  []*analyzer.FileAnalysis{},
		Issues: issues,
	}

	reporter := NewJSONReporter(cfg)
	var buf bytes.Buffer
	reporter.output = &buf

	err := reporter.Report(result, nil)
	require.NoError(t, err)

	// Parse JSON
	var parsed JSONOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify issues are sorted by severity (error > warning > info)
	assert.Len(t, parsed.Result.Issues, 3)
	assert.Equal(t, "error", parsed.Result.Issues[0].Severity)
	assert.Equal(t, "warning", parsed.Result.Issues[1].Severity)
	assert.Equal(t, "info", parsed.Result.Issues[2].Severity)
}
