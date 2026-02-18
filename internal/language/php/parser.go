// Package php provides PHP language support for code analysis.
package php

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	phpgrammar "github.com/tree-sitter/tree-sitter-php/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/parallel"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// PHPParser implements parser.Parser for PHP source files using tree-sitter.
type PHPParser struct {
	language *sitter.Language // PHP-only grammar (pure PHP, no embedded HTML)
	workers  int             // Number of parallel workers (0=auto, 1=sequential)
}

// NewParser creates a new PHP parser.
// The workers parameter controls parallel parsing (0=auto, 1=sequential).
func NewParser(workers int) *PHPParser {
	return &PHPParser{
		language: sitter.NewLanguage(phpgrammar.LanguagePHPOnly()),
		workers:  workers,
	}
}

// ParseFile parses a single PHP file and extracts comprehensive metrics.
func (p *PHPParser) ParseFile(path string) (*parser.FileMetrics, error) {
	if !isValidExtension(path) {
		return nil, fmt.Errorf("not a PHP file: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create parser
	tsParser := sitter.NewParser()
	defer tsParser.Close()
	tsParser.SetLanguage(p.language)

	tree := tsParser.Parse(content, nil)
	defer tree.Close()

	root := tree.RootNode()

	// Initialize metrics
	metrics := &parser.FileMetrics{
		FilePath:    path,
		PackageName: extractNamespace(root, content),
		Language:    "php",
		Functions:   []*parser.FunctionMetrics{},
		Imports:     []string{},
	}

	// Extract functions, methods, and closures
	p.extractFunctions(root, content, metrics, "")

	// Extract use statements (imports)
	p.extractImports(root, content, metrics)

	// Count lines
	metrics.TotalLines, metrics.CodeLines, metrics.CommentLines, metrics.BlankLines = countLines(content)

	return metrics, nil
}

// ParseDirectory recursively parses all PHP files in a directory.
func (p *PHPParser) ParseDirectory(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
	workers := p.workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	if workers == 1 {
		return p.parseDirectorySequential(rootPath, excludePatterns, extensions, statusReporter)
	}
	return p.parseDirectoryParallel(rootPath, excludePatterns, extensions, statusReporter, workers)
}

// parseDirectorySequential parses files one at a time.
func (p *PHPParser) parseDirectorySequential(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
	var allMetrics []*parser.FileMetrics
	var errors []error
	var fileCount int

	statusReporter.Update("[PARSE] Discovering files...")

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, fmt.Errorf("error accessing path %s: %w", path, err))
			return nil
		}

		if info.IsDir() {
			if shouldExclude(path, rootPath, excludePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasMatchingExtension(path, extensions) {
			return nil
		}

		if shouldExclude(path, rootPath, excludePatterns) {
			return nil
		}

		fileCount++
		baseName := filepath.Base(path)
		statusReporter.UpdateProgress("[PARSE] Parsing files", len(allMetrics)+1, fileCount, baseName)

		metrics, parseErr := p.ParseFile(path)
		if parseErr != nil {
			errors = append(errors, fmt.Errorf("error parsing %s: %w", path, parseErr))
			return nil
		}

		allMetrics = append(allMetrics, metrics)
		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Errorf("error walking directory: %w", err))
	}

	return allMetrics, errors
}

// parseResult holds the result of parsing a single file.
type parseResult struct {
	metrics *parser.FileMetrics
	err     error
}

// parseDirectoryParallel parses files concurrently using a worker pool.
func (p *PHPParser) parseDirectoryParallel(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter, workers int) ([]*parser.FileMetrics, []error) {
	// Phase 1: Discover all files to parse
	statusReporter.Update("[PARSE] Discovering files...")
	var filePaths []string
	var walkErrors []error

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			walkErrors = append(walkErrors, fmt.Errorf("error accessing path %s: %w", path, err))
			return nil
		}

		if info.IsDir() {
			if shouldExclude(path, rootPath, excludePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if !hasMatchingExtension(path, extensions) {
			return nil
		}

		if shouldExclude(path, rootPath, excludePatterns) {
			return nil
		}

		filePaths = append(filePaths, path)
		return nil
	})

	if err != nil {
		walkErrors = append(walkErrors, fmt.Errorf("error walking directory: %w", err))
	}

	if len(filePaths) == 0 {
		return nil, walkErrors
	}

	// Phase 2: Parse files in parallel with progress reporting
	progressReporter := parallel.NewProgressReporter(statusReporter, "[PARSE] Parsing files", len(filePaths))

	pool := parallel.NewWorkerPool(workers, func(path string) parseResult {
		metrics, parseErr := p.ParseFile(path)
		progressReporter.RecordProgress(filepath.Base(path))
		if parseErr != nil {
			return parseResult{err: fmt.Errorf("error parsing %s: %w", path, parseErr)}
		}
		return parseResult{metrics: metrics}
	})
	pool.Start()

	go func() {
		for _, path := range filePaths {
			pool.Submit(path)
		}
		pool.Close()
	}()

	// Collect results
	var allMetrics []*parser.FileMetrics
	var parseErrors []error

	for result := range pool.Results() {
		if result.err != nil {
			parseErrors = append(parseErrors, result.err)
		} else if result.metrics != nil {
			allMetrics = append(allMetrics, result.metrics)
		}
	}

	allErrors := append(walkErrors, parseErrors...)
	return allMetrics, allErrors
}

