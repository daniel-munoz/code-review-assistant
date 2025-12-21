package comparison

import (
	"testing"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComparator(t *testing.T) {
	tests := []struct {
		name            string
		stableThreshold float64
		expected        float64
	}{
		{
			name:            "uses provided threshold",
			stableThreshold: 10.0,
			expected:        10.0,
		},
		{
			name:            "uses default when threshold is zero",
			stableThreshold: 0,
			expected:        5.0,
		},
		{
			name:            "uses default when threshold is negative",
			stableThreshold: -5.0,
			expected:        5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := NewComparator(tt.stableThreshold)
			assert.Equal(t, tt.expected, comp.stableThreshold)
		})
	}
}

func TestComparator_Compare_NilPrevious(t *testing.T) {
	comp := NewComparator(5.0)
	current := createTestResult(10, 1000, 50)

	result := comp.Compare(current, nil, time.Now())

	assert.Nil(t, result)
}

func TestComparator_Compare_CalculatesDeltas(t *testing.T) {
	comp := NewComparator(5.0)

	previous := createTestResult(10, 1000, 50)
	previous.Metrics = &analyzer.AggregateMetrics{
		AverageComplexity: 5.0,
	}
	previous.Coverage = &analyzer.CoverageReport{
		AverageCoverage: 70.0,
	}
	previous.Issues = []*analyzer.Issue{
		{Type: "test", Severity: "warning"},
	}

	current := createTestResult(12, 1200, 60)
	current.Metrics = &analyzer.AggregateMetrics{
		AverageComplexity: 4.5,
	}
	current.Coverage = &analyzer.CoverageReport{
		AverageCoverage: 75.0,
	}
	current.Issues = []*analyzer.Issue{
		{Type: "test", Severity: "warning"},
		{Type: "test2", Severity: "error"},
	}

	timestamp := time.Now().Add(-24 * time.Hour)
	result := comp.Compare(current, previous, timestamp)

	require.NotNil(t, result)
	assert.Equal(t, current, result.Current)
	assert.Equal(t, previous, result.Previous)
	assert.Equal(t, timestamp, result.PreviousTimestamp)

	// Check deltas
	assert.Equal(t, 10, result.Deltas.TotalFiles.Previous)
	assert.Equal(t, 12, result.Deltas.TotalFiles.Current)
	assert.Equal(t, 2, result.Deltas.TotalFiles.Change)
	assert.Equal(t, 20.0, result.Deltas.TotalFiles.Percent)

	assert.Equal(t, 1000, result.Deltas.TotalLines.Previous)
	assert.Equal(t, 1200, result.Deltas.TotalLines.Current)
	assert.Equal(t, 200, result.Deltas.TotalLines.Change)
	assert.Equal(t, 20.0, result.Deltas.TotalLines.Percent)

	assert.Equal(t, 50, result.Deltas.TotalFunctions.Previous)
	assert.Equal(t, 60, result.Deltas.TotalFunctions.Current)
	assert.Equal(t, 10, result.Deltas.TotalFunctions.Change)
	assert.Equal(t, 20.0, result.Deltas.TotalFunctions.Percent)

	assert.Equal(t, 5.0, result.Deltas.AvgComplexity.Previous)
	assert.Equal(t, 4.5, result.Deltas.AvgComplexity.Current)
	assert.InDelta(t, -0.5, result.Deltas.AvgComplexity.Change, 0.01)
	assert.InDelta(t, -10.0, result.Deltas.AvgComplexity.Percent, 0.1)

	assert.Equal(t, 70.0, result.Deltas.AvgCoverage.Previous)
	assert.Equal(t, 75.0, result.Deltas.AvgCoverage.Current)
	assert.InDelta(t, 5.0, result.Deltas.AvgCoverage.Change, 0.01)
	assert.InDelta(t, 7.14, result.Deltas.AvgCoverage.Percent, 0.1)

	assert.Equal(t, 1, result.Deltas.IssueCount.Previous)
	assert.Equal(t, 2, result.Deltas.IssueCount.Current)
	assert.Equal(t, 1, result.Deltas.IssueCount.Change)
	assert.Equal(t, 100.0, result.Deltas.IssueCount.Percent)
}

