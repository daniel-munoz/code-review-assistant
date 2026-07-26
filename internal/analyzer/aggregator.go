package analyzer

import (
	"sort"

	"github.com/daniel-munoz/code-review-assistant/internal/constants"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// calculateAggregateMetrics computes aggregate metrics from file metrics
func calculateAggregateMetrics(fileMetrics []*parser.FileMetrics, allFunctions []*parser.FunctionMetrics) *AggregateMetrics {
	metrics := &AggregateMetrics{
		LargestFiles: make([]*FileSize, 0, constants.TopFilesLimit),
	}

	if len(fileMetrics) == 0 {
		return metrics
	}

	calculateFunctionAverages(metrics, allFunctions)
	calculateMedians(metrics, allFunctions)
	calculatePercentiles(metrics, allFunctions)
	metrics.CommentRatio = calculateCommentRatio(fileMetrics)
	metrics.LargestFiles = findLargestFiles(fileMetrics, constants.TopFilesLimit)
	metrics.MostComplexFunctions = findMostComplexFunctions(fileMetrics, constants.TopComplexFunctionsLimit)

	return metrics
}

// calculateFunctionAverages calculates average function length and complexity
func calculateFunctionAverages(metrics *AggregateMetrics, allFunctions []*parser.FunctionMetrics) {
	if len(allFunctions) == 0 {
		return
	}

	totalLines := 0
	totalComplexity := 0
	for _, fn := range allFunctions {
		totalLines += fn.Lines
		totalComplexity += fn.Complexity
	}

	metrics.AverageFunctionLength = float64(totalLines) / float64(len(allFunctions))
	metrics.AverageComplexity = float64(totalComplexity) / float64(len(allFunctions))
}

// calculatePercentiles calculates 95th percentile for function length and complexity
func calculatePercentiles(metrics *AggregateMetrics, allFunctions []*parser.FunctionMetrics) {
	if len(allFunctions) == 0 {
		return
	}

	metrics.FunctionLengthP95 = calculateP95(allFunctions, func(fn *parser.FunctionMetrics) int {
		return fn.Lines
	})

	metrics.ComplexityP95 = calculateP95(allFunctions, func(fn *parser.FunctionMetrics) int {
		return fn.Complexity
	})
}

// calculateMedians calculates median function length and complexity
func calculateMedians(metrics *AggregateMetrics, allFunctions []*parser.FunctionMetrics) {
	if len(allFunctions) == 0 {
		return
	}

	metrics.MedianFunctionLength = calculateMedian(allFunctions, func(fn *parser.FunctionMetrics) int {
		return fn.Lines
	})

	metrics.MedianComplexity = calculateMedian(allFunctions, func(fn *parser.FunctionMetrics) int {
		return fn.Lines
	})
}

// calculateMedian calculates the median of a metric. For an even number of
// values it returns the mean of the two middle values.
func calculateMedian(functions []*parser.FunctionMetrics, getValue func(*parser.FunctionMetrics) int) int {
	values := make([]int, len(functions))
	for i, fn := range functions {
		values[i] = getValue(fn)
	}

	mid := len(values) / 2
	if len(values)%2 == 0 {
		return (values[mid-1] + values[mid]) / 2
	}
	return values[mid]
}

// calculateP95 calculates the 95th percentile of a metric
func calculateP95(functions []*parser.FunctionMetrics, getValue func(*parser.FunctionMetrics) int) int {
	values := make([]int, len(functions))
	for i, fn := range functions {
		values[i] = getValue(fn)
	}

	sort.Ints(values)
	p95Index := int(float64(len(values)) * constants.Percentile95)
	if p95Index >= len(values) {
		p95Index = len(values) - 1
	}

	return values[p95Index]
}

// calculateCommentRatio calculates the overall comment ratio across all files
func calculateCommentRatio(fileMetrics []*parser.FileMetrics) float64 {
	totalLines := 0
	totalComments := 0

	for _, fm := range fileMetrics {
		totalLines += fm.TotalLines
		totalComments += fm.CommentLines
	}

	if totalLines == 0 {
		return 0
	}

	return float64(totalComments) / float64(totalLines)
}

// findLargestFiles returns the top N largest files by line count
func findLargestFiles(fileMetrics []*parser.FileMetrics, limit int) []*FileSize {
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

	if len(fileSizes) < limit {
		limit = len(fileSizes)
	}

	return fileSizes[:limit]
}

// findMostComplexFunctions returns the top N most complex functions
func findMostComplexFunctions(fileMetrics []*parser.FileMetrics, limit int) []*FunctionInfo {
	var functions []*FunctionInfo

	for _, fm := range fileMetrics {
		for _, fn := range fm.Functions {
			functions = append(functions, &FunctionInfo{
				File:       fm.FilePath,
				Function:   fn.FullName(),
				Complexity: fn.Complexity,
				Lines:      fn.Lines,
			})
		}
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Complexity > functions[j].Complexity
	})

	if len(functions) < limit {
		limit = len(functions)
	}

	return functions[:limit]
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