// extractFunctions walks the AST and extracts function/method metrics.
func (p *PHPParser) extractFunctions(node *sitter.Node, content []byte, metrics *parser.FileMetrics, className string) {
	cursor := node.Walk()
	defer cursor.Close()

	if !cursor.GotoFirstChild() {
		return
	}

	for {
		child := cursor.Node()
		nodeType := child.Kind()

		switch nodeType {
		case "function_definition":
			fm := p.extractFunctionDefinition(child, content, className)
			metrics.Functions = append(metrics.Functions, fm)

		case "class_declaration":
			classNameNode := child.ChildByFieldName("name")
			var newClassName string
			if classNameNode != nil {
				newClassName = string(content[classNameNode.StartByte():classNameNode.EndByte()])
			}
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractClassMembers(bodyNode, content, metrics, newClassName)
			}

		case "interface_declaration":
			classNameNode := child.ChildByFieldName("name")
			var newClassName string
			if classNameNode != nil {
				newClassName = string(content[classNameNode.StartByte():classNameNode.EndByte()])
			}
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractClassMembers(bodyNode, content, metrics, newClassName)
			}

		case "trait_declaration":
			classNameNode := child.ChildByFieldName("name")
			var newClassName string
			if classNameNode != nil {
				newClassName = string(content[classNameNode.StartByte():classNameNode.EndByte()])
			}
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractClassMembers(bodyNode, content, metrics, newClassName)
			}

		case "enum_declaration":
			classNameNode := child.ChildByFieldName("name")
			var newClassName string
			if classNameNode != nil {
				newClassName = string(content[classNameNode.StartByte():classNameNode.EndByte()])
			}
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractClassMembers(bodyNode, content, metrics, newClassName)
			}

		case "namespace_definition":
			// Recurse into namespace body
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractFunctions(bodyNode, content, metrics, className)
			}
		}

		if !cursor.GotoNextSibling() {
			break
		}
	}
}

// extractClassMembers extracts methods from a class/interface/trait/enum body.
func (p *PHPParser) extractClassMembers(classBody *sitter.Node, content []byte, metrics *parser.FileMetrics, className string) {
	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "method_declaration":
			fm := p.extractMethodDeclaration(child, content, className)
			metrics.Functions = append(metrics.Functions, fm)
		}
	}
}

// extractFunctionDefinition extracts metrics from a function_definition node.
func (p *PHPParser) extractFunctionDefinition(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
	fm := &parser.FunctionMetrics{
		StartLine:    int(node.StartPosition().Row) + 1,
		EndLine:      int(node.EndPosition().Row) + 1,
		ReceiverType: className,
	}
	fm.Lines = fm.EndLine - fm.StartLine + 1

	// Extract function name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		fm.Name = string(content[nameNode.StartByte():nameNode.EndByte()])
	}

	// Count parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fm.Parameters = countParameters(paramsNode)
	}

	fm.ReturnValues = 1 // PHP functions return a single value

	// Calculate complexity
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode, content)
	} else {
		fm.Complexity = 1
	}

	return fm
}

// extractMethodDeclaration extracts metrics from a method_declaration node.
func (p *PHPParser) extractMethodDeclaration(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
	fm := &parser.FunctionMetrics{
		StartLine:    int(node.StartPosition().Row) + 1,
		EndLine:      int(node.EndPosition().Row) + 1,
		ReceiverType: className,
	}
	fm.Lines = fm.EndLine - fm.StartLine + 1

	// Extract method name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		fm.Name = string(content[nameNode.StartByte():nameNode.EndByte()])
	}

	// Count parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fm.Parameters = countParameters(paramsNode)
	}

	fm.ReturnValues = 1

	// Calculate complexity
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode, content)
	} else {
		fm.Complexity = 1 // Abstract methods
	}

	return fm
}

// countParameters counts the number of parameters in a formal_parameters node.
func countParameters(node *sitter.Node) int {
	count := 0

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "simple_parameter", "variadic_parameter", "property_promotion_parameter":
			count++
		}
	}

	return count
}

// calculateComplexity calculates cyclomatic complexity for a function body.
func calculateComplexity(node *sitter.Node, content []byte) int {
	complexity := 1 // Base complexity

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		complexity += complexityIncrement(n, content)

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(node)
	return complexity
}

