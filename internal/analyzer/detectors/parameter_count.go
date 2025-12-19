package detectors

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// ParameterCountDetector detects functions with too many parameters
type ParameterCountDetector struct{}

// NewParameterCountDetector creates a new parameter count detector
func NewParameterCountDetector() *ParameterCountDetector {
	return &ParameterCountDetector{}
}

// Name returns the detector name
func (d *ParameterCountDetector) Name() string {
	return "parameter_count"
}

// Enabled returns whether this detector is enabled
func (d *ParameterCountDetector) Enabled(cfg *config.AnalysisConfig) bool {
	return cfg.MaxParameters > 0
}

// Detect checks if a function has too many parameters
func (d *ParameterCountDetector) Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue {
	var issues []*Issue

	threshold := cfg.MaxParameters

	if fn.Parameters > threshold {
		issues = append(issues, &Issue{
			Severity:  "warning",
			Type:      "too_many_parameters",
			File:      file.FilePath,
			Line:      fn.StartLine,
			Function:  fn.FullName(),
			Message:   fmt.Sprintf("Function has too many parameters (%d)", fn.Parameters),
			Value:     fn.Parameters,
			Threshold: threshold,
		})
	}

	return issues
}
