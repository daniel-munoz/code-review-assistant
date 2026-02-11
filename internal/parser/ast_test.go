package parser

import (
	"path/filepath"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	testCases := []struct {
		name             string
		filePath         string
		expectError      bool
		expectedPackage  string
		expectedFuncCount int
		validateFunc     func(t *testing.T, metrics *FileMetrics)
	}{
		{
			name:             "parse main.go",
			filePath:         "../../testdata/sample/main.go",
			expectError:      false,
			expectedPackage:  "sample",
			expectedFuncCount: 4, // main, Add, Calculate, checkEnvironment
			validateFunc: func(t *testing.T, metrics *FileMetrics) {
				assert.Greater(t, metrics.TotalLines, 0, "should have lines")
				assert.Greater(t, metrics.CodeLines, 0, "should have code lines")
				assert.Greater(t, metrics.CommentLines, 0, "should have comments")

				// Verify specific functions exist
				funcNames := make(map[string]bool)
				for _, fn := range metrics.Functions {
					funcNames[fn.Name] = true
				}
				assert.True(t, funcNames["main"], "should have main function")
				assert.True(t, funcNames["Add"], "should have Add function")
				assert.True(t, funcNames["Calculate"], "should have Calculate function")

				// Check Add function details
				for _, fn := range metrics.Functions {
					if fn.Name == "Add" {
						assert.Equal(t, 2, fn.Parameters, "Add should have 2 parameters")
						assert.Equal(t, 1, fn.ReturnValues, "Add should have 1 return value")
						assert.False(t, fn.IsMethod(), "Add should not be a method")
					}
					if fn.Name == "Calculate" {
						assert.Equal(t, 2, fn.Parameters, "Calculate should have 2 parameters")
						assert.Equal(t, 2, fn.ReturnValues, "Calculate should have 2 return values")
					}
				}

				// Verify imports
				assert.Contains(t, metrics.Imports, "fmt", "should import fmt")
				assert.Contains(t, metrics.Imports, "os", "should import os")
			},
		},
		{
			name:             "parse util.go with methods",
			filePath:         "../../testdata/sample/util.go",
			expectError:      false,
			expectedPackage:  "sample",
			expectedFuncCount: 6, // 3 methods + 3 functions
			validateFunc: func(t *testing.T, metrics *FileMetrics) {
				// Count methods vs functions
				methodCount := 0
				funcCount := 0

				for _, fn := range metrics.Functions {
					if fn.IsMethod() {
						methodCount++
						assert.Equal(t, "*User", fn.ReceiverType, "User methods should have *User receiver")
					} else {
						funcCount++
					}
				}

				assert.Equal(t, 3, methodCount, "should have 3 methods")
				assert.Equal(t, 3, funcCount, "should have 3 functions")

				// Verify method full names
				for _, fn := range metrics.Functions {
					if fn.Name == "IsValid" {
						assert.Equal(t, "*User.IsValid", fn.FullName(), "method should have full name")
					}
				}
			},
		},
		{
			name:             "parse large.go",
			filePath:         "../../testdata/sample/large.go",
			expectError:      false,
			expectedPackage:  "sample",
			expectedFuncCount: 61, // 60 ProcessData + 1 LongFunction
			validateFunc: func(t *testing.T, metrics *FileMetrics) {
				assert.Greater(t, metrics.TotalLines, 500, "large.go should exceed 500 lines")

				// Find LongFunction and verify it exceeds threshold
				for _, fn := range metrics.Functions {
					if fn.Name == "LongFunction" {
						assert.Greater(t, fn.Lines, 50, "LongFunction should exceed 50 lines")
					}
				}
			},
		},
		{
			name:        "parse non-existent file",
			filePath:    "nonexistent.go",
			expectError: true,
		},
		{
			name:        "parse non-go file",
			filePath:    "../../README.md",
			expectError: true,
		},
	}

	parser := NewParser(1)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics, err := parser.ParseFile(tc.filePath)

			if tc.expectError {
				assert.Error(t, err, "expected error parsing file")
				return
			}

			require.NoError(t, err, "should parse file without error")
			require.NotNil(t, metrics, "metrics should not be nil")

			assert.Equal(t, tc.expectedPackage, metrics.PackageName, "package name should match")
			assert.Len(t, metrics.Functions, tc.expectedFuncCount, "function count should match")

			// Validate line counts add up
			assert.Equal(t, metrics.TotalLines, metrics.CodeLines+metrics.CommentLines+metrics.BlankLines,
				"line counts should add up")

			if tc.validateFunc != nil {
				tc.validateFunc(t, metrics)
			}
		})
	}
}

