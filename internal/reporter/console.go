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

		if issue.Type != "low_comment_ratio" {
			fmt.Printf("  %s: %d (threshold: %d)\n", formatIssueType(issue.Type), issue.Value, issue.Threshold)
		} else {
			fmt.Printf("  Current: %d%% (recommended: >%d%%)\n", issue.Value, issue.Threshold)
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
	default:
		return "Value"
	}
}

func sumCommentLines(result *analyzer.AnalysisResult) int {
	total := 0
	for _, file := range result.Files {
		total += file.Metrics.CommentLines
	}
	return total
}
