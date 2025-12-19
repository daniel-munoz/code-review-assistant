package detectors

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// ReturnCountDetector detects functions with too many return statements
type ReturnCountDetector struct{}

// NewReturnCountDetector creates a new return count detector
func NewReturnCountDetector() *ReturnCountDetector {
	return &ReturnCountDetector{}
}

// Name returns the detector name
func (d *ReturnCountDetector) Name() string {
	return "return_count"
}

// Enabled returns whether this detector is enabled
func (d *ReturnCountDetector) Enabled(cfg *config.AnalysisConfig) bool {
	return cfg.MaxReturnStatements > 0
}

// Detect checks if a function has too many return statements
func (d *ReturnCountDetector) Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue {
	if funcDecl.Body == nil {
		return nil
	}

	returnCount := countReturnStatements(funcDecl.Body)
	threshold := cfg.MaxReturnStatements

	if returnCount > threshold {
		return []*Issue{
			{
				Severity:  "info",
				Type:      "too_many_returns",
				File:      file.FilePath,
				Line:      fn.StartLine,
				Function:  fn.FullName(),
				Message:   fmt.Sprintf("Function has %d return statements", returnCount),
				Value:     returnCount,
				Threshold: threshold,
			},
		}
	}

	return nil
}

// countReturnStatements counts return statements in a function (excluding nested functions)
func countReturnStatements(body *ast.BlockStmt) int {
	count := 0

	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.ReturnStmt); ok {
			count++
		}
		// Don't count returns in nested functions
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		return true
	})

	return count
}
