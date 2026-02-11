// Package language provides abstractions for multi-language code analysis.
//
// This package defines the core interfaces that language-specific implementations
// must satisfy. Each supported language provides its own implementation of these
// interfaces, encapsulating language-specific parsing, detection, coverage, and
// dependency analysis.
//
// The Language interface is the primary abstraction that orchestrates all
// language-specific components. Language providers are registered at init time
// and can be retrieved by name or auto-detected from project files.
package language

import (
	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// Language defines the interface for language-specific analysis components.
//
// Each supported programming language implements this interface to provide
// its own parser, detectors, coverage runner, and dependency analyzer.
// Language providers are registered at init time and can be retrieved
// by name or auto-detected from project files.
type Language interface {
	// Name returns the lowercase identifier for this language (e.g., "go", "python", "typescript").
	// This is used for CLI flags and configuration.
	Name() string

	// DisplayName returns the human-readable name (e.g., "Go", "Python", "TypeScript").
	// This is used in user-facing messages and reports.
	DisplayName() string

	// Extensions returns the file extensions for this language (e.g., [".go"], [".py"], [".ts", ".tsx"]).
	// Extensions should include the leading dot.
	Extensions() []string

	// DefaultExcludePatterns returns glob patterns to exclude by default for this language.
	// For example, Go excludes "vendor/**" and "**/*_test.go".
	DefaultExcludePatterns() []string

	// Parser returns a parser for this language's source files.
	// The workers parameter controls parallel parsing (0=auto, 1=sequential).
	Parser(workers int) parser.Parser

	// DetectorRunner returns a runner for anti-pattern detectors.
	// Each language handles its own AST internally - the analyzer never sees language-specific types.
	DetectorRunner(cfg *config.AnalysisConfig) DetectorRunner

	// CoverageRunner returns a runner for test coverage analysis.
	// Returns nil if coverage analysis is not supported for this language.
	CoverageRunner(cfg *config.AnalysisConfig, statusReporter status.Reporter) CoverageRunner

	// DependencyAnalyzer returns an analyzer for dependency analysis.
	// Returns nil if dependency analysis is not supported for this language.
	DependencyAnalyzer(projectPath string) (DependencyAnalyzer, error)
}

// DetectorRunner runs anti-pattern detectors for a specific language.
//
// This interface decouples the generic analyzer from language-specific AST types.
// Each language's implementation handles its own parsing and AST walking internally,
// returning generic Issue objects that the analyzer can process.
type DetectorRunner interface {
	// RunDetectors analyzes a file and returns any issues found by enabled detectors.
	// The implementation may re-parse the file to obtain language-specific AST.
	RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue
}

// CoverageRunner runs test coverage analysis for a specific language.
//
// Each language has its own test framework and coverage tools. This interface
// abstracts the language-specific details of running coverage analysis.
type CoverageRunner interface {
	// RunCoverage runs test coverage analysis for the project.
	// Returns per-package coverage results or an error if coverage cannot be run.
	RunCoverage(projectPath string, excludePatterns []string) ([]*coverage.PackageCoverage, error)
}

// DependencyAnalyzer analyzes dependencies for a specific language.
//
// Each language has its own module system and import conventions. This interface
// abstracts the language-specific details of dependency analysis.
type DependencyAnalyzer interface {
	// Analyze extracts and categorizes dependencies from parsed files.
	// Returns per-package dependency information.
	Analyze(files []*parser.FileMetrics) ([]*dependencies.PackageDependencies, error)

	// DetectCircularDependencies finds circular import chains.
	// Returns a list of detected cycles.
	DetectCircularDependencies(files []*parser.FileMetrics) ([]*dependencies.CircularDependency, error)
}
