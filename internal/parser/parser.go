package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/parallel"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// Parser defines the interface for parsing source files and extracting metrics.
//
// Implementations analyze source code structure and calculate metrics without
// executing the code. Each language has its own Parser implementation.
type Parser interface {
	// ParseFile parses a single source file and returns its metrics.
	//
	// Returns FileMetrics containing:
	//   - Line counts (total, code, comments, blank)
	//   - Package/module information
	//   - Import list
	//   - Function/method metrics (size, complexity, signature)
	//
	// Returns an error if:
	//   - file doesn't have the correct extension for this parser
	//   - file doesn't exist
	//   - file contains syntax errors
	ParseFile(path string) (*FileMetrics, error)

	// ParseDirectory recursively parses all source files in a directory tree.
	//
	// Discovers and parses all files with matching extensions under the specified
	// path, excluding files and directories matching the excludePatterns (glob patterns).
	//
	// The extensions parameter specifies which file extensions to include (e.g., [".go"]).
	// The statusReporter parameter is used to report parsing progress.
	//
	// This method is fault-tolerant: parse errors for individual files are
	// collected but don't stop processing. Returns:
	//   - metrics: FileMetrics for all successfully parsed files
	//   - errors: parse errors for files that failed (empty if all succeeded)
	ParseDirectory(path string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*FileMetrics, []error)
}

// ASTParser implements Parser using Go's AST (Abstract Syntax Tree) packages.
//
// This implementation uses go/parser to build an AST and go/ast to traverse it,
// extracting structural metrics without requiring the code to compile or run.
type ASTParser struct {
	Workers int // Number of parallel workers (0=auto, 1=sequential)
}

// NewParser creates a new Parser instance using AST-based parsing for Go.
//
// Parameters:
//   - workers: number of parallel workers (0=auto uses runtime.NumCPU, 1=sequential)
//
// Returns an ASTParser that can parse individual Go files or entire directory trees.
func NewParser(workers int) Parser {
	return &ASTParser{Workers: workers}
}

// ParseFile parses a single Go file and extracts comprehensive metrics.
//
// Uses go/parser.ParseFile to build an AST, then walks the tree to extract:
//   - File-level metrics (line counts, imports, package name)
//   - Function-level metrics (size, complexity, parameters, returns)
//
// Returns an error if the file doesn't exist, isn't a .go file, or contains
// syntax errors that prevent parsing.
func (p *ASTParser) ParseFile(path string) (*FileMetrics, error) {
	// Verify file exists and is a .go file
	if !strings.HasSuffix(path, ".go") {
		return nil, fmt.Errorf("not a Go file: %s", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// Parse the file using AST
	return parseGoFile(path)
}

// ParseDirectory recursively discovers and parses all source files in a directory.
//
// Uses filepath.Walk to traverse the directory tree, parsing all files with
// matching extensions that don't match the exclude patterns. Patterns support glob syntax:
//   - vendor/** - exclude vendor directory and subdirectories
//   - **/*_test.go - exclude all test files
//   - **/testdata/** - exclude testdata directories
//
// This method is designed to be fault-tolerant:
//   - Parse errors for individual files are collected but don't stop processing
//   - Directory access errors are logged and skipped
//   - Returns metrics for all successfully parsed files
//
// The excludePatterns are matched against paths relative to rootPath.
// The extensions parameter specifies which file extensions to include (e.g., [".go"]).
//
// If Workers is set to 1, parsing is sequential. Otherwise, files are parsed
// in parallel using a worker pool.
func (p *ASTParser) ParseDirectory(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*FileMetrics, []error) {
	workers := p.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	if workers == 1 {
		return p.parseDirectorySequential(rootPath, excludePatterns, extensions, statusReporter)
	}
	return p.parseDirectoryParallel(rootPath, excludePatterns, extensions, statusReporter, workers)
}

// parseDirectorySequential parses files one at a time (original implementation).
func (p *ASTParser) parseDirectorySequential(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter) ([]*FileMetrics, []error) {
	var allMetrics []*FileMetrics
	var errors []error
	var fileCount int

	// Initial status
	statusReporter.Update("[PARSE] Discovering files...")

	// Walk the directory tree
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, fmt.Errorf("error accessing path %s: %w", path, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			// Check if this directory should be excluded
			if ShouldExclude(path, rootPath, excludePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process files with matching extensions
		if !HasMatchingExtension(path, extensions) {
			return nil
		}

		// Check if file matches exclude patterns
		if ShouldExclude(path, rootPath, excludePatterns) {
			return nil
		}

		// Increment file count and update progress
		fileCount++
		baseName := filepath.Base(path)
		statusReporter.UpdateProgress("[PARSE] Parsing files", len(allMetrics)+1, fileCount, baseName)

		// Parse the file
		metrics, parseErr := p.ParseFile(path)
		if parseErr != nil {
			errors = append(errors, fmt.Errorf("error parsing %s: %w", path, parseErr))
			return nil // Continue parsing other files
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
	metrics *FileMetrics
	err     error
}

// parseDirectoryParallel parses files concurrently using a worker pool.
func (p *ASTParser) parseDirectoryParallel(rootPath string, excludePatterns []string, extensions []string, statusReporter status.Reporter, workers int) ([]*FileMetrics, []error) {
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
			if ShouldExclude(path, rootPath, excludePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if !HasMatchingExtension(path, extensions) {
			return nil
		}

		if ShouldExclude(path, rootPath, excludePatterns) {
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

	// Create worker pool
	pool := parallel.NewWorkerPool(workers, func(path string) parseResult {
		metrics, parseErr := p.ParseFile(path)
		progressReporter.RecordProgress(filepath.Base(path))
		if parseErr != nil {
			return parseResult{err: fmt.Errorf("error parsing %s: %w", path, parseErr)}
		}
		return parseResult{metrics: metrics}
	})
	pool.Start()

	// Submit all files in a goroutine to avoid blocking
	go func() {
		for _, path := range filePaths {
			pool.Submit(path)
		}
		pool.Close()
	}()

	// Collect results (single-threaded - no mutex needed)
	var allMetrics []*FileMetrics
	var parseErrors []error

	for result := range pool.Results() {
		if result.err != nil {
			parseErrors = append(parseErrors, result.err)
		} else if result.metrics != nil {
			allMetrics = append(allMetrics, result.metrics)
		}
	}

	// Combine walk errors and parse errors
	allErrors := append(walkErrors, parseErrors...)
	return allMetrics, allErrors
}
