// Package javascript provides JavaScript/TypeScript language support for code analysis.
package javascript

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsts "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/parallel"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// JavaScriptParser implements parser.Parser for JavaScript/TypeScript source files using tree-sitter.
type JavaScriptParser struct {
	tsLanguage  *sitter.Language // TypeScript grammar for .ts, .js, .mjs, .cjs
	tsxLanguage *sitter.Language // TSX grammar for .tsx, .jsx
	workers     int              // Number of parallel workers (0=auto, 1=sequential)
}

// NewParser creates a new JavaScript/TypeScript parser.
// The workers parameter controls parallel parsing (0=auto, 1=sequential).
func NewParser(workers int) *JavaScriptParser {
	return &JavaScriptParser{
		tsLanguage:  sitter.NewLanguage(tsts.LanguageTypescript()),
		tsxLanguage: sitter.NewLanguage(tsts.LanguageTSX()),
		workers:     workers,
	}
}

// getLanguageForFile returns the appropriate tree-sitter language for a file extension.
func (p *JavaScriptParser) getLanguageForFile(path string) *sitter.Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".tsx", ".jsx":
		return p.tsxLanguage
	default:
		return p.tsLanguage
	}
}

// isValidExtension checks if the file has a valid JavaScript/TypeScript extension.
func isValidExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

// ParseFile parses a single JavaScript/TypeScript file and extracts comprehensive metrics.
func (p *JavaScriptParser) ParseFile(path string) (*parser.FileMetrics, error) {
	if !isValidExtension(path) {
		return nil, fmt.Errorf("not a JavaScript/TypeScript file: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create parser with appropriate language
	tsParser := sitter.NewParser()
	defer tsParser.Close()
	tsParser.SetLanguage(p.getLanguageForFile(path))

	tree := tsParser.Parse(content, nil)
	defer tree.Close()

	root := tree.RootNode()

	// Initialize metrics
	metrics := &parser.FileMetrics{
		FilePath:    path,
		PackageName: extractModuleName(path),
		Language:    "javascript",
		Functions:   []*parser.FunctionMetrics{},
		Imports:     []string{},
	}

	// Extract functions, methods, and arrow functions
	p.extractFunctions(root, content, metrics, "")

	// Extract imports
	p.extractImports(root, content, metrics)

	// Count lines
	metrics.TotalLines, metrics.CodeLines, metrics.CommentLines, metrics.BlankLines = countLines(content)

	return metrics, nil
}

// ParseDirectory recursively parses all JavaScript/TypeScript files in a directory.
// If Workers is set to 1, parsing is sequential. Otherwise, files are parsed in parallel.
func (p *JavaScriptParser) ParseDirectory(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
	workers := p.workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	if workers == 1 {
		return p.parseDirectorySequential(rootPath, excludePatterns, extensions, statusReporter)
	}
	return p.parseDirectoryParallel(rootPath, excludePatterns, extensions, statusReporter, workers)
}

// parseDirectorySequential parses files one at a time (original implementation).
func (p *JavaScriptParser) parseDirectorySequential(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
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
func (p *JavaScriptParser) parseDirectoryParallel(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter, workers int) ([]*parser.FileMetrics, []error) {
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

	// Collect results (single-threaded - no mutex needed)
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
func (p *JavaScriptParser) extractFunctions(node *sitter.Node, content []byte, metrics *parser.FileMetrics, className string) {
	cursor := node.Walk()
	defer cursor.Close()

	if !cursor.GotoFirstChild() {
		return
	}

	for {
		child := cursor.Node()
		nodeType := child.Kind()

		switch nodeType {
		case "function_declaration", "generator_function_declaration":
			fm := p.extractFunctionDeclaration(child, content, className)
			metrics.Functions = append(metrics.Functions, fm)

		case "lexical_declaration", "variable_declaration":
			// Check for arrow functions or function expressions assigned to variables
			fns := p.extractVariableFunctions(child, content, className)
			metrics.Functions = append(metrics.Functions, fns...)

		case "class_declaration":
			// Extract class name and process its body
			classNameNode := child.ChildByFieldName("name")
			var newClassName string
			if classNameNode != nil {
				newClassName = string(content[classNameNode.StartByte():classNameNode.EndByte()])
			}
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractClassMethods(bodyNode, content, metrics, newClassName)
			}

		case "export_statement":
			// Handle exported functions/classes
			p.extractFunctions(child, content, metrics, className)

		case "method_definition":
			// Standalone method (shouldn't happen at top level, but handle it)
			fm := p.extractMethodDefinition(child, content, className)
			metrics.Functions = append(metrics.Functions, fm)
		}

		if !cursor.GotoNextSibling() {
			break
		}
	}
}

// extractClassMethods extracts methods from a class body.
func (p *JavaScriptParser) extractClassMethods(classBody *sitter.Node, content []byte, metrics *parser.FileMetrics, className string) {
	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "method_definition":
			fm := p.extractMethodDefinition(child, content, className)
			metrics.Functions = append(metrics.Functions, fm)

		case "public_field_definition", "field_definition":
			// Check for arrow function fields: field = () => {}
			fm := p.extractFieldArrowFunction(child, content, className)
			if fm != nil {
				metrics.Functions = append(metrics.Functions, fm)
			}
		}
	}
}

// extractFunctionDeclaration extracts metrics from a function_declaration node.
func (p *JavaScriptParser) extractFunctionDeclaration(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
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
		fm.Parameters = countParameters(paramsNode, content)
	}

	fm.ReturnValues = 1 // JS functions return single value

	// Calculate complexity
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode, content)
	} else {
		fm.Complexity = 1
	}

	return fm
}

