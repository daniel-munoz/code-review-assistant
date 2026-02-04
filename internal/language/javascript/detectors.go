// Package javascript provides JavaScript/TypeScript language support for code analysis.
package javascript

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsts "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// JavaScriptDetectorRunner implements language.DetectorRunner for JavaScript/TypeScript source code.
type JavaScriptDetectorRunner struct {
	config      *config.AnalysisConfig
	tsLanguage  *sitter.Language
	tsxLanguage *sitter.Language
}

// NewDetectorRunner creates a new detector runner for JavaScript/TypeScript source code.
func NewDetectorRunner(cfg *config.AnalysisConfig) *JavaScriptDetectorRunner {
	return &JavaScriptDetectorRunner{
		config:      cfg,
		tsLanguage:  sitter.NewLanguage(tsts.LanguageTypescript()),
		tsxLanguage: sitter.NewLanguage(tsts.LanguageTSX()),
	}
}

// RunDetectors parses a JavaScript/TypeScript file and runs all enabled detectors.
func (r *JavaScriptDetectorRunner) RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue {
	content, err := os.ReadFile(file.FilePath)
	if err != nil {
		return nil
	}

	root, cleanup := r.parseFile(file.FilePath, content)
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
func (r *JavaScriptDetectorRunner) parseFile(path string, content []byte) (*sitter.Node, func()) {
	tsParser := sitter.NewParser()

	// Choose language based on file extension
	lang := r.tsLanguage
	if isTSXFile(path) {
		lang = r.tsxLanguage
	}
	tsParser.SetLanguage(lang)

	tree := tsParser.Parse(content, nil)
	cleanup := func() {
		tree.Close()
		tsParser.Close()
	}

	return tree.RootNode(), cleanup
}

// isTSXFile returns true if the file is a TSX or JSX file.
func isTSXFile(path string) bool {
	return strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".jsx")
}

// detectFunctionIssues runs all detectors on a single function.
func (r *JavaScriptDetectorRunner) detectFunctionIssues(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, root *sitter.Node, content []byte) []*detectors.Issue {
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
		case "function_declaration", "generator_function_declaration":
			if matchesFunctionDeclaration(n, content, name, startLine) {
				result = n
				return
			}

		case "method_definition":
			if matchesMethodDefinition(n, content, name, startLine) {
				result = n
				return
			}

		case "variable_declarator":
			// Check for arrow functions or function expressions
			if matchesVariableFunction(n, content, name, startLine) {
				// Return the arrow_function or function_expression node
				valueNode := n.ChildByFieldName("value")
				if valueNode != nil {
					result = valueNode
					return
				}
			}

		case "public_field_definition", "field_definition":
			// Check for class field arrow functions
			if matchesFieldFunction(n, content, name, startLine) {
				for i := uint(0); i < n.ChildCount(); i++ {
					child := n.Child(i)
					if child != nil && (child.Kind() == "arrow_function" || child.Kind() == "function_expression") {
						result = child
						return
					}
				}
			}
		}

		walkChildren(n, walk)
	}

	walk(root)
	return result
}

// matchesFunctionDeclaration checks if a function_declaration matches the expected name and line.
func matchesFunctionDeclaration(n *sitter.Node, content []byte, name string, startLine int) bool {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	nodeName := string(content[nameNode.StartByte():nameNode.EndByte()])
	nodeLine := int(n.StartPosition().Row) + 1
	return nodeName == name && nodeLine == startLine
}

// matchesMethodDefinition checks if a method_definition matches the expected name and line.
func matchesMethodDefinition(n *sitter.Node, content []byte, name string, startLine int) bool {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	nodeName := string(content[nameNode.StartByte():nameNode.EndByte()])
	nodeLine := int(n.StartPosition().Row) + 1
	return nodeName == name && nodeLine == startLine
}

// matchesVariableFunction checks if a variable_declarator with a function matches.
func matchesVariableFunction(n *sitter.Node, content []byte, name string, startLine int) bool {
	nameNode := n.ChildByFieldName("name")
	valueNode := n.ChildByFieldName("value")
	if nameNode == nil || valueNode == nil {
		return false
	}

	valueKind := valueNode.Kind()
	if valueKind != "arrow_function" && valueKind != "function_expression" && valueKind != "generator_function" {
		return false
	}

	nodeName := string(content[nameNode.StartByte():nameNode.EndByte()])
	nodeLine := int(valueNode.StartPosition().Row) + 1
	return nodeName == name && nodeLine == startLine
}

// matchesFieldFunction checks if a class field function matches.
func matchesFieldFunction(n *sitter.Node, content []byte, name string, startLine int) bool {
	nodeLine := int(n.StartPosition().Row) + 1
	if nodeLine != startLine {
		return false
	}

	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child != nil && child.Kind() == "property_identifier" {
			nodeName := string(content[child.StartByte():child.EndByte()])
			return nodeName == name
		}
	}
	return false
}

// calculateMaxNestingDepth calculates the maximum nesting depth in a JavaScript function body.
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
	case "if_statement", "for_statement", "for_in_statement", "for_of_statement",
		"while_statement", "do_statement",
		"try_statement", "with_statement", "switch_statement":
		return true
	}
	return false
}

// countReturnStatements counts return statements in a JavaScript function body.
func countReturnStatements(node *sitter.Node) int {
	count := 0

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		kind := n.Kind()
		if kind == "return_statement" {
			count++
		}

		// Don't count returns in nested functions
		if kind == "function_declaration" || kind == "arrow_function" ||
			kind == "function_expression" || kind == "generator_function_declaration" {
			return
		}

		walkChildren(n, walk)
	}

	walk(node)
	return count
}

// walkChildren iterates over all children of a node and calls the walker function.
func walkChildren(n *sitter.Node, walker func(*sitter.Node)) {
	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child != nil {
			walker(child)
		}
	}
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

	// Check if we're entering a constant assignment (const UPPER_CASE = ...)
	if kind == "variable_declarator" {
		inConstant = ctx.isConstantDeclaration(n)
	}

	// Check for magic numbers
	if kind == "number" {
		ctx.checkMagicNumber(n, inConstant)
	}

	// Recurse into children
	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child != nil {
			ctx.walk(child, inConstant)
		}
	}
}

// isConstantDeclaration checks if the variable declarator uses UPPER_CASE naming.
func (ctx *magicNumberContext) isConstantDeclaration(n *sitter.Node) bool {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	name := string(ctx.content[nameNode.StartByte():nameNode.EndByte()])
	return ctx.constPattern.MatchString(name)
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