func TestComparator_DetectTrends(t *testing.T) {
	tests := []struct {
		name               string
		stableThreshold    float64
		complexityChange   float64
		coverageChange     float64
		issueCountChange   float64
		expectedComplexity TrendDirection
		expectedCoverage   TrendDirection
		expectedIssues     TrendDirection
	}{
		{
			name:               "all improving",
			stableThreshold:    5.0,
			complexityChange:   -10.0, // Lower complexity is better
			coverageChange:     10.0,  // Higher coverage is better
			issueCountChange:   -15.0, // Fewer issues is better
			expectedComplexity: TrendImproving,
			expectedCoverage:   TrendImproving,
			expectedIssues:     TrendImproving,
		},
		{
			name:               "all worsening",
			stableThreshold:    5.0,
			complexityChange:   10.0,  // Higher complexity is worse
			coverageChange:     -10.0, // Lower coverage is worse
			issueCountChange:   15.0,  // More issues is worse
			expectedComplexity: TrendWorsening,
			expectedCoverage:   TrendWorsening,
			expectedIssues:     TrendWorsening,
		},
		{
			name:               "all stable",
			stableThreshold:    5.0,
			complexityChange:   2.0, // Within threshold
			coverageChange:     -3.0,
			issueCountChange:   4.0,
			expectedComplexity: TrendStable,
			expectedCoverage:   TrendStable,
			expectedIssues:     TrendStable,
		},
		{
			name:               "mixed trends",
			stableThreshold:    5.0,
			complexityChange:   -8.0, // Improving
			coverageChange:     3.0,  // Stable
			issueCountChange:   12.0, // Worsening
			expectedComplexity: TrendImproving,
			expectedCoverage:   TrendStable,
			expectedIssues:     TrendWorsening,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := NewComparator(tt.stableThreshold)

			deltas := &MetricDeltas{
				AvgComplexity: FloatDelta{Percent: tt.complexityChange},
				AvgCoverage:   FloatDelta{Percent: tt.coverageChange},
				IssueCount:    IntDelta{Percent: tt.issueCountChange},
			}

			trends := comp.detectTrends(deltas)

			assert.Equal(t, tt.expectedComplexity, trends.Complexity)
			assert.Equal(t, tt.expectedCoverage, trends.Coverage)
			assert.Equal(t, tt.expectedIssues, trends.IssueCount)
		})
	}
}

func TestComparator_CategorizeIssues(t *testing.T) {
	comp := NewComparator(5.0)

	previous := []*analyzer.Issue{
		{
			Type:     "large_file",
			File:     "file1.go",
			Line:     1,
			Function: "main",
			Severity: "warning",
		},
		{
			Type:     "high_complexity",
			File:     "file2.go",
			Line:     10,
			Function: "complexFunc",
			Severity: "error",
		},
		{
			Type:     "low_coverage",
			File:     "file3.go",
			Line:     5,
			Function: "uncovered",
			Severity: "warning",
		},
	}

	current := []*analyzer.Issue{
		{
			Type:     "high_complexity",
			File:     "file2.go",
			Line:     10,
			Function: "complexFunc",
			Severity: "error",
		},
		{
			Type:     "long_function",
			File:     "file4.go",
			Line:     20,
			Function: "longFunc",
			Severity: "warning",
		},
		{
			Type:     "magic_number",
			File:     "file5.go",
			Line:     15,
			Function: "calculate",
			Severity: "info",
		},
	}

	newIssues, fixedIssues := comp.categorizeIssues(current, previous)

	// Verify new issues
	require.Len(t, newIssues, 2)
	assert.Equal(t, "long_function", newIssues[0].Type)
	assert.Equal(t, "magic_number", newIssues[1].Type)

	// Verify fixed issues
	require.Len(t, fixedIssues, 2)
	assert.Equal(t, "large_file", fixedIssues[0].Type)
	assert.Equal(t, "low_coverage", fixedIssues[1].Type)
}

