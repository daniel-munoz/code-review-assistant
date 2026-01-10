package reporter

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildComplexityDistribution(t *testing.T) {
	result := &analyzer.AnalysisResult{
		Files: []*analyzer.FileAnalysis{
			{
				Metrics: &parser.FileMetrics{
					Functions: []*parser.FunctionMetrics{
						{Complexity: 3},
						{Complexity: 7},
						{Complexity: 12},
						{Complexity: 18},
						{Complexity: 25},
						{Complexity: 5},
						{Complexity: 10},
					},
				},
			},
		},
	}

	dist := buildComplexityDistribution(result)

	require.NotNil(t, dist)
	assert.Equal(t, []string{"1-5", "6-10", "11-15", "16-20", "20+"}, dist.Ranges)
	assert.Equal(t, []int{2, 2, 1, 1, 1}, dist.Counts) // 3,5 | 7,10 | 12 | 18 | 25
}

func TestBuildComplexityDistribution_EmptyResult(t *testing.T) {
	result := &analyzer.AnalysisResult{
		Files: []*analyzer.FileAnalysis{},
	}

	dist := buildComplexityDistribution(result)

	require.NotNil(t, dist)
	assert.Equal(t, []string{"1-5", "6-10", "11-15", "16-20", "20+"}, dist.Ranges)
	assert.Equal(t, []int{0, 0, 0, 0, 0}, dist.Counts)
}

func TestBuildCoverageBreakdown(t *testing.T) {
	coverage := &analyzer.CoverageReport{
		Packages: []*analyzer.PackageCoverage{
			{PackagePath: "pkg/a", Coverage: 85.0},
			{PackagePath: "pkg/b", Coverage: 55.0},
			{PackagePath: "pkg/c", Coverage: 30.0},
			{PackagePath: "pkg/d", Skipped: true},       // Should be excluded
			{PackagePath: "pkg/e", Error: "test failed"}, // Should be excluded
		},
	}

	data := buildCoverageBreakdown(coverage)

	require.NotNil(t, data)
	assert.Equal(t, []string{"pkg/a", "pkg/b", "pkg/c"}, data.Packages)
	assert.Equal(t, []float64{85.0, 55.0, 30.0}, data.Coverages)
	assert.Equal(t, []string{"#10b981", "#f59e0b", "#ef4444"}, data.Colors)
}

func TestBuildCoverageBreakdown_Nil(t *testing.T) {
	data := buildCoverageBreakdown(nil)
	assert.Nil(t, data)
}

func TestBuildCoverageBreakdown_EmptyPackages(t *testing.T) {
	coverage := &analyzer.CoverageReport{
		Packages: []*analyzer.PackageCoverage{},
	}

	data := buildCoverageBreakdown(coverage)
	assert.Nil(t, data)
}

