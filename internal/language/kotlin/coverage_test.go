package kotlin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
)

func findCoverage(t *testing.T, results []*coverage.PackageCoverage, pkg string) *coverage.PackageCoverage {
	t.Helper()
	for _, r := range results {
		if r.PackagePath == pkg {
			return r
		}
	}
	t.Fatalf("package %q not found in coverage results", pkg)
	return nil
}

func TestParseReports_SingleModule(t *testing.T) {
	results, err := parseReports([]string{"../../../testdata/kotlin/coverage/jacoco-single.xml"})
	require.NoError(t, err)
	require.Len(t, results, 3)

	alpha := findCoverage(t, results, "com.example.alpha")
	assert.InDelta(t, 75.0, alpha.Coverage, 0.001, "15 covered / 20 total lines; class-level counters ignored")
	assert.False(t, alpha.Skipped)

	beta := findCoverage(t, results, "com.example.beta")
	assert.InDelta(t, 100.0, beta.Coverage, 0.001)

	empty := findCoverage(t, results, "com.example.empty")
	assert.True(t, empty.Skipped, "package without line data is skipped")
}

func TestParseReports_MultiModuleMergesBySummingCounters(t *testing.T) {
	results, err := parseReports([]string{
		"../../../testdata/kotlin/coverage/jacoco-module-a.xml",
		"../../../testdata/kotlin/coverage/jacoco-module-b.xml",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	shared := findCoverage(t, results, "com.example.shared")
	assert.InDelta(t, 50.0, shared.Coverage, 0.001, "(15+5) covered / 40 total lines")
}

func TestParseReports_MissingFile(t *testing.T) {
	_, err := parseReports([]string{"../../../testdata/kotlin/coverage/does-not-exist.xml"})
	assert.Error(t, err)
}
