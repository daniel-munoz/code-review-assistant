package analyzer

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer/detectors"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDetectorRunner is a test helper that returns no issues
type mockDetectorRunner struct{}

func (m *mockDetectorRunner) RunDetectors(cfg *config.AnalysisConfig, file *parser.FileMetrics) []*detectors.Issue {
	return nil
}

// newTestAnalyzer creates an analyzer for testing with minimal dependencies
func newTestAnalyzer(cfg *config.AnalysisConfig) Analyzer {
	return NewAnalyzer(cfg, status.NewSilentReporter(), &mockDetectorRunner{}, nil, nil)
}

func TestNewAnalyzer(t *testing.T) {
	t.Run("creates analyzer with default config", func(t *testing.T) {
		cfg := config.Default()

		analyzer := newTestAnalyzer(&cfg.Analysis)

		require.NotNil(t, analyzer, "analyzer should not be nil")
		assert.IsType(t, &MetricsAnalyzer{}, analyzer, "should be MetricsAnalyzer")
	})

	t.Run("creates analyzer with custom config", func(t *testing.T) {
		cfg := &config.AnalysisConfig{
			LargeFileThreshold:    1000,
			LongFunctionThreshold: 100,
			MinCommentRatio:       0.20,
			ComplexityThreshold:   15,
		}

		analyzer := newTestAnalyzer(cfg)

		require.NotNil(t, analyzer, "analyzer should not be nil")
	})
}

func TestAnalyze_BasicMetrics(t *testing.T) {
	t.Run("analyzes empty project", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err, "should analyze empty project without error")
		require.NotNil(t, result, "result should not be nil")
		assert.Equal(t, "/test/project", result.ProjectPath)
		assert.Equal(t, 0, result.TotalFiles)
		assert.Equal(t, 0, result.TotalLines)
		assert.Equal(t, 0, result.TotalCodeLines)
		assert.Equal(t, 0, result.TotalFunctions)
	})

	t.Run("analyzes single file", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:     "test.go",
				PackageName:  "test",
				TotalLines:   100,
				CodeLines:    80,
				CommentLines: 10,
				BlankLines:   10,
				Functions: []*parser.FunctionMetrics{
					{Name: "TestFunc", Lines: 20, Complexity: 5, StartLine: 10},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalFiles)
		assert.Equal(t, 100, result.TotalLines)
		assert.Equal(t, 80, result.TotalCodeLines)
		assert.Equal(t, 1, result.TotalFunctions)
	})

	t.Run("aggregates multiple files", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:     "file1.go",
				PackageName:  "test",
				TotalLines:   100,
				CodeLines:    80,
				CommentLines: 10,
				Functions: []*parser.FunctionMetrics{
					{Name: "Func1", Lines: 20, Complexity: 5},
				},
			},
			{
				FilePath:     "file2.go",
				PackageName:  "test",
				TotalLines:   200,
				CodeLines:    160,
				CommentLines: 20,
				Functions: []*parser.FunctionMetrics{
					{Name: "Func2", Lines: 30, Complexity: 8},
					{Name: "Func3", Lines: 40, Complexity: 12},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		assert.Equal(t, 2, result.TotalFiles)
		assert.Equal(t, 300, result.TotalLines)
		assert.Equal(t, 240, result.TotalCodeLines)
		assert.Equal(t, 3, result.TotalFunctions)
	})
}

func TestAnalyze_LargeFileDetection(t *testing.T) {
	t.Run("detects large files", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LargeFileThreshold = 100
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "large.go",
				PackageName: "test",
				TotalLines:  150, // Exceeds threshold of 100
				CodeLines:   120,
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		assert.Len(t, result.Files, 1)
		assert.True(t, result.Files[0].LargeFile, "file should be marked as large")

		// Find large_file issue
		var largeFileIssue *Issue
		for _, issue := range result.Issues {
			if issue.Type == "large_file" {
				largeFileIssue = issue
				break
			}
		}

		require.NotNil(t, largeFileIssue, "should have large_file issue")
		assert.Equal(t, "warning", largeFileIssue.Severity)
		assert.Equal(t, "large.go", largeFileIssue.File)
		assert.Equal(t, 150, largeFileIssue.Value)
		assert.Equal(t, 100, largeFileIssue.Threshold)
	})

	t.Run("does not flag files below threshold", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LargeFileThreshold = 100
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "small.go",
				PackageName: "test",
				TotalLines:  50, // Below threshold
				CodeLines:   40,
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		assert.Len(t, result.Files, 1)
		assert.False(t, result.Files[0].LargeFile, "file should not be marked as large")

		// No large_file issue
		for _, issue := range result.Issues {
			assert.NotEqual(t, "large_file", issue.Type, "should not have large_file issue")
		}
	})
}

