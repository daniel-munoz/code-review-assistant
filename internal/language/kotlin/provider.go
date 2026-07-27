// Package kotlin provides Kotlin language support for code analysis.
//
// This package implements the language.Language interface for Kotlin,
// using tree-sitter for parsing and custom detectors for anti-pattern detection.
package kotlin

import (
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/language"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func init() {
	language.Register(&KotlinLanguage{})
}

// KotlinLanguage implements language.Language for Kotlin source code.
type KotlinLanguage struct{}

// Name returns "kotlin".
func (k *KotlinLanguage) Name() string {
	return "kotlin"
}

// DisplayName returns "Kotlin".
func (k *KotlinLanguage) DisplayName() string {
	return "Kotlin"
}

// Extensions returns [".kt"].
// .kts files (Gradle build scripts) are deliberately excluded: they are build
// configuration, not production code, and would pollute codebase metrics.
func (k *KotlinLanguage) Extensions() []string {
	return []string{".kt"}
}

// DefaultExcludePatterns returns Kotlin/Gradle-specific exclude patterns.
func (k *KotlinLanguage) DefaultExcludePatterns() []string {
	return []string{
		"**/build/**",
		"**/.gradle/**",
		"**/generated/**",
		"**/src/test/**",
		"**/src/testFixtures/**",
		"**/*Test.kt",
		"**/*Spec.kt",
	}
}

// Parser returns a tree-sitter based Kotlin parser with the specified number of workers.
func (k *KotlinLanguage) Parser(workers int) parser.Parser {
	return NewParser(workers)
}

// DetectorRunner returns a Kotlin-specific detector runner.
func (k *KotlinLanguage) DetectorRunner(cfg *config.AnalysisConfig) language.DetectorRunner {
	return NewDetectorRunner(cfg)
}

// CoverageRunner returns a Gradle-based (Kover/JaCoCo) coverage runner.
func (k *KotlinLanguage) CoverageRunner(cfg *config.AnalysisConfig, statusReporter status.Reporter) language.CoverageRunner {
	return NewCoverageRunner(cfg.CoverageTimeout, statusReporter)
}

// DependencyAnalyzer returns a Kotlin dependency analyzer grouping by
// declared package.
func (k *KotlinLanguage) DependencyAnalyzer(projectPath string) (language.DependencyAnalyzer, error) {
	return NewDependencyAnalyzer(projectPath)
}
