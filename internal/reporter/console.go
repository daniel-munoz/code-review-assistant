package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/olekukonko/tablewriter"
)

// ConsoleReporter implements Reporter for console output
type ConsoleReporter struct {
	config *config.OutputConfig
}

// NewConsoleReporter creates a new ConsoleReporter
func NewConsoleReporter(cfg *config.OutputConfig) *ConsoleReporter {
	return &ConsoleReporter{
		config: cfg,
	}
}

// Report outputs the analysis results to the console
func (cr *ConsoleReporter) Report(result *analyzer.AnalysisResult) error {
	// Sort issues by severity
	analyzer.SortIssuesBySeverity(result.Issues)

	// Header
	fmt.Println("Code Review Assistant - Analysis Report")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Printf("Project: %s\n", result.ProjectPath)
	fmt.Printf("Analyzed: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()

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

	// Verbose mode: per-file details
	if cr.config.Verbose {
		cr.printFileDetails(result.Files)
		fmt.Println()
	}

	fmt.Println("Analysis complete.")

	return nil
}

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

// Helper functions

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s", addThousandsSeparator(n))
}

func addThousandsSeparator(n int) string {
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return (float64(part) / float64(total)) * 100
}

func formatPath(path string) string {
	// Try to make path relative to current directory
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	relPath, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	// If relative path is shorter, use it
	if len(relPath) < len(path) {
		return relPath
	}
	return path
}

func formatIssueType(issueType string) string {
	switch issueType {
	case "large_file":
		return "Lines"
	case "long_function":
		return "Lines"
	case "high_complexity":
		return "Complexity"
	case "too_many_parameters":
		return "Parameters"
	case "deep_nesting":
		return "Nesting Depth"
	case "too_many_returns":
		return "Return Statements"
	case "low_coverage":
		return "Coverage"
	case "too_many_imports":
		return "Imports"
	case "too_many_external_deps":
		return "External Dependencies"
	case "magic_number", "duplicate_error_handling", "circular_dependency":
		return "" // These don't need value formatting
	default:
		return "Value"
	}
}

func formatDependencyCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	return strings.Join(cycle, " -> ")
}

func sumCommentLines(result *analyzer.AnalysisResult) int {
	total := 0
	for _, file := range result.Files {
		total += file.Metrics.CommentLines
	}
	return total
}
