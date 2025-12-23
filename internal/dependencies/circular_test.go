package dependencies

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCircularDependencies(t *testing.T) {
	analyzer, err := NewAnalyzer("../..")
	require.NoError(t, err)

	t.Run("detects no cycles in acyclic graph", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../main.go",
				PackageName: "main",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/config",
				},
			},
			{
				FilePath:    "../../internal/config/config.go",
				PackageName: "config",
				Imports:     []string{"fmt"}, // stdlib only
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		assert.Empty(t, cycles, "should find no circular dependencies")
	})

	t.Run("detects simple 2-package cycle", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../internal/pkg1/file.go",
				PackageName: "pkg1",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg2",
				},
			},
			{
				FilePath:    "../../internal/pkg2/file.go",
				PackageName: "pkg2",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg1",
				},
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		require.Len(t, cycles, 1, "should detect one cycle")
		// Now checks for full import paths
		assert.Contains(t, cycles[0].Cycle, "github.com/daniel-munoz/code-review-assistant/internal/pkg1")
		assert.Contains(t, cycles[0].Cycle, "github.com/daniel-munoz/code-review-assistant/internal/pkg2")
	})

	t.Run("detects 3-package cycle", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../internal/pkg1/file.go",
				PackageName: "pkg1",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg2",
				},
			},
			{
				FilePath:    "../../internal/pkg2/file.go",
				PackageName: "pkg2",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg3",
				},
			},
			{
				FilePath:    "../../internal/pkg3/file.go",
				PackageName: "pkg3",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg1",
				},
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		require.Len(t, cycles, 1, "should detect one cycle")
		assert.Len(t, cycles[0].Cycle, 4, "cycle should have 4 elements (including closing)")
		assert.Contains(t, cycles[0].Cycle, "github.com/daniel-munoz/code-review-assistant/internal/pkg1")
		assert.Contains(t, cycles[0].Cycle, "github.com/daniel-munoz/code-review-assistant/internal/pkg2")
		assert.Contains(t, cycles[0].Cycle, "github.com/daniel-munoz/code-review-assistant/internal/pkg3")
	})

	t.Run("handles self-cycle", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../internal/pkg1/file.go",
				PackageName: "pkg1",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg1",
				},
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		// A self-cycle should be detected
		require.Len(t, cycles, 1, "should detect self-cycle")
	})

	t.Run("ignores stdlib and external dependencies", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../internal/pkg1/file.go",
				PackageName: "pkg1",
				Imports: []string{
					"fmt",                       // stdlib - ignored
					"github.com/spf13/cobra",    // external - ignored
					"github.com/daniel-munoz/code-review-assistant/internal/pkg2", // internal
				},
			},
			{
				FilePath:    "../../internal/pkg2/file.go",
				PackageName: "pkg2",
				Imports: []string{
					"os",                        // stdlib - ignored
					"github.com/spf13/viper",    // external - ignored
				},
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		assert.Empty(t, cycles, "should not detect cycles from stdlib/external imports")
	})

	t.Run("handles empty file list", func(t *testing.T) {
		files := []*parser.FileMetrics{}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		assert.Empty(t, cycles, "should handle empty file list")
	})

	t.Run("handles package with no imports", func(t *testing.T) {
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../standalone/file.go",
				PackageName: "standalone",
				Imports:     []string{},
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		assert.Empty(t, cycles, "should handle package with no imports")
	})

	t.Run("deduplicates identical cycles", func(t *testing.T) {
		// Two files in pkg1, both importing pkg2
		files := []*parser.FileMetrics{
			{
				FilePath:    "../../internal/pkg1/file1.go",
				PackageName: "pkg1",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg2",
				},
			},
			{
				FilePath:    "../../internal/pkg1/file2.go",
				PackageName: "pkg1", // Same package, different file
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg2",
				},
			},
			{
				FilePath:    "../../internal/pkg2/file.go",
				PackageName: "pkg2",
				Imports: []string{
					"github.com/daniel-munoz/code-review-assistant/internal/pkg1",
				},
			},
		}

		cycles, err := analyzer.DetectCircularDependencies(files)

		require.NoError(t, err)
		// Should detect the cycle only once, not twice
		assert.LessOrEqual(t, len(cycles), 1, "should not report duplicate cycles")
	})
}