func TestAnalyze_LongFunctionDetection(t *testing.T) {
	t.Run("detects long functions", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LongFunctionThreshold = 50
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "test.go",
				PackageName: "test",
				TotalLines:  100,
				CodeLines:   80,
				Functions: []*parser.FunctionMetrics{
					{
						Name:      "LongFunc",
						Lines:     75, // Exceeds threshold
						StartLine: 10,
					},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)

		// Find long_function issue
		var longFuncIssue *Issue
		for _, issue := range result.Issues {
			if issue.Type == "long_function" {
				longFuncIssue = issue
				break
			}
		}

		require.NotNil(t, longFuncIssue, "should have long_function issue")
		assert.Equal(t, "warning", longFuncIssue.Severity)
		assert.Equal(t, "test.go", longFuncIssue.File)
		assert.Equal(t, "LongFunc", longFuncIssue.Function)
		assert.Equal(t, 75, longFuncIssue.Value)
		assert.Equal(t, 50, longFuncIssue.Threshold)
		assert.Equal(t, 10, longFuncIssue.Line)
	})

	t.Run("does not flag short functions", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LongFunctionThreshold = 50
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "test.go",
				PackageName: "test",
				Functions: []*parser.FunctionMetrics{
					{Name: "ShortFunc", Lines: 20}, // Below threshold
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)

		// No long_function issue
		for _, issue := range result.Issues {
			assert.NotEqual(t, "long_function", issue.Type)
		}
	})
}

func TestAnalyze_HighComplexityDetection(t *testing.T) {
	t.Run("detects high complexity functions", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.ComplexityThreshold = 10
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "complex.go",
				PackageName: "test",
				Functions: []*parser.FunctionMetrics{
					{
						Name:       "ComplexFunc",
						Complexity: 15, // Exceeds threshold
						Lines:      50,
						StartLine:  20,
					},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)

		// Find high_complexity issue
		var complexIssue *Issue
		for _, issue := range result.Issues {
			if issue.Type == "high_complexity" {
				complexIssue = issue
				break
			}
		}

		require.NotNil(t, complexIssue, "should have high_complexity issue")
		assert.Equal(t, "warning", complexIssue.Severity)
		assert.Equal(t, "complex.go", complexIssue.File)
		assert.Equal(t, "ComplexFunc", complexIssue.Function)
		assert.Equal(t, 15, complexIssue.Value)
		assert.Equal(t, 10, complexIssue.Threshold)
	})
}

func TestAnalyze_CommentRatio(t *testing.T) {
	t.Run("flags low comment ratio", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.MinCommentRatio = 0.15
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:     "test.go",
				PackageName:  "test",
				TotalLines:   100,
				CodeLines:    90,
				CommentLines: 5, // 5% comment ratio, below 15% threshold
				BlankLines:   5,
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotNil(t, result.Metrics)
		assert.Less(t, result.Metrics.CommentRatio, 0.15)

		// Find low_comment_ratio issue
		var commentIssue *Issue
		for _, issue := range result.Issues {
			if issue.Type == "low_comment_ratio" {
				commentIssue = issue
				break
			}
		}

		require.NotNil(t, commentIssue, "should have low_comment_ratio issue")
		assert.Equal(t, "info", commentIssue.Severity)
	})

	t.Run("does not flag good comment ratio", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.MinCommentRatio = 0.15
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:     "test.go",
				PackageName:  "test",
				TotalLines:   100,
				CodeLines:    70,
				CommentLines: 20, // 20% comment ratio, above threshold
				BlankLines:   10,
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotNil(t, result.Metrics)
		assert.GreaterOrEqual(t, result.Metrics.CommentRatio, 0.15)

		// No low_comment_ratio issue
		for _, issue := range result.Issues {
			assert.NotEqual(t, "low_comment_ratio", issue.Type)
		}
	})
}

