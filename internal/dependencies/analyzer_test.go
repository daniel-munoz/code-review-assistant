package dependencies

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyzer(t *testing.T) {
	t.Run("creates analyzer for valid project", func(t *testing.T) {
		// Use the current project directory (two levels up)
		analyzer, err := NewAnalyzer("../..")

		require.NoError(t, err, "should create analyzer without error")
		require.NotNil(t, analyzer, "analyzer should not be nil")
		assert.NotEmpty(t, analyzer.moduleName, "module name should be set")
		assert.Contains(t, analyzer.moduleName, "code-review-assistant", "module name should contain project name")
	})

	// Note: NewAnalyzer will create an analyzer even if go.mod doesn't exist
	// It only fails if it can't read the go.mod file when one exists
}

func TestAnalyze(t *testing.T) {
	// Create analyzer using current project
	analyzer, err := NewAnalyzer("../..")
	require.NoError(t, err, "should create analyzer")

	t.Run("analyzes empty file list", func(t *testing.T) {
		files := []*parser.FileMetrics{}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err, "should analyze empty list without error")
		assert.Empty(t, results, "results should be empty")
	})

	t.Run("analyzes single package with stdlib imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				PackageName: "main",
				Imports:     []string{"fmt", "os", "strings"},
			},
		}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err)
		require.Len(t, results, 1)

		pkg := results[0]
		assert.Equal(t, "main", pkg.PackageName)
		assert.Len(t, pkg.StdlibImports, 3)
		assert.Contains(t, pkg.StdlibImports, "fmt")
		assert.Contains(t, pkg.StdlibImports, "os")
		assert.Contains(t, pkg.StdlibImports, "strings")
		assert.Empty(t, pkg.ExternalImports)
		assert.Empty(t, pkg.InternalImports)
		assert.Equal(t, 3, pkg.TotalImports)
		assert.Equal(t, 0, pkg.ExternalImportCount)
	})

	t.Run("analyzes package with external imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				PackageName: "test",
				Imports: []string{
					"fmt",
					"github.com/spf13/cobra",
					"github.com/stretchr/testify/assert",
				},
			},
		}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err)
		require.Len(t, results, 1)

		pkg := results[0]
		assert.Equal(t, 1, len(pkg.StdlibImports), "should have 1 stdlib import")
		assert.Equal(t, 2, len(pkg.ExternalImports), "should have 2 external imports")
		assert.Contains(t, pkg.ExternalImports, "github.com/spf13/cobra")
		assert.Contains(t, pkg.ExternalImports, "github.com/stretchr/testify/assert")
		assert.Equal(t, 3, pkg.TotalImports)
		assert.Equal(t, 2, pkg.ExternalImportCount)
	})

	t.Run("analyzes package with internal imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				PackageName: "cmd",
				Imports: []string{
					"fmt",
					"github.com/daniel-munoz/code-review-assistant/internal/config",
					"github.com/daniel-munoz/code-review-assistant/internal/parser",
				},
			},
		}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err)
		require.Len(t, results, 1)

		pkg := results[0]
		assert.Equal(t, 1, len(pkg.StdlibImports), "should have 1 stdlib import")
		assert.Equal(t, 2, len(pkg.InternalImports), "should have 2 internal imports")
		assert.Contains(t, pkg.InternalImports, "github.com/daniel-munoz/code-review-assistant/internal/config")
		assert.Contains(t, pkg.InternalImports, "github.com/daniel-munoz/code-review-assistant/internal/parser")
		assert.Equal(t, 3, pkg.TotalImports)
		assert.Equal(t, 0, pkg.ExternalImportCount)
	})

	t.Run("analyzes package with mixed imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				PackageName: "analyzer",
				Imports: []string{
					"fmt",                 // stdlib
					"strings",             // stdlib
					"github.com/daniel-munoz/code-review-assistant/internal/parser", // internal
					"github.com/spf13/viper",                                        // external
				},
			},
		}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err)
		require.Len(t, results, 1)

		pkg := results[0]
		assert.Equal(t, 2, len(pkg.StdlibImports))
		assert.Equal(t, 1, len(pkg.InternalImports))
		assert.Equal(t, 1, len(pkg.ExternalImports))
		assert.Equal(t, 4, pkg.TotalImports)
		assert.Equal(t, 1, pkg.ExternalImportCount)
	})

	t.Run("deduplicates imports across multiple files in same package", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				PackageName: "test",
				Imports:     []string{"fmt", "os"},
			},
			{
				PackageName: "test",
				Imports:     []string{"fmt", "strings"}, // fmt is duplicate
			},
		}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err)
		require.Len(t, results, 1)

		pkg := results[0]
		// Should have 3 unique imports: fmt, os, strings
		assert.Equal(t, 3, pkg.TotalImports)
		assert.Len(t, pkg.StdlibImports, 3)
	})

	t.Run("groups files by package", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{PackageName: "pkg1", Imports: []string{"fmt"}},
			{PackageName: "pkg2", Imports: []string{"os"}},
			{PackageName: "pkg1", Imports: []string{"strings"}},
		}

		results, err := analyzer.Analyze(files)

		require.NoError(t, err)
		require.Len(t, results, 2, "should have 2 packages")

		// Find each package in results
		var pkg1, pkg2 *PackageDependencies
		for _, result := range results {
			if result.PackageName == "pkg1" {
				pkg1 = result
			} else if result.PackageName == "pkg2" {
				pkg2 = result
			}
		}

		require.NotNil(t, pkg1, "should find pkg1")
		require.NotNil(t, pkg2, "should find pkg2")

		// pkg1 has 2 imports (fmt and strings)
		assert.Equal(t, 2, pkg1.TotalImports)
		// pkg2 has 1 import (os)
		assert.Equal(t, 1, pkg2.TotalImports)
	})
}

