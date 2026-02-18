// Package php provides PHP language support for code analysis.
//
// This package implements the language.Language interface for PHP,
// using tree-sitter for parsing and custom detectors for anti-pattern detection.
package php

import (
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func init() {
	language.Register(&PHPLanguage{})
}

// PHPLanguage implements language.Language for PHP source code.
type PHPLanguage struct{}

// Name returns "php".
func (p *PHPLanguage) Name() string {
	return "php"
}

// DisplayName returns "PHP".
func (p *PHPLanguage) DisplayName() string {
	return "PHP"
}

// Extensions returns PHP file extensions.
func (p *PHPLanguage) Extensions() []string {
	return []string{".php"}
}

// DefaultExcludePatterns returns PHP-specific exclude patterns.
func (p *PHPLanguage) DefaultExcludePatterns() []string {
	return []string{
		// Dependencies
		"**/vendor/**",
		"**/node_modules/**",

		// Test files
		"**/*Test.php",
		"**/*_test.php",
		"**/tests/**",
		"**/Tests/**",

		// Database migrations
		"**/migrations/**",
		"**/database/migrations/**",

		// Template files (mixed HTML/PHP)
		"**/*.blade.php",

		// Cache and generated files
		"**/var/**",
		"**/storage/framework/**",
		"**/bootstrap/cache/**",
	}
}

// Parser returns a tree-sitter based PHP parser with the specified number of workers.
func (p *PHPLanguage) Parser(workers int) parser.Parser {
	return NewParser(workers)
}

// DetectorRunner returns a PHP-specific detector runner.
func (p *PHPLanguage) DetectorRunner(cfg *config.AnalysisConfig) language.DetectorRunner {
	return NewDetectorRunner(cfg)
}

// CoverageRunner returns a PHP test coverage runner.
// Supports PHPUnit with Clover XML output.
func (p *PHPLanguage) CoverageRunner(cfg *config.AnalysisConfig, statusReporter status.Reporter) language.CoverageRunner {
	return NewCoverageRunner(cfg.CoverageTimeout, statusReporter)
}

// DependencyAnalyzer returns a PHP dependency analyzer.
func (p *PHPLanguage) DependencyAnalyzer(projectPath string) (language.DependencyAnalyzer, error) {
	return NewDependencyAnalyzer(projectPath)
}
