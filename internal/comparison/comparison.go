package comparison

import (
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
)

// Comparator compares analysis results to identify trends and changes.
//
// The comparator takes two AnalysisResults (current and previous) and
// generates a ComparisonResult showing:
//   - Metric deltas (changes in numeric values)
//   - Trends (improving, worsening, or stable)
//   - Issue changes (new issues, fixed issues)
//
// Used for historical comparison when --compare flag is used.
type Comparator struct {
	stableThreshold float64 // Percentage change threshold for "stable" (default: 5%)
}

// NewComparator creates a new comparator with the given stable threshold.
//
// The stable threshold determines what percentage change is considered
// "stable" vs "improving/worsening". For example, with a 5% threshold:
//   - Complexity change of 2% is "stable"
//   - Complexity change of 8% is "improving" or "worsening"
//
// Parameters:
//   - stableThreshold: Percentage (0-100) for stable detection (e.g., 5.0 for 5%)
//
// Example:
//   comp := NewComparator(5.0)
//   result := comp.Compare(currentResult, previousResult)
func NewComparator(stableThreshold float64) *Comparator {
	if stableThreshold <= 0 {
		stableThreshold = 5.0 // Default to 5%
	}
	return &Comparator{
		stableThreshold: stableThreshold,
	}
}

// Compare generates a comparison between current and previous analysis results.
//
// Returns nil if previous is nil (nothing to compare against).
// Calculates deltas for all metrics, detects trends, and categorizes issues.
//
// Parameters:
//   - current: The current analysis result
//   - previous: The previous analysis result to compare against
//   - previousTimestamp: The timestamp of the previous report
//
// Returns:
//   - ComparisonResult with deltas, trends, and issue changes
//
// Example:
//   comp := NewComparator(5.0)
//   comparison := comp.Compare(currentResult, previousResult, previousTimestamp)
//   if comparison.Trends.Complexity == TrendImproving {
//       fmt.Println("Complexity is improving!")
//   }
func (c *Comparator) Compare(current, previous *analyzer.AnalysisResult, previousTimestamp time.Time) *ComparisonResult {
	if previous == nil {
		return nil
	}

	result := &ComparisonResult{
		Current:           current,
		Previous:          previous,
		PreviousTimestamp: previousTimestamp,
	}

	// Calculate metric deltas
	result.Deltas = c.calculateDeltas(current, previous)

	// Detect trends
	result.Trends = c.detectTrends(result.Deltas)

	// Categorize issues
	result.NewIssues, result.FixedIssues = c.categorizeIssues(current.Issues, previous.Issues)

	return result
}

// calculateDeltas computes the difference between current and previous metrics.
func (c *Comparator) calculateDeltas(current, previous *analyzer.AnalysisResult) *MetricDeltas {
	deltas := &MetricDeltas{}

	// File metrics
	deltas.TotalFiles = IntDelta{
		Previous: previous.TotalFiles,
		Current:  current.TotalFiles,
		Change:   current.TotalFiles - previous.TotalFiles,
	}
	deltas.TotalFiles.Percent = c.percentChange(deltas.TotalFiles.Previous, deltas.TotalFiles.Current)

	deltas.TotalLines = IntDelta{
		Previous: previous.TotalLines,
		Current:  current.TotalLines,
		Change:   current.TotalLines - previous.TotalLines,
	}
	deltas.TotalLines.Percent = c.percentChange(deltas.TotalLines.Previous, deltas.TotalLines.Current)

	deltas.TotalFunctions = IntDelta{
		Previous: previous.TotalFunctions,
		Current:  current.TotalFunctions,
		Change:   current.TotalFunctions - previous.TotalFunctions,
	}
	deltas.TotalFunctions.Percent = c.percentChange(deltas.TotalFunctions.Previous, deltas.TotalFunctions.Current)

	// Complexity metrics
	if current.Metrics != nil && previous.Metrics != nil {
		deltas.AvgComplexity = FloatDelta{
			Previous: previous.Metrics.AverageComplexity,
			Current:  current.Metrics.AverageComplexity,
			Change:   current.Metrics.AverageComplexity - previous.Metrics.AverageComplexity,
		}
		deltas.AvgComplexity.Percent = c.percentChange(
			int(deltas.AvgComplexity.Previous*100),
			int(deltas.AvgComplexity.Current*100),
		)
	}

	// Coverage metrics
	if current.Coverage != nil && previous.Coverage != nil {
		deltas.AvgCoverage = FloatDelta{
			Previous: previous.Coverage.AverageCoverage,
			Current:  current.Coverage.AverageCoverage,
			Change:   current.Coverage.AverageCoverage - previous.Coverage.AverageCoverage,
		}
		deltas.AvgCoverage.Percent = c.percentChange(
			int(deltas.AvgCoverage.Previous*100),
			int(deltas.AvgCoverage.Current*100),
		)
	}

	// Issue count
	deltas.IssueCount = IntDelta{
		Previous: len(previous.Issues),
		Current:  len(current.Issues),
		Change:   len(current.Issues) - len(previous.Issues),
	}
	deltas.IssueCount.Percent = c.percentChange(deltas.IssueCount.Previous, deltas.IssueCount.Current)

	return deltas
}