func TestParseDirectory(t *testing.T) {
	parser := NewParser(1)

	t.Run("parse sample directory", func(t *testing.T) {
		metricsSlice, errors := parser.ParseDirectory("../../testdata/sample", []string{}, []string{".go"}, status.NewSilentReporter())

		// Should have no errors
		assert.Empty(t, errors, "should parse all files without errors")

		// Should find all Go files (main.go, util.go, large.go, errors.go, antipatterns.go, complex.go)
		assert.Len(t, metricsSlice, 6, "should find 6 Go files")

		// Verify we got metrics for each file
		fileNames := make(map[string]bool)
		for _, m := range metricsSlice {
			fileName := filepath.Base(m.FilePath)
			fileNames[fileName] = true
		}

		assert.True(t, fileNames["main.go"], "should have parsed main.go")
		assert.True(t, fileNames["util.go"], "should have parsed util.go")
		assert.True(t, fileNames["large.go"], "should have parsed large.go")
		assert.True(t, fileNames["errors.go"], "should have parsed errors.go")
		assert.True(t, fileNames["antipatterns.go"], "should have parsed antipatterns.go")
		assert.True(t, fileNames["complex.go"], "should have parsed complex.go")
	})

	t.Run("parse with exclusions", func(t *testing.T) {
		excludePatterns := []string{"**/large.go"}
		metricsSlice, errors := parser.ParseDirectory("../../testdata/sample", excludePatterns, []string{".go"}, status.NewSilentReporter())

		assert.Empty(t, errors, "should parse without errors")

		// Should have 5 files (excluding large.go from 6 total)
		assert.Len(t, metricsSlice, 5, "should find 5 files after exclusion")

		// Verify large.go was excluded
		for _, m := range metricsSlice {
			fileName := filepath.Base(m.FilePath)
			assert.NotEqual(t, "large.go", fileName, "large.go should be excluded")
		}
	})

	t.Run("parse non-existent directory", func(t *testing.T) {
		_, errors := parser.ParseDirectory("nonexistent", []string{}, []string{".go"}, status.NewSilentReporter())
		assert.NotEmpty(t, errors, "should have errors for non-existent directory")
	})
}

func TestMetricsHelpers(t *testing.T) {
	t.Run("CommentRatio calculation", func(t *testing.T) {
		metrics := &FileMetrics{
			TotalLines:   100,
			CodeLines:    70,
			CommentLines: 20,
			BlankLines:   10,
		}

		ratio := metrics.CommentRatio()
		assert.Equal(t, 0.2, ratio, "comment ratio should be 20%")
	})

	t.Run("CodeRatio calculation", func(t *testing.T) {
		metrics := &FileMetrics{
			TotalLines:   100,
			CodeLines:    70,
			CommentLines: 20,
			BlankLines:   10,
		}

		ratio := metrics.CodeRatio()
		assert.Equal(t, 0.7, ratio, "code ratio should be 70%")
	})

	t.Run("zero lines edge case", func(t *testing.T) {
		metrics := &FileMetrics{
			TotalLines: 0,
		}

		assert.Equal(t, 0.0, metrics.CommentRatio(), "should handle zero total lines")
		assert.Equal(t, 0.0, metrics.CodeRatio(), "should handle zero total lines")
	})
}

func TestFunctionMetrics(t *testing.T) {
	t.Run("IsMethod", func(t *testing.T) {
		method := &FunctionMetrics{
			Name:         "Process",
			ReceiverType: "*User",
		}
		assert.True(t, method.IsMethod(), "should be a method")

		function := &FunctionMetrics{
			Name:         "Process",
			ReceiverType: "",
		}
		assert.False(t, function.IsMethod(), "should not be a method")
	})

	t.Run("FullName", func(t *testing.T) {
		method := &FunctionMetrics{
			Name:         "Process",
			ReceiverType: "*User",
		}
		assert.Equal(t, "*User.Process", method.FullName(), "method should have receiver in full name")

		function := &FunctionMetrics{
			Name:         "Process",
			ReceiverType: "",
		}
		assert.Equal(t, "Process", function.FullName(), "function should just have name")
	})
}

