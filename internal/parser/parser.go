package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Parser defines the interface for parsing Go source files
type Parser interface {
	// ParseFile parses a single Go file and returns its metrics
	ParseFile(path string) (*FileMetrics, error)

	// ParseDirectory recursively parses all Go files in a directory
	// Returns metrics for all successfully parsed files and any errors encountered
	ParseDirectory(path string, excludePatterns []string) ([]*FileMetrics, []error)
}

// ASTParser implements Parser using Go's AST packages
type ASTParser struct{}

// NewParser creates a new Parser instance
func NewParser() Parser {
	return &ASTParser{}
}

// ParseFile parses a single Go file and extracts metrics
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

// ParseDirectory recursively finds and parses all Go files in a directory
func (p *ASTParser) ParseDirectory(rootPath string, excludePatterns []string) ([]*FileMetrics, []error) {
	var allMetrics []*FileMetrics
	var errors []error

	// Walk the directory tree
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, fmt.Errorf("error accessing path %s: %w", path, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			// Check if this directory should be excluded
			if shouldExclude(path, rootPath, excludePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Check if file matches exclude patterns
		if shouldExclude(path, rootPath, excludePatterns) {
			return nil
		}

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

// shouldExclude checks if a path matches any of the exclude patterns
func shouldExclude(path string, rootPath string, patterns []string) bool {
	// Get relative path from root
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		relPath = path
	}

	// Normalize path separators for pattern matching
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		if matchPattern(relPath, pattern) {
			return true
		}
	}
	return false
}

// matchPattern performs simple glob-style pattern matching
// Supports: **, *, ?
func matchPattern(path, pattern string) bool {
	// Simple implementation for common patterns
	pattern = filepath.ToSlash(pattern)

	// Handle ** (match any directories)
	if strings.Contains(pattern, "**") {
		// Pattern types:
		// "vendor/**" - matches vendor/ and everything under it
		// "**/vendor/**" - matches any path with vendor as a directory component
		// "**/*.go" - matches any .go file
		// "**/testdata/**" - matches any path containing testdata directory

		parts := strings.Split(pattern, "**")

		// Check if pattern starts with a specific directory (not **)
		startsWithPrefix := parts[0] != "" && !strings.HasPrefix(pattern, "**")

		// Extract non-empty parts (these must appear in the path)
		var matchers []string
		for i, part := range parts {
			part = strings.Trim(part, "/")
			if part != "" {
				// For suffix (last part), handle specially if it has wildcards
				if i == len(parts)-1 && (strings.Contains(part, "*") || strings.Contains(part, "?")) {
					// Suffix with wildcards - match basename
					matched, _ := filepath.Match(part, filepath.Base(path))
					return matched
				}
				matchers = append(matchers, part)
			}
		}

		// If no matchers, pattern is just "**" - matches everything
		if len(matchers) == 0 {
			return true
		}

		// Check matchers against path
		pathComponents := strings.Split(path, "/")
		matcherIdx := 0
		startIdx := 0

		// If pattern starts with a prefix (like "vendor/**"), first matcher must be at start
		if startsWithPrefix && len(pathComponents) > 0 {
			if pathComponents[0] != matchers[0] {
				return false
			}
			matcherIdx = 1
			startIdx = 1
		}

		// Match remaining matchers in order
		for i := startIdx; i < len(pathComponents) && matcherIdx < len(matchers); i++ {
			if pathComponents[i] == matchers[matcherIdx] {
				matcherIdx++
			}
		}

		// All matchers should be found
		return matcherIdx == len(matchers)
	}

	// Handle patterns with * but not **
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		// Try matching the full path first
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also try matching just the basename
		matched, err = filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		return false
	}

	// Exact match
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}
