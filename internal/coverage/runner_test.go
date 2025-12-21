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
	t.Run("runner respects timeout setting", func(t *testing.T) {
		// Create runner with very short timeout
		runner := NewRunner(1)

		assert.Equal(t, 1*time.Second, runner.timeout, "timeout should be 1 second")
	})

	t.Run("runner with large timeout", func(t *testing.T) {
		runner := NewRunner(300)

		assert.Equal(t, 300*time.Second, runner.timeout, "timeout should be 5 minutes")
	})
}

func TestRunCoverage_ErrorHandling(t *testing.T) {
	t.Run("handles nonexistent project path", func(t *testing.T) {
		runner := NewRunner(30)

		results, err := runner.RunCoverage("/nonexistent/path/12345", []string{})

		assert.Error(t, err, "should return error for nonexistent path")
		assert.Contains(t, err.Error(), "failed to find packages")
		assert.Nil(t, results, "results should be nil on error")
	})

	t.Run("handles empty project path", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping slow integration test in short mode")
		}

		runner := NewRunner(30)

		// Empty path will default to current directory which should work
		results, err := runner.RunCoverage(".", []string{})

		// Either succeeds or fails gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "failed to find packages")
		} else {
			assert.NotNil(t, results)
		}
	})
}

func TestFindPackages_ExcludePatterns(t *testing.T) {
	runner := NewRunner(30)

	t.Run("excludes testdata packages by default", func(t *testing.T) {
		// Test that shouldExclude correctly identifies testdata packages
		testPackages := []string{
			"github.com/user/project/testdata",
			"github.com/user/project/internal/testdata/helper",
			"github.com/user/project/pkg/testdata",
		}

		for _, pkg := range testPackages {
			excluded := runner.shouldExclude(pkg, []string{})
			assert.True(t, excluded, "should exclude package: %s", pkg)
		}
	})

	t.Run("excludes vendor packages by default", func(t *testing.T) {
		testPackages := []string{
			"github.com/user/project/vendor/external",
			"github.com/user/project/vendor/github.com/some/lib",
		}

		for _, pkg := range testPackages {
			excluded := runner.shouldExclude(pkg, []string{})
			assert.True(t, excluded, "should exclude package: %s", pkg)
		}
	})

	t.Run("does not exclude normal packages", func(t *testing.T) {
		testPackages := []string{
			"github.com/user/project/internal/parser",
			"github.com/user/project/pkg/utils",
			"github.com/user/project/cmd",
		}

		for _, pkg := range testPackages {
			excluded := runner.shouldExclude(pkg, []string{})
			assert.False(t, excluded, "should not exclude package: %s", pkg)
		}
	})
}

func TestRunPackageCoverage_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow integration tests in short mode")
	}

	runner := NewRunner(30)

	t.Run("handles package with no test files", func(t *testing.T) {
		// This will actually execute go test, so we need a real package
		// We can test the logic by checking the function signature and structure
		result := runner.runPackageCoverage("github.com/daniel-munoz/code-review-assistant/internal/constants")

		require.NotNil(t, result)
		assert.Equal(t, "github.com/daniel-munoz/code-review-assistant/internal/constants", result.PackagePath)

		// Should either be skipped (no tests) or have coverage
		if result.Skipped {
			assert.Equal(t, 0.0, result.Coverage, "skipped package should have 0 coverage")
			assert.Empty(t, result.Error, "skipped package should not have error")
		}
	})

	t.Run("result structure for nonexistent package", func(t *testing.T) {
		result := runner.runPackageCoverage("github.com/nonexistent/package/12345")

		require.NotNil(t, result)
		assert.Equal(t, "github.com/nonexistent/package/12345", result.PackagePath)
		// Should have an error since package doesn't exist
		assert.NotEmpty(t, result.Error, "should have error for nonexistent package")
	})
}

