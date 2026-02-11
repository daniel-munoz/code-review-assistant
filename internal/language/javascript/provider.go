// Package javascript provides JavaScript/TypeScript language support for code analysis.
//
// This package implements the language.Language interface for JavaScript and TypeScript,
// using tree-sitter for parsing and custom detectors for anti-pattern detection.
// A single language handler ("javascript") parses both JS and TS using the TypeScript grammar.
package javascript

import (
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func init() {
	language.Register(&JavaScriptLanguage{})
}

// JavaScriptLanguage implements language.Language for JavaScript/TypeScript source code.
type JavaScriptLanguage struct{}

// Name returns "javascript".
func (j *JavaScriptLanguage) Name() string {
	return "javascript"
}

// DisplayName returns "JavaScript/TypeScript".
func (j *JavaScriptLanguage) DisplayName() string {
	return "JavaScript/TypeScript"
}

// Extensions returns all JavaScript and TypeScript file extensions.
func (j *JavaScriptLanguage) Extensions() []string {
	return []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"}
}

// DefaultExcludePatterns returns JavaScript/TypeScript-specific exclude patterns.
func (j *JavaScriptLanguage) DefaultExcludePatterns() []string {
	return []string{
		// Dependencies and build artifacts
		"**/node_modules/**",
		"**/dist/**",
		"**/build/**",
		"**/.next/**",
		"**/out/**",
		"**/.nuxt/**",
		"**/.output/**",

		// Test files
		"**/*.test.js",
		"**/*.test.ts",
		"**/*.test.jsx",
		"**/*.test.tsx",
		"**/*.spec.js",
		"**/*.spec.ts",
		"**/*.spec.jsx",
		"**/*.spec.tsx",
		"**/__tests__/**",
		"**/__mocks__/**",

		// Type declarations and minified files
		"**/*.d.ts",
		"**/*.min.js",
		"**/*.bundle.js",

		// Config files
		"**/jest.config.js",
		"**/jest.config.ts",
		"**/webpack.config.js",
		"**/vite.config.ts",
		"**/vite.config.js",
		"**/rollup.config.js",
		"**/next.config.js",
		"**/next.config.mjs",
	}
}

// Parser returns a tree-sitter based JavaScript/TypeScript parser with the specified number of workers.
func (j *JavaScriptLanguage) Parser(workers int) parser.Parser {
	return NewParser(workers)
}

// DetectorRunner returns a JavaScript/TypeScript-specific detector runner.
func (j *JavaScriptLanguage) DetectorRunner(cfg *config.AnalysisConfig) language.DetectorRunner {
	return NewDetectorRunner(cfg)
}

// CoverageRunner returns a JavaScript/TypeScript test coverage runner.
// Supports Jest and Vitest test frameworks.
func (j *JavaScriptLanguage) CoverageRunner(cfg *config.AnalysisConfig, statusReporter status.Reporter) language.CoverageRunner {
	return NewCoverageRunner(cfg.CoverageTimeout, statusReporter)
}

// DependencyAnalyzer returns a JavaScript/TypeScript dependency analyzer.
func (j *JavaScriptLanguage) DependencyAnalyzer(projectPath string) (language.DependencyAnalyzer, error) {
	return NewDependencyAnalyzer(projectPath)
}