func TestComparator_PercentChange(t *testing.T) {
	comp := NewComparator(5.0)

	tests := []struct {
		name     string
		previous int
		current  int
		expected float64
	}{
		{
			name:     "increase from 100 to 150",
			previous: 100,
			current:  150,
			expected: 50.0,
		},
		{
			name:     "decrease from 100 to 50",
			previous: 100,
			current:  50,
			expected: -50.0,
		},
		{
			name:     "no change",
			previous: 100,
			current:  100,
			expected: 0.0,
		},
		{
			name:     "from zero to something",
			previous: 0,
			current:  50,
			expected: 100.0,
		},
		{
			name:     "both zero",
			previous: 0,
			current:  0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comp.percentChange(tt.previous, tt.current)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestTrendDirection_String(t *testing.T) {
	tests := []struct {
		trend    TrendDirection
		expected string
	}{
		{TrendImproving, "IMPROVING"},
		{TrendWorsening, "WORSENING"},
		{TrendStable, "STABLE"},
		{TrendDirection(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.trend.String())
		})
	}
}

func TestTrendDirection_Icon(t *testing.T) {
	tests := []struct {
		trend    TrendDirection
		expected string
	}{
		{TrendImproving, "✅"},
		{TrendWorsening, "📉"},
		{TrendStable, "➡️"},
		{TrendDirection(999), "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.trend.Icon())
		})
	}
}

func TestComparator_IssueKey(t *testing.T) {
	comp := NewComparator(5.0)

	issue := &analyzer.Issue{
		Type:     "large_file",
		File:     "test.go",
		Line:     42,
		Function: "TestFunc",
	}

	// The key format doesn't matter as long as it's consistent
	key := comp.issueKey(issue)
	assert.NotEmpty(t, key)

	// Same issue should produce same key
	key2 := comp.issueKey(issue)
	assert.Equal(t, key, key2)

	// Different issue should produce different key
	differentIssue := &analyzer.Issue{
		Type:     "high_complexity",
		File:     "test.go",
		Line:     42,
		Function: "TestFunc",
	}
	differentKey := comp.issueKey(differentIssue)
	assert.NotEqual(t, key, differentKey)
}

func TestComparator_EdgeCases(t *testing.T) {
	comp := NewComparator(5.0)

	t.Run("handles nil metrics gracefully", func(t *testing.T) {
		previous := createTestResult(10, 1000, 50)
		previous.Metrics = nil
		previous.Coverage = nil

		current := createTestResult(12, 1200, 60)
		current.Metrics = nil
		current.Coverage = nil

		result := comp.Compare(current, previous, time.Now())
		require.NotNil(t, result)

		// Should have zero values for metrics deltas
		assert.Equal(t, 0.0, result.Deltas.AvgComplexity.Previous)
		assert.Equal(t, 0.0, result.Deltas.AvgComplexity.Current)
		assert.Equal(t, 0.0, result.Deltas.AvgCoverage.Previous)
		assert.Equal(t, 0.0, result.Deltas.AvgCoverage.Current)
	})

	t.Run("handles empty issue lists", func(t *testing.T) {
		previous := createTestResult(10, 1000, 50)
		previous.Issues = []*analyzer.Issue{}

		current := createTestResult(10, 1000, 50)
		current.Issues = []*analyzer.Issue{}

		result := comp.Compare(current, previous, time.Now())
		require.NotNil(t, result)

		assert.Empty(t, result.NewIssues)
		assert.Empty(t, result.FixedIssues)
	})

	t.Run("handles identical results", func(t *testing.T) {
		previous := createTestResult(10, 1000, 50)
		previous.Metrics = &analyzer.AggregateMetrics{AverageComplexity: 5.0}

		current := createTestResult(10, 1000, 50)
		current.Metrics = &analyzer.AggregateMetrics{AverageComplexity: 5.0}

		result := comp.Compare(current, previous, time.Now())
		require.NotNil(t, result)

		assert.Equal(t, 0, result.Deltas.TotalFiles.Change)
		assert.Equal(t, 0, result.Deltas.TotalLines.Change)
		assert.Equal(t, 0.0, result.Deltas.AvgComplexity.Change)
		assert.Equal(t, TrendStable, result.Trends.Complexity)
	})
}

// Helper function to create test results
func createTestResult(files, lines, functions int) *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		ProjectPath:    "/test/project",
		TotalFiles:     files,
		TotalLines:     lines,
		TotalFunctions: functions,
		Issues:         []*analyzer.Issue{},
	}
}
