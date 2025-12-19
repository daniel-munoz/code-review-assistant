package analyzer

import (
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// Analyzer defines the interface for analyzing parsed code metrics
type Analyzer interface {
	// Analyze takes parsed file metrics and produces an analysis result
	Analyze(projectPath string, metrics []*parser.FileMetrics) (*AnalysisResult, error)
}

// MetricsAnalyzer implements Analyzer for basic metrics analysis
type MetricsAnalyzer struct {
	config *config.AnalysisConfig
}

// NewAnalyzer creates a new MetricsAnalyzer with the given configuration
func NewAnalyzer(cfg *config.AnalysisConfig) Analyzer {
	return &MetricsAnalyzer{
		config: cfg,
	}
}

// Analyze performs analysis on the parsed metrics
func (ma *MetricsAnalyzer) Analyze(projectPath string, metrics []*parser.FileMetrics) (*AnalysisResult, error) {
	result := &AnalysisResult{
		ProjectPath: projectPath,
		TotalFiles:  len(metrics),
		Files:       make([]*FileAnalysis, 0, len(metrics)),
		Issues:      make([]*Issue, 0),
	}

	// Collect all functions for aggregate calculations
	var allFunctions []*parser.FunctionMetrics

	// Process each file
	for _, fileMetrics := range metrics {
		// Update totals
		result.TotalLines += fileMetrics.TotalLines
		result.TotalCodeLines += fileMetrics.CodeLines
		result.TotalFunctions += len(fileMetrics.Functions)

		// Create file analysis
		fileAnalysis := &FileAnalysis{
			Path:      fileMetrics.FilePath,
			Metrics:   fileMetrics,
			LargeFile: fileMetrics.TotalLines > ma.config.LargeFileThreshold,
		}
		result.Files = append(result.Files, fileAnalysis)

		// Check for large files
		if fileAnalysis.LargeFile {
			result.Issues = append(result.Issues, &Issue{
				Severity:  "warning",
				Type:      "large_file",
				File:      fileMetrics.FilePath,
				Line:      0,
				Message:   "File exceeds size threshold",
				Value:     fileMetrics.TotalLines,
				Threshold: ma.config.LargeFileThreshold,
			})
		}

		// Check for long functions
		for _, fn := range fileMetrics.Functions {
			allFunctions = append(allFunctions, fn)

			if fn.Lines > ma.config.LongFunctionThreshold {
				result.Issues = append(result.Issues, &Issue{
					Severity:  "warning",
					Type:      "long_function",
					File:      fileMetrics.FilePath,
					Line:      fn.StartLine,
					Function:  fn.FullName(),
					Message:   "Function exceeds length threshold",
					Value:     fn.Lines,
					Threshold: ma.config.LongFunctionThreshold,
				})
			}
		}
	}

	// Calculate aggregate metrics
	result.Metrics = calculateAggregateMetrics(metrics, allFunctions)

	// Check overall comment ratio
	if result.Metrics.CommentRatio < ma.config.MinCommentRatio {
		result.Issues = append(result.Issues, &Issue{
			Severity:  "info",
			Type:      "low_comment_ratio",
			File:      "",
			Line:      0,
			Message:   "Overall comment ratio is below recommended threshold",
			Value:     int(result.Metrics.CommentRatio * 100),
			Threshold: int(ma.config.MinCommentRatio * 100),
		})
	}

	return result, nil
}
