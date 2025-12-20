package detectors

import (
	"go/ast"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// DuplicateErrorDetector detects repetitive error handling patterns
type DuplicateErrorDetector struct{}

// NewDuplicateErrorDetector creates a new duplicate error detector
func NewDuplicateErrorDetector() *DuplicateErrorDetector {
	return &DuplicateErrorDetector{}
}

// Name returns the detector name
func (d *DuplicateErrorDetector) Name() string {
	return "duplicate_errors"
}

// Enabled returns whether this detector is enabled
func (d *DuplicateErrorDetector) Enabled(cfg *config.AnalysisConfig) bool {
	return cfg.DetectDuplicateErrors
}

// Detect checks for repetitive error handling patterns
func (d *DuplicateErrorDetector) Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue {
	if funcDecl.Body == nil {
		return nil
	}

	errorCheckCount := countErrorChecks(funcDecl.Body)

	// Flag functions with more than 5 error checks
	if errorCheckCount > 5 {
		return []*Issue{
			{
				Severity:  "info",
				Type:      "duplicate_error_handling",
				File:      file.FilePath,
				Line:      fn.StartLine,
				Function:  fn.FullName(),
				Message:   "Function has repetitive error handling patterns - consider refactoring",
				Value:     errorCheckCount,
				Threshold: 5,
			},
		}
	}

	return nil
}

// countErrorChecks counts "if err != nil" patterns
func countErrorChecks(body *ast.BlockStmt) int {
	count := 0

	ast.Inspect(body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		// Check if condition is "err != nil" or "nil != err"
		if isErrorCheck(ifStmt.Cond) {
			count++
		}

		return true
	})

	return count
}

// isErrorCheck determines if an AST expression is an error nil-check pattern.
//
// This function detects the common Go idiom of checking if an error is not nil.
// It recognizes both standard patterns:
//   - "err != nil" (left-hand error, right-hand nil)
//   - "nil != err" (left-hand nil, right-hand error)
//
// The check is strict: it only matches binary expressions with the NEQ operator
// where one operand is an identifier named "err" and the other is the identifier
// "nil". This conservative approach avoids false positives from custom error
// variables or comparison expressions.
//
// Returns true if the expression matches either error-checking pattern,
// false otherwise.
func isErrorCheck(expr ast.Expr) bool {
	binExpr, ok := expr.(*ast.BinaryExpr)
	if !ok || binExpr.Op != token.NEQ {
		return false
	}

	// Check for "err != nil" pattern
	if ident, ok := binExpr.X.(*ast.Ident); ok && ident.Name == "err" {
		if nilIdent, ok := binExpr.Y.(*ast.Ident); ok && nilIdent.Name == "nil" {
			return true
		}
	}

	// Check for "nil != err" pattern
	if nilIdent, ok := binExpr.X.(*ast.Ident); ok && nilIdent.Name == "nil" {
		if ident, ok := binExpr.Y.(*ast.Ident); ok && ident.Name == "err" {
			return true
		}
	}

	return false
}
