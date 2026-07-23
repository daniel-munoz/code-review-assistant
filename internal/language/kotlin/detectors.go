package kotlin

import (
	"fmt"
	"os"

	sitter "github.com/tree-sitter/go-tree-sitter"
	kotlinlang "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// KotlinDetectorRunner implements language.DetectorRunner for Kotlin source code.
type KotlinDetectorRunner struct {
	config   *config.AnalysisConfig
	language *sitter.Language
}

// NewDetectorRunner creates a new detector runner for Kotlin source code.
func NewDetectorRunner(cfg *config.AnalysisConfig) *KotlinDetectorRunner {
	return &KotlinDetectorRunner{
		config:   cfg,
		language: sitter.NewLanguage(kotlinlang.Language()),
	}
}

// RunDetectors parses a Kotlin file and runs all enabled detectors.
func (r *KotlinDetectorRunner) RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue {
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
		issues = append(issues, r.detectFunctionIssues(cfg, file, fn, root, content)...)
	}

	return issues
}

// parseFile creates a tree-sitter parser and parses the content.
func (r *KotlinDetectorRunner) parseFile(content []byte) (*sitter.Node, func()) {
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
func (r *KotlinDetectorRunner) detectFunctionIssues(cfg *config.AnalysisConfig, file *parser.FileMetrics, fn *parser.FunctionMetrics, root *sitter.Node, content []byte) []*detectors.Issue {
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

	if issue := detectDeepNesting(cfg, file, fn, bodyNode); issue != nil {
		issues = append(issues, issue)
	}

	if issue := detectTooManyReturns(cfg, file, fn, bodyNode); issue != nil {
		issues = append(issues, issue)
	}

	if cfg.DetectMagicNumbers {
		issues = append(issues, detectMagicNumbers(bodyNode, content, file.FilePath, fn)...)
	}

	if cfg.DetectNonNullAssertions {
		issues = append(issues, detectNonNullAssertions(file, fn, bodyNode)...)
	}

	// runBlocking inside fun main is the legitimate idiom - don't flag it there
	if cfg.DetectRunBlocking && fn.Name != "main" {
		issues = append(issues, detectRunBlocking(file, fn, bodyNode, content)...)
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

// detectNonNullAssertions flags each use of the !! operator.
func detectNonNullAssertions(file *parser.FileMetrics, fn *parser.FunctionMetrics, bodyNode *sitter.Node) []*detectors.Issue {
	var issues []*detectors.Issue

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "!!" {
			issues = append(issues, &detectors.Issue{
				Severity: "warning",
				Type:     "non_null_assertion",
				File:     file.FilePath,
				Line:     int(n.StartPosition().Row) + 1,
				Function: fn.FullName(),
				Message:  "Non-null assertion (!!) bypasses null safety; prefer safe calls or explicit checks",
			})
		}
		walkChildren(n, walk)
	}

	walk(bodyNode)
	return issues
}

// detectRunBlocking flags runBlocking calls (thread-blocking coroutine bridge).
func detectRunBlocking(file *parser.FileMetrics, fn *parser.FunctionMetrics, bodyNode *sitter.Node, content []byte) []*detectors.Issue {
	var issues []*detectors.Issue

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "call_expression" {
			callee := n.Child(0)
			if callee != nil && callee.Kind() == "identifier" &&
				string(content[callee.StartByte():callee.EndByte()]) == "runBlocking" {
				issues = append(issues, &detectors.Issue{
					Severity: "warning",
					Type:     "run_blocking",
					File:     file.FilePath,
					Line:     int(n.StartPosition().Row) + 1,
					Function: fn.FullName(),
					Message:  "runBlocking blocks the current thread; prefer suspending functions or structured concurrency",
				})
			}
		}
		walkChildren(n, walk)
	}

	walk(bodyNode)
	return issues
}

// findFunctionBody locates a function's body node by name and start line.
func findFunctionBody(root *sitter.Node, content []byte, name string, startLine int) *sitter.Node {
	funcNode := findFunctionNode(root, content, name, startLine)
	if funcNode == nil {
		return nil
	}
	return childOfKind(funcNode, "function_body")
}

// findFunctionNode locates a function_declaration node by name and start line.
func findFunctionNode(root *sitter.Node, content []byte, name string, startLine int) *sitter.Node {
	var result *sitter.Node

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if result != nil {
			return
		}

		if n.Kind() == "function_declaration" && matchesFunctionSignature(n, content, name, startLine) {
			result = n
			return
		}

		walkChildren(n, walk)
	}

	walk(root)
	return result
}