func TestNormalizeCycle(t *testing.T) {
	testCases := []struct {
		name     string
		cycle    []string
		expected []string
	}{
		{
			name:     "normalizes cycle starting with smallest",
			cycle:    []string{"pkg2", "pkg3", "pkg1"},
			expected: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:     "handles already normalized cycle",
			cycle:    []string{"pkg1", "pkg2", "pkg3"},
			expected: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:     "normalizes single element",
			cycle:    []string{"pkg1"},
			expected: []string{"pkg1"},
		},
		{
			name:     "normalizes empty cycle",
			cycle:    []string{},
			expected: []string{},
		},
		{
			name:     "normalizes 2-element cycle",
			cycle:    []string{"pkg2", "pkg1"},
			expected: []string{"pkg1", "pkg2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeCycle(tc.cycle)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAreCyclesEquivalent(t *testing.T) {
	testCases := []struct {
		name     string
		cycle1   []string
		cycle2   []string
		expected bool
	}{
		{
			name:     "identical cycles are equivalent",
			cycle1:   []string{"pkg1", "pkg2", "pkg3"},
			cycle2:   []string{"pkg1", "pkg2", "pkg3"},
			expected: true,
		},
		{
			name:     "rotated cycles are equivalent",
			cycle1:   []string{"pkg1", "pkg2", "pkg3"},
			cycle2:   []string{"pkg2", "pkg3", "pkg1"},
			expected: true,
		},
		{
			name:     "another rotation is equivalent",
			cycle1:   []string{"pkg1", "pkg2", "pkg3"},
			cycle2:   []string{"pkg3", "pkg1", "pkg2"},
			expected: true,
		},
		{
			name:     "different cycles are not equivalent",
			cycle1:   []string{"pkg1", "pkg2", "pkg3"},
			cycle2:   []string{"pkg1", "pkg3", "pkg2"},
			expected: false,
		},
		{
			name:     "different length cycles are not equivalent",
			cycle1:   []string{"pkg1", "pkg2"},
			cycle2:   []string{"pkg1", "pkg2", "pkg3"},
			expected: false,
		},
		{
			name:     "empty cycles are equivalent",
			cycle1:   []string{},
			cycle2:   []string{},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := areCyclesEquivalent(tc.cycle1, tc.cycle2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsDuplicateCycle(t *testing.T) {
	existingCycles := []*CircularDependency{
		{Cycle: []string{"pkg1", "pkg2", "pkg3"}},
		{Cycle: []string{"pkg4", "pkg5"}},
	}

	t.Run("detects duplicate cycle", func(t *testing.T) {
		newCycle := []string{"pkg2", "pkg3", "pkg1"} // Rotation of existing cycle
		result := isDuplicateCycle(existingCycles, newCycle)
		assert.True(t, result, "should detect duplicate (rotated) cycle")
	})

	t.Run("detects non-duplicate cycle", func(t *testing.T) {
		newCycle := []string{"pkg6", "pkg7"}
		result := isDuplicateCycle(existingCycles, newCycle)
		assert.False(t, result, "should not consider unique cycle as duplicate")
	})

	t.Run("handles empty existing cycles", func(t *testing.T) {
		emptyList := []*CircularDependency{}
		newCycle := []string{"pkg1", "pkg2"}
		result := isDuplicateCycle(emptyList, newCycle)
		assert.False(t, result, "should not find duplicates in empty list")
	})
}

func TestCircularDependency_FormatCycle(t *testing.T) {
	t.Run("formats simple cycle", func(t *testing.T) {
		cd := &CircularDependency{
			Cycle: []string{"pkg1", "pkg2", "pkg1"},
		}
		formatted := cd.FormatCycle()
		assert.Equal(t, "pkg1 -> pkg2 -> pkg1", formatted)
	})

	t.Run("formats 3-package cycle", func(t *testing.T) {
		cd := &CircularDependency{
			Cycle: []string{"pkg1", "pkg2", "pkg3", "pkg1"},
		}
		formatted := cd.FormatCycle()
		assert.Equal(t, "pkg1 -> pkg2 -> pkg3 -> pkg1", formatted)
	})

	t.Run("formats empty cycle", func(t *testing.T) {
		cd := &CircularDependency{
			Cycle: []string{},
		}
		formatted := cd.FormatCycle()
		assert.Equal(t, "", formatted)
	})

	t.Run("formats single element cycle", func(t *testing.T) {
		cd := &CircularDependency{
			Cycle: []string{"pkg1"},
		}
		formatted := cd.FormatCycle()
		assert.Equal(t, "pkg1", formatted)
	})
}