func TestPatternMatching(t *testing.T) {
	testCases := []struct {
		path     string
		pattern  string
		expected bool
	}{
		{"vendor/foo.go", "vendor/**", true},
		{"pkg/vendor/foo.go", "vendor/**", false},
		{"vendor/foo.go", "**/vendor/**", true},
		{"pkg/vendor/foo.go", "**/vendor/**", true},
		{"foo_test.go", "**/*_test.go", true},
		{"pkg/foo_test.go", "**/*_test.go", true},
		{"foo.go", "**/*_test.go", false},
		{"testdata/sample.go", "**/testdata/**", true},
		{"pkg/testdata/sample.go", "**/testdata/**", true},
		{"foo.pb.go", "**/*.pb.go", true},
		{"pkg/foo.pb.go", "**/*.pb.go", true},
		{"foo.go", "**/*.pb.go", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path+" vs "+tc.pattern, func(t *testing.T) {
			result := matchPattern(tc.path, tc.pattern)
			assert.Equal(t, tc.expected, result, "pattern match should be correct")
		})
	}
}

func TestComplexity(t *testing.T) {
	parser := NewParser(1)

	metrics, err := parser.ParseFile("../../testdata/sample/util.go")
	require.NoError(t, err)

	for _, fn := range metrics.Functions {
		// Every function should have complexity >= 1
		assert.GreaterOrEqual(t, fn.Complexity, 1, "function %s should have complexity >= 1", fn.Name)

		// IsValid has multiple if statements, should have higher complexity
		if fn.Name == "IsValid" {
			assert.Greater(t, fn.Complexity, 1, "IsValid should have complexity > 1")
		}

		// Contains has a for loop, should have higher complexity
		if fn.Name == "Contains" {
			assert.Greater(t, fn.Complexity, 1, "Contains should have complexity > 1")
		}
	}
}

// TestParseDirectory_ParallelVsSequential verifies that parallel parsing produces
// the same results as sequential parsing (order-independent).
func TestParseDirectory_ParallelVsSequential(t *testing.T) {
	testDir := "../../testdata/sample"

	// Parse sequentially (workers=1)
	seqParser := NewParser(1)
	seqMetrics, seqErrors := seqParser.ParseDirectory(testDir, []string{}, []string{".go"}, status.NewSilentReporter())

	// Parse in parallel (workers=4)
	parParser := NewParser(4)
	parMetrics, parErrors := parParser.ParseDirectory(testDir, []string{}, []string{".go"}, status.NewSilentReporter())

	// Should have same number of files
	assert.Equal(t, len(seqMetrics), len(parMetrics), "parallel and sequential should find same number of files")

	// Should have same number of errors
	assert.Equal(t, len(seqErrors), len(parErrors), "parallel and sequential should have same number of errors")

	// Build maps for order-independent comparison
	seqFileMap := make(map[string]*FileMetrics)
	for _, m := range seqMetrics {
		seqFileMap[m.FilePath] = m
	}

	parFileMap := make(map[string]*FileMetrics)
	for _, m := range parMetrics {
		parFileMap[m.FilePath] = m
	}

	// Compare each file's metrics
	for path, seqFile := range seqFileMap {
		parFile, exists := parFileMap[path]
		require.True(t, exists, "parallel should have file %s", path)

		assert.Equal(t, seqFile.TotalLines, parFile.TotalLines, "TotalLines should match for %s", path)
		assert.Equal(t, seqFile.CodeLines, parFile.CodeLines, "CodeLines should match for %s", path)
		assert.Equal(t, seqFile.CommentLines, parFile.CommentLines, "CommentLines should match for %s", path)
		assert.Equal(t, len(seqFile.Functions), len(parFile.Functions), "Function count should match for %s", path)
	}
}
