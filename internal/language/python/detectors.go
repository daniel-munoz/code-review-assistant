// Package python provides Python language support for code analysis.
package python

import (
	"fmt"
	"os"
	"regexp"

	sitter "github.com/tree-sitter/go-tree-sitter"
	pythonlang "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// PythonDetectorRunner implements language.DetectorRunner for Python source code.
//
// This runner uses tree-sitter to parse Python files and run anti-pattern
// detection, similar to how GoDetectorRunner works with Go AST.
type PythonDetectorRunner struct {
	config   *config.AnalysisConfig
	language *sitter.Language
}

// NewDetectorRunner creates a new detector runner for Python source code.
func NewDetectorRunner(cfg *config.AnalysisConfig) *PythonDetectorRunner {
	return &PythonDetectorRunner{
		config:   cfg,
		language: sitter.NewLanguage(pythonlang.Language()),
	}
}

// RunDetectors parses a Python file and runs all enabled detectors.
func (r *PythonDetectorRunner) RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue {
	// Read file content
	content, err := os.ReadFile(file.FilePath)
	if err != nil {
		return nil
	}

	// Parse with tree-sitter
	tsParser := sitter.NewParser()
	defer tsParser.Close()
	tsParser.SetLanguage(r.language)

	tree := tsParser.Parse(content, nil)
	defer tree.Close()

	root := tree.RootNode()

	var issues []*detectors.Issue

	// Run detectors on each function
	for _, fn := range file.Functions {
		// Parameter count detector
		if cfg.MaxParameters > 0 && fn.Parameters > cfg.MaxParameters {
			issues = append(issues, &detectors.Issue{
				Severity:  "warning",
				Type:      "too_many_parameters",
				File:      file.FilePath,
				Line:      fn.StartLine,
				Function:  fn.FullName(),
				Message:   fmt.Sprintf("Function has too many parameters (%d)", fn.Parameters),
				Value:     fn.Parameters,
				Threshold: cfg.MaxParameters,
			})
		}

		// Find the function node for AST-based detectors
		funcNode := findFunctionNode(root, content, fn.Name, fn.StartLine)
		if funcNode == nil {
			continue
		}

		bodyNode := funcNode.ChildByFieldName("body")
		if bodyNode == nil {
			continue
		}

		// Nesting depth detector
		if cfg.MaxNestingDepth > 0 {
			maxDepth := calculateMaxNestingDepth(bodyNode)
			if maxDepth > cfg.MaxNestingDepth {
				issues = append(issues, &detectors.Issue{
					Severity:  "warning",
					Type:      "deep_nesting",
					File:      file.FilePath,
					Line:      fn.StartLine,
					Function:  fn.FullName(),
					Message:   fmt.Sprintf("Function has deep nesting (depth: %d)", maxDepth),
					Value:     maxDepth,
					Threshold: cfg.MaxNestingDepth,
				})
			}
		}

		// Return count detector
		if cfg.MaxReturnStatements > 0 {
			returnCount := countReturnStatements(bodyNode)
			if returnCount > cfg.MaxReturnStatements {
				issues = append(issues, &detectors.Issue{
					Severity:  "info",
					Type:      "too_many_returns",
					File:      file.FilePath,
					Line:      fn.StartLine,
					Function:  fn.FullName(),
					Message:   fmt.Sprintf("Function has %d return statements", returnCount),
					Value:     returnCount,
					Threshold: cfg.MaxReturnStatements,
				})
			}
		}

		// Magic number detector
		if cfg.DetectMagicNumbers {
			magicIssues := detectMagicNumbers(bodyNode, content, file.FilePath, fn)
			issues = append(issues, magicIssues...)
		}
	}

	return issues
}

// findFunctionNode locates a function_definition node by name and start line.
func findFunctionNode(root *sitter.Node, content []byte, name string, startLine int) *sitter.Node {
	var result *sitter.Node

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if result != nil {
			return
		}

		if n.Kind() == "function_definition" {
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				nodeName := string(content[nameNode.StartByte():nameNode.EndByte()])
				nodeLine := int(n.StartPosition().Row) + 1
				if nodeName == name && nodeLine == startLine {
					result = n
					return
				}
			}
		}

		// Recurse into children
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}

	walk(root)
	return result
}

// calculateMaxNestingDepth calculates the maximum nesting depth in a Python function body.
func calculateMaxNestingDepth(node *sitter.Node) int {
	maxDepth := 0

	var traverse func(n *sitter.Node, currentDepth int)
	traverse = func(n *sitter.Node, currentDepth int) {
		nodeKind := n.Kind()

		// Check if this is a nesting construct
		isNestingConstruct := false
		switch nodeKind {
		case "if_statement", "for_statement", "while_statement", "try_statement",
			"with_statement", "match_statement":
			isNestingConstruct = true
		}

		newDepth := currentDepth
		if isNestingConstruct {
			newDepth = currentDepth + 1
			if newDepth > maxDepth {
				maxDepth = newDepth
			}
		}

		// Recurse into children
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child != nil {
				traverse(child, newDepth)
			}
		}
	}

	traverse(node, 0)
	return maxDepth
}

// countReturnStatements counts return statements in a Python function body.
func countReturnStatements(node *sitter.Node) int {
	count := 0

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "return_statement" {
			count++
		}

		// Don't count returns in nested functions
		if n.Kind() == "function_definition" || n.Kind() == "lambda" {
			return
		}

		// Recurse into children
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}

	walk(node)
	return count
}

// detectMagicNumbers finds numeric literals that aren't common constants and returns issues.
func detectMagicNumbers(node *sitter.Node, content []byte, filePath string, fn *parser.FunctionMetrics) []*detectors.Issue {
	var issues []*detectors.Issue
	seen := make(map[string]bool)

	// Common acceptable numbers
	commonNumbers := map[string]bool{
		"0": true, "1": true, "-1": true,
		"0.0": true, "1.0": true, "-1.0": true,
	}

	// Pattern to detect constant assignment (UPPER_CASE = number)
	constantPattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

	var walk func(n *sitter.Node, inConstant bool)
	walk = func(n *sitter.Node, inConstant bool) {
		nodeKind := n.Kind()

		// Check if we're in a constant assignment
		if nodeKind == "assignment" {
			// Check if left side is an UPPER_CASE identifier
			for i := uint(0); i < n.ChildCount(); i++ {
				child := n.Child(i)
				if child != nil && child.Kind() == "identifier" {
					name := string(content[child.StartByte():child.EndByte()])
					if constantPattern.MatchString(name) {
						inConstant = true
						break
					}
				}
			}
		}

		// Detect magic numbers
		if nodeKind == "integer" || nodeKind == "float" {
			if !inConstant {
				numText := string(content[n.StartByte():n.EndByte()])
				line := int(n.StartPosition().Row) + 1

				// Avoid duplicate reporting
				key := fmt.Sprintf("%s:%d", numText, line)
				if !seen[key] && !commonNumbers[numText] {
					seen[key] = true
					issues = append(issues, &detectors.Issue{
						Severity:  "info",
						Type:      "magic_number",
						File:      filePath,
						Line:      line,
						Function:  fn.FullName(),
						Message:   "Magic number should be replaced with a named constant: " + numText,
						Value:     0,
						Threshold: 0,
					})
				}
			}
		}

		// Recurse into children
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child, inConstant)
			}
		}
	}

	walk(node, false)
	return issues
}
