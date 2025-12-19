package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// parseGoFile parses a Go source file and extracts metrics using AST
func parseGoFile(path string) (*FileMetrics, error) {
	// Create a new file set for position information
	fset := token.NewFileSet()

	// Parse the file with comments
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Initialize metrics
	metrics := &FileMetrics{
		FilePath:    path,
		PackageName: node.Name.Name,
		Functions:   []*FunctionMetrics{},
		Imports:     []string{},
	}

	// Extract imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		metrics.Imports = append(metrics.Imports, importPath)
	}

	// Create visitor to collect function metrics
	visitor := &metricsVisitor{
		fset:    fset,
		metrics: metrics,
	}

	// Walk the AST to collect functions
	ast.Walk(visitor, node)

	// Count lines (total, code, comment, blank)
	totalLines, codeLines, commentLines, blankLines, err := countLinesInFile(path, node, fset)
	if err != nil {
		return nil, fmt.Errorf("failed to count lines: %w", err)
	}

	metrics.TotalLines = totalLines
	metrics.CodeLines = codeLines
	metrics.CommentLines = commentLines
	metrics.BlankLines = blankLines

	return metrics, nil
}

// metricsVisitor is an AST visitor that collects function metrics
type metricsVisitor struct {
	fset    *token.FileSet
	metrics *FileMetrics
}

// Visit implements ast.Visitor interface
func (v *metricsVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	// Look for function declarations
	funcDecl, ok := node.(*ast.FuncDecl)
	if !ok {
		return v
	}

	fm := &FunctionMetrics{
		Name:      funcDecl.Name.Name,
		StartLine: v.fset.Position(funcDecl.Pos()).Line,
		EndLine:   v.fset.Position(funcDecl.End()).Line,
	}

	// Calculate lines in function
	fm.Lines = fm.EndLine - fm.StartLine + 1

	// Extract receiver type if this is a method
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		fm.ReceiverType = exprToString(funcDecl.Recv.List[0].Type)
	}

	// Count parameters
	if funcDecl.Type.Params != nil {
		fm.Parameters = countFields(funcDecl.Type.Params.List)
	}

	// Count return values
	if funcDecl.Type.Results != nil {
		fm.ReturnValues = countFields(funcDecl.Type.Results.List)
	}

	// Calculate simple cyclomatic complexity
	if funcDecl.Body != nil {
		fm.Complexity = calculateComplexity(funcDecl.Body)
	}

	v.metrics.Functions = append(v.metrics.Functions, fm)

	return v
}

// exprToString converts an AST expression to a string representation
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.IndexExpr:
		return exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	default:
		return "unknown"
	}
}

// countFields counts the number of fields in a field list (for parameters/returns)
func countFields(fields []*ast.Field) int {
	count := 0
	for _, field := range fields {
		if len(field.Names) == 0 {
			// Unnamed parameter/return
			count++
		} else {
			// Named parameters/returns
			count += len(field.Names)
		}
	}
	return count
}

// calculateComplexity calculates cyclomatic complexity using McCabe's method
// Complexity = 1 (base) + number of decision points
// Decision points: if, for, range, case (non-default), select case, &&, ||
func calculateComplexity(body *ast.BlockStmt) int {
	complexity := 1 // Base complexity

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			complexity++
			// Note: else-if is handled by nested IfStmt

		case *ast.ForStmt:
			complexity++

		case *ast.RangeStmt:
			complexity++

		case *ast.SwitchStmt:
			// Counted through CaseClause below

		case *ast.TypeSwitchStmt:
			// Counted through CaseClause below

		case *ast.CaseClause:
			// Each case adds a decision point (except default)
			if node.List != nil && len(node.List) > 0 {
				complexity++
			}

		case *ast.CommClause:
			// Select case statements
			if node.Comm != nil {
				complexity++
			}

		case *ast.BinaryExpr:
			// Count logical operators (they create additional paths)
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	return complexity
}

// countLinesInFile counts different types of lines in a file
func countLinesInFile(path string, node *ast.File, fset *token.FileSet) (total, code, comment, blank int, err error) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to read file: %w", err)
	}

	lines := bytes.Split(content, []byte("\n"))
	total = len(lines)

	// Build a set of comment line numbers from the AST
	commentLines := make(map[int]bool)
	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			startLine := fset.Position(comment.Pos()).Line
			endLine := fset.Position(comment.End()).Line
			for lineNum := startLine; lineNum <= endLine; lineNum++ {
				commentLines[lineNum] = true
			}
		}
	}

	// Classify each line
	for i, line := range lines {
		lineNum := i + 1
		trimmed := bytes.TrimSpace(line)

		if len(trimmed) == 0 {
			blank++
		} else if commentLines[lineNum] {
			comment++
		} else {
			code++
		}
	}

	return total, code, comment, blank, nil
}
