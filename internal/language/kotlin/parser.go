// Package kotlin provides Kotlin language support for code analysis.
package kotlin

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	kotlinlang "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"

	"github.com/daniel-munoz/code-review-assistant/internal/parallel"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// KotlinParser implements parser.Parser for Kotlin source files using tree-sitter.
type KotlinParser struct {
	language *sitter.Language
	workers  int // Number of parallel workers (0=auto, 1=sequential)
}

// NewParser creates a new Kotlin parser.
// The workers parameter controls parallel parsing (0=auto, 1=sequential).
func NewParser(workers int) *KotlinParser {
	return &KotlinParser{
		language: sitter.NewLanguage(kotlinlang.Language()),
		workers:  workers,
	}
}

// ParseFile parses a single Kotlin file and extracts comprehensive metrics.
func (p *KotlinParser) ParseFile(path string) (*parser.FileMetrics, error) {
	if !strings.HasSuffix(path, ".kt") {
		return nil, fmt.Errorf("not a Kotlin file: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	tsParser := sitter.NewParser()
	defer tsParser.Close()
	tsParser.SetLanguage(p.language)

	tree := tsParser.Parse(content, nil)
	defer tree.Close()

	root := tree.RootNode()

	metrics := &parser.FileMetrics{
		FilePath:    path,
		PackageName: extractPackageName(root, content),
		Language:    "kotlin",
		Functions:   []*parser.FunctionMetrics{},
		Imports:     []string{},
	}

	p.extractDeclarations(root, content, metrics, "")
	extractImports(root, content, metrics)
	metrics.TotalLines, metrics.CodeLines, metrics.CommentLines, metrics.BlankLines = countLines(content)

	return metrics, nil
}

// ParseDirectory recursively parses all Kotlin files in a directory.
// If Workers is set to 1, parsing is sequential. Otherwise, files are parsed in parallel.
func (p *KotlinParser) ParseDirectory(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
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
func (p *KotlinParser) parseDirectorySequential(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*parser.FileMetrics, []error) {
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
func (p *KotlinParser) parseDirectoryParallel(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter, workers int) ([]*parser.FileMetrics, []error) {
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

// extractDeclarations walks a source_file or class_body node collecting functions.
// typeName is the enclosing class/object/interface name ("" at top level).
// Local functions inside function bodies are intentionally not collected,
// matching the Python provider's behavior.
func (p *KotlinParser) extractDeclarations(node *sitter.Node, content []byte, metrics *parser.FileMetrics, typeName string) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "function_declaration":
			metrics.Functions = append(metrics.Functions, p.extractFunction(child, content, typeName))

		case "class_declaration", "object_declaration":
			// class_declaration covers classes and interfaces
			name := declarationName(child, content)
			if body := childOfKind(child, "class_body"); body != nil {
				p.extractDeclarations(body, content, metrics, name)
			}

		case "companion_object":
			// Companion members belong to the enclosing class
			if body := childOfKind(child, "class_body"); body != nil {
				p.extractDeclarations(body, content, metrics, typeName)
			}
		}
	}
}

// extractFunction extracts metrics from a function_declaration node.
func (p *KotlinParser) extractFunction(node *sitter.Node, content []byte, typeName string) *parser.FunctionMetrics {
	fm := &parser.FunctionMetrics{
		StartLine:    int(node.StartPosition().Row) + 1, // 0-indexed to 1-indexed
		EndLine:      int(node.EndPosition().Row) + 1,
		ReceiverType: typeName,
	}
	fm.Lines = fm.EndLine - fm.StartLine + 1

	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		fm.Name = string(content[nameNode.StartByte():nameNode.EndByte()])
	}

	if paramsNode := childOfKind(node, "function_value_parameters"); paramsNode != nil {
		fm.Parameters = countChildrenOfKind(paramsNode, "parameter")
	}

	// Kotlin functions return a single value (Unit if not explicit)
	fm.ReturnValues = 1

	if bodyNode := childOfKind(node, "function_body"); bodyNode != nil {
		fm.Complexity = calculateComplexity(bodyNode)
	} else {
		fm.Complexity = 1 // abstract/interface methods without a body
	}

	return fm
}

// declarationName returns the name of a class/object declaration.
func declarationName(node *sitter.Node, content []byte) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return string(content[nameNode.StartByte():nameNode.EndByte()])
	}
	if idNode := childOfKind(node, "identifier"); idNode != nil {
		return string(content[idNode.StartByte():idNode.EndByte()])
	}
	return ""
}