// detectTrends analyzes deltas to determine if metrics are improving, worsening, or stable.
func (c *Comparator) detectTrends(deltas *MetricDeltas) *Trends {
	trends := &Trends{}

	// Complexity: Lower is better
	trends.Complexity = c.detectTrend(deltas.AvgComplexity.Percent, false)

	// Coverage: Higher is better
	trends.Coverage = c.detectTrend(deltas.AvgCoverage.Percent, true)

	// Issues: Lower is better
	trends.IssueCount = c.detectTrend(deltas.IssueCount.Percent, false)

	return trends
}

// detectTrend determines the trend direction based on percent change.
//
// Parameters:
//   - percentChange: The percentage change (can be positive or negative)
//   - higherIsBetter: True if higher values are better (e.g., coverage), false otherwise (e.g., complexity)
//
// Returns:
//   - TrendImproving if the change is in the good direction and exceeds threshold
//   - TrendWorsening if the change is in the bad direction and exceeds threshold
//   - TrendStable if the absolute change is within the stable threshold
func (c *Comparator) detectTrend(percentChange float64, higherIsBetter bool) TrendDirection {
	absChange := percentChange
	if absChange < 0 {
		absChange = -absChange
	}

	// Within stable threshold?
	if absChange <= c.stableThreshold {
		return TrendStable
	}

	// Determine if improving or worsening
	if higherIsBetter {
		if percentChange > 0 {
			return TrendImproving
		}
		return TrendWorsening
	}

	// Lower is better
	if percentChange < 0 {
		return TrendImproving
	}
	return TrendWorsening
}

// categorizeIssues compares current and previous issues to find new and fixed issues.
//
// Issues are matched by: Type + File + Line + Function
// - New issues: In current but not in previous
// - Fixed issues: In previous but not in current
func (c *Comparator) categorizeIssues(current, previous []*analyzer.Issue) ([]*analyzer.Issue, []*analyzer.Issue) {
	// Create a map of previous issues for fast lookup
	prevMap := make(map[string]*analyzer.Issue)
	for _, issue := range previous {
		key := c.issueKey(issue)
		prevMap[key] = issue
	}

	// Find new issues (in current but not in previous)
	var newIssues []*analyzer.Issue
	currMap := make(map[string]bool)
	for _, issue := range current {
		key := c.issueKey(issue)
		currMap[key] = true
		if _, exists := prevMap[key]; !exists {
			newIssues = append(newIssues, issue)
		}
	}

	// Find fixed issues (in previous but not in current)
	var fixedIssues []*analyzer.Issue
	for _, issue := range previous {
		key := c.issueKey(issue)
		if !currMap[key] {
			fixedIssues = append(fixedIssues, issue)
		}
	}

	return newIssues, fixedIssues
}

// issueKey generates a unique key for an issue based on type, file, line, and function.
func (c *Comparator) issueKey(issue *analyzer.Issue) string {
	return issue.Type + "|" + issue.File + "|" + string(rune(issue.Line)) + "|" + issue.Function
}

// percentChange calculates the percentage change between two values.
func (c *Comparator) percentChange(previous, current int) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100.0 // Changed from 0 to something
	}
	return (float64(current-previous) / float64(previous)) * 100.0
}

// ComparisonResult contains the results of comparing two analysis runs.
type ComparisonResult struct {
	Current           *analyzer.AnalysisResult
	Previous          *analyzer.AnalysisResult
	PreviousTimestamp time.Time // Timestamp of the previous report
	Deltas            *MetricDeltas
	Trends            *Trends
	NewIssues         []*analyzer.Issue
	FixedIssues       []*analyzer.Issue
}

// MetricDeltas contains the changes in metrics between two runs.
type MetricDeltas struct {
	TotalFiles     IntDelta
	TotalLines     IntDelta
	TotalFunctions IntDelta
	AvgComplexity  FloatDelta
	AvgCoverage    FloatDelta
	IssueCount     IntDelta
}

// IntDelta represents a change in an integer metric.
type IntDelta struct {
	Previous int
	Current  int
	Change   int
	Percent  float64
}

// FloatDelta represents a change in a float metric.
type FloatDelta struct {
	Previous float64
	Current  float64
	Change   float64
	Percent  float64
}

// Trends indicates the direction of change for key metrics.
type Trends struct {
	Complexity TrendDirection
	Coverage   TrendDirection
	IssueCount TrendDirection
}

// TrendDirection indicates whether a metric is improving, worsening, or stable.
type TrendDirection int

const (
	TrendStable TrendDirection = iota
	TrendImproving
	TrendWorsening
)

// String returns a human-readable representation of the trend.
func (t TrendDirection) String() string {
	switch t {
	case TrendImproving:
		return "IMPROVING"
	case TrendWorsening:
		return "WORSENING"
	case TrendStable:
		return "STABLE"
	default:
		return "UNKNOWN"
	}
}

// Icon returns an emoji icon representing the trend.
func (t TrendDirection) Icon() string {
	switch t {
	case TrendImproving:
		return "✅"
	case TrendWorsening:
		return "📉"
	case TrendStable:
		return "➡️"
	default:
		return "❓"
	}
}