func TestGetCoverageColor(t *testing.T) {
	tests := []struct {
		coverage float64
		expected string
	}{
		{100.0, "#10b981"}, // Green
		{80.0, "#10b981"},  // Green
		{75.0, "#f59e0b"},  // Yellow
		{50.0, "#f59e0b"},  // Yellow
		{30.0, "#ef4444"},  // Red
		{0.0, "#ef4444"},   // Red
	}

	for _, tt := range tests {
		t.Run(formatFloat(tt.coverage), func(t *testing.T) {
			result := getCoverageColor(tt.coverage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildIssueCountData(t *testing.T) {
	issues := []*analyzer.Issue{
		{Type: "high_complexity", Severity: "error"},
		{Type: "high_complexity", Severity: "warning"},
		{Type: "high_complexity", Severity: "info"},
		{Type: "large_file", Severity: "warning"},
		{Type: "large_file", Severity: "warning"},
		{Type: "magic_number", Severity: "info"},
	}

	data := buildIssueCountData(issues)

	require.NotNil(t, data)
	assert.Len(t, data.Types, 3)
	assert.Contains(t, data.Types, "High Complexity")
	assert.Contains(t, data.Types, "Large File")
	assert.Contains(t, data.Types, "Magic Number")

	// Find indices for verification
	var complexityIdx, largeFileIdx, magicIdx int
	for i, t := range data.Types {
		switch t {
		case "High Complexity":
			complexityIdx = i
		case "Large File":
			largeFileIdx = i
		case "Magic Number":
			magicIdx = i
		}
	}

	assert.Equal(t, 1, data.ErrorCount[complexityIdx])
	assert.Equal(t, 1, data.WarnCount[complexityIdx])
	assert.Equal(t, 1, data.InfoCount[complexityIdx])

	assert.Equal(t, 0, data.ErrorCount[largeFileIdx])
	assert.Equal(t, 2, data.WarnCount[largeFileIdx])
	assert.Equal(t, 0, data.InfoCount[largeFileIdx])

	assert.Equal(t, 0, data.ErrorCount[magicIdx])
	assert.Equal(t, 0, data.WarnCount[magicIdx])
	assert.Equal(t, 1, data.InfoCount[magicIdx])
}

func TestBuildIssueCountData_EmptyIssues(t *testing.T) {
	data := buildIssueCountData([]*analyzer.Issue{})
	assert.Nil(t, data)
}

func TestFormatIssueTypeForChart(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"large_file", "Large File"},
		{"long_function", "Long Function"},
		{"high_complexity", "High Complexity"},
		{"too_many_parameters", "Too Many Params"},
		{"deep_nesting", "Deep Nesting"},
		{"too_many_returns", "Too Many Returns"},
		{"low_coverage", "Low Coverage"},
		{"too_many_imports", "Too Many Imports"},
		{"too_many_external_deps", "Too Many External Deps"},
		{"magic_number", "Magic Number"},
		{"duplicate_error_handling", "Duplicate Error Handling"},
		{"circular_dependency", "Circular Dependency"},
		{"low_comment_ratio", "Low Comment Ratio"},
		{"unknown_type", "unknown_type"}, // Fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatIssueTypeForChart(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildHeatmapData(t *testing.T) {
	result := &analyzer.AnalysisResult{
		Files: []*analyzer.FileAnalysis{
			{
				Path: "internal/pkg/handler.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 150,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 4},
						{Complexity: 6},
					},
				},
			},
			{
				Path: "internal/utils.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 80,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 12},
						{Complexity: 8},
					},
				},
			},
			{
				Path: "cmd/main.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 450,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 22},
					},
				},
			},
			{
				Path: "nocode.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 20,
					Functions:  []*parser.FunctionMetrics{}, // Skip - no functions
				},
			},
		},
	}

	cells := buildHeatmapData(result)

	require.NotNil(t, cells)
	assert.Len(t, cells, 3) // Fourth file excluded (no functions)

	// Verify handler.go cell
	handlerCell := cells[0]
	assert.Equal(t, "internal/pkg/handler.go", handlerCell.FilePath)
	assert.Equal(t, "handler.go", handlerCell.FileName)
	assert.Equal(t, 5.0, handlerCell.Complexity) // avg (4+6)/2
	assert.Equal(t, 150, handlerCell.LOC)
	assert.Equal(t, 2, handlerCell.Functions)
	assert.Equal(t, "#10b981", handlerCell.Color) // Green (complexity <= 5)
	assert.Equal(t, 3, handlerCell.Size)          // 100-200 lines

	// Verify utils.go cell
	utilsCell := cells[1]
	assert.Equal(t, "utils.go", utilsCell.FileName)
	assert.Equal(t, 10.0, utilsCell.Complexity) // avg (12+8)/2
	assert.Equal(t, "#84cc16", utilsCell.Color) // Light green (complexity <= 10)

	// Verify main.go cell
	mainCell := cells[2]
	assert.Equal(t, "main.go", mainCell.FileName)
	assert.Equal(t, 22.0, mainCell.Complexity)
	assert.Equal(t, "#ef4444", mainCell.Color) // Red (complexity > 20)
	assert.Equal(t, 5, mainCell.Size)          // 400+ lines
}

