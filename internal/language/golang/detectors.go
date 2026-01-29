package golang

import (
	"go/ast"
	goparser "go/parser"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// GoDetectorRunner implements language.DetectorRunner for Go source code.
//
// This runner re-parses Go files to obtain AST access and runs all enabled
// detectors on each function declaration. It encapsulates the Go-specific
// AST walking logic that was previously in analyzer.runDetectors().
type GoDetectorRunner struct {
	registry *detectors.Registry
	config   *config.AnalysisConfig
}

// NewGoDetectorRunner creates a new detector runner for Go source code.
func NewGoDetectorRunner(cfg *config.AnalysisConfig) *GoDetectorRunner {
	return &GoDetectorRunner{
		registry: detectors.NewRegistry(cfg),
		config:   cfg,
	}
}

// RunDetectors re-parses a Go file and runs all enabled detectors.
//
// This method:
//  1. Re-parses the file to obtain a Go AST
//  2. Walks all function declarations
//  3. Matches each AST declaration to its corresponding FunctionMetrics
//  4. Runs all enabled detectors on each matched function
//
// Returns nil if the file cannot be parsed (e.g., syntax errors).
func (r *GoDetectorRunner) RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue {
	// Re-parse file to get AST
	fset := token.NewFileSet()
	node, err := goparser.ParseFile(fset, file.FilePath, nil, goparser.ParseComments)
	if err != nil {
		// Skip detector analysis if parse fails
		return nil
	}

	var issues []*detectors.Issue

	// Run detectors on each function declaration
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Find matching function metrics by name and start line
		for _, fn := range file.Functions {
			declLine := fset.Position(funcDecl.Pos()).Line
			if funcDecl.Name.Name == fn.Name && declLine == fn.StartLine {
				detectorIssues := r.registry.RunAll(file, fn, fset, funcDecl)
				issues = append(issues, detectorIssues...)
				break
			}
		}
	}

	return issues
}
