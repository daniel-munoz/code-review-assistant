package kotlin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
)

// runDetectorsOnFixture parses antipatterns.kt and runs all detectors on it.
func runDetectorsOnFixture(t *testing.T, cfg *config.AnalysisConfig) []*detectors.Issue {
	t.Helper()
	absPath := fixturePath(t, "antipatterns.kt")

	p := NewParser(1)
	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err)

	runner := NewDetectorRunner(cfg)
	return runner.RunDetectors(cfg, metrics)
}

// defaultTestConfig returns an AnalysisConfig with all detectors enabled.
func defaultTestConfig() *config.AnalysisConfig {
	return &config.AnalysisConfig{
		MaxParameters:           5,
		MaxNestingDepth:         4,
		MaxReturnStatements:     3,
		DetectMagicNumbers:      true,
		DetectNonNullAssertions: true,
		DetectRunBlocking:       true,
	}
}

// issuesByType groups issues by their Type field.
func issuesByType(issues []*detectors.Issue) map[string][]*detectors.Issue {
	byType := make(map[string][]*detectors.Issue)
	for _, issue := range issues {
		byType[issue.Type] = append(byType[issue.Type], issue)
	}
	return byType
}

func TestRunDetectors_GenericDetectors(t *testing.T) {
	byType := issuesByType(runDetectorsOnFixture(t, defaultTestConfig()))

	// tooManyParams has 7 params (threshold 5)
	require.NotEmpty(t, byType["too_many_parameters"])
	assert.Equal(t, "tooManyParams", byType["too_many_parameters"][0].Function)
	assert.Equal(t, 7, byType["too_many_parameters"][0].Value)

	// deepNesting nests 5 levels (threshold 4)
	require.NotEmpty(t, byType["deep_nesting"])
	found := false
	for _, issue := range byType["deep_nesting"] {
		if issue.Function == "deepNesting" {
			found = true
		}
	}
	assert.True(t, found, "deepNesting should be flagged")

	// manyReturns has 5 returns (threshold 3)
	require.NotEmpty(t, byType["too_many_returns"])
	assert.Equal(t, "manyReturns", byType["too_many_returns"][0].Function)
}

func TestRunDetectors_MagicNumbers(t *testing.T) {
	byType := issuesByType(runDetectorsOnFixture(t, defaultTestConfig()))

	messages := make([]string, 0)
	for _, issue := range byType["magic_number"] {
		messages = append(messages, issue.Message)
	}

	assert.NotEmpty(t, byType["magic_number"])
	found4500, found86400, found250 := false, false, false
	for _, m := range messages {
		if m == "Magic number should be replaced with a named constant: 4500" {
			found4500 = true
		}
		if m == "Magic number should be replaced with a named constant: 86400" {
			found86400 = true
		}
		if m == "Magic number should be replaced with a named constant: 250" {
			found250 = true
		}
	}
	assert.True(t, found4500, "4500 should be flagged")
	assert.True(t, found86400, "86400 should be flagged")
	assert.False(t, found250, "250 is assigned to UPPER_CASE val MAX_BATCH and should be exempt")
}

func TestRunDetectors_NonNullAssertion(t *testing.T) {
	byType := issuesByType(runDetectorsOnFixture(t, defaultTestConfig()))

	require.NotEmpty(t, byType["non_null_assertion"], "!! in unsafeAccess should be flagged")
	assert.Equal(t, "unsafeAccess", byType["non_null_assertion"][0].Function)
	assert.Equal(t, "warning", byType["non_null_assertion"][0].Severity)
}

func TestRunDetectors_RunBlocking(t *testing.T) {
	byType := issuesByType(runDetectorsOnFixture(t, defaultTestConfig()))

	require.NotEmpty(t, byType["run_blocking"], "runBlocking in blockingCall should be flagged")
	for _, issue := range byType["run_blocking"] {
		assert.NotEqual(t, "main", issue.Function, "runBlocking inside fun main is the legitimate idiom")
	}
}

func TestRunDetectors_KotlinDetectorsDisabled(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.DetectNonNullAssertions = false
	cfg.DetectRunBlocking = false

	byType := issuesByType(runDetectorsOnFixture(t, cfg))
	assert.Empty(t, byType["non_null_assertion"])
	assert.Empty(t, byType["run_blocking"])
}