// matchesFunctionSignature checks if a node matches the expected function name and line.
func matchesFunctionSignature(n *sitter.Node, content []byte, name string, startLine int) bool {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	nodeName := string(content[nameNode.StartByte():nameNode.EndByte()])
	nodeLine := int(n.StartPosition().Row) + 1
	return nodeName == name && nodeLine == startLine
}

// calculateMaxNestingDepth calculates the maximum nesting depth in a function body.
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
	case "if_expression", "for_statement", "while_statement",
		"do_while_statement", "try_expression", "when_expression":
		return true
	}
	return false
}

// countReturnStatements counts return expressions in a function body.
// Returns inside nested functions and lambdas are not counted.
func countReturnStatements(node *sitter.Node) int {
	count := 0

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		kind := n.Kind()
		if kind == "return_expression" {
			count++
		}

		if kind == "function_declaration" || kind == "lambda_literal" {
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
	content  []byte
	filePath string
	fn       *parser.FunctionMetrics
	seen     map[string]bool
	issues   []*detectors.Issue
}

// commonNumbers that are acceptable and not flagged as magic numbers.
var commonNumbers = map[string]bool{
	"0": true, "1": true, "-1": true,
	"0.0": true, "1.0": true, "-1.0": true,
}

// detectMagicNumbers finds numeric literals that aren't common constants.
func detectMagicNumbers(node *sitter.Node, content []byte, filePath string, fn *parser.FunctionMetrics) []*detectors.Issue {
	ctx := &magicNumberContext{
		content:  content,
		filePath: filePath,
		fn:       fn,
		seen:     make(map[string]bool),
	}

	ctx.walk(node, false)
	return ctx.issues
}

// walk recursively traverses the AST looking for magic numbers.
func (ctx *magicNumberContext) walk(n *sitter.Node, inConstant bool) {
	kind := n.Kind()

	// Entering a constant-style property declaration exempts its initializer
	if kind == "property_declaration" {
		inConstant = ctx.isConstantProperty(n)
	}

	if kind == "number_literal" {
		ctx.checkMagicNumber(n, inConstant)
	}

	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child != nil {
			ctx.walk(child, inConstant)
		}
	}
}

// isConstantProperty checks for a `const` modifier or an UPPER_CASE name.
func (ctx *magicNumberContext) isConstantProperty(n *sitter.Node) bool {
	if mods := childOfKind(n, "modifiers"); mods != nil && hasDescendantOfKind(mods, "const") {
		return true
	}
	if varDecl := childOfKind(n, "variable_declaration"); varDecl != nil {
		if id := childOfKind(varDecl, "identifier"); id != nil {
			name := string(ctx.content[id.StartByte():id.EndByte()])
			return constNamePattern.MatchString(name)
		}
	}
	return false
}

// hasDescendantOfKind reports whether any descendant of n has the given kind.
func hasDescendantOfKind(n *sitter.Node, kind string) bool {
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == kind || hasDescendantOfKind(child, kind) {
			return true
		}
	}
	return false
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
		Severity: "info",
		Type:     "magic_number",
		File:     ctx.filePath,
		Line:     line,
		Function: ctx.fn.FullName(),
		Message:  "Magic number should be replaced with a named constant: " + numText,
	})
}
