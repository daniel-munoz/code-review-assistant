package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
)

// ConsoleReporter implements Reporter for console output.
//
// This reporter formats analysis results for terminal display with:
// - Colored severity indicators (❌/⚠️/ℹ️)
// - Formatted tables for metrics and comparisons
// - Verbose mode for detailed per-file breakdowns
// - Historical comparison summaries (Phase 3)
//
// The output is organized into logical sections:
// 1. Header with project info and timestamp
// 2. Comparison summary (if previous report exists)
// 3. Summary statistics
// 4. Aggregate metrics
// 5. Issues list (sorted by severity)
// 6. Largest files
// 7. Most complex functions
// 8. Test coverage report
// 9. Dependency analysis
// 10. Detailed comparison (verbose mode)
// 11. Per-file details (verbose mode)
//
// Printing logic is split across:
// - console.go (this file): Main struct and Report() orchestration
// - console_printers.go: Individual section print methods
// - console_formatters.go: Formatting helper functions
type ConsoleReporter struct {
	config *config.OutputConfig
}

// NewConsoleReporter creates a new ConsoleReporter with the given configuration.
func NewConsoleReporter(cfg *config.OutputConfig) *ConsoleReporter {
	return &ConsoleReporter{
		config: cfg,
	}
}

// Report outputs the analysis results to the console.
//
// This is the main orchestration method that coordinates all report sections.
// It sorts issues by severity, prints a formatted header, then delegates to
// specialized print methods for each section.
//
// The report structure adapts based on:
// - Available data (coverage, dependencies, comparison)
// - Verbose mode setting
// - Issue count
func (cr *ConsoleReporter) Report(result *analyzer.AnalysisResult, comp *comparison.ComparisonResult) error {
	// Sort issues by severity
	analyzer.SortIssuesBySeverity(result.Issues)

	// Header
	fmt.Println("Code Review Assistant - Analysis Report")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Printf("Project: %s\n", result.ProjectPath)
	fmt.Printf("Analyzed: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Phase 3: Comparison Summary (if available)
	if comp != nil {
		cr.printComparisonSummary(comp)
		fmt.Println()
	}

	// Summary
	cr.printSummary(result)
	fmt.Println()

	// Aggregate Metrics
	cr.printAggregateMetrics(result.Metrics)
	fmt.Println()

	// Issues
	if len(result.Issues) > 0 {
		cr.printIssues(result.Issues)
		fmt.Println()
	}

	// Largest Files
	if len(result.Metrics.LargestFiles) > 0 {
		cr.printLargestFiles(result.Metrics.LargestFiles)
		fmt.Println()
	}

	// Most Complex Functions
	if len(result.Metrics.MostComplexFunctions) > 0 {
		cr.printMostComplexFunctions(result.Metrics.MostComplexFunctions)
		fmt.Println()
	}

	// Coverage Report
	if result.Coverage != nil {
		cr.printCoverageReport(result.Coverage)
		fmt.Println()
	}

	// Dependency Report
	if result.Dependencies != nil {
		cr.printDependencyReport(result.Dependencies)
		fmt.Println()
	}

	// Phase 3: Detailed Comparison (if available and verbose)
	if comp != nil && cr.config.Verbose {
		cr.printDetailedComparison(comp)
		fmt.Println()
	}

	// Verbose mode: per-file details
	if cr.config.Verbose {
		cr.printFileDetails(result.Files)
		fmt.Println()
	}

	fmt.Println("Analysis complete.")

	return nil
}
