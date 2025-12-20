package coverage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	t.Run("creates runner with default timeout", func(t *testing.T) {
		runner := NewRunner(30)

		require.NotNil(t, runner, "runner should not be nil")
		assert.Equal(t, 30*time.Second, runner.timeout, "timeout should be set correctly")
	})

	t.Run("creates runner with custom timeout", func(t *testing.T) {
		runner := NewRunner(60)

		require.NotNil(t, runner, "runner should not be nil")
		assert.Equal(t, 60*time.Second, runner.timeout, "timeout should be set to 60 seconds")
	})

	t.Run("creates runner with zero timeout", func(t *testing.T) {
		runner := NewRunner(0)

		require.NotNil(t, runner, "runner should not be nil")
		assert.Equal(t, time.Duration(0), runner.timeout, "timeout should be zero")
	})
}

func TestParseCoverage(t *testing.T) {
	runner := NewRunner(30)

	testCases := []struct {
		name        string
		output      string
		expected    float64
		expectError bool
	}{
		{
			name:        "parses standard coverage output",
			output:      "PASS\ncoverage: 75.5% of statements\nok      github.com/test/package  0.123s",
			expected:    75.5,
			expectError: false,
		},
		{
			name:        "parses 100% coverage",
			output:      "coverage: 100.0% of statements",
			expected:    100.0,
			expectError: false,
		},
		{
			name:        "parses 0% coverage",
			output:      "coverage: 0.0% of statements",
			expected:    0.0,
			expectError: false,
		},
		{
			name:        "parses fractional coverage",
			output:      "coverage: 42.3% of statements",
			expected:    42.3,
			expectError: false,
		},
		{
			name:        "handles coverage with extra whitespace",
			output:      "coverage:   85.2%   of   statements",
			expected:    85.2,
			expectError: false,
		},
		{
			name:        "returns error when coverage not found",
			output:      "PASS\nok      github.com/test/package  0.123s",
			expectError: true,
		},
		{
			name:        "returns error for malformed output",
			output:      "coverage: XX.X% of statements",
			expectError: true,
		},
		{
			name:        "returns error for empty output",
			output:      "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			coverage, err := runner.parseCoverage(tc.output)

			if tc.expectError {
				assert.Error(t, err, "should return error for invalid input")
			} else {
				require.NoError(t, err, "should parse without error")
				assert.Equal(t, tc.expected, coverage, "coverage value should match")
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	runner := NewRunner(30)

	testCases := []struct {
		name     string
		pkg      string
		patterns []string
		expected bool
	}{
		{
			name:     "excludes testdata packages",
			pkg:      "github.com/user/project/testdata",
			patterns: []string{},
			expected: true,
		},
		{
			name:     "excludes packages containing testdata",
			pkg:      "github.com/user/project/internal/testdata/helper",
			patterns: []string{},
			expected: true,
		},
		{
			name:     "excludes vendor packages",
			pkg:      "github.com/user/project/vendor/external",
			patterns: []string{},
			expected: true,
		},
		{
			name:     "does not exclude normal packages",
			pkg:      "github.com/user/project/internal/parser",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "does not exclude packages with test in name",
			pkg:      "github.com/user/project/testutil",
			patterns: []string{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := runner.shouldExclude(tc.pkg, tc.patterns)
			assert.Equal(t, tc.expected, result, "exclusion check should match expected")
		})
	}
}

func TestRunCoverage_Integration(t *testing.T) {
	t.Skip("Skipping integration test - runs actual go test which can hang test suite")

	t.Run("runs coverage on current package", func(t *testing.T) {
		runner := NewRunner(60)

		// Run coverage on the current working directory (which should have this test)
		results, err := runner.RunCoverage(".", []string{})

		require.NoError(t, err, "should run coverage without error")
		require.NotEmpty(t, results, "should return coverage results")

		// Verify we got results for the coverage package
		foundSelf := false
		for _, result := range results {
			if result.PackagePath == "github.com/daniel-munoz/code-review-assistant/internal/coverage" {
				foundSelf = true
				// This package should have tests now
				assert.False(t, result.Skipped, "coverage package should have tests")
				if result.Error == "" {
					assert.Greater(t, result.Coverage, 0.0, "coverage should be > 0")
				}
			}
		}

		assert.True(t, foundSelf, "should find coverage package in results")
	})
}

func TestPackageCoverage_Struct(t *testing.T) {
	t.Run("creates package coverage with all fields", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg",
			Coverage:    75.5,
			Error:       "",
			Skipped:     false,
		}

		assert.Equal(t, "github.com/user/project/pkg", pc.PackagePath)
		assert.Equal(t, 75.5, pc.Coverage)
		assert.Empty(t, pc.Error)
		assert.False(t, pc.Skipped)
	})

	t.Run("represents skipped package", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg",
			Coverage:    0,
			Error:       "",
			Skipped:     true,
		}

		assert.True(t, pc.Skipped, "package should be marked as skipped")
		assert.Equal(t, 0.0, pc.Coverage, "skipped package has 0 coverage")
	})

	t.Run("represents package with error", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg",
			Coverage:    0,
			Error:       "compilation failed",
			Skipped:     false,
		}

		assert.NotEmpty(t, pc.Error, "package should have error message")
		assert.Equal(t, 0.0, pc.Coverage, "failed package has 0 coverage")
	})
}

