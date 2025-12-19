package detectors

import (
	"go/ast"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// Detector defines the interface for anti-pattern detection
type Detector interface {
	// Name returns the detector name
	Name() string

	// Detect analyzes a function and returns any issues found
	Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue

	// Enabled returns whether this detector is enabled
	Enabled(cfg *config.AnalysisConfig) bool
}

// Registry manages all available detectors
type Registry struct {
	detectors []Detector
	config    *config.AnalysisConfig
}

// NewRegistry creates a new detector registry with all detectors
func NewRegistry(cfg *config.AnalysisConfig) *Registry {
	return &Registry{
		config: cfg,
		detectors: []Detector{
			NewParameterCountDetector(),
			NewNestingDepthDetector(),
			NewReturnCountDetector(),
			NewMagicNumberDetector(),
			NewDuplicateErrorDetector(),
		},
	}
}

// RunAll runs all enabled detectors on a function
func (r *Registry) RunAll(file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue {
	var issues []*Issue

	for _, detector := range r.detectors {
		if detector.Enabled(r.config) {
			detectedIssues := detector.Detect(r.config, file, fn, fset, funcDecl)
			issues = append(issues, detectedIssues...)
		}
	}

	return issues
}