// extractMethodDefinition extracts metrics from a method_definition node.
func (p *JavaScriptParser) extractMethodDefinition(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
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
		fm.Parameters = countParameters(paramsNode, content)
	}

	fm.ReturnValues = 1

	// Calculate complexity
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode, content)
	} else {
		fm.Complexity = 1
	}

	return fm
}

// extractVariableFunctions extracts arrow functions and function expressions from variable declarations.
func (p *JavaScriptParser) extractVariableFunctions(node *sitter.Node, content []byte, className string) []*parser.FunctionMetrics {
	var functions []*parser.FunctionMetrics

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "variable_declarator" {
			nameNode := n.ChildByFieldName("name")
			valueNode := n.ChildByFieldName("value")

			if nameNode != nil && valueNode != nil {
				valueKind := valueNode.Kind()
				if valueKind == "arrow_function" || valueKind == "function_expression" || valueKind == "generator_function" {
					fm := p.extractArrowOrFunctionExpression(valueNode, content, className)
					fm.Name = string(content[nameNode.StartByte():nameNode.EndByte()])
					functions = append(functions, fm)
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
	return functions
}

// extractFieldArrowFunction extracts arrow functions from class field definitions.
func (p *JavaScriptParser) extractFieldArrowFunction(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
	var name string
	var valueNode *sitter.Node

	// Look for property name and value
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "property_identifier":
			name = string(content[child.StartByte():child.EndByte()])
		case "arrow_function", "function_expression":
			valueNode = child
		}
	}

	if valueNode == nil {
		return nil
	}

	fm := p.extractArrowOrFunctionExpression(valueNode, content, className)
	fm.Name = name
	return fm
}

// extractArrowOrFunctionExpression extracts metrics from arrow_function or function_expression nodes.
func (p *JavaScriptParser) extractArrowOrFunctionExpression(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
	fm := &parser.FunctionMetrics{
		StartLine:    int(node.StartPosition().Row) + 1,
		EndLine:      int(node.EndPosition().Row) + 1,
		ReceiverType: className,
		ReturnValues: 1,
	}
	fm.Lines = fm.EndLine - fm.StartLine + 1

	// Count parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fm.Parameters = countParameters(paramsNode, content)
	} else {
		// Single parameter without parentheses: x => x + 1
		paramNode := node.ChildByFieldName("parameter")
		if paramNode != nil {
			fm.Parameters = 1
		}
	}

	// Calculate complexity
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode, content)
	} else {
		fm.Complexity = 1
	}

	return fm
}

// countParameters counts the number of parameters in a formal_parameters node.
func countParameters(node *sitter.Node, _ []byte) int {
	count := 0

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "identifier", "required_parameter", "optional_parameter",
			"rest_pattern", "assignment_pattern", "object_pattern", "array_pattern":
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
	case "if_statement",
		"for_statement", "for_in_statement", "for_of_statement",
		"while_statement", "do_statement",
		"switch_case",
		"catch_clause",
		"ternary_expression", "conditional_expression":
		return 1
	case "binary_expression":
		// Check for logical operators && and || which create decision points
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "&&" || child.Kind() == "||" {
				return 1
			}
			// Also check the actual text for operators
			if child != nil && child.IsNamed() == false {
				op := string(content[child.StartByte():child.EndByte()])
				if op == "&&" || op == "||" {
					return 1
				}
			}
		}
		return 0
	}
	return 0
}

// extractImports extracts import statements from the AST.
func (p *JavaScriptParser) extractImports(node *sitter.Node, content []byte, metrics *parser.FileMetrics) {
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "import_statement":
			source := extractImportSource(n, content)
			if source != "" {
				metrics.Imports = append(metrics.Imports, source)
			}
		case "call_expression":
			// Check for require() calls
			funcNode := n.ChildByFieldName("function")
			if funcNode != nil && string(content[funcNode.StartByte():funcNode.EndByte()]) == "require" {
				argsNode := n.ChildByFieldName("arguments")
				if argsNode != nil {
					source := extractRequireSource(argsNode, content)
					if source != "" {
						metrics.Imports = append(metrics.Imports, source)
					}
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
}

// extractImportSource extracts the source module from an import statement.
func extractImportSource(node *sitter.Node, content []byte) string {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode != nil {
		text := string(content[sourceNode.StartByte():sourceNode.EndByte()])
		// Remove quotes
		return strings.Trim(text, "\"'`")
	}
	return ""
}

// extractRequireSource extracts the source module from require() arguments.
func extractRequireSource(argsNode *sitter.Node, content []byte) string {
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child != nil && child.Kind() == "string" {
			text := string(content[child.StartByte():child.EndByte()])
			return strings.Trim(text, "\"'`")
		}
	}
	return ""
}

// countLines counts total, code, comment, and blank lines in JavaScript/TypeScript source.
func countLines(content []byte) (total, code, comment, blank int) {
	lines := bytes.Split(content, []byte("\n"))
	total = len(lines)

	inMultilineComment := false

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// Handle multiline comments
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
		} else if bytes.HasPrefix(trimmed, []byte("//")) {
			comment++
		} else {
			code++
		}
	}

	return
}

// extractModuleName extracts a module name from the file path.
func extractModuleName(path string) string {
	base := filepath.Base(path)
	// Remove extension
	for _, ext := range []string{".tsx", ".jsx", ".ts", ".js", ".mjs", ".cjs"} {
		if strings.HasSuffix(base, ext) {
			return strings.TrimSuffix(base, ext)
		}
	}
	return base
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

	// Handle patterns like "**/node_modules/**" (3 parts: "", "/node_modules/", "")
	if len(parts) == 3 && parts[0] == "" && parts[2] == "" {
		// Pattern is **/something/** - match if "something" appears anywhere in path
		middle := strings.Trim(parts[1], "/")
		if middle != "" {
			// Check if the middle part appears as a path component
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
