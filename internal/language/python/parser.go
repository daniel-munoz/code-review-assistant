// Package python provides Python language support for code analysis.
package python

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/parallel"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// PythonParser implements parser.Parser for Python source files using tree-sitter.
type PythonParser struct {
	language *sitter.Language
	workers  int // Number of parallel workers (0=auto, 1=sequential)
}

// NewParser creates a new Python parser.
// The workers parameter controls parallel parsing (0=auto, 1=sequential).
func NewParser(workers int) *PythonParser {
	return &PythonParser{
		language: sitter.NewLanguage(python.Language()),
		workers:  workers,
	}
}

// ParseFile parses a single Python file and extracts comprehensive metrics.
func (p *PythonParser) ParseFile(path string) (*parser.FileMetrics, error) {
	// Verify file exists and is a Python file
	if !strings.HasSuffix(path, ".py") {
		return nil, fmt.Errorf("not a Python file: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create parser and parse
	tsParser := sitter.NewParser()
	defer tsParser.Close()
	tsParser.SetLanguage(p.language)

	tree := tsParser.Parse(content, nil)
	defer tree.Close()

	root := tree.RootNode()

	// Initialize metrics
	metrics := &parser.FileMetrics{
		FilePath:    path,
		PackageName: extractModuleName(path),
		Language:    "python",
		Functions:   []*parser.FunctionMetrics{},
		Imports:     []string{},
	}

	// Extract functions and classes
	p.extractFunctionsAndClasses(root, content, metrics, "")

	// Extract imports
	p.extractImports(root, content, metrics)

	// Count lines
	metrics.TotalLines, metrics.CodeLines, metrics.CommentLines, metrics.BlankLines = countLines(content)

	return metrics, nil
}

// ParseDirectory recursively parses all Python files in a directory.
// If Workers is set to 1, parsing is sequential. Otherwise, files are parsed in parallel.
func (p *PythonParser) ParseDirectory(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
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
func (p *PythonParser) parseDirectorySequential(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
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
func (p *PythonParser) parseDirectoryParallel(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter, workers int) ([]*parser.FileMetrics, []error) {
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

// extractFunctionsAndClasses walks the AST and extracts function/class metrics.
func (p *PythonParser) extractFunctionsAndClasses(node *sitter.Node, content []byte, metrics *parser.FileMetrics, className string) {
	cursor := node.Walk()
	defer cursor.Close()

	// Move to first child
	if !cursor.GotoFirstChild() {
		return
	}

	for {
		child := cursor.Node()
		nodeType := child.Kind()

		switch nodeType {
		case "function_definition":
			fm := p.extractFunction(child, content, className)
			metrics.Functions = append(metrics.Functions, fm)

		case "class_definition":
			// Extract class name and process its body
			classNameNode := child.ChildByFieldName("name")
			var newClassName string
			if classNameNode != nil {
				newClassName = string(content[classNameNode.StartByte():classNameNode.EndByte()])
			}

			// Process class body for methods
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				p.extractFunctionsAndClasses(bodyNode, content, metrics, newClassName)
			}

		case "decorated_definition":
			// Handle decorated functions/classes
			p.extractFunctionsAndClasses(child, content, metrics, className)
		}

		if !cursor.GotoNextSibling() {
			break
		}
	}
}

// extractFunction extracts metrics from a function_definition node.
func (p *PythonParser) extractFunction(node *sitter.Node, content []byte, className string) *parser.FunctionMetrics {
	fm := &parser.FunctionMetrics{
		StartLine:    int(node.StartPosition().Row) + 1, // 0-indexed to 1-indexed
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

	// Count return values (Python functions return single value, but we count return statements)
	fm.ReturnValues = 1 // Python always returns something (None if not explicit)

	// Calculate complexity
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode)
	} else {
		fm.Complexity = 1
	}

	return fm
}

// countParameters counts the number of parameters in a parameters node.
func countParameters(node *sitter.Node, content []byte) int {
	count := 0
	cursor := node.Walk()
	defer cursor.Close()

	if !cursor.GotoFirstChild() {
		return 0
	}

	for {
		child := cursor.Node()
		nodeType := child.Kind()

		// Count various parameter types
		switch nodeType {
		case "identifier", "typed_parameter", "default_parameter", "typed_default_parameter",
			"list_splat_pattern", "dictionary_splat_pattern":
			// Skip 'self' and 'cls' for methods
			paramName := extractParameterName(child, content)
			if paramName == "self" || paramName == "cls" {
				// Don't count self/cls
			} else {
				count++
			}
		}

		if !cursor.GotoNextSibling() {
			break
		}
	}

	return count
}

// extractParameterName extracts the parameter name from any parameter node type.
// For typed_parameter (e.g., "self: MyClass"), it extracts just the identifier.
// For default_parameter (e.g., "x=5"), it extracts just the name.
func extractParameterName(node *sitter.Node, content []byte) string {
	switch node.Kind() {
	case "identifier":
		return string(content[node.StartByte():node.EndByte()])
	case "typed_parameter", "default_parameter", "typed_default_parameter":
		// The first child is typically the identifier
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "identifier" {
				return string(content[child.StartByte():child.EndByte()])
			}
		}
	}
	return ""
}

// calculateComplexity calculates cyclomatic complexity for a function body.
func calculateComplexity(node *sitter.Node) int {
	complexity := 1 // Base complexity

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		complexity += complexityIncrement(n.Kind())

		// Recurse into children
		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(node)
	return complexity
}

// complexityIncrement returns the complexity contribution for a given node kind.
func complexityIncrement(kind string) int {
	switch kind {
	case "if_statement", "elif_clause",
		"for_statement", "while_statement",
		"except_clause",
		"conditional_expression",
		"boolean_operator",
		"list_comprehension", "dict_comprehension", "set_comprehension", "generator_expression",
		"match_statement", "case_clause":
		return 1
	}
	return 0
}

// extractImports extracts import statements from the AST.
func (p *PythonParser) extractImports(node *sitter.Node, content []byte, metrics *parser.FileMetrics) {
	cursor := node.Walk()
	defer cursor.Close()

	var walk func()
	walk = func() {
		n := cursor.Node()

		imports := extractImportsFromNode(n, content)
		metrics.Imports = append(metrics.Imports, imports...)

		// Recurse into children
		if cursor.GotoFirstChild() {
			for {
				walk()
				if !cursor.GotoNextSibling() {
					break
				}
			}
			cursor.GotoParent()
		}
	}

	walk()
}

// extractImportsFromNode extracts imports from a single node.
func extractImportsFromNode(n *sitter.Node, content []byte) []string {
	switch n.Kind() {
	case "import_statement":
		return extractDirectImports(n, content)
	case "import_from_statement":
		return extractFromImport(n, content)
	}
	return nil
}

// extractDirectImports handles "import foo, bar" statements.
func extractDirectImports(n *sitter.Node, content []byte) []string {
	var imports []string
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child == nil {
			continue
		}

		kind := child.Kind()
		if kind != "dotted_name" && kind != "aliased_import" {
			continue
		}

		importText := string(content[child.StartByte():child.EndByte()])
		// Extract just the module name from "module as alias"
		if idx := strings.Index(importText, " as "); idx >= 0 {
			importText = importText[:idx]
		}
		imports = append(imports, strings.TrimSpace(importText))
	}
	return imports
}

// extractFromImport handles "from foo import bar" statements.
func extractFromImport(n *sitter.Node, content []byte) []string {
	moduleNode := n.ChildByFieldName("module_name")
	if moduleNode == nil {
		return nil
	}
	moduleName := string(content[moduleNode.StartByte():moduleNode.EndByte()])
	return []string{moduleName}
}

// countLines counts total, code, comment, and blank lines in Python source.
func countLines(content []byte) (total, code, comment, blank int) {
	lines := bytes.Split(content, []byte("\n"))
	total = len(lines)

	inMultilineString := false
	multilineDelim := []byte("")

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// Handle multiline strings (docstrings)
		if inMultilineString {
			comment++
			if bytes.Contains(trimmed, multilineDelim) {
				inMultilineString = false
			}
			continue
		}

		// Check for multiline string start
		if bytes.HasPrefix(trimmed, []byte(`"""`)) || bytes.HasPrefix(trimmed, []byte(`'''`)) {
			if bytes.HasPrefix(trimmed, []byte(`"""`)) {
				multilineDelim = []byte(`"""`)
			} else {
				multilineDelim = []byte(`'''`)
			}

			// Check if it ends on the same line
			rest := trimmed[3:]
			if !bytes.Contains(rest, multilineDelim) {
				inMultilineString = true
			}
			comment++
			continue
		}

		if len(trimmed) == 0 {
			blank++
		} else if bytes.HasPrefix(trimmed, []byte("#")) {
			comment++
		} else {
			code++
		}
	}

	return
}

// extractModuleName extracts a module name from the file path.
func extractModuleName(path string) string {
	// Get base name without extension
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".py")
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
