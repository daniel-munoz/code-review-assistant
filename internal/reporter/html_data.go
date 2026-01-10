package reporter

import (
	"context"
	"encoding/json"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/storage"
)

// ChartData contains all pre-processed data for JavaScript charts
type ChartData struct {
	ComplexityDist    *ComplexityDistribution  `json:"complexityDist"`
	CoverageBreakdown *CoverageBreakdownData   `json:"coverageBreakdown"`
	IssueCounts       *IssueCountData          `json:"issueCounts"`
	Heatmap           []*HeatmapCell           `json:"heatmap"`
	DependencyGraph   *DependencyGraphData     `json:"dependencyGraph"`
	MetricsTimeSeries *MetricsTimeSeriesData   `json:"metricsTimeSeries,omitempty"`
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

// DependencyGraphData for network visualization
type DependencyGraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a package in the dependency graph
type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"` // "stdlib", "internal", "external"
	Value int    `json:"value"` // Size based on import count
	Title string `json:"title"` // Tooltip text
}

// GraphEdge represents a dependency relationship
type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Color string `json:"color,omitempty"` // Red for circular deps
	Width int    `json:"width"`           // 1-3
}

// buildChartData transforms AnalysisResult into chart-ready data
func buildChartData(result *analyzer.AnalysisResult, store storage.Storage) *ChartData {
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

	// Only build dependency graph if dependency data exists
	if result.Dependencies != nil {
		data.DependencyGraph = buildDependencyGraph(result.Dependencies)
	}

	// Only build time series if storage is available
	if store != nil {
		data.MetricsTimeSeries = buildTimeSeriesData(store, result.ProjectPath)
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

// buildDependencyGraph creates network graph data from dependency analysis
func buildDependencyGraph(deps *analyzer.DependencyReport) *DependencyGraphData {
	if deps == nil || len(deps.Packages) == 0 {
		return nil
	}

	graph := &DependencyGraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	nodeMap := make(map[string]bool)
	circularEdges := make(map[string]bool)

	// Build set of circular dependency edges for highlighting
	for _, circular := range deps.CircularDependencies {
		if len(circular.Cycle) < 2 {
			continue
		}
		// Mark all edges in the circular chain
		for i := 0; i < len(circular.Cycle)-1; i++ {
			edgeKey := circular.Cycle[i] + "->" + circular.Cycle[i+1]
			circularEdges[edgeKey] = true
		}
	}

	// Create nodes for each package
	for _, pkg := range deps.Packages {
		if !nodeMap[pkg.PackageName] {
			// Extract short label from full package path
			label := pkg.PackageName
			if idx := len(label) - 1; idx >= 0 {
				for i := idx; i >= 0; i-- {
					if label[i] == '/' {
						label = label[i+1:]
						break
					}
				}
			}

			// Determine group (for coloring)
			group := "internal"
			if len(pkg.ExternalImports) > 0 && len(pkg.InternalImports) == 0 {
				group = "external"
			}

			// Create tooltip with details
			title := pkg.PackageName + "\\n" +
				"Total imports: " + formatNumber(pkg.TotalImports) + "\\n" +
				"Internal: " + formatNumber(len(pkg.InternalImports)) + "\\n" +
				"External: " + formatNumber(len(pkg.ExternalImports))

			node := GraphNode{
				ID:    pkg.PackageName,
				Label: label,
				Group: group,
				Value: pkg.TotalImports,
				Title: title,
			}
			graph.Nodes = append(graph.Nodes, node)
			nodeMap[pkg.PackageName] = true
		}

		// Create edges for internal dependencies
		for _, imp := range pkg.InternalImports {
			// Add imported package as node if not exists
			if !nodeMap[imp] {
				label := imp
				if idx := len(label) - 1; idx >= 0 {
					for i := idx; i >= 0; i-- {
						if label[i] == '/' {
							label = label[i+1:]
							break
						}
					}
				}

				node := GraphNode{
					ID:    imp,
					Label: label,
					Group: "internal",
					Value: 1,
					Title: imp,
				}
				graph.Nodes = append(graph.Nodes, node)
				nodeMap[imp] = true
			}

			// Create edge
			edgeKey := pkg.PackageName + "->" + imp
			edge := GraphEdge{
				From:  pkg.PackageName,
				To:    imp,
				Width: 1,
			}

			// Highlight circular dependencies in red
			if circularEdges[edgeKey] {
				edge.Color = "#ef4444" // Red
				edge.Width = 3
			}

			graph.Edges = append(graph.Edges, edge)
		}
	}

	return graph
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

// MetricsTimeSeriesData for line chart showing metrics over time
type MetricsTimeSeriesData struct {
	Labels       []string  `json:"labels"`       // Timestamps (formatted)
	Complexity   []float64 `json:"complexity"`   // Average complexity over time
	Coverage     []float64 `json:"coverage"`     // Coverage percentage over time
	IssueCount   []int     `json:"issueCount"`   // Total issues over time
	TotalLines   []int     `json:"totalLines"`   // Total lines of code over time
	RawTimestamps []string `json:"timestamps"`   // ISO timestamps for tooltips
}

// buildTimeSeriesData creates time series data from historical reports
func buildTimeSeriesData(store storage.Storage, projectPath string) *MetricsTimeSeriesData {
	if store == nil {
		return nil
	}

	// Query last 20 reports
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := storage.ListOptions{
		Limit: 20,
	}

	metadata, err := store.List(ctx, projectPath, opts)
	if err != nil || len(metadata) == 0 {
		return nil
	}

	// Need at least 2 data points for a meaningful trend
	if len(metadata) < 2 {
		return nil
	}

	// Reverse to get chronological order (List returns newest first)
	for i := 0; i < len(metadata)/2; i++ {
		j := len(metadata) - i - 1
		metadata[i], metadata[j] = metadata[j], metadata[i]
	}

	data := &MetricsTimeSeriesData{
		Labels:        make([]string, 0, len(metadata)),
		Complexity:    make([]float64, 0, len(metadata)),
		Coverage:      make([]float64, 0, len(metadata)),
		IssueCount:    make([]int, 0, len(metadata)),
		TotalLines:    make([]int, 0, len(metadata)),
		RawTimestamps: make([]string, 0, len(metadata)),
	}

	for _, meta := range metadata {
		// Format timestamp for display
		label := meta.Timestamp.Format("Jan 2 15:04")
		data.Labels = append(data.Labels, label)
		data.RawTimestamps = append(data.RawTimestamps, meta.Timestamp.Format(time.RFC3339))

		// Extract metrics from metadata
		data.Complexity = append(data.Complexity, meta.AvgComplexity)
		data.Coverage = append(data.Coverage, meta.AvgCoverage)
		data.IssueCount = append(data.IssueCount, meta.IssueCount)
		data.TotalLines = append(data.TotalLines, meta.TotalLines)
	}

	return data
}
