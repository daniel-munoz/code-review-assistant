package orchestrator

import (
	"errors"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/daniel-munoz/code-review-assistant/internal/comparison"
	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockParser struct {
	metrics []*parser.FileMetrics
	errors  []error
}

func (m *mockParser) ParseFile(filePath string) (*parser.FileMetrics, error) {
	if len(m.metrics) > 0 {
		return m.metrics[0], nil
	}
	if len(m.errors) > 0 {
		return nil, m.errors[0]
	}
	return nil, errors.New("no metrics or errors configured")
}

func (m *mockParser) ParseDirectory(dirPath string, excludePatterns []string) ([]*parser.FileMetrics, []error) {
	return m.metrics, m.errors
}

type mockAnalyzer struct {
	result    *analyzer.AnalysisResult
	shouldErr bool
	errMsg    string
}

func (m *mockAnalyzer) Analyze(projectPath string, metrics []*parser.FileMetrics) (*analyzer.AnalysisResult, error) {
	if m.shouldErr {
		return nil, errors.New(m.errMsg)
	}
	return m.result, nil
}

type mockReporter struct {
	reportCalled bool
	shouldErr    bool
	errMsg       string
}

func (m *mockReporter) Report(result *analyzer.AnalysisResult, comp *comparison.ComparisonResult) error {
	m.reportCalled = true
	if m.shouldErr {
		return errors.New(m.errMsg)
	}
	return nil
}

func TestNew(t *testing.T) {
	t.Run("creates orchestrator with valid config", func(t *testing.T) {
		cfg := config.Default()

		orch, err := New(cfg)

		require.NoError(t, err, "should create orchestrator without error")
		require.NotNil(t, orch, "orchestrator should not be nil")
		assert.NotNil(t, orch.config, "config should be set")
		assert.NotNil(t, orch.parser, "parser should be set")
		assert.NotNil(t, orch.analyzer, "analyzer should be set")
		assert.NotNil(t, orch.reporter, "reporter should be set")
	})

	t.Run("creates orchestrator with custom config", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.LargeFileThreshold = 1000
		cfg.Output.Format = "console"
		cfg.Output.Verbose = true

		orch, err := New(cfg)

		require.NoError(t, err, "should create orchestrator without error")
		require.NotNil(t, orch, "orchestrator should not be nil")
		assert.Equal(t, cfg, orch.config, "config should match input")
	})
}

func TestRun_SuccessCase(t *testing.T) {
	cfg := config.Default()

	// Create mock components
	mockMetrics := []*parser.FileMetrics{
		{
			FilePath:    "test.go",
			PackageName: "test",
			TotalLines:  100,
			CodeLines:   80,
		},
	}

	mockResult := &analyzer.AnalysisResult{
		ProjectPath: "/test/path",
		TotalFiles:  1,
		TotalLines:  100,
	}

	mp := &mockParser{metrics: mockMetrics}
	ma := &mockAnalyzer{result: mockResult}
	mr := &mockReporter{}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	err := orch.Run("/test/path")

	assert.NoError(t, err, "should complete successfully")
	assert.True(t, mr.reportCalled, "reporter should have been called")
}

func TestRun_NoFilesFound(t *testing.T) {
	cfg := config.Default()

	// Mock parser that returns no metrics
	mp := &mockParser{
		metrics: []*parser.FileMetrics{},
		errors:  []error{},
	}
	ma := &mockAnalyzer{}
	mr := &mockReporter{}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	err := orch.Run("/empty/path")

	assert.Error(t, err, "should return error when no files found")
	assert.Contains(t, err.Error(), "no Go files found", "error message should indicate no files")
	assert.False(t, mr.reportCalled, "reporter should not be called when no files found")
}

func TestRun_ParseErrors(t *testing.T) {
	cfg := config.Default()

	// Mock parser that returns some metrics and some errors
	mockMetrics := []*parser.FileMetrics{
		{
			FilePath:    "good.go",
			PackageName: "test",
			TotalLines:  100,
		},
	}

	parseErrors := []error{
		errors.New("failed to parse file1.go: syntax error"),
		errors.New("failed to parse file2.go: invalid code"),
	}

	mockResult := &analyzer.AnalysisResult{
		ProjectPath: "/test/path",
		TotalFiles:  1,
	}

	mp := &mockParser{
		metrics: mockMetrics,
		errors:  parseErrors,
	}
	ma := &mockAnalyzer{result: mockResult}
	mr := &mockReporter{}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	// Should succeed despite parse errors (continues with successfully parsed files)
	err := orch.Run("/test/path")

	assert.NoError(t, err, "should succeed with some successfully parsed files")
	assert.True(t, mr.reportCalled, "reporter should be called")
}

