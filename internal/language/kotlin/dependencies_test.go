package kotlin

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// parseDepsFixtures parses every fixture in testdata/kotlin/deps with the real
// Kotlin parser, so these tests also cover package/import extraction end to end.
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

	root := findPackage(t, results, "<root>")
	assert.Equal(t, []string{"com.example.gamma.Gamma"}, root.InternalImports)
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
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, categorizeImport(tc.imp, declared), tc.imp)
	}
}
