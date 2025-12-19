package detectors

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// MagicNumberDetector detects magic numbers (numeric literals)
type MagicNumberDetector struct{}

// NewMagicNumberDetector creates a new magic number detector
func NewMagicNumberDetector() *MagicNumberDetector {
	return &MagicNumberDetector{}
}

// Name returns the detector name
func (d *MagicNumberDetector) Name() string {
	return "magic_numbers"
}

// Enabled returns whether this detector is enabled
func (d *MagicNumberDetector) Enabled(cfg *config.AnalysisConfig) bool {
	return cfg.DetectMagicNumbers
}

// Detect checks for magic numbers in a function
func (d *MagicNumberDetector) Detect(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, fset *token.FileSet, funcDecl *ast.FuncDecl) []*Issue {
	if funcDecl.Body == nil {
		return nil
	}

	var issues []*Issue

	// Track magic numbers to avoid duplicates
	seen := make(map[string]bool)

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || (lit.Kind != token.INT && lit.Kind != token.FLOAT) {
			return true
		}

		// Ignore allowed numbers: 0, 1, -1
		if isAllowedNumber(lit.Value) {
			return true
		}

		// Check if in const declaration (not a magic number)
		if isInConstDecl(n, funcDecl.Body) {
			return true
		}

		// Avoid duplicate reporting
		key := lit.Value + ":" + strconv.Itoa(fset.Position(lit.Pos()).Line)
		if seen[key] {
			return true
		}
		seen[key] = true

		issues = append(issues, &Issue{
			Severity:  "info",
			Type:      "magic_number",
			File:      file.FilePath,
			Line:      fset.Position(lit.Pos()).Line,
			Function:  fn.FullName(),
			Message:   "Magic number should be replaced with a named constant: " + lit.Value,
			Value:     0,
			Threshold: 0,
		})

		return true
	})

	return issues
}

func isAllowedNumber(value string) bool {
	return value == "0" || value == "1" || value == "-1" ||
		value == "0.0" || value == "1.0" || value == "-1.0"
}

func isInConstDecl(n ast.Node, body *ast.BlockStmt) bool {
	// Check if this literal is part of a const declaration
	var inConst bool

	ast.Inspect(body, func(node ast.Node) bool {
		if genDecl, ok := node.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			// Check if n is within this const declaration
			if genDecl.Pos() <= n.Pos() && n.Pos() <= genDecl.End() {
				inConst = true
				return false
			}
		}
		return true
	})

	return inConst
}
