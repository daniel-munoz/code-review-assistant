package reporter

import (
	"encoding/json"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
)

// ChartData contains all pre-processed data for JavaScript charts
type ChartData struct {
	ComplexityDist    *ComplexityDistribution `json:"complexityDist"`
	CoverageBreakdown *CoverageBreakdownData  `json:"coverageBreakdown"`
	IssueCounts       *IssueCountData         `json:"issueCounts"`
	Heatmap           []*HeatmapCell          `json:"heatmap"`
}

// ComplexityDistribution for histogram showing function complexity distribution
type ComplexityDistribution struct {
	Ranges []string `json:"ranges"` // ["1-5", "6-10", "11-15", "16-20", "20+"]
	Counts []int    `json:"counts"` // Number of functions in each range
}

// CoverageBreakdownData for per-package coverage visualization
type CoverageBreakdownData struct {
	Packages  []string  `json:"packages"`  // Package names
	Coverages []float64 `json:"coverages"` // Coverage percentages
	Colors    []string  `json:"colors"`    // Color codes based on threshold
}

// IssueCountData for stacked bar chart showing issues by type and severity
type IssueCountData struct {
	Types      []string `json:"types"`      // Issue type labels
	ErrorCount []int    `json:"errorCount"` // Count of errors per type
	WarnCount  []int    `json:"warnCount"`  // Count of warnings per type
	InfoCount  []int    `json:"infoCount"`  // Count of info per type
}

// HeatmapCell represents a single file in the complexity heatmap
type HeatmapCell struct {
	FilePath   string  `json:"filePath"`   // Relative file path
	FileName   string  `json:"fileName"`   // Just the filename
	Complexity float64 `json:"complexity"` // Average complexity
	LOC        int     `json:"loc"`        // Lines of code
	Functions  int     `json:"functions"`  // Number of functions
	Color      string  `json:"color"`      // Color based on complexity
	Size       int     `json:"size"`       // Relative size for grid (1-5)
}

// buildChartData transforms AnalysisResult into chart-ready data
func buildChartData(result *analyzer.AnalysisResult) *ChartData {
	if result == nil {
		return nil
	}

	data := &ChartData{
		ComplexityDist:    buildComplexityDistribution(result),
		IssueCounts:       buildIssueCountData(result.Issues),
		Heatmap:           buildHeatmapData(result),
	}

	// Only build coverage breakdown if coverage data exists
	if result.Coverage != nil {
		data.CoverageBreakdown = buildCoverageBreakdown(result.Coverage)
	}

	return data
}

// buildComplexityDistribution creates histogram data for function complexity
func buildComplexityDistribution(result *analyzer.AnalysisResult) *ComplexityDistribution {
	ranges := []string{"1-5", "6-10", "11-15", "16-20", "20+"}
	counts := make([]int, len(ranges))

	// Iterate through all files and functions
	for _, file := range result.Files {
		for _, fn := range file.Metrics.Functions {
			switch {
			case fn.Complexity <= 5:
				counts[0]++
			case fn.Complexity <= 10:
				counts[1]++
			case fn.Complexity <= 15:
				counts[2]++
			case fn.Complexity <= 20:
				counts[3]++
			default:
				counts[4]++
			}
		}
	}

	return &ComplexityDistribution{
		Ranges: ranges,
		Counts: counts,
	}
}

// buildCoverageBreakdown creates per-package coverage data
func buildCoverageBreakdown(coverage *analyzer.CoverageReport) *CoverageBreakdownData {
	if coverage == nil || len(coverage.Packages) == 0 {
		return nil
	}

	data := &CoverageBreakdownData{
		Packages:  make([]string, 0),
		Coverages: make([]float64, 0),
		Colors:    make([]string, 0),
	}

	// Only include packages with actual coverage (not skipped or errored)
	for _, pkg := range coverage.Packages {
		if pkg.Skipped || pkg.Error != "" {
			continue
		}

		data.Packages = append(data.Packages, pkg.PackagePath)
		data.Coverages = append(data.Coverages, pkg.Coverage)

		// Assign color based on coverage threshold
		color := getCoverageColor(pkg.Coverage)
		data.Colors = append(data.Colors, color)
	}

	return data
}

// getCoverageColor returns a color code based on coverage percentage
func getCoverageColor(coverage float64) string {
	switch {
	case coverage >= 80:
		return "#10b981" // Green
	case coverage >= 50:
		return "#f59e0b" // Yellow
	default:
		return "#ef4444" // Red
	}
}

