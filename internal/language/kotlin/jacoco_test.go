package kotlin

import (
	"os"
	"path/filepath"
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
	assert.InDelta(t, 66.667, shared.Coverage, 0.01, "summed (15+5)/(20+10)=66.7%, NOT averaged (75+50)/2=62.5%")
}

func TestParseReports_MissingFile(t *testing.T) {
	_, err := parseReports([]string{"../../../testdata/kotlin/coverage/does-not-exist.xml"})
	assert.Error(t, err)
}

func TestParseReports_GroupedAggregateReport(t *testing.T) {
	results, err := parseReports([]string{"../../../testdata/kotlin/coverage/jacoco-grouped.xml"})
	require.NoError(t, err)
	require.Len(t, results, 2)

	core := findCoverage(t, results, "com.example.core")
	assert.InDelta(t, 80.0, core.Coverage, 0.001)

	root := findCoverage(t, results, "(root)")
	assert.InDelta(t, 100.0, root.Coverage, 0.001, "JaCoCo default package maps to (root)")
}

func TestParseReports_MalformedXML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.xml")
	require.NoError(t, os.WriteFile(path, []byte("not xml at all"), 0o644))

	_, err := parseReports([]string{path})
	assert.Error(t, err)
}

func TestParseReports_WrongRootElementRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "other.xml")
	require.NoError(t, os.WriteFile(path, []byte(`<coverage><package name="x"><counter type="LINE" missed="1" covered="1"/></package></coverage>`), 0o644))

	_, err := parseReports([]string{path})
	assert.Error(t, err, "non-JaCoCo XML swept up by future globbing must fail loudly")
}

func TestParseReports_NoPackagesAnywhereErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.xml")
	require.NoError(t, os.WriteFile(path, []byte(`<?xml version="1.0"?><report name="empty"></report>`), 0o644))

	_, err := parseReports([]string{path})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no coverage packages")
}

func TestParseReports_EmptyReportMergedWithRealOneSucceeds(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.xml")
	require.NoError(t, os.WriteFile(empty, []byte(`<?xml version="1.0"?><report name="empty"></report>`), 0o644))

	results, err := parseReports([]string{
		"../../../testdata/kotlin/coverage/jacoco-module-a.xml",
		empty,
	})
	require.NoError(t, err, "an empty module report alongside a real one is fine")
	require.Len(t, results, 1)
}