func TestRunCoverage_RealProject(t *testing.T) {
	t.Skip("Skipping integration test - runs actual go test which can hang test suite")

	t.Run("runs coverage on project root", func(t *testing.T) {
		runner := NewRunner(120) // Longer timeout for full project

		// Run coverage on the project root (two levels up from this package)
		results, err := runner.RunCoverage("../..", []string{})

		require.NoError(t, err, "should run coverage without error")
		require.NotEmpty(t, results, "should return coverage results")

		// Verify we got results for multiple packages
		packagesWithCoverage := 0
		packagesSkipped := 0
		packagesWithErrors := 0

		for _, result := range results {
			if result.Skipped {
				packagesSkipped++
			} else if result.Error != "" {
				packagesWithErrors++
			} else if result.Coverage > 0 {
				packagesWithCoverage++
			}
		}

		// Should have at least some packages with coverage
		assert.Greater(t, packagesWithCoverage, 0, "should have packages with coverage")

		// Total packages should equal sum of categories
		total := packagesWithCoverage + packagesSkipped + packagesWithErrors
		assert.Equal(t, len(results), total, "all results should be categorized")
	})
}

func TestParseCoverage_EdgeCases(t *testing.T) {
	runner := NewRunner(30)

	t.Run("handles very high precision coverage", func(t *testing.T) {
		output := "coverage: 99.99999% of statements"
		coverage, err := runner.parseCoverage(output)

		require.NoError(t, err)
		assert.Equal(t, 99.99999, coverage)
	})

	t.Run("handles single digit coverage", func(t *testing.T) {
		output := "coverage: 5% of statements"
		coverage, err := runner.parseCoverage(output)

		require.NoError(t, err)
		assert.Equal(t, 5.0, coverage)
	})

	t.Run("handles coverage with newlines", func(t *testing.T) {
		output := "PASS\n\ncoverage: 68.5% of statements\n\nok  \tpkg\t0.123s"
		coverage, err := runner.parseCoverage(output)

		require.NoError(t, err)
		assert.Equal(t, 68.5, coverage)
	})
}

func TestTimeout(t *testing.T) {
	t.Skip("Skipping timeout test - not essential for coverage")

	t.Run("runner respects timeout setting", func(t *testing.T) {
		// Create runner with very short timeout
		runner := NewRunner(1)

		assert.Equal(t, 1*time.Second, runner.timeout, "timeout should be 1 second")
	})
}