// buildIssueCountData creates issue breakdown by type and severity
func buildIssueCountData(issues []*analyzer.Issue) *IssueCountData {
	if len(issues) == 0 {
		return nil
	}

	// Count issues by type and severity
	issueMap := make(map[string]map[string]int)

	for _, issue := range issues {
		if _, exists := issueMap[issue.Type]; !exists {
			issueMap[issue.Type] = make(map[string]int)
		}
		issueMap[issue.Type][issue.Severity]++
	}

	// Convert map to arrays for chart
	data := &IssueCountData{
		Types:      make([]string, 0, len(issueMap)),
		ErrorCount: make([]int, 0, len(issueMap)),
		WarnCount:  make([]int, 0, len(issueMap)),
		InfoCount:  make([]int, 0, len(issueMap)),
	}

	for issueType, severityCounts := range issueMap {
		// Format issue type for display
		label := formatIssueTypeForChart(issueType)
		data.Types = append(data.Types, label)
		data.ErrorCount = append(data.ErrorCount, severityCounts["error"])
		data.WarnCount = append(data.WarnCount, severityCounts["warning"])
		data.InfoCount = append(data.InfoCount, severityCounts["info"])
	}

	return data
}

// formatIssueTypeForChart formats issue types for chart display
func formatIssueTypeForChart(issueType string) string {
	labels := map[string]string{
		"large_file":               "Large File",
		"long_function":            "Long Function",
		"high_complexity":          "High Complexity",
		"too_many_parameters":      "Too Many Params",
		"deep_nesting":             "Deep Nesting",
		"too_many_returns":         "Too Many Returns",
		"low_coverage":             "Low Coverage",
		"too_many_imports":         "Too Many Imports",
		"too_many_external_deps":   "Too Many External Deps",
		"magic_number":             "Magic Number",
		"duplicate_error_handling": "Duplicate Error Handling",
		"circular_dependency":      "Circular Dependency",
		"low_comment_ratio":        "Low Comment Ratio",
	}

	if label, ok := labels[issueType]; ok {
		return label
	}
	return issueType
}

// buildHeatmapData creates heatmap cells for file complexity visualization
func buildHeatmapData(result *analyzer.AnalysisResult) []*HeatmapCell {
	if result == nil || len(result.Files) == 0 {
		return nil
	}

	cells := make([]*HeatmapCell, 0, len(result.Files))

	for _, file := range result.Files {
		// Skip files with no functions
		if len(file.Metrics.Functions) == 0 {
			continue
		}

		// Calculate average complexity for the file
		totalComplexity := 0
		for _, fn := range file.Metrics.Functions {
			totalComplexity += fn.Complexity
		}
		avgComplexity := float64(totalComplexity) / float64(len(file.Metrics.Functions))

		// Determine color based on complexity
		color := getComplexityColor(avgComplexity)

		// Determine relative size (1-5) based on LOC
		size := getRelativeSize(file.Metrics.TotalLines)

		// Extract filename from path
		fileName := file.Path
		if idx := len(file.Path) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if file.Path[i] == '/' {
					fileName = file.Path[i+1:]
					break
				}
			}
		}

		cell := &HeatmapCell{
			FilePath:   file.Path,
			FileName:   fileName,
			Complexity: avgComplexity,
			LOC:        file.Metrics.TotalLines,
			Functions:  len(file.Metrics.Functions),
			Color:      color,
			Size:       size,
		}

		cells = append(cells, cell)
	}

	return cells
}

// getComplexityColor returns a color based on average complexity
func getComplexityColor(complexity float64) string {
	switch {
	case complexity <= 5:
		return "#10b981" // Green
	case complexity <= 10:
		return "#84cc16" // Light green
	case complexity <= 15:
		return "#f59e0b" // Yellow/Orange
	case complexity <= 20:
		return "#fb923c" // Orange
	default:
		return "#ef4444" // Red
	}
}

// getRelativeSize returns a size class (1-5) based on lines of code
func getRelativeSize(loc int) int {
	switch {
	case loc < 50:
		return 1 // Small
	case loc < 100:
		return 2 // Medium-small
	case loc < 200:
		return 3 // Medium
	case loc < 400:
		return 4 // Medium-large
	default:
		return 5 // Large
	}
}

// toJSON converts ChartData to JSON string for embedding in HTML
func (cd *ChartData) toJSON() string {
	if cd == nil {
		return "null"
	}
	bytes, err := json.Marshal(cd)
	if err != nil {
		return "null"
	}
	return string(bytes)
}
