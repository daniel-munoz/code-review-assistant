// Package python provides Python language support for code analysis.
//
// This package implements the language.Language interface for Python,
// using tree-sitter for parsing and custom detectors for anti-pattern detection.
package python

import (
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func init() {
	language.Register(&PythonLanguage{})
}

// PythonLanguage implements language.Language for Python source code.
type PythonLanguage struct{}

// Name returns "python".
func (p *PythonLanguage) Name() string {
	return "python"
}

// DisplayName returns "Python".
func (p *PythonLanguage) DisplayName() string {
	return "Python"
}

// Extensions returns [".py"].
func (p *PythonLanguage) Extensions() []string {
	return []string{".py"}
}

// DefaultExcludePatterns returns Python-specific exclude patterns.
func (p *PythonLanguage) DefaultExcludePatterns() []string {
	return []string{
		"**/__pycache__/**",
		"**/venv/**",
		"**/.venv/**",
		"**/env/**",
		"**/.env/**",
		"**/site-packages/**",
		"**/test_*.py",
		"**/*_test.py",
		"**/conftest.py",
		"**/tests/**",
		"**/.tox/**",
		"**/.pytest_cache/**",
	}
}

// Parser returns a tree-sitter based Python parser with the specified number of workers.
func (p *PythonLanguage) Parser(workers int) parser.Parser {
	return NewParser(workers)
}

// DetectorRunner returns a Python-specific detector runner.
func (p *PythonLanguage) DetectorRunner(cfg *config.AnalysisConfig) language.DetectorRunner {
	return NewDetectorRunner(cfg)
}

// CoverageRunner returns nil as Python coverage is not yet implemented.
// Python coverage support will be added in a future phase using pytest-cov.
func (p *PythonLanguage) CoverageRunner(cfg *config.AnalysisConfig, statusReporter status.Reporter) language.CoverageRunner {
	return nil
}

// DependencyAnalyzer returns nil as Python dependency analysis is not yet implemented.
// Python dependency analysis will be added in a future phase using requirements.txt/pyproject.toml parsing.
func (p *PythonLanguage) DependencyAnalyzer(projectPath string) (language.DependencyAnalyzer, error) {
	return nil, nil
}
