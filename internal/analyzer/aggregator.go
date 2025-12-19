package analyzer

import (
	"sort"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// calculateAggregateMetrics computes aggregate metrics from file metrics
func calculateAggregateMetrics(fileMetrics []*parser.FileMetrics, allFunctions []*parser.FunctionMetrics) *AggregateMetrics {
	metrics := &AggregateMetrics{
		LargestFiles: make([]*FileSize, 0, 10),
	}

	if len(fileMetrics) == 0 {
		return metrics
	}

	// Calculate average function length
	if len(allFunctions) > 0 {
		totalLines := 0
		for _, fn := range allFunctions {
			totalLines += fn.Lines
		}
		metrics.AverageFunctionLength = float64(totalLines) / float64(len(allFunctions))
	}

	// Calculate 95th percentile of function length
	if len(allFunctions) > 0 {
		lengths := make([]int, len(allFunctions))
		for i, fn := range allFunctions {
			lengths[i] = fn.Lines
		}
		sort.Ints(lengths)
		p95Index := int(float64(len(lengths)) * 0.95)
		if p95Index >= len(lengths) {
			p95Index = len(lengths) - 1
		}
		metrics.FunctionLengthP95 = lengths[p95Index]
	}

	// Calculate overall comment ratio
	totalLines := 0
	totalComments := 0
	for _, fm := range fileMetrics {
		totalLines += fm.TotalLines
		totalComments += fm.CommentLines
	}
	if totalLines > 0 {
		metrics.CommentRatio = float64(totalComments) / float64(totalLines)
	}

	// Find largest files (top 10)
	fileSizes := make([]*FileSize, len(fileMetrics))
	for i, fm := range fileMetrics {
		fileSizes[i] = &FileSize{
			Path:  fm.FilePath,
			Lines: fm.TotalLines,
		}
	}
	sort.Slice(fileSizes, func(i, j int) bool {
		return fileSizes[i].Lines > fileSizes[j].Lines
	})

	// Take top 10 (or fewer if less than 10 files)
	maxFiles := 10
	if len(fileSizes) < maxFiles {
		maxFiles = len(fileSizes)
	}
	metrics.LargestFiles = fileSizes[:maxFiles]

	return metrics
}

// SortIssuesBySeverity sorts issues by severity and then by file
func SortIssuesBySeverity(issues []*Issue) {
	severityOrder := map[string]int{
		"error":   0,
		"warning": 1,
		"info":    2,
	}

	sort.Slice(issues, func(i, j int) bool {
		// First by severity
		if severityOrder[issues[i].Severity] != severityOrder[issues[j].Severity] {
			return severityOrder[issues[i].Severity] < severityOrder[issues[j].Severity]
		}
		// Then by file
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		// Then by line
		return issues[i].Line < issues[j].Line
	})
}