// childOfKind returns the first direct child with the given kind, or nil.
func childOfKind(node *sitter.Node, kind string) *sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		if child := node.Child(i); child != nil && child.Kind() == kind {
			return child
		}
	}
	return nil
}

// countChildrenOfKind counts direct children with the given kind.
func countChildrenOfKind(node *sitter.Node, kind string) int {
	count := 0
	for i := uint(0); i < node.ChildCount(); i++ {
		if child := node.Child(i); child != nil && child.Kind() == kind {
			count++
		}
	}
	return count
}

// calculateComplexity calculates cyclomatic complexity for a function body.
func calculateComplexity(node *sitter.Node) int {
	complexity := 1 // Base complexity

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		complexity += complexityIncrement(n)

		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(node)
	return complexity
}

// complexityIncrement returns the complexity contribution of a node.
// Operator tokens (&&, ||, ?:) appear as children of binary_expression,
// so visiting every node (including anonymous tokens) counts them naturally.
func complexityIncrement(n *sitter.Node) int {
	switch n.Kind() {
	case "if_expression",
		"for_statement", "while_statement", "do_while_statement",
		"catch_block",
		"&&", "||", "?:":
		return 1
	case "when_entry":
		if isElseWhenEntry(n) {
			return 0 // the else branch is the default path, like Go's default case
		}
		return 1
	}
	return 0
}

// isElseWhenEntry reports whether a when_entry is the else branch.
func isElseWhenEntry(n *sitter.Node) bool {
	return childOfKind(n, "else") != nil
}

// extractPackageName extracts the package declaration (e.g., "com.example.service").
func extractPackageName(root *sitter.Node, content []byte) string {
	if header := childOfKind(root, "package_header"); header != nil {
		if qual := childOfKind(header, "qualified_identifier"); qual != nil {
			return string(content[qual.StartByte():qual.EndByte()])
		}
	}
	return ""
}

// extractImports collects import statements. Imports are only legal at the
// top of a Kotlin file, so only direct children of source_file are scanned.
func extractImports(root *sitter.Node, content []byte, metrics *parser.FileMetrics) {
	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil || child.Kind() != "import" {
			continue
		}
		if qual := childOfKind(child, "qualified_identifier"); qual != nil {
			metrics.Imports = append(metrics.Imports, string(content[qual.StartByte():qual.EndByte()]))
		}
	}
}

// countLines counts total, code, comment, and blank lines in Kotlin source.
// Handles // line comments and /* */ block comments (including KDoc).
func countLines(content []byte) (total, code, comment, blank int) {
	lines := bytes.Split(content, []byte("\n"))
	total = len(lines)

	inBlockComment := false

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if inBlockComment {
			comment++
			if bytes.Contains(trimmed, []byte("*/")) {
				inBlockComment = false
			}
			continue
		}

		switch {
		case len(trimmed) == 0:
			blank++
		case bytes.HasPrefix(trimmed, []byte("//")):
			comment++
		case bytes.HasPrefix(trimmed, []byte("/*")):
			comment++
			if !bytes.Contains(trimmed[2:], []byte("*/")) {
				inBlockComment = true
			}
		default:
			code++
		}
	}

	return
}

// constNamePattern matches UPPER_CASE constant-style identifiers.
var constNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

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