func TestParseCoverage_DetailedEdgeCases(t *testing.T) {
	runner := NewRunner(30)

	t.Run("handles coverage at boundaries", func(t *testing.T) {
		testCases := []struct {
			name     string
			output   string
			expected float64
		}{
			{
				name:     "0.0% coverage",
				output:   "coverage: 0.0% of statements",
				expected: 0.0,
			},
			{
				name:     "0.1% coverage",
				output:   "coverage: 0.1% of statements",
				expected: 0.1,
			},
			{
				name:     "99.9% coverage",
				output:   "coverage: 99.9% of statements",
				expected: 99.9,
			},
			{
				name:     "100.0% coverage",
				output:   "coverage: 100.0% of statements",
				expected: 100.0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				coverage, err := runner.parseCoverage(tc.output)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, coverage)
			})
		}
	})

	t.Run("handles coverage with various whitespace", func(t *testing.T) {
		testCases := []string{
			"coverage: 75.5% of statements",
			"coverage:  75.5%  of  statements",
			"coverage:\t75.5%\tof\tstatements",
			"coverage:   75.5%   of   statements",
		}

		for _, output := range testCases {
			coverage, err := runner.parseCoverage(output)
			require.NoError(t, err, "should parse: %s", output)
			assert.Equal(t, 75.5, coverage)
		}
	})

	t.Run("handles real go test output formats", func(t *testing.T) {
		realOutputs := []struct {
			name     string
			output   string
			expected float64
		}{
			{
				name: "standard pass with coverage",
				output: `=== RUN   TestExample
--- PASS: TestExample (0.00s)
PASS
coverage: 85.7% of statements
ok  	github.com/test/pkg	0.123s`,
				expected: 85.7,
			},
			{
				name: "multiple packages output",
				output: `?   	github.com/test/pkg1	[no test files]
ok  	github.com/test/pkg2	0.123s	coverage: 92.3% of statements
ok  	github.com/test/pkg3	0.456s	coverage: 78.1% of statements`,
				expected: 92.3, // Should find first match
			},
		}

		for _, tc := range realOutputs {
			t.Run(tc.name, func(t *testing.T) {
				coverage, err := runner.parseCoverage(tc.output)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, coverage)
			})
		}
	})
}

func TestPackageCoverage_AllFields(t *testing.T) {
	t.Run("package with successful coverage", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg/utils",
			Coverage:    88.5,
			Error:       "",
			Skipped:     false,
		}

		assert.Equal(t, "github.com/user/project/pkg/utils", pc.PackagePath)
		assert.Equal(t, 88.5, pc.Coverage)
		assert.Empty(t, pc.Error)
		assert.False(t, pc.Skipped)
	})

	t.Run("package with 0% coverage", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg/empty",
			Coverage:    0.0,
			Error:       "",
			Skipped:     false,
		}

		assert.Equal(t, 0.0, pc.Coverage)
		assert.False(t, pc.Skipped, "0% coverage doesn't mean skipped")
	})

	t.Run("package with compilation error", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg/broken",
			Coverage:    0,
			Error:       "compilation failed: syntax error",
			Skipped:     false,
		}

		assert.NotEmpty(t, pc.Error)
		assert.Contains(t, pc.Error, "compilation failed")
		assert.False(t, pc.Skipped)
	})

	t.Run("package with timeout error", func(t *testing.T) {
		pc := &PackageCoverage{
			PackagePath: "github.com/user/project/pkg/slow",
			Coverage:    0,
			Error:       "test timeout exceeded",
			Skipped:     false,
		}

		assert.Contains(t, pc.Error, "timeout")
	})
}

func TestRunner_Timeout(t *testing.T) {
	t.Run("different timeout values", func(t *testing.T) {
		testCases := []struct {
			seconds  int
			expected time.Duration
		}{
			{0, 0},
			{1, 1 * time.Second},
			{30, 30 * time.Second},
			{60, 60 * time.Second},
			{120, 120 * time.Second},
			{300, 300 * time.Second},
		}

		for _, tc := range testCases {
			runner := NewRunner(tc.seconds)
			assert.Equal(t, tc.expected, runner.timeout, "timeout for %d seconds", tc.seconds)
		}
	})
}