func TestRun_ManyParseErrors(t *testing.T) {
	cfg := config.Default()

	// Mock parser with many errors (tests error limiting)
	mockMetrics := []*parser.FileMetrics{
		{FilePath: "good.go", PackageName: "test"},
	}

	parseErrors := make([]error, 10)
	for i := 0; i < 10; i++ {
		parseErrors[i] = errors.New("parse error")
	}

	mockResult := &analyzer.AnalysisResult{
		ProjectPath: "/test/path",
	}

	mp := &mockParser{
		metrics: mockMetrics,
		errors:  parseErrors,
	}
	ma := &mockAnalyzer{result: mockResult}
	mr := &mockReporter{}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	err := orch.Run("/test/path")

	// Should still succeed (prints warning but continues)
	assert.NoError(t, err, "should succeed despite many parse errors")
	assert.True(t, mr.reportCalled, "reporter should be called")
}

func TestRun_AnalyzerFailure(t *testing.T) {
	cfg := config.Default()

	mockMetrics := []*parser.FileMetrics{
		{FilePath: "test.go", PackageName: "test"},
	}

	mp := &mockParser{metrics: mockMetrics}
	ma := &mockAnalyzer{
		shouldErr: true,
		errMsg:    "analysis failed: internal error",
	}
	mr := &mockReporter{}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	err := orch.Run("/test/path")

	assert.Error(t, err, "should return error when analyzer fails")
	assert.Contains(t, err.Error(), "analysis failed", "error message should indicate analysis failure")
	assert.False(t, mr.reportCalled, "reporter should not be called when analysis fails")
}

func TestRun_ReporterFailure(t *testing.T) {
	cfg := config.Default()

	mockMetrics := []*parser.FileMetrics{
		{FilePath: "test.go", PackageName: "test"},
	}

	mockResult := &analyzer.AnalysisResult{
		ProjectPath: "/test/path",
	}

	mp := &mockParser{metrics: mockMetrics}
	ma := &mockAnalyzer{result: mockResult}
	mr := &mockReporter{
		shouldErr: true,
		errMsg:    "failed to write report",
	}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	err := orch.Run("/test/path")

	assert.Error(t, err, "should return error when reporter fails")
	assert.Contains(t, err.Error(), "failed to generate report", "error message should indicate report failure")
	assert.True(t, mr.reportCalled, "reporter should have been called (even though it failed)")
}

func TestRun_ExcludePatterns(t *testing.T) {
	cfg := config.Default()
	cfg.Analysis.ExcludePatterns = []string{"vendor/**", "**/*_test.go"}

	mp := &mockParser{
		metrics: []*parser.FileMetrics{
			{FilePath: "main.go", PackageName: "main"},
		},
	}

	ma := &mockAnalyzer{
		result: &analyzer.AnalysisResult{ProjectPath: "/test"},
	}
	mr := &mockReporter{}

	orch := &Orchestrator{
		config:   cfg,
		parser:   mp,
		analyzer: ma,
		reporter: mr,
	}

	err := orch.Run("/test/path")

	assert.NoError(t, err, "should execute successfully with exclude patterns")
}

func TestRun_IntegrationFlow(t *testing.T) {
	t.Run("complete pipeline execution", func(t *testing.T) {
		cfg := config.Default()

		mp := &mockParser{
			metrics: []*parser.FileMetrics{
				{FilePath: "test.go", PackageName: "test", TotalLines: 100},
			},
		}

		ma := &mockAnalyzer{
			result: &analyzer.AnalysisResult{
				ProjectPath: "/test",
				TotalFiles:  1,
				TotalLines:  100,
			},
		}

		mr := &mockReporter{}

		orch := &Orchestrator{
			config:   cfg,
			parser:   mp,
			analyzer: ma,
			reporter: mr,
		}

		err := orch.Run("/test/path")

		assert.NoError(t, err, "complete pipeline should execute successfully")
		assert.True(t, mr.reportCalled, "reporter should be called as final step")
	})
}