// complexityIncrement returns the complexity contribution for a given node.
func complexityIncrement(node *sitter.Node, content []byte) int {
	switch node.Kind() {
	case "if_statement", "else_if_clause",
		"for_statement", "foreach_statement",
		"while_statement", "do_statement",
		"case_statement",
		"catch_clause",
		"conditional_expression":
		return 1
	case "match_expression":
		// Each arm in a match expression adds complexity
		return countMatchArms(node)
	case "binary_expression":
		// Check for logical operators && and || which create decision points
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && !child.IsNamed() {
				op := string(content[child.StartByte():child.EndByte()])
				if op == "&&" || op == "||" || op == "and" || op == "or" {
					return 1
				}
			}
		}
		return 0
	}
	return 0
}

// countMatchArms counts the number of arms in a match expression.
func countMatchArms(node *sitter.Node) int {
	count := 0
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "match_conditional_expression" {
			count++
		}
	}
	// If no arms found, count at least 1 for the match itself
	if count == 0 {
		return 1
	}
	return count
}

// extractImports extracts use statements from the AST.
func (p *PHPParser) extractImports(node *sitter.Node, content []byte, metrics *parser.FileMetrics) {
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "namespace_use_declaration" {
			imports := extractUseStatements(n, content)
			metrics.Imports = append(metrics.Imports, imports...)
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(node)
}

// extractUseStatements extracts fully qualified names from a namespace_use_declaration node.
func extractUseStatements(node *sitter.Node, content []byte) []string {
	var imports []string

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "namespace_use_clause":
			// Extract the name from the clause
			for i := uint(0); i < n.ChildCount(); i++ {
				child := n.Child(i)
				if child != nil && (child.Kind() == "qualified_name" || child.Kind() == "name") {
					name := string(content[child.StartByte():child.EndByte()])
					name = strings.TrimPrefix(name, "\\")
					imports = append(imports, name)
				}
			}
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(node)

	// If no clauses found, try to extract the name directly
	if len(imports) == 0 {
		text := string(content[node.StartByte():node.EndByte()])
		text = strings.TrimPrefix(text, "use ")
		text = strings.TrimSuffix(text, ";")
		text = strings.TrimSpace(text)
		text = strings.TrimPrefix(text, "\\")
		if text != "" {
			imports = append(imports, text)
		}
	}

	return imports
}

// extractNamespace extracts the namespace declaration from the AST root.
func extractNamespace(root *sitter.Node, content []byte) string {
	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}

		if child.Kind() == "namespace_definition" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				return string(content[nameNode.StartByte():nameNode.EndByte()])
			}
		}
	}

	// Fall back to extracting module name from file path
	return ""
}

// countLines counts total, code, comment, and blank lines in PHP source.
func countLines(content []byte) (total, code, comment, blank int) {
	lines := bytes.Split(content, []byte("\n"))
	total = len(lines)

	inMultilineComment := false

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// Handle multiline comments (/* ... */ and /** ... */)
		if inMultilineComment {
			comment++
			if bytes.Contains(trimmed, []byte("*/")) {
				inMultilineComment = false
			}
			continue
		}

		// Check for multiline comment start
		if bytes.HasPrefix(trimmed, []byte("/*")) {
			comment++
			if !bytes.Contains(trimmed, []byte("*/")) {
				inMultilineComment = true
			}
			continue
		}

		if len(trimmed) == 0 {
			blank++
		} else if bytes.HasPrefix(trimmed, []byte("//")) || bytes.HasPrefix(trimmed, []byte("#")) {
			comment++
		} else {
			code++
		}
	}

	return
}

// isValidExtension checks if the file has a valid PHP extension.
func isValidExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".php"
}

// shouldExclude checks if a path should be excluded based on patterns.
func shouldExclude(path, rootPath string, patterns []string) bool {
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		relPath = path
	}
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		if matchPattern(relPath, pattern) {
			return true
		}
	}
	return false
}

// hasMatchingExtension checks if a path has one of the specified extensions.
func hasMatchingExtension(path string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// matchPattern performs glob-style pattern matching.
func matchPattern(path, pattern string) bool {
	pattern = filepath.ToSlash(pattern)

	if strings.Contains(pattern, "**") {
		return matchDoubleStarPattern(path, pattern)
	}

	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}
	matched, _ = filepath.Match(pattern, filepath.Base(path))
	return matched
}

// matchDoubleStarPattern handles ** glob patterns.
func matchDoubleStarPattern(path, pattern string) bool {
	parts := strings.Split(pattern, "**")

	// Handle patterns like "**/vendor/**" (3 parts: "", "/vendor/", "")
	if len(parts) == 3 && parts[0] == "" && parts[2] == "" {
		middle := strings.Trim(parts[1], "/")
		if middle != "" {
			return strings.Contains(path, middle+"/") || strings.Contains(path, "/"+middle+"/") ||
				strings.HasPrefix(path, middle+"/") || path == middle
		}
		return true
	}

	if len(parts) == 2 {
		prefix := strings.Trim(parts[0], "/")
		suffix := strings.Trim(parts[1], "/")

		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}

		if suffix != "" {
			if strings.Contains(suffix, "*") {
				matched, _ := filepath.Match(suffix, filepath.Base(path))
				return matched
			}
			return strings.HasSuffix(path, suffix) || strings.Contains(path, suffix+"/")
		}

		return true
	}

	return false
}
