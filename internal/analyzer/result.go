package analyzer

import "github.com/daniel-munoz/code-review-assistant/internal/parser"

// AnalysisResult contains the complete analysis of a codebase
type AnalysisResult struct {
	ProjectPath    string            `json:"project_path"`
	TotalFiles     int               `json:"total_files"`
	TotalLines     int               `json:"total_lines"`
	TotalCodeLines int               `json:"total_code_lines"`
	TotalFunctions int               `json:"total_functions"`
	Metrics        *AggregateMetrics `json:"metrics"`
	Files          []*FileAnalysis   `json:"files"`
	Issues         []*Issue          `json:"issues"`
}

// AggregateMetrics contains aggregated metrics across all files
type AggregateMetrics struct {
	AverageFunctionLength float64     `json:"average_function_length"`
	FunctionLengthP95     int         `json:"function_length_p95"` // 95th percentile
	CommentRatio          float64     `json:"comment_ratio"`       // Overall comment ratio
	LargestFiles          []*FileSize `json:"largest_files"`       // Top 10
}

// FileSize represents a file and its line count
type FileSize struct {
	Path  string `json:"path"`
	Lines int    `json:"lines"`
}

// FileAnalysis contains analysis for a single file
type FileAnalysis struct {
	Path      string              `json:"path"`
	Metrics   *parser.FileMetrics `json:"metrics"`
	LargeFile bool                `json:"large_file"` // Exceeds threshold
}

// Issue represents a code quality issue found during analysis
type Issue struct {
	Severity  string `json:"severity"`  // "warning", "info", "error"
	Type      string `json:"type"`      // "large_file", "long_function", "low_comment_ratio"
	File      string `json:"file"`      // File path
	Line      int    `json:"line"`      // Line number (0 if not applicable)
	Function  string `json:"function"`  // Function name (empty if not applicable)
	Message   string `json:"message"`   // Human-readable message
	Value     int    `json:"value"`     // Actual value (e.g., line count)
	Threshold int    `json:"threshold"` // Threshold value
}