func TestCategorizeImport(t *testing.T) {
	analyzer, err := NewAnalyzer("../..")
	require.NoError(t, err)

	testCases := []struct {
		name     string
		imp      string
		expected string
	}{
		{
			name:     "categorizes stdlib fmt",
			imp:      "fmt",
			expected: "stdlib",
		},
		{
			name:     "categorizes stdlib os",
			imp:      "os",
			expected: "stdlib",
		},
		{
			name:     "categorizes stdlib strings",
			imp:      "strings",
			expected: "stdlib",
		},
		{
			name:     "categorizes stdlib path/filepath",
			imp:      "path/filepath",
			expected: "stdlib",
		},
		{
			name:     "categorizes internal package",
			imp:      "github.com/daniel-munoz/code-review-assistant/internal/parser",
			expected: "internal",
		},
		{
			name:     "categorizes internal config",
			imp:      "github.com/daniel-munoz/code-review-assistant/internal/config",
			expected: "internal",
		},
		{
			name:     "categorizes external cobra",
			imp:      "github.com/spf13/cobra",
			expected: "external",
		},
		{
			name:     "categorizes external viper",
			imp:      "github.com/spf13/viper",
			expected: "external",
		},
		{
			name:     "categorizes external testify",
			imp:      "github.com/stretchr/testify/assert",
			expected: "external",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			category := analyzer.categorizeImport(tc.imp)
			assert.Equal(t, tc.expected, category, "category should match expected")
		})
	}
}

func TestIsStdlib(t *testing.T) {
	testCases := []struct {
		name     string
		imp      string
		expected bool
	}{
		{
			name:     "fmt is stdlib",
			imp:      "fmt",
			expected: true,
		},
		{
			name:     "os is stdlib",
			imp:      "os",
			expected: true,
		},
		{
			name:     "strings is stdlib",
			imp:      "strings",
			expected: true,
		},
		{
			name:     "path/filepath is stdlib",
			imp:      "path/filepath",
			expected: true,
		},
		{
			name:     "io is stdlib",
			imp:      "io",
			expected: true,
		},
		{
			name:     "testing is stdlib",
			imp:      "testing",
			expected: true,
		},
		{
			name:     "github.com/spf13/cobra is not stdlib",
			imp:      "github.com/spf13/cobra",
			expected: false,
		},
		{
			name:     "github.com/stretchr/testify is not stdlib",
			imp:      "github.com/stretchr/testify/assert",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isStdlib(tc.imp)
			assert.Equal(t, tc.expected, result, "stdlib check should match expected")
		})
	}
}

func TestPackageDependencies_Struct(t *testing.T) {
	t.Run("creates package dependencies with all fields", func(t *testing.T) {
		pkg := &PackageDependencies{
			PackageName:         "test",
			StdlibImports:       []string{"fmt", "os"},
			InternalImports:     []string{"internal/config"},
			ExternalImports:     []string{"github.com/spf13/cobra"},
			TotalImports:        4,
			ExternalImportCount: 1,
		}

		assert.Equal(t, "test", pkg.PackageName)
		assert.Len(t, pkg.StdlibImports, 2)
		assert.Len(t, pkg.InternalImports, 1)
		assert.Len(t, pkg.ExternalImports, 1)
		assert.Equal(t, 4, pkg.TotalImports)
		assert.Equal(t, 1, pkg.ExternalImportCount)
	})

	t.Run("represents package with no imports", func(t *testing.T) {
		pkg := &PackageDependencies{
			PackageName:         "empty",
			StdlibImports:       []string{},
			InternalImports:     []string{},
			ExternalImports:     []string{},
			TotalImports:        0,
			ExternalImportCount: 0,
		}

		assert.Equal(t, 0, pkg.TotalImports)
		assert.Equal(t, 0, pkg.ExternalImportCount)
		assert.Empty(t, pkg.StdlibImports)
		assert.Empty(t, pkg.InternalImports)
		assert.Empty(t, pkg.ExternalImports)
	})
}
