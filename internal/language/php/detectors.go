// Package php provides PHP language support for code analysis.
package php

import (
	"fmt"
	"os"
	"regexp"

	sitter "github.com/tree-sitter/go-tree-sitter"
	phpgrammar "github.com/tree-sitter/tree-sitter-php/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// PHPDetectorRunner implements language.DetectorRunner for PHP source code.
type PHPDetectorRunner struct {
	config   *config.AnalysisConfig
	language *sitter.Language
}

// NewDetectorRunner creates a new detector runner for PHP source code.
func NewDetectorRunner(cfg *config.AnalysisConfig) *PHPDetectorRunner {
	return &PHPDetectorRunner{
		config:   cfg,
		language: sitter.NewLanguage(phpgrammar.LanguagePHPOnly()),
	}
}

// RunDetectors parses a PHP file and runs all enabled detectors.
func (r *PHPDetectorRunner) RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue {
	content, err := os.ReadFile(file.FilePath)
	if err != nil {
		return nil
	}

	root, cleanup := r.parseFile(content)
	if root == nil {
		return nil
	}
	defer cleanup()

	var issues []*detectors.Issue
	for _, fn := range file.Functions {
		fnIssues := r.detectFunctionIssues(cfg, file, fn, root, content)
		issues = append(issues, fnIssues...)
	}

	return issues
}

// parseFile creates a tree-sitter parser and parses the content.
func (r *PHPDetectorRunner) parseFile(content []byte) (*sitter.Node, func()) {
	tsParser := sitter.NewParser()
	tsParser.SetLanguage(r.language)

	tree := tsParser.Parse(content, nil)
	cleanup := func() {
		tree.Close()
		tsParser.Close()
	}

	return tree.RootNode(), cleanup
}

// detectFunctionIssues runs all detectors on a single function.
func (r *PHPDetectorRunner) detectFunctionIssues(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, root *sitter.Node, content []byte) []*detectors.Issue {
	var issues []*detectors.Issue

	// Parameter count detector (doesn't need AST)
	if issue := detectTooManyParameters(cfg, file, fn); issue != nil {
		issues = append(issues, issue)
	}

	// Find function body for AST-based detectors
	bodyNode := findFunctionBody(root, content, fn.Name, fn.StartLine)
	if bodyNode == nil {
		return issues
	}

	// Nesting depth detector
	if issue := detectDeepNesting(cfg, file, fn, bodyNode); issue != nil {
		issues = append(issues, issue)
	}

	// Return count detector
	if issue := detectTooManyReturns(cfg, file, fn, bodyNode); issue != nil {
		issues = append(issues, issue)
	}

	// Magic number detector
	if cfg.DetectMagicNumbers {
		magicIssues := detectMagicNumbers(bodyNode, content, file.FilePath, fn)
		issues = append(issues, magicIssues...)
	}

	return issues
}

// detectTooManyParameters checks if a function has too many parameters.
func detectTooManyParameters(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics) *detectors.Issue {
	if cfg.MaxParameters <= 0 || fn.Parameters <= cfg.MaxParameters {
		return nil
	}

	return &detectors.Issue{
		Severity:  "warning",
		Type:      "too_many_parameters",
		File:      file.FilePath,
		Line:      fn.StartLine,
		Function:  fn.FullName(),
		Message:   fmt.Sprintf("Function has too many parameters (%d)", fn.Parameters),
		Value:     fn.Parameters,
		Threshold: cfg.MaxParameters,
	}
}

// detectDeepNesting checks if a function has excessive nesting depth.
func detectDeepNesting(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, bodyNode *sitter.Node) *detectors.Issue {
	if cfg.MaxNestingDepth <= 0 {
		return nil
	}

	maxDepth := calculateMaxNestingDepth(bodyNode)
	if maxDepth <= cfg.MaxNestingDepth {
		return nil
	}

	return &detectors.Issue{
		Severity:  "warning",
		Type:      "deep_nesting",
		File:      file.FilePath,
		Line:      fn.StartLine,
		Function:  fn.FullName(),
		Message:   fmt.Sprintf("Function has deep nesting (depth: %d)", maxDepth),
		Value:     maxDepth,
		Threshold: cfg.MaxNestingDepth,
	}
}

// detectTooManyReturns checks if a function has too many return statements.
func detectTooManyReturns(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, bodyNode *sitter.Node) *detectors.Issue {
	if cfg.MaxReturnStatements <= 0 {
		return nil
	}

	returnCount := countReturnStatements(bodyNode)
	if returnCount <= cfg.MaxReturnStatements {
		return nil
	}

	return &detectors.Issue{
		Severity:  "info",
		Type:      "too_many_returns",
		File:      file.FilePath,
		Line:      fn.StartLine,
		Function:  fn.FullName(),
		Message:   fmt.Sprintf("Function has %d return statements", returnCount),
		Value:     returnCount,
		Threshold: cfg.MaxReturnStatements,
	}
}

// findFunctionBody locates a function's body node by name and start line.
func findFunctionBody(root *sitter.Node, content []byte, name string, startLine int) *sitter.Node {
	funcNode := findFunctionNode(root, content, name, startLine)
	if funcNode == nil {
		return nil
	}
	return funcNode.ChildByFieldName("body")
}

// findFunctionNode locates a function node by name and start line.
func findFunctionNode(root *sitter.Node, content []byte, name string, startLine int) *sitter.Node {
	var result *sitter.Node

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if result != nil {
			return
		}

		kind := n.Kind()
		switch kind {
		case "function_definition":
			if matchesFunctionNode(n, content, name, startLine) {
				result = n
				return
			}

		case "method_declaration":
			if matchesFunctionNode(n, content, name, startLine) {
				result = n
				return
			}
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(root)
	return result
}

// matchesFunctionNode checks if a function/method node matches the expected name and line.
func matchesFunctionNode(n *sitter.Node, content []byte, name string, startLine int) bool {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	nodeName := string(content[nameNode.StartByte():nameNode.EndByte()])
	nodeLine := int(n.StartPosition().Row) + 1
	return nodeName == name && nodeLine == startLine
}

// calculateMaxNestingDepth calculates the maximum nesting depth in a PHP function body.
func calculateMaxNestingDepth(node *sitter.Node) int {
	maxDepth := 0

	var traverse func(n *sitter.Node, currentDepth int)
	traverse = func(n *sitter.Node, currentDepth int) {
		newDepth := currentDepth
		if isNestingConstruct(n.Kind()) {
			newDepth++
			if newDepth > maxDepth {
				maxDepth = newDepth
			}
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				traverse(child, newDepth)
			}
		}
	}

	traverse(node, 0)
	return maxDepth
}

// isNestingConstruct returns true if the node kind increases nesting depth.
func isNestingConstruct(kind string) bool {
	switch kind {
	case "if_statement", "for_statement", "foreach_statement",
		"while_statement", "do_statement",
		"try_statement", "switch_statement", "match_expression":
		return true
	}
	return false
}

// countReturnStatements counts return statements in a PHP function body.
func countReturnStatements(node *sitter.Node) int {
	count := 0

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		kind := n.Kind()
		if kind == "return_statement" {
			count++
		}

		// Don't count returns in nested functions/closures
		if kind == "function_definition" || kind == "anonymous_function_creation_expression" ||
			kind == "arrow_function" {
			return
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(node)
	return count
}

// magicNumberContext holds state for magic number detection.
type magicNumberContext struct {
	content      []byte
	filePath     string
	fn           *parser.FunctionMetrics
	seen         map[string]bool
	issues       []*detectors.Issue
	constPattern *regexp.Regexp
}

// commonNumbers that are acceptable and not flagged as magic numbers.
var commonNumbers = map[string]bool{
	"0": true, "1": true, "-1": true,
	"0.0": true, "1.0": true, "-1.0": true,
	"100": true, "1000": true,
}

// detectMagicNumbers finds numeric literals that aren't common constants.
func detectMagicNumbers(node *sitter.Node, content []byte, filePath string, fn *parser.FunctionMetrics) []*detectors.Issue {
	ctx := &magicNumberContext{
		content:      content,
		filePath:     filePath,
		fn:           fn,
		seen:         make(map[string]bool),
		constPattern: regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`),
	}

	ctx.walk(node, false)
	return ctx.issues
}

// walk recursively traverses the AST looking for magic numbers.
func (ctx *magicNumberContext) walk(n *sitter.Node, inConstant bool) {
	kind := n.Kind()

	// Check if we're inside a const declaration
	if kind == "const_element" {
		inConstant = true
	}

	// Check for magic numbers
	if kind == "integer" || kind == "float" {
		ctx.checkMagicNumber(n, inConstant)
	}

	// Recurse into children
	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child != nil {
			ctx.walk(child, inConstant)
		}
	}
}

// checkMagicNumber checks if a numeric node is a magic number and records an issue.
func (ctx *magicNumberContext) checkMagicNumber(n *sitter.Node, inConstant bool) {
	if inConstant {
		return
	}

	numText := string(ctx.content[n.StartByte():n.EndByte()])
	if commonNumbers[numText] {
		return
	}

	line := int(n.StartPosition().Row) + 1
	key := fmt.Sprintf("%s:%d", numText, line)

	if ctx.seen[key] {
		return
	}
	ctx.seen[key] = true

	ctx.issues = append(ctx.issues, &detectors.Issue{
		Severity:  "info",
		Type:      "magic_number",
		File:      ctx.filePath,
		Line:      line,
		Function:  ctx.fn.FullName(),
		Message:   "Magic number should be replaced with a named constant: " + numText,
		Value:     0,
		Threshold: 0,
	})
}
