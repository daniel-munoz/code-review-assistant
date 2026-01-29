// Package golang provides Go language support for code analysis.
//
// This package implements the language.Language interface for Go,
// wrapping the existing Go-specific parsing, detection, coverage,
// and dependency analysis components.
package golang

import (
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func init() {
	language.Register(&GoLanguage{})
}

// GoLanguage implements language.Language for Go source code.
type GoLanguage struct{}

// Name returns "go".
func (g *GoLanguage) Name() string {
	return "go"
}

// DisplayName returns "Go".
func (g *GoLanguage) DisplayName() string {
	return "Go"
}

// Extensions returns [".go"].
func (g *GoLanguage) Extensions() []string {
	return []string{".go"}
}

// DefaultExcludePatterns returns Go-specific exclude patterns.
func (g *GoLanguage) DefaultExcludePatterns() []string {
	return []string{
		"vendor/**",
		"**/*_test.go",
		"**/testdata/**",
		"**/*.pb.go",
	}
}

// Parser returns a Go AST-based parser.
func (g *GoLanguage) Parser() parser.Parser {
	return parser.NewParser()
}

// DetectorRunner returns a Go-specific detector runner.
func (g *GoLanguage) DetectorRunner(cfg *config.AnalysisConfig) language.DetectorRunner {
	return NewGoDetectorRunner(cfg)
}

// CoverageRunner returns a Go test coverage runner.
func (g *GoLanguage) CoverageRunner(cfg *config.AnalysisConfig, statusReporter status.Reporter) language.CoverageRunner {
	return coverage.NewRunner(cfg.CoverageTimeout, statusReporter)
}

// DependencyAnalyzer returns a Go dependency analyzer.
func (g *GoLanguage) DependencyAnalyzer(projectPath string) (language.DependencyAnalyzer, error) {
	return dependencies.NewAnalyzer(projectPath)
}
