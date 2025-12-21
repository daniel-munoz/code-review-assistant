package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/olekukonko/tablewriter"
)

// printSummary prints the summary statistics
func (cr *ConsoleReporter) printSummary(result *analyzer.AnalysisResult) {
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("-", 60))

	data := [][]string{
		{"Total Files", formatNumber(result.TotalFiles)},
		{"Total Lines", formatNumber(result.TotalLines)},
		{"Code Lines", fmt.Sprintf("%s (%.1f%%)", formatNumber(result.TotalCodeLines), percentage(result.TotalCodeLines, result.TotalLines))},
		{"Comment Lines", fmt.Sprintf("%d (%.1f%%)", result.TotalLines-result.TotalCodeLines-(result.TotalLines-result.TotalCodeLines-sumCommentLines(result)), percentage(sumCommentLines(result), result.TotalLines))},
		{"Blank Lines", fmt.Sprintf("%d (%.1f%%)", result.TotalLines-result.TotalCodeLines-sumCommentLines(result), percentage(result.TotalLines-result.TotalCodeLines-sumCommentLines(result), result.TotalLines))},
		{"Total Functions", formatNumber(result.TotalFunctions)},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Metric", "Value"})
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()
}

// printAggregateMetrics prints aggregate code quality metrics
func (cr *ConsoleReporter) printAggregateMetrics(metrics *analyzer.AggregateMetrics) {
	fmt.Println("AGGREGATE METRICS")
	fmt.Println(strings.Repeat("-", 60))

	data := [][]string{
		{"Average Function Length", fmt.Sprintf("%.1f lines", metrics.AverageFunctionLength)},
		{"Function Length (95th %%ile)", fmt.Sprintf("%d lines", metrics.FunctionLengthP95)},
		{"Comment Ratio", fmt.Sprintf("%.1f%%", metrics.CommentRatio*100)},
		{"Average Complexity", fmt.Sprintf("%.1f", metrics.AverageComplexity)},
		{"Complexity (95th %%ile)", fmt.Sprintf("%d", metrics.ComplexityP95)},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()
}

// printIssues displays all detected issues with contextual formatting.
//
// This function handles different issue types with appropriate visual markers
// and formatting. Each issue displays:
// - Severity icon (❌ for errors, ⚠️ for warnings, ℹ️ for info)
// - File location with optional line number
// - Function name (if applicable)
// - Issue-specific metrics (value vs threshold)
//
// Issues are expected to be pre-sorted by severity using SortIssuesBySeverity.
// The function adapts its output based on issue type to provide the most
// relevant information for each category.
func (cr *ConsoleReporter) printIssues(issues []*analyzer.Issue) {
	fmt.Printf("ISSUES FOUND (%d)\n", len(issues))
	fmt.Println(strings.Repeat("-", 60))

	for _, issue := range issues {
		// Icon based on severity
		icon := ""
		switch issue.Severity {
		case "error":
			icon = "❌"
		case "warning":
			icon = "⚠️ "
		case "info":
			icon = "ℹ️ "
		}

		fmt.Printf("%s [%s] %s\n", icon, strings.ToUpper(issue.Severity), issue.Message)

		if issue.File != "" {
			fmt.Printf("  File: %s", formatPath(issue.File))
			if issue.Line > 0 {
				fmt.Printf(":%d", issue.Line)
			}
			fmt.Println()
		}

		if issue.Function != "" {
			fmt.Printf("  Function: %s\n", issue.Function)
		}

		// Handle display based on issue type
		issueTypeStr := formatIssueType(issue.Type)
		if issue.Type == "magic_number" {
			// Already in message
		} else if issue.Type == "duplicate_error_handling" {
			fmt.Printf("  Error checks: %d (threshold: %d)\n", issue.Value, issue.Threshold)
		} else if issue.Type == "low_comment_ratio" {
			fmt.Printf("  Current: %d%% (recommended: >%d%%)\n", issue.Value, issue.Threshold)
		} else if issueTypeStr != "" {
			fmt.Printf("  %s: %d (threshold: %d)\n", issueTypeStr, issue.Value, issue.Threshold)
		}

		fmt.Println()
	}
}

// printLargestFiles prints the largest files by line count
func (cr *ConsoleReporter) printLargestFiles(files []*analyzer.FileSize) {
	fmt.Println("LARGEST FILES")
	fmt.Println(strings.Repeat("-", 60))

	data := make([][]string, 0, len(files))
	for i, file := range files {
		data = append(data, []string{
			fmt.Sprintf("%d.", i+1),
			formatPath(file.Path),
			fmt.Sprintf("%d lines", file.Lines),
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()
}

// printMostComplexFunctions prints functions with highest cyclomatic complexity
func (cr *ConsoleReporter) printMostComplexFunctions(functions []*analyzer.FunctionInfo) {
	fmt.Println("MOST COMPLEX FUNCTIONS")
	fmt.Println(strings.Repeat("-", 60))

	data := make([][]string, 0, len(functions))
	for i, fn := range functions {
		data = append(data, []string{
			fmt.Sprintf("%d.", i+1),
			fn.Function,
			fmt.Sprintf("CC=%d", fn.Complexity),
			fmt.Sprintf("%d lines", fn.Lines),
			formatPath(fn.File),
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()
}

// printCoverageReport prints test coverage statistics
func (cr *ConsoleReporter) printCoverageReport(coverage *analyzer.CoverageReport) {
	fmt.Println("TEST COVERAGE")
	fmt.Println(strings.Repeat("-", 60))

	// Summary
	data := [][]string{
		{"Average Coverage", fmt.Sprintf("%.1f%%", coverage.AverageCoverage)},
		{"Total Packages", fmt.Sprintf("%d", len(coverage.Packages))},
	}

	if coverage.LowCoverageCount > 0 {
		data = append(data, []string{"Packages Below Threshold", fmt.Sprintf("%d", coverage.LowCoverageCount)})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()

	// Verbose mode: per-package details
	if cr.config.Verbose && len(coverage.Packages) > 0 {
		fmt.Println()
		fmt.Println("Package Coverage Details:")
		fmt.Println()

		packageData := make([][]string, 0, len(coverage.Packages))
		for _, pkg := range coverage.Packages {
			status := ""
			if pkg.Error != "" {
				status = fmt.Sprintf("Error: %s", pkg.Error)
			} else if pkg.Skipped {
				status = "No tests"
			} else {
				status = fmt.Sprintf("%.1f%%", pkg.Coverage)
			}

			packageData = append(packageData, []string{
				pkg.PackagePath,
				status,
			})
		}

		pkgTable := tablewriter.NewWriter(os.Stdout)
		pkgTable.SetHeader([]string{"Package", "Coverage"})
		pkgTable.SetBorder(false)
		pkgTable.SetColumnSeparator(" | ")
		pkgTable.SetAlignment(tablewriter.ALIGN_LEFT)
		pkgTable.AppendBulk(packageData)
		pkgTable.Render()
	}
}

// printDependencyReport displays package dependency analysis results.
//
// This function provides both summary statistics and detailed breakdowns:
// - Total package count
// - Packages exceeding import thresholds
// - Packages with high external dependency counts
// - Circular dependency warnings with cycle visualization
//
// In verbose mode, it also shows per-package details including:
// - Import categorization (stdlib, internal, external)
// - List of external dependencies
//
// Circular dependencies are highlighted with warning icons and formatted
// as dependency chains (A -> B -> C -> A) for easy identification.
func (cr *ConsoleReporter) printDependencyReport(dependencies *analyzer.DependencyReport) {
	fmt.Println("DEPENDENCIES")
	fmt.Println(strings.Repeat("-", 60))

	// Summary
	data := [][]string{
		{"Total Packages", fmt.Sprintf("%d", dependencies.TotalPackages)},
	}

	if dependencies.HighImportCount > 0 {
		data = append(data, []string{"Packages with High Imports", fmt.Sprintf("%d", dependencies.HighImportCount)})
	}

	if dependencies.HighExternalCount > 0 {
		data = append(data, []string{"Packages with High External Deps", fmt.Sprintf("%d", dependencies.HighExternalCount)})
	}

	if len(dependencies.CircularDependencies) > 0 {
		data = append(data, []string{"Circular Dependencies", fmt.Sprintf("%d", len(dependencies.CircularDependencies))})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()

	// Show circular dependencies if found
	if len(dependencies.CircularDependencies) > 0 {
		fmt.Println()
		fmt.Println("⚠️  Circular Dependencies Detected:")
		for i, cd := range dependencies.CircularDependencies {
			fmt.Printf("  %d. %s\n", i+1, formatDependencyCycle(cd.Cycle))
		}
	}

	// Verbose mode: per-package details
	if cr.config.Verbose && len(dependencies.Packages) > 0 {
		fmt.Println()
		fmt.Println("Package Dependency Details:")
		fmt.Println()

		for _, pkg := range dependencies.Packages {
			fmt.Printf("%s:\n", pkg.PackageName)
			fmt.Printf("  Total Imports: %d (Stdlib: %d, Internal: %d, External: %d)\n",
				pkg.TotalImports,
				len(pkg.StdlibImports),
				len(pkg.InternalImports),
				len(pkg.ExternalImports))

			if len(pkg.ExternalImports) > 0 {
				fmt.Printf("  External Dependencies: %s\n", strings.Join(pkg.ExternalImports, ", "))
			}
			fmt.Println()
		}
	}
}

// printFileDetails prints detailed per-file metrics (verbose mode)
func (cr *ConsoleReporter) printFileDetails(files []*analyzer.FileAnalysis) {
	fmt.Println("FILE DETAILS")
	fmt.Println(strings.Repeat("-", 60))

	for _, file := range files {
		fmt.Printf("\n%s\n", formatPath(file.Path))
		fmt.Printf("  Lines: %d (Code: %d, Comments: %d, Blank: %d)\n",
			file.Metrics.TotalLines,
			file.Metrics.CodeLines,
			file.Metrics.CommentLines,
			file.Metrics.BlankLines)
		fmt.Printf("  Functions: %d\n", len(file.Metrics.Functions))
		fmt.Printf("  Comment Ratio: %.1f%%\n", file.Metrics.CommentRatio()*100)

		if file.LargeFile {
			fmt.Println("  ⚠️  Large file")
		}
	}
}

// printComparisonSummary prints a summary of changes compared to the previous report
func (cr *ConsoleReporter) printComparisonSummary(comp *comparison.ComparisonResult) {
	fmt.Println("COMPARISON WITH PREVIOUS REPORT")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Previous: %s\n", comp.PreviousTimestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Trends summary
	fmt.Printf("Complexity:    %s  %s\n", comp.Trends.Complexity.Icon(), comp.Trends.Complexity.String())
	fmt.Printf("Coverage:      %s  %s\n", comp.Trends.Coverage.Icon(), comp.Trends.Coverage.String())
	fmt.Printf("Issue Count:   %s  %s\n", comp.Trends.IssueCount.Icon(), comp.Trends.IssueCount.String())
	fmt.Println()

	// Key metrics table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Metric", "Previous", "Current", "Change"})
	table.SetBorder(false)

	table.Append([]string{
		"Files",
		formatNumber(comp.Deltas.TotalFiles.Previous),
		formatNumber(comp.Deltas.TotalFiles.Current),
		formatDelta(comp.Deltas.TotalFiles.Change, comp.Deltas.TotalFiles.Percent),
	})
	table.Append([]string{
		"Lines",
		formatNumber(comp.Deltas.TotalLines.Previous),
		formatNumber(comp.Deltas.TotalLines.Current),
		formatDelta(comp.Deltas.TotalLines.Change, comp.Deltas.TotalLines.Percent),
	})
	table.Append([]string{
		"Functions",
		formatNumber(comp.Deltas.TotalFunctions.Previous),
		formatNumber(comp.Deltas.TotalFunctions.Current),
		formatDelta(comp.Deltas.TotalFunctions.Change, comp.Deltas.TotalFunctions.Percent),
	})
	table.Append([]string{
		"Avg Complexity",
		fmt.Sprintf("%.2f", comp.Deltas.AvgComplexity.Previous),
		fmt.Sprintf("%.2f", comp.Deltas.AvgComplexity.Current),
		formatFloatDelta(comp.Deltas.AvgComplexity.Change, comp.Deltas.AvgComplexity.Percent),
	})
	table.Append([]string{
		"Avg Coverage",
		fmt.Sprintf("%.1f%%", comp.Deltas.AvgCoverage.Previous),
		fmt.Sprintf("%.1f%%", comp.Deltas.AvgCoverage.Current),
		formatFloatDelta(comp.Deltas.AvgCoverage.Change, comp.Deltas.AvgCoverage.Percent),
	})
	table.Append([]string{
		"Issues",
		formatNumber(comp.Deltas.IssueCount.Previous),
		formatNumber(comp.Deltas.IssueCount.Current),
		formatDelta(comp.Deltas.IssueCount.Change, comp.Deltas.IssueCount.Percent),
	})

	table.Render()
}

// printDetailedComparison prints detailed information about new and fixed issues
func (cr *ConsoleReporter) printDetailedComparison(comp *comparison.ComparisonResult) {
	// New Issues
	if len(comp.NewIssues) > 0 {
		fmt.Println("NEW ISSUES")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Found %d new issue(s) since last analysis:\n\n", len(comp.NewIssues))

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Severity", "Type", "File", "Line", "Function"})
		table.SetBorder(false)

		for _, issue := range comp.NewIssues {
			table.Append([]string{
				formatSeverity(issue.Severity),
				issue.Type,
				filepath.Base(issue.File),
				fmt.Sprintf("%d", issue.Line),
				truncateString(issue.Function, 30),
			})
		}

		table.Render()
		fmt.Println()
	}

	// Fixed Issues
	if len(comp.FixedIssues) > 0 {
		fmt.Println("FIXED ISSUES")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Fixed %d issue(s) since last analysis:\n\n", len(comp.FixedIssues))

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Severity", "Type", "File", "Line", "Function"})
		table.SetBorder(false)

		for _, issue := range comp.FixedIssues {
			table.Append([]string{
				formatSeverity(issue.Severity),
				issue.Type,
				filepath.Base(issue.File),
				fmt.Sprintf("%d", issue.Line),
				truncateString(issue.Function, 30),
			})
		}

		table.Render()
	}
}
