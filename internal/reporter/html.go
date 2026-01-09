package reporter

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/constants"
)

// HTMLReporter implements Reporter for HTML dashboard output
type HTMLReporter struct {
	config *config.OutputConfig
	output io.Writer
}

// NewHTMLReporter creates a new HTMLReporter
func NewHTMLReporter(cfg *config.OutputConfig) *HTMLReporter {
	return &HTMLReporter{
		config: cfg,
		output: os.Stdout, // Default to stdout
	}
}

// Report generates an HTML formatted dashboard report
func (hr *HTMLReporter) Report(result *analyzer.AnalysisResult, comp *comparison.ComparisonResult) error {
	// Sort issues by severity
	analyzer.SortIssuesBySeverity(result.Issues)

	// If output file is specified, write to file instead of stdout
	if hr.config.OutputFile != "" {
		file, err := hr.createOutputFile(hr.config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		hr.output = file
	}

	// Build template data
	data := hr.buildTemplateData(result, comp)

	// Parse and execute template
	tmpl, err := template.New("html").Funcs(hr.templateFuncs()).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	if err := tmpl.Execute(hr.output, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// createOutputFile creates the output file, including any necessary directories
func (hr *HTMLReporter) createOutputFile(path string) (*os.File, error) {
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

// TemplateData contains all data needed for the HTML template
type TemplateData struct {
	// Basic info
	ProjectPath string
	ProjectName string
	Timestamp   string
	Duration    string

	// Summary metrics
	TotalFiles     int
	TotalLines     int
	TotalCodeLines int
	CodePercent    float64
	CommentLines   int
	CommentPercent float64
	BlankLines     int
	BlankPercent   float64
	TotalFunctions int

	// Aggregate metrics
	Metrics *analyzer.AggregateMetrics

	// Issues
	Issues       []*analyzer.Issue
	IssueCount   int
	ErrorCount   int
	WarningCount int
	InfoCount    int

	// Coverage
	Coverage *analyzer.CoverageReport

	// Dependencies
	Dependencies *analyzer.DependencyReport

	// Comparison
	Comparison *comparison.ComparisonResult

	// Files (verbose mode)
	Files   []*analyzer.FileAnalysis
	Verbose bool
}

// buildTemplateData constructs the data structure for the HTML template
func (hr *HTMLReporter) buildTemplateData(result *analyzer.AnalysisResult, comp *comparison.ComparisonResult) *TemplateData {
	data := &TemplateData{
		ProjectPath:    result.ProjectPath,
		ProjectName:    filepath.Base(result.ProjectPath),
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		TotalFiles:     result.TotalFiles,
		TotalLines:     result.TotalLines,
		TotalCodeLines: result.TotalCodeLines,
		TotalFunctions: result.TotalFunctions,
		Metrics:        result.Metrics,
		Issues:         result.Issues,
		IssueCount:     len(result.Issues),
		Coverage:       result.Coverage,
		Dependencies:   result.Dependencies,
		Comparison:     comp,
		Files:          result.Files,
		Verbose:        hr.config.Verbose,
	}

	// Calculate percentages
	if result.TotalLines > 0 {
		data.CodePercent = float64(result.TotalCodeLines) / float64(result.TotalLines) * constants.PercentageMultiplier
	}

	// Calculate comment and blank lines
	data.CommentLines = sumCommentLines(result)
	if result.TotalLines > 0 {
		data.CommentPercent = float64(data.CommentLines) / float64(result.TotalLines) * constants.PercentageMultiplier
	}

	data.BlankLines = result.TotalLines - result.TotalCodeLines - data.CommentLines
	if result.TotalLines > 0 {
		data.BlankPercent = float64(data.BlankLines) / float64(result.TotalLines) * constants.PercentageMultiplier
	}

	// Count issues by severity
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			data.ErrorCount++
		case "warning":
			data.WarningCount++
		case "info":
			data.InfoCount++
		}
	}

	return data
}

// templateFuncs returns the template functions for HTML rendering
func (hr *HTMLReporter) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatNumber":   formatNumber,
		"formatFloat":    formatFloat,
		"severityIcon":   severityIcon,
		"severityClass":  severityClass,
		"issueTypeLabel": formatIssueType,
		"trendIcon":      trendIcon,
		"trendClass":     trendClass,
		"formatDelta":    formatDelta,
		"add":            add,
		"slice":          slice,
	}
}

// severityIcon returns an icon for the given severity
func severityIcon(severity string) string {
	switch severity {
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return ""
	}
}

// severityClass returns a CSS class for the given severity
func severityClass(severity string) string {
	switch severity {
	case "error":
		return "severity-error"
	case "warning":
		return "severity-warning"
	case "info":
		return "severity-info"
	default:
		return ""
	}
}

// formatFloat formats a float with 1 decimal place
func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

// trendIcon returns an icon for a trend direction
func trendIcon(trend comparison.TrendDirection) string {
	return trend.Icon()
}

// trendClass returns a CSS class for a trend direction
func trendClass(trend comparison.TrendDirection) string {
	switch trend {
	case comparison.TrendImproving:
		return "trend-improving"
	case comparison.TrendWorsening:
		return "trend-worsening"
	case comparison.TrendStable:
		return "trend-stable"
	default:
		return ""
	}
}

// add adds two integers (template helper)
func add(a, b int) int {
	return a + b
}

// slice returns a slice from index start to end (template helper)
func slice(arr []string, start int) []string {
	if start >= len(arr) {
		return []string{}
	}
	return arr[start:]
}
