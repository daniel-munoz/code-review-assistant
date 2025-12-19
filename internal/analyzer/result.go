package analyzer

import (
	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// Issue is an alias for detectors.Issue for backward compatibility
type Issue = detectors.Issue

// AnalysisResult contains the complete analysis of a codebase
type AnalysisResult struct {
	ProjectPath    string              `json:"project_path"`
	TotalFiles     int                 `json:"total_files"`
	TotalLines     int                 `json:"total_lines"`
	TotalCodeLines int                 `json:"total_code_lines"`
	TotalFunctions int                 `json:"total_functions"`
	Metrics        *AggregateMetrics   `json:"metrics"`
	Files          []*FileAnalysis     `json:"files"`
	Issues         []*Issue            `json:"issues"`
	Coverage       *CoverageReport     `json:"coverage,omitempty"`
	Dependencies   *DependencyReport   `json:"dependencies,omitempty"`
}

// AggregateMetrics contains aggregated metrics across all files
type AggregateMetrics struct {
	AverageFunctionLength float64         `json:"average_function_length"`
	FunctionLengthP95     int             `json:"function_length_p95"` // 95th percentile
	CommentRatio          float64         `json:"comment_ratio"`       // Overall comment ratio
	LargestFiles          []*FileSize     `json:"largest_files"`       // Top 10
	AverageComplexity     float64         `json:"average_complexity"`
	ComplexityP95         int             `json:"complexity_p95"` // 95th percentile
	MostComplexFunctions  []*FunctionInfo `json:"most_complex_functions"` // Top 10
}

// FileSize represents a file and its line count
type FileSize struct {
	Path  string `json:"path"`
	Lines int    `json:"lines"`
}

// FunctionInfo represents function information for reporting
type FunctionInfo struct {
	File       string `json:"file"`
	Function   string `json:"function"`
	Complexity int    `json:"complexity"`
	Lines      int    `json:"lines"`
}

// FileAnalysis contains analysis for a single file
type FileAnalysis struct {
	Path      string              `json:"path"`
	Metrics   *parser.FileMetrics `json:"metrics"`
	LargeFile bool                `json:"large_file"` // Exceeds threshold
}

// CoverageReport contains test coverage information
type CoverageReport struct {
	Packages         []*PackageCoverage `json:"packages"`
	AverageCoverage  float64            `json:"average_coverage"`
	LowCoverageCount int                `json:"low_coverage_count"`
}

// PackageCoverage represents coverage for a single package
type PackageCoverage struct {
	PackagePath string  `json:"package_path"`
	Coverage    float64 `json:"coverage"`
	Error       string  `json:"error,omitempty"`
	Skipped     bool    `json:"skipped"`
}

// DependencyReport contains dependency analysis information
type DependencyReport struct {
	Packages             []*PackageDependencies `json:"packages"`
	CircularDependencies []*CircularDependency  `json:"circular_dependencies,omitempty"`
	TotalPackages        int                    `json:"total_packages"`
	HighImportCount      int                    `json:"high_import_count"`      // Packages exceeding import threshold
	HighExternalCount    int                    `json:"high_external_count"`    // Packages with too many external deps
}

// PackageDependencies represents dependency information for a single package
type PackageDependencies struct {
	PackageName         string   `json:"package_name"`
	StdlibImports       []string `json:"stdlib_imports"`
	InternalImports     []string `json:"internal_imports"`
	ExternalImports     []string `json:"external_imports"`
	TotalImports        int      `json:"total_imports"`
	ExternalImportCount int      `json:"external_import_count"`
}

// CircularDependency represents a circular dependency between packages
type CircularDependency struct {
	Cycle []string `json:"cycle"` // The circular dependency chain
}