func TestBuildHeatmapData_Nil(t *testing.T) {
	cells := buildHeatmapData(nil)
	assert.Nil(t, cells)
}

func TestGetComplexityColor(t *testing.T) {
	tests := []struct {
		complexity float64
		expected   string
	}{
		{1.0, "#10b981"},   // Green
		{5.0, "#10b981"},   // Green
		{7.0, "#84cc16"},   // Light green
		{10.0, "#84cc16"},  // Light green
		{12.0, "#f59e0b"},  // Yellow/Orange
		{15.0, "#f59e0b"},  // Yellow/Orange
		{18.0, "#fb923c"},  // Orange
		{20.0, "#fb923c"},  // Orange
		{25.0, "#ef4444"},  // Red
		{100.0, "#ef4444"}, // Red
	}

	for _, tt := range tests {
		t.Run(formatFloat(tt.complexity), func(t *testing.T) {
			result := getComplexityColor(tt.complexity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRelativeSize(t *testing.T) {
	tests := []struct {
		loc      int
		expected int
	}{
		{10, 1},   // Small
		{49, 1},   // Small
		{50, 2},   // Medium-small
		{99, 2},   // Medium-small
		{100, 3},  // Medium
		{199, 3},  // Medium
		{200, 4},  // Medium-large
		{399, 4},  // Medium-large
		{400, 5},  // Large
		{1000, 5}, // Large
	}

	for _, tt := range tests {
		t.Run(formatNumber(tt.loc), func(t *testing.T) {
			result := getRelativeSize(tt.loc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	deps := &analyzer.DependencyReport{
		Packages: []*analyzer.PackageDependencies{
			{
				PackageName:     "github.com/user/proj/internal/pkg/handler",
				TotalImports:    5,
				InternalImports: []string{"github.com/user/proj/internal/pkg/models"},
				ExternalImports: []string{"github.com/gorilla/mux"},
			},
			{
				PackageName:     "github.com/user/proj/internal/pkg/models",
				TotalImports:    2,
				InternalImports: []string{},
				ExternalImports: []string{},
			},
		},
		CircularDependencies: []*analyzer.CircularDependency{
			{
				Cycle: []string{
					"github.com/user/proj/pkg/a",
					"github.com/user/proj/pkg/b",
					"github.com/user/proj/pkg/a",
				},
			},
		},
	}

	graph := buildDependencyGraph(deps)

	require.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 1)

	// Verify nodes
	handlerNode := graph.Nodes[0]
	assert.Equal(t, "github.com/user/proj/internal/pkg/handler", handlerNode.ID)
	assert.Equal(t, "handler", handlerNode.Label) // Short label
	assert.Equal(t, "internal", handlerNode.Group)
	assert.Equal(t, 5, handlerNode.Value)
	assert.Contains(t, handlerNode.Title, "Total imports: 5")

	// Verify edge
	edge := graph.Edges[0]
	assert.Equal(t, "github.com/user/proj/internal/pkg/handler", edge.From)
	assert.Equal(t, "github.com/user/proj/internal/pkg/models", edge.To)
	assert.Equal(t, 1, edge.Width)
}

func TestBuildDependencyGraph_CircularDepsHighlighted(t *testing.T) {
	deps := &analyzer.DependencyReport{
		Packages: []*analyzer.PackageDependencies{
			{
				PackageName:     "pkg/a",
				TotalImports:    1,
				InternalImports: []string{"pkg/b"},
			},
			{
				PackageName:     "pkg/b",
				TotalImports:    1,
				InternalImports: []string{"pkg/a"},
			},
		},
		CircularDependencies: []*analyzer.CircularDependency{
			{
				Cycle: []string{"pkg/a", "pkg/b", "pkg/a"},
			},
		},
	}

	graph := buildDependencyGraph(deps)

	require.NotNil(t, graph)
	assert.Len(t, graph.Edges, 2)

	// Both edges should be highlighted as circular
	for _, edge := range graph.Edges {
		if edge.From == "pkg/a" && edge.To == "pkg/b" {
			assert.Equal(t, "#ef4444", edge.Color)
			assert.Equal(t, 3, edge.Width)
		}
		if edge.From == "pkg/b" && edge.To == "pkg/a" {
			assert.Equal(t, "#ef4444", edge.Color)
			assert.Equal(t, 3, edge.Width)
		}
	}
}

func TestBuildDependencyGraph_Nil(t *testing.T) {
	graph := buildDependencyGraph(nil)
	assert.Nil(t, graph)
}

func TestBuildDependencyGraph_EmptyPackages(t *testing.T) {
	deps := &analyzer.DependencyReport{
		Packages: []*analyzer.PackageDependencies{},
	}

	graph := buildDependencyGraph(deps)
	assert.Nil(t, graph)
}

func TestChartDataToJSON(t *testing.T) {
	data := &ChartData{
		ComplexityDist: &ComplexityDistribution{
			Ranges: []string{"1-5", "6-10"},
			Counts: []int{10, 5},
		},
	}

	json := data.toJSON()

	assert.NotEmpty(t, json)
	assert.Contains(t, json, "complexityDist")
	assert.Contains(t, json, "1-5")
	assert.Contains(t, json, "6-10")
}

func TestChartDataToJSON_Nil(t *testing.T) {
	var data *ChartData
	json := data.toJSON()
	assert.Equal(t, "null", json)
}

func TestBuildChartData(t *testing.T) {
	result := &analyzer.AnalysisResult{
		ProjectPath: "/test/project",
		Files: []*analyzer.FileAnalysis{
			{
				Path: "test.go",
				Metrics: &parser.FileMetrics{
					TotalLines: 100,
					Functions: []*parser.FunctionMetrics{
						{Complexity: 5},
						{Complexity: 10},
					},
				},
			},
		},
		Issues: []*analyzer.Issue{
			{Type: "high_complexity", Severity: "warning"},
		},
		Coverage: &analyzer.CoverageReport{
			Packages: []*analyzer.PackageCoverage{
				{PackagePath: "pkg/test", Coverage: 75.0},
			},
		},
		Dependencies: &analyzer.DependencyReport{
			Packages: []*analyzer.PackageDependencies{
				{
					PackageName:  "pkg/test",
					TotalImports: 3,
				},
			},
		},
	}

	data := buildChartData(result, nil)

	require.NotNil(t, data)
	assert.NotNil(t, data.ComplexityDist)
	assert.NotNil(t, data.IssueCounts)
	assert.NotNil(t, data.Heatmap)
	assert.NotNil(t, data.CoverageBreakdown)
	assert.NotNil(t, data.DependencyGraph)
	assert.Nil(t, data.MetricsTimeSeries) // No storage provided
}

func TestBuildChartData_Nil(t *testing.T) {
	data := buildChartData(nil, nil)
	assert.Nil(t, data)
}

func TestBuildChartData_MinimalResult(t *testing.T) {
	result := &analyzer.AnalysisResult{
		ProjectPath: "/test/project",
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
		Issues: []*analyzer.Issue{},
		// No coverage, no dependencies
	}

	data := buildChartData(result, nil)

	require.NotNil(t, data)
	assert.NotNil(t, data.ComplexityDist)
	assert.Nil(t, data.IssueCounts)     // No issues
	assert.NotNil(t, data.Heatmap)
	assert.Nil(t, data.CoverageBreakdown) // No coverage
	assert.Nil(t, data.DependencyGraph)   // No dependencies
}
