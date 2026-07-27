package kotlin

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// parseDepsFixtures parses the dependency fixtures in testdata/kotlin/deps with
// the real Kotlin parser, so these tests also cover package/import extraction
// end to end.
func parseDepsFixtures(t *testing.T) []*parser.FileMetrics {
	t.Helper()

	p := NewParser(1)
	names := []string{"alpha.kt", "beta.kt", "gamma.kt", "root.kt"}

	var files []*parser.FileMetrics
	for _, name := range names {
		fm, err := p.ParseFile(filepath.Join("../../../testdata/kotlin/deps", name))
		require.NoError(t, err, name)
		files = append(files, fm)
	}
	return files
}

func findPackage(t *testing.T, results []*dependencies.PackageDependencies, name string) *dependencies.PackageDependencies {
	t.Helper()
	for _, r := range results {
		if r.PackageName == name {
			return r
		}
	}
	t.Fatalf("package %q not found in results", name)
	return nil
}

func TestDependencyAnalyzer_Analyze_Categorization(t *testing.T) {
	analyzer, err := NewDependencyAnalyzer(".")
	require.NoError(t, err)

	results, err := analyzer.Analyze(parseDepsFixtures(t))
	require.NoError(t, err)
	require.Len(t, results, 4)

	alpha := findPackage(t, results, "com.example.alpha")
	assert.Equal(t, []string{"java.util.UUID", "kotlin.math.abs"}, alpha.StdlibImports)
	assert.Equal(t, []string{"com.example.beta.Beta"}, alpha.InternalImports)
	assert.Equal(t, []string{"com.thirdparty.http.Client", "kotlinx.coroutines.flow.Flow"}, alpha.ExternalImports,
		"kotlinx is a separate artifact, categorized external")
	assert.Equal(t, 5, alpha.TotalImports)
	assert.Equal(t, 2, alpha.ExternalImportCount)

	beta := findPackage(t, results, "com.example.beta")
	assert.Equal(t, []string{"com.example.alpha.Alpha"}, beta.InternalImports)
	assert.Equal(t, 1, beta.TotalImports)
}

func TestDependencyAnalyzer_Analyze_WildcardImportIsInternal(t *testing.T) {
	analyzer, err := NewDependencyAnalyzer(".")
	require.NoError(t, err)

	results, err := analyzer.Analyze(parseDepsFixtures(t))
	require.NoError(t, err)

	gamma := findPackage(t, results, "com.example.gamma")
	// The Kotlin parser normalizes wildcard imports: `import com.example.alpha.*`
	// is stored as "com.example.alpha" (the .* is dropped at parse time).
	assert.Equal(t, []string{"com.example.alpha"}, gamma.InternalImports)
	assert.Empty(t, gamma.ExternalImports)
}

func TestDependencyAnalyzer_Analyze_FileWithoutPackageGroupsUnderRoot(t *testing.T) {
	analyzer, err := NewDependencyAnalyzer(".")
	require.NoError(t, err)

	results, err := analyzer.Analyze(parseDepsFixtures(t))
	require.NoError(t, err)

	root := findPackage(t, results, "(root)")
	assert.Equal(t, []string{"com.example.gamma.Gamma"}, root.InternalImports)
}

func TestDependencyAnalyzer_Analyze_EdgeCases(t *testing.T) {
	analyzer, err := NewDependencyAnalyzer(".")
	require.NoError(t, err)

	t.Run("nil input", func(t *testing.T) {
		results, err := analyzer.Analyze(nil)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("multiple files in one package dedupe imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{FilePath: "a.kt", PackageName: "com.example.p", Imports: []string{"java.util.UUID", "com.other.Thing"}},
			{FilePath: "b.kt", PackageName: "com.example.p", Imports: []string{"java.util.UUID"}},
		}

		results, err := analyzer.Analyze(files)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, 2, results[0].TotalImports, "duplicate import counted once")
		assert.Equal(t, []string{"java.util.UUID"}, results[0].StdlibImports)
		assert.Equal(t, []string{"com.other.Thing"}, results[0].ExternalImports)
	})

	t.Run("package with no imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{FilePath: "c.kt", PackageName: "com.example.empty"},
		}

		results, err := analyzer.Analyze(files)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, 0, results[0].TotalImports)
		assert.Equal(t, 0, results[0].ExternalImportCount)
	})

	t.Run("nested class and enum entry imports resolve to declared package", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{FilePath: "d.kt", PackageName: "com.example.alpha"},
			{FilePath: "e.kt", PackageName: "com.example.beta", Imports: []string{
				"com.example.alpha.Alpha.Companion",
				"com.example.alpha.Color.RED",
			}},
		}

		results, err := analyzer.Analyze(files)
		require.NoError(t, err)
		beta := findPackage(t, results, "com.example.beta")
		assert.Len(t, beta.InternalImports, 2)
		assert.Empty(t, beta.ExternalImports)
	})
}

func TestCategorizeImport(t *testing.T) {
	declared := map[string]bool{
		"com.example.alpha": true,
		"com.example.beta":  true,
	}

	cases := []struct {
		imp  string
		want string
	}{
		{"com.example.alpha.Alpha", "internal"},
		{"com.example.alpha.*", "internal"},
		{"com.example.alpha", "internal"}, // package-level facade import
		{"kotlin.math.abs", "stdlib"},
		{"java.util.UUID", "stdlib"},
		{"javax.inject.Inject", "stdlib"},
		{"kotlinx.coroutines.flow.Flow", "external"},
		{"com.thirdparty.http.Client", "external"},
		{"com.example.alpha.Alpha.Companion", "internal"}, // nested member resolves via longest declared prefix
		{"com.example.alphax.Thing", "external"},          // prefix similarity is not a match
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, categorizeImport(tc.imp, declared), tc.imp)
	}
}

func TestDependencyAnalyzer_DetectCircularDependencies(t *testing.T) {
	analyzer, err := NewDependencyAnalyzer(".")
	require.NoError(t, err)

	cycles, err := analyzer.DetectCircularDependencies(parseDepsFixtures(t))
	require.NoError(t, err)

	// alpha -> beta -> alpha is the only cycle; gamma and (root) only
	// point INTO the cycle and must not appear in it.
	require.Len(t, cycles, 1)
	require.Len(t, cycles[0].Cycle, 3)
	assert.ElementsMatch(t, []string{"com.example.alpha", "com.example.beta"}, cycles[0].Cycle[:2])
}

func TestDependencyAnalyzer_DetectCircularDependencies_SelfImportExcluded(t *testing.T) {
	analyzer, err := NewDependencyAnalyzer(".")
	require.NoError(t, err)

	// A package importing a sibling class from its own package must not
	// produce a self-loop cycle.
	files := []*parser.FileMetrics{
		{
			FilePath:    "solo.kt",
			PackageName: "com.example.solo",
			Imports:     []string{"com.example.solo.Helper"},
		},
	}

	cycles, err := analyzer.DetectCircularDependencies(files)
	require.NoError(t, err)
	assert.Empty(t, cycles)
}
