package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/constants"
)

// formatNumber formats a number with thousands separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s", addThousandsSeparator(n))
}

// addThousandsSeparator adds commas as thousands separators
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

// percentage calculates percentage of part to total
func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return (float64(part) / float64(total)) * constants.PercentageMultiplier
}

// formatPath tries to make path relative to current directory for shorter display
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

// formatIssueType maps internal issue type identifiers to human-readable labels.
//
// This function translates issue type codes into display-friendly metric names
// that describe what value is being measured. For example:
// - "large_file" -> "Lines" (file size in lines)
// - "high_complexity" -> "Complexity" (cyclomatic complexity)
// - "too_many_parameters" -> "Parameters" (parameter count)
//
// Some issue types like "magic_number", "duplicate_error_handling", and
// "circular_dependency" return an empty string because their messages already
// contain all necessary context and don't need additional value formatting.
//
// Returns the appropriate label string, or "Value" as a generic fallback.
var issueTypeLabels = map[string]string{
	"large_file":               "Lines",
	"long_function":            "Lines",
	"high_complexity":          "Complexity",
	"too_many_parameters":      "Parameters",
	"deep_nesting":             "Nesting Depth",
	"too_many_returns":         "Return Statements",
	"low_coverage":             "Coverage",
	"too_many_imports":         "Imports",
	"too_many_external_deps":   "External Dependencies",
	"magic_number":             "", // These don't need value formatting
	"duplicate_error_handling": "",
	"circular_dependency":      "",
}

func formatIssueType(issueType string) string {
	if label, ok := issueTypeLabels[issueType]; ok {
		return label
	}
	return "Value" // Default for unknown types
}

// formatDependencyCycle formats a dependency cycle for display
func formatDependencyCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	return strings.Join(cycle, " -> ")
}

// sumCommentLines sums total comment lines across all files
func sumCommentLines(result *analyzer.AnalysisResult) int {
	total := 0
	for _, file := range result.Files {
		total += file.Metrics.CommentLines
	}
	return total
}

// formatDelta formats an integer delta with sign and percentage
func formatDelta(change int, percent float64) string {
	sign := ""
	if change > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%d (%s%.1f%%)", sign, change, sign, percent)
}

// formatFloatDelta formats a float delta with sign and percentage
func formatFloatDelta(change float64, percent float64) string {
	sign := ""
	if change > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.2f (%s%.1f%%)", sign, change, sign, percent)
}

// formatSeverity returns a formatted severity string
func formatSeverity(severity string) string {
	switch severity {
	case "error":
		return "ERROR"
	case "warning":
		return "WARNING"
	case "info":
		return "INFO"
	default:
		return strings.ToUpper(severity)
	}
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
