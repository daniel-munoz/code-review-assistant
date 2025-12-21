package reporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
)

// MarkdownReporter implements Reporter for Markdown output
type MarkdownReporter struct {
	config *config.OutputConfig
	output io.Writer
}

// NewMarkdownReporter creates a new MarkdownReporter
func NewMarkdownReporter(cfg *config.OutputConfig) *MarkdownReporter {
	return &MarkdownReporter{
		config: cfg,
		output: os.Stdout, // Default to stdout
	}
}

// Report generates a Markdown formatted report
func (mr *MarkdownReporter) Report(result *analyzer.AnalysisResult, comp *comparison.ComparisonResult) error {
	// Sort issues by severity
	analyzer.SortIssuesBySeverity(result.Issues)

	// If output file is specified, write to file instead of stdout
	if mr.config.OutputFile != "" {
		file, err := mr.createOutputFile(mr.config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		mr.output = file
	}

	// Generate report sections
	mr.writeHeader(result)

	// Phase 3: Add comparison summary if available
	if comp != nil {
		mr.writeComparisonSummary(comp)
	}

	mr.writeSummary(result)
	mr.writeAggregateMetrics(result.Metrics)
	mr.writeIssues(result.Issues)
	mr.writeLargestFiles(result.Metrics.LargestFiles)
	mr.writeMostComplexFunctions(result.Metrics.MostComplexFunctions)
	mr.writeCoverageReport(result.Coverage)
	mr.writeDependencyReport(result.Dependencies)

	// Phase 3: Add detailed comparison if available and verbose
	if comp != nil && mr.config.Verbose {
		mr.writeDetailedComparison(comp)
	}

	// Verbose mode: per-file details
	if mr.config.Verbose {
		mr.writeFileDetails(result.Files)
	}

	mr.writeFooter()

	return nil
}

func (mr *MarkdownReporter) createOutputFile(path string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Create or truncate file
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (mr *MarkdownReporter) writeHeader(result *analyzer.AnalysisResult) {
	fmt.Fprintf(mr.output, "# Code Review Assistant - Analysis Report\n\n")
	fmt.Fprintf(mr.output, "**Project:** %s  \n", result.ProjectPath)
	fmt.Fprintf(mr.output, "**Analyzed:** %s  \n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(mr.output, "---\n\n")
}

func (mr *MarkdownReporter) writeSummary(result *analyzer.AnalysisResult) {
	fmt.Fprintf(mr.output, "## Summary\n\n")

	fmt.Fprintf(mr.output, "| Metric | Value |\n")
	fmt.Fprintf(mr.output, "|--------|-------|\n")
	fmt.Fprintf(mr.output, "| Total Files | %s |\n", formatNumber(result.TotalFiles))
	fmt.Fprintf(mr.output, "| Total Lines | %s |\n", formatNumber(result.TotalLines))
	fmt.Fprintf(mr.output, "| Code Lines | %s (%.1f%%) |\n",
		formatNumber(result.TotalCodeLines),
		percentage(result.TotalCodeLines, result.TotalLines))

	commentLines := sumCommentLines(result)
	blankLines := result.TotalLines - result.TotalCodeLines - commentLines

	fmt.Fprintf(mr.output, "| Comment Lines | %s (%.1f%%) |\n",
		formatNumber(commentLines),
		percentage(commentLines, result.TotalLines))
	fmt.Fprintf(mr.output, "| Blank Lines | %s (%.1f%%) |\n",
		formatNumber(blankLines),
		percentage(blankLines, result.TotalLines))
	fmt.Fprintf(mr.output, "| Total Functions | %s |\n\n",
		formatNumber(result.TotalFunctions))
}

func (mr *MarkdownReporter) writeAggregateMetrics(metrics *analyzer.AggregateMetrics) {
	fmt.Fprintf(mr.output, "## Aggregate Metrics\n\n")

	fmt.Fprintf(mr.output, "| Metric | Value |\n")
	fmt.Fprintf(mr.output, "|--------|-------|\n")
	fmt.Fprintf(mr.output, "| Average Function Length | %.1f lines |\n", metrics.AverageFunctionLength)
	fmt.Fprintf(mr.output, "| Function Length (95th %%ile) | %d lines |\n", metrics.FunctionLengthP95)
	fmt.Fprintf(mr.output, "| Comment Ratio | %.1f%% |\n", metrics.CommentRatio*100)
	fmt.Fprintf(mr.output, "| Average Complexity | %.1f |\n", metrics.AverageComplexity)
	fmt.Fprintf(mr.output, "| Complexity (95th %%ile) | %d |\n\n", metrics.ComplexityP95)
}

func (mr *MarkdownReporter) writeIssues(issues []*analyzer.Issue) {
	if len(issues) == 0 {
		return
	}

	fmt.Fprintf(mr.output, "## Issues Found (%d)\n\n", len(issues))

	for _, issue := range issues {
		// Icon based on severity
		icon := ""
		switch issue.Severity {
		case "error":
			icon = "❌"
		case "warning":
			icon = "⚠️"
		case "info":
			icon = "ℹ️"
		}

		fmt.Fprintf(mr.output, "%s **[%s]** %s\n",
			icon, strings.ToUpper(issue.Severity), issue.Message)

		if issue.File != "" {
			location := fmt.Sprintf("`%s`", issue.File)
			if issue.Line > 0 {
				location = fmt.Sprintf("`%s:%d`", issue.File, issue.Line)
			}
			fmt.Fprintf(mr.output, "  - **File:** %s\n", location)
		}

		if issue.Function != "" {
			fmt.Fprintf(mr.output, "  - **Function:** `%s`\n", issue.Function)
		}

		// Handle display based on issue type
		issueTypeStr := formatIssueType(issue.Type)
		if issue.Type == "magic_number" {
			// Already in message
		} else if issue.Type == "duplicate_error_handling" {
			fmt.Fprintf(mr.output, "  - **Error checks:** %d (threshold: %d)\n", issue.Value, issue.Threshold)
		} else if issue.Type == "low_comment_ratio" {
			fmt.Fprintf(mr.output, "  - **Current:** %d%% (recommended: >%d%%)\n", issue.Value, issue.Threshold)
		} else if issueTypeStr != "" {
			fmt.Fprintf(mr.output, "  - **%s:** %d (threshold: %d)\n", issueTypeStr, issue.Value, issue.Threshold)
		}

		fmt.Fprintf(mr.output, "\n")
	}
}

func (mr *MarkdownReporter) writeLargestFiles(files []*analyzer.FileSize) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(mr.output, "## Largest Files\n\n")
	fmt.Fprintf(mr.output, "| Rank | File | Lines |\n")
	fmt.Fprintf(mr.output, "|------|------|-------|\n")

	for i, file := range files {
		fmt.Fprintf(mr.output, "| %d | `%s` | %s |\n",
			i+1,
			file.Path,
			formatNumber(file.Lines))
	}
	fmt.Fprintf(mr.output, "\n")
}

func (mr *MarkdownReporter) writeMostComplexFunctions(functions []*analyzer.FunctionInfo) {
	if len(functions) == 0 {
		return
	}

	fmt.Fprintf(mr.output, "## Most Complex Functions\n\n")
	fmt.Fprintf(mr.output, "| Rank | Function | Complexity | Lines | File |\n")
	fmt.Fprintf(mr.output, "|------|----------|------------|-------|------|\n")

	for i, fn := range functions {
		fmt.Fprintf(mr.output, "| %d | `%s` | %d | %d | `%s` |\n",
			i+1,
			fn.Function,
			fn.Complexity,
			fn.Lines,
			fn.File)
	}
	fmt.Fprintf(mr.output, "\n")
}

func (mr *MarkdownReporter) writeCoverageReport(coverage *analyzer.CoverageReport) {
	if coverage == nil {
		return
	}

	fmt.Fprintf(mr.output, "## Test Coverage\n\n")
	fmt.Fprintf(mr.output, "| Metric | Value |\n")
	fmt.Fprintf(mr.output, "|--------|-------|\n")
	fmt.Fprintf(mr.output, "| Average Coverage | %.1f%% |\n", coverage.AverageCoverage)
	fmt.Fprintf(mr.output, "| Total Packages | %d |\n", len(coverage.Packages))

	if coverage.LowCoverageCount > 0 {
		fmt.Fprintf(mr.output, "| Packages Below Threshold | %d |\n", coverage.LowCoverageCount)
	}
	fmt.Fprintf(mr.output, "\n")

	// Verbose mode: per-package details
	if mr.config.Verbose && len(coverage.Packages) > 0 {
		fmt.Fprintf(mr.output, "<details>\n")
		fmt.Fprintf(mr.output, "<summary>Package Coverage Details</summary>\n\n")

		fmt.Fprintf(mr.output, "| Package | Coverage |\n")
		fmt.Fprintf(mr.output, "|---------|----------|\n")

		for _, pkg := range coverage.Packages {
			status := ""
			if pkg.Error != "" {
				status = fmt.Sprintf("Error: %s", pkg.Error)
			} else if pkg.Skipped {
				status = "No tests"
			} else {
				status = fmt.Sprintf("%.1f%%", pkg.Coverage)
			}

			fmt.Fprintf(mr.output, "| `%s` | %s |\n", pkg.PackagePath, status)
		}

		fmt.Fprintf(mr.output, "\n</details>\n\n")
	}
}

func (mr *MarkdownReporter) writeDependencyReport(dependencies *analyzer.DependencyReport) {
	if dependencies == nil {
		return
	}

	fmt.Fprintf(mr.output, "## Dependencies\n\n")
	fmt.Fprintf(mr.output, "| Metric | Value |\n")
	fmt.Fprintf(mr.output, "|--------|-------|\n")
	fmt.Fprintf(mr.output, "| Total Packages | %d |\n", dependencies.TotalPackages)

	if dependencies.HighImportCount > 0 {
		fmt.Fprintf(mr.output, "| Packages with High Imports | %d |\n", dependencies.HighImportCount)
	}

	if dependencies.HighExternalCount > 0 {
		fmt.Fprintf(mr.output, "| Packages with High External Deps | %d |\n", dependencies.HighExternalCount)
	}

	if len(dependencies.CircularDependencies) > 0 {
		fmt.Fprintf(mr.output, "| Circular Dependencies | %d |\n", len(dependencies.CircularDependencies))
	}
	fmt.Fprintf(mr.output, "\n")

	// Show circular dependencies if found
	if len(dependencies.CircularDependencies) > 0 {
		fmt.Fprintf(mr.output, "### ⚠️ Circular Dependencies Detected\n\n")
		for i, cd := range dependencies.CircularDependencies {
			fmt.Fprintf(mr.output, "%d. %s\n", i+1, formatDependencyCycle(cd.Cycle))
		}
		fmt.Fprintf(mr.output, "\n")
	}

	// Verbose mode: per-package details
	if mr.config.Verbose && len(dependencies.Packages) > 0 {
		fmt.Fprintf(mr.output, "<details>\n")
		fmt.Fprintf(mr.output, "<summary>Package Dependency Details</summary>\n\n")

		for _, pkg := range dependencies.Packages {
			fmt.Fprintf(mr.output, "**%s:**\n", pkg.PackageName)
			fmt.Fprintf(mr.output, "- Total Imports: %d (Stdlib: %d, Internal: %d, External: %d)\n",
				pkg.TotalImports,
				len(pkg.StdlibImports),
				len(pkg.InternalImports),
				len(pkg.ExternalImports))

			if len(pkg.ExternalImports) > 0 {
				fmt.Fprintf(mr.output, "- External Dependencies: %s\n", strings.Join(pkg.ExternalImports, ", "))
			}
			fmt.Fprintf(mr.output, "\n")
		}

		fmt.Fprintf(mr.output, "</details>\n\n")
	}
}

func (mr *MarkdownReporter) writeFileDetails(files []*analyzer.FileAnalysis) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(mr.output, "<details>\n")
	fmt.Fprintf(mr.output, "<summary>File Details</summary>\n\n")

	for _, file := range files {
		fmt.Fprintf(mr.output, "### `%s`\n\n", file.Path)
		fmt.Fprintf(mr.output, "- **Lines:** %d (Code: %d, Comments: %d, Blank: %d)\n",
			file.Metrics.TotalLines,
			file.Metrics.CodeLines,
			file.Metrics.CommentLines,
			file.Metrics.BlankLines)
		fmt.Fprintf(mr.output, "- **Functions:** %d\n", len(file.Metrics.Functions))
		fmt.Fprintf(mr.output, "- **Comment Ratio:** %.1f%%\n", file.Metrics.CommentRatio()*100)

		if file.LargeFile {
			fmt.Fprintf(mr.output, "- ⚠️ **Large file**\n")
		}
		fmt.Fprintf(mr.output, "\n")
	}

	fmt.Fprintf(mr.output, "</details>\n\n")
}

// Phase 3: Comparison reporting methods

func (mr *MarkdownReporter) writeComparisonSummary(comp *comparison.ComparisonResult) {
	fmt.Fprintf(mr.output, "## 📊 Comparison with Previous Report\n\n")
	fmt.Fprintf(mr.output, "**Previous Report:** %s\n\n", comp.PreviousTimestamp.Format("2006-01-02 15:04:05"))

	// Trends
	fmt.Fprintf(mr.output, "### Trends\n\n")
	fmt.Fprintf(mr.output, "- **Complexity:** %s %s\n", comp.Trends.Complexity.Icon(), comp.Trends.Complexity.String())
	fmt.Fprintf(mr.output, "- **Coverage:** %s %s\n", comp.Trends.Coverage.Icon(), comp.Trends.Coverage.String())
	fmt.Fprintf(mr.output, "- **Issue Count:** %s %s\n\n", comp.Trends.IssueCount.Icon(), comp.Trends.IssueCount.String())

	// Metrics comparison table
	fmt.Fprintf(mr.output, "### Metrics\n\n")
	fmt.Fprintf(mr.output, "| Metric | Previous | Current | Change |\n")
	fmt.Fprintf(mr.output, "|--------|----------|---------|--------|\n")
	fmt.Fprintf(mr.output, "| Files | %d | %d | %s |\n",
		comp.Deltas.TotalFiles.Previous,
		comp.Deltas.TotalFiles.Current,
		formatMarkdownDelta(comp.Deltas.TotalFiles.Change, comp.Deltas.TotalFiles.Percent))
	fmt.Fprintf(mr.output, "| Lines | %d | %d | %s |\n",
		comp.Deltas.TotalLines.Previous,
		comp.Deltas.TotalLines.Current,
		formatMarkdownDelta(comp.Deltas.TotalLines.Change, comp.Deltas.TotalLines.Percent))
	fmt.Fprintf(mr.output, "| Functions | %d | %d | %s |\n",
		comp.Deltas.TotalFunctions.Previous,
		comp.Deltas.TotalFunctions.Current,
		formatMarkdownDelta(comp.Deltas.TotalFunctions.Change, comp.Deltas.TotalFunctions.Percent))
	fmt.Fprintf(mr.output, "| Avg Complexity | %.2f | %.2f | %s |\n",
		comp.Deltas.AvgComplexity.Previous,
		comp.Deltas.AvgComplexity.Current,
		formatMarkdownFloatDelta(comp.Deltas.AvgComplexity.Change, comp.Deltas.AvgComplexity.Percent))
	fmt.Fprintf(mr.output, "| Avg Coverage | %.1f%% | %.1f%% | %s |\n",
		comp.Deltas.AvgCoverage.Previous,
		comp.Deltas.AvgCoverage.Current,
		formatMarkdownFloatDelta(comp.Deltas.AvgCoverage.Change, comp.Deltas.AvgCoverage.Percent))
	fmt.Fprintf(mr.output, "| Issues | %d | %d | %s |\n\n",
		comp.Deltas.IssueCount.Previous,
		comp.Deltas.IssueCount.Current,
		formatMarkdownDelta(comp.Deltas.IssueCount.Change, comp.Deltas.IssueCount.Percent))
}

func (mr *MarkdownReporter) writeDetailedComparison(comp *comparison.ComparisonResult) {
	// New Issues
	if len(comp.NewIssues) > 0 {
		fmt.Fprintf(mr.output, "### 🆕 New Issues (%d)\n\n", len(comp.NewIssues))
		fmt.Fprintf(mr.output, "| Severity | Type | File | Line | Function |\n")
		fmt.Fprintf(mr.output, "|----------|------|------|------|----------|\n")
		for _, issue := range comp.NewIssues {
			fmt.Fprintf(mr.output, "| %s | %s | `%s` | %d | `%s` |\n",
				formatMarkdownSeverity(issue.Severity),
				issue.Type,
				filepath.Base(issue.File),
				issue.Line,
				truncateString(issue.Function, 40))
		}
		fmt.Fprintf(mr.output, "\n")
	}

	// Fixed Issues
	if len(comp.FixedIssues) > 0 {
		fmt.Fprintf(mr.output, "### ✅ Fixed Issues (%d)\n\n", len(comp.FixedIssues))
		fmt.Fprintf(mr.output, "| Severity | Type | File | Line | Function |\n")
		fmt.Fprintf(mr.output, "|----------|------|------|------|----------|\n")
		for _, issue := range comp.FixedIssues {
			fmt.Fprintf(mr.output, "| %s | %s | `%s` | %d | `%s` |\n",
				formatMarkdownSeverity(issue.Severity),
				issue.Type,
				filepath.Base(issue.File),
				issue.Line,
				truncateString(issue.Function, 40))
		}
		fmt.Fprintf(mr.output, "\n")
	}
}

// formatMarkdownDelta formats an integer delta for Markdown
func formatMarkdownDelta(change int, percent float64) string {
	if change == 0 {
		return "0"
	}
	sign := "+"
	if change < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%d (%s%.1f%%)", sign, change, sign, percent)
}

// formatMarkdownFloatDelta formats a float delta for Markdown
func formatMarkdownFloatDelta(change float64, percent float64) string {
	if change == 0 {
		return "0"
	}
	sign := "+"
	if change < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.2f (%s%.1f%%)", sign, change, sign, percent)
}

// formatMarkdownSeverity formats severity with emoji for Markdown
func formatMarkdownSeverity(severity string) string {
	switch severity {
	case "error":
		return "🔴 Error"
	case "warning":
		return "🟡 Warning"
	case "info":
		return "🔵 Info"
	default:
		return severity
	}
}

func (mr *MarkdownReporter) writeFooter() {
	fmt.Fprintf(mr.output, "---\n\n")
	fmt.Fprintf(mr.output, "*Analysis complete.*\n")
}
