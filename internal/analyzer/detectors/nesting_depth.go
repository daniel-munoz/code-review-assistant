package detectors

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// NestingDepthDetector detects functions with excessive nesting
type NestingDepthDetector struct{}

// NewNestingDepthDetector creates a new nesting depth detector
func NewNestingDepthDetector() *NestingDepthDetector {
	return &NestingDepthDetector{}
}

// Name returns the detector name
func (d *NestingDepthDetector) Name() string {
	return "nesting_depth"
}

// Enabled returns whether this detector is enabled
func (d *NestingDepthDetector) Enabled(cfg *config.AnalysisConfig) bool {
	return cfg.MaxNestingDepth > 0
}

// Detect checks if a function has excessive nesting depth
func (d *NestingDepthDetector) Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue {
	if funcDecl.Body == nil {
		return nil
	}

	maxDepth := calculateMaxNestingDepth(funcDecl.Body)
	threshold := cfg.MaxNestingDepth

	if maxDepth > threshold {
		return []*Issue{
			{
				Severity:  "warning",
				Type:      "deep_nesting",
				File:      file.FilePath,
				Line:      fn.StartLine,
				Function:  fn.FullName(),
				Message:   fmt.Sprintf("Function has deep nesting (depth: %d)", maxDepth),
				Value:     maxDepth,
				Threshold: threshold,
			},
		}
	}

	return nil
}

// calculateMaxNestingDepth calculates the maximum nesting depth in a function
func calculateMaxNestingDepth(body *ast.BlockStmt) int {
	maxDepth := 0

	var traverse func(ast.Node, int)
	traverse = func(n ast.Node, currentDepth int) {
		if currentDepth > maxDepth {
			maxDepth = currentDepth
		}

		ast.Inspect(n, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.SelectStmt, *ast.TypeSwitchStmt:
				// Increment depth for block statements
				if block := getBlockStmt(node); block != nil {
					traverse(block, currentDepth+1)
					return false // Don't recurse automatically
				}
			}
			return true
		})
	}

	traverse(body, 0)
	return maxDepth
}

func getBlockStmt(node ast.Node) *ast.BlockStmt {
	switch n := node.(type) {
	case *ast.IfStmt:
		return n.Body
	case *ast.ForStmt:
		return n.Body
	case *ast.RangeStmt:
		return n.Body
	case *ast.SwitchStmt:
		return n.Body
	case *ast.SelectStmt:
		return n.Body
	case *ast.TypeSwitchStmt:
		return n.Body
	}
	return nil
}