func TestAnalyze_AggregateMetrics(t *testing.T) {
	t.Run("calculates aggregate metrics", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:     "file1.go",
				PackageName:  "test",
				TotalLines:   200,
				CodeLines:    160,
				CommentLines: 30,
				Functions: []*parser.FunctionMetrics{
					{Name: "Func1", Lines: 20, Complexity: 5},
					{Name: "Func2", Lines: 30, Complexity: 8},
				},
			},
			{
				FilePath:     "file2.go",
				PackageName:  "test",
				TotalLines:   300,
				CodeLines:    240,
				CommentLines: 45,
				Functions: []*parser.FunctionMetrics{
					{Name: "Func3", Lines: 40, Complexity: 12},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotNil(t, result.Metrics)

		// Average function length: (20 + 30 + 40) / 3 = 30
		assert.Equal(t, 30.0, result.Metrics.AverageFunctionLength)

		// Comment ratio: (30 + 45) / (200 + 300) = 75 / 500 = 0.15
		assert.Equal(t, 0.15, result.Metrics.CommentRatio)

		// Average complexity: (5 + 8 + 12) / 3 = 8.33...
		assert.InDelta(t, 8.33, result.Metrics.AverageComplexity, 0.01)
	})

	t.Run("identifies largest files", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{FilePath: "small.go", TotalLines: 100},
			{FilePath: "medium.go", TotalLines: 200},
			{FilePath: "large.go", TotalLines: 500},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotNil(t, result.Metrics)
		require.NotNil(t, result.Metrics.LargestFiles)
		assert.GreaterOrEqual(t, len(result.Metrics.LargestFiles), 1)

		// Largest file should be large.go
		assert.Equal(t, "large.go", result.Metrics.LargestFiles[0].Path)
		assert.Equal(t, 500, result.Metrics.LargestFiles[0].Lines)
	})

	t.Run("identifies most complex functions", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "test.go",
				PackageName: "test",
				Functions: []*parser.FunctionMetrics{
					{Name: "Simple", Complexity: 3, Lines: 10},
					{Name: "Moderate", Complexity: 7, Lines: 20},
					{Name: "Complex", Complexity: 15, Lines: 50},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotNil(t, result.Metrics)
		require.NotNil(t, result.Metrics.MostComplexFunctions)
		assert.GreaterOrEqual(t, len(result.Metrics.MostComplexFunctions), 1)

		// Most complex function should be Complex
		assert.Equal(t, "Complex", result.Metrics.MostComplexFunctions[0].Function)
		assert.Equal(t, 15, result.Metrics.MostComplexFunctions[0].Complexity)
	})
}

func TestAnalyze_MultipleIssueTypes(t *testing.T) {
	t.Run("reports multiple issue types", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LargeFileThreshold = 100
		cfg.Analysis.LongFunctionThreshold = 50
		cfg.Analysis.ComplexityThreshold = 10
		cfg.Analysis.MinCommentRatio = 0.15
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:     "problematic.go",
				PackageName:  "test",
				TotalLines:   150, // Triggers large_file
				CodeLines:    140,
				CommentLines: 5, // Low comment ratio
				Functions: []*parser.FunctionMetrics{
					{
						Name:       "BadFunc",
						Lines:      75,  // Triggers long_function
						Complexity: 20,  // Triggers high_complexity
						StartLine:  10,
					},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		require.NotEmpty(t, result.Issues)

		// Count issue types
		issueTypes := make(map[string]int)
		for _, issue := range result.Issues {
			issueTypes[issue.Type]++
		}

		assert.Contains(t, issueTypes, "large_file")
		assert.Contains(t, issueTypes, "long_function")
		assert.Contains(t, issueTypes, "high_complexity")
		assert.Contains(t, issueTypes, "low_comment_ratio")
	})
}

func TestAnalyze_MethodDetection(t *testing.T) {
	t.Run("handles methods with receiver types", func(t *testing.T) {
		cfg := config.Default()
		analyzer := newTestAnalyzer(&cfg.Analysis)

		metrics := []*parser.FileMetrics{
			{
				FilePath:    "methods.go",
				PackageName: "test",
				Functions: []*parser.FunctionMetrics{
					{
						Name:         "Process",
						ReceiverType: "*User",
						Lines:        20,
						Complexity:   5,
					},
				},
			},
		}

		result, err := analyzer.Analyze("/test/project", metrics)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalFunctions)
	})
}
