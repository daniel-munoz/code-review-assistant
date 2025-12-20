package detectors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	parserPkg "github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Run("creates registry with all detectors", func(t *testing.T) {
		cfg := config.Default()

		registry := NewRegistry(&cfg.Analysis)

		require.NotNil(t, registry, "registry should not be nil")
		assert.Len(t, registry.detectors, 5, "should have 5 detectors")
		assert.NotNil(t, registry.config, "config should be set")
	})

	t.Run("creates registry with custom config", func(t *testing.T) {
		cfg := &config.AnalysisConfig{
			MaxParameters:         7,
			MaxNestingDepth:       5,
			MaxReturnStatements:   4,
			DetectMagicNumbers:    false,
			DetectDuplicateErrors: false,
		}

		registry := NewRegistry(cfg)

		require.NotNil(t, registry)
		assert.Equal(t, cfg, registry.config)
	})
}

func TestRegistry_RunAll(t *testing.T) {
	t.Run("runs all enabled detectors", func(t *testing.T) {
		cfg := config.Default()
		registry := NewRegistry(&cfg.Analysis)

		// Parse a test function with too many parameters
		code := `package test
		func TooManyParams(a, b, c, d, e, f, g int) {
			// This function has 7 parameters, exceeding default threshold of 5
		}`

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", code, 0)
		require.NoError(t, err)

		// Extract the function declaration
		var funcDecl *ast.FuncDecl
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				funcDecl = fd
				break
			}
		}
		require.NotNil(t, funcDecl)

		// Create file and function metrics
		fileMetrics := &parserPkg.FileMetrics{
			FilePath: "test.go",
		}
		funcMetrics := &parserPkg.FunctionMetrics{
			Name:       "TooManyParams",
			Parameters: 7,
		}

		issues := registry.RunAll(fileMetrics, funcMetrics, fset, funcDecl)

		// Should detect too many parameters
		assert.NotEmpty(t, issues, "should detect issues")

		// Find parameter count issue
		var paramIssue *Issue
		for _, issue := range issues {
			if issue.Type == "too_many_parameters" {
				paramIssue = issue
				break
			}
		}

		require.NotNil(t, paramIssue, "should detect too many parameters")
		assert.Equal(t, "warning", paramIssue.Severity)
		assert.Equal(t, 7, paramIssue.Value)
		assert.Equal(t, 5, paramIssue.Threshold)
	})

	t.Run("only runs enabled detectors", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.DetectMagicNumbers = false // Disable magic number detection
		registry := NewRegistry(&cfg.Analysis)

		// Parse a test function with magic numbers
		code := `package test
		func WithMagicNumber() int {
			return 42 // Magic number
		}`

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", code, 0)
		require.NoError(t, err)

		var funcDecl *ast.FuncDecl
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				funcDecl = fd
				break
			}
		}
		require.NotNil(t, funcDecl)

		fileMetrics := &parserPkg.FileMetrics{FilePath: "test.go"}
		funcMetrics := &parserPkg.FunctionMetrics{Name: "WithMagicNumber"}

		issues := registry.RunAll(fileMetrics, funcMetrics, fset, funcDecl)

		// Should NOT detect magic numbers since it's disabled
		for _, issue := range issues {
			assert.NotEqual(t, "magic_number", issue.Type, "should not detect magic numbers when disabled")
		}
	})

	t.Run("returns empty issues for clean function", func(t *testing.T) {
		cfg := config.Default()
		registry := NewRegistry(&cfg.Analysis)

		// Parse a clean function
		code := `package test
		func CleanFunction(a int) int {
			return a
		}`

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", code, 0)
		require.NoError(t, err)

		var funcDecl *ast.FuncDecl
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				funcDecl = fd
				break
			}
		}
		require.NotNil(t, funcDecl)

		fileMetrics := &parserPkg.FileMetrics{FilePath: "test.go"}
		funcMetrics := &parserPkg.FunctionMetrics{
			Name:       "CleanFunction",
			Parameters: 1,
		}

		issues := registry.RunAll(fileMetrics, funcMetrics, fset, funcDecl)

		// Clean function should have no issues (or very few)
		assert.LessOrEqual(t, len(issues), 0, "clean function should have minimal issues")
	})
}

func TestParameterCountDetector(t *testing.T) {
	detector := NewParameterCountDetector()

	t.Run("name returns parameter_count", func(t *testing.T) {
		assert.Equal(t, "parameter_count", detector.Name())
	})

	t.Run("enabled returns true by default", func(t *testing.T) {
		cfg := config.Default()
		assert.True(t, detector.Enabled(&cfg.Analysis))
	})

	t.Run("detects too many parameters", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.MaxParameters = 3

		fileMetrics := &parserPkg.FileMetrics{FilePath: "test.go"}
		funcMetrics := &parserPkg.FunctionMetrics{
			Name:       "TooManyParams",
			Parameters: 5, // Exceeds threshold of 3
			StartLine:  10,
		}

		issues := detector.Detect(&cfg.Analysis, fileMetrics, funcMetrics, nil, nil)

		require.Len(t, issues, 1)
		assert.Equal(t, "too_many_parameters", issues[0].Type)
		assert.Equal(t, "warning", issues[0].Severity)
		assert.Equal(t, 5, issues[0].Value)
		assert.Equal(t, 3, issues[0].Threshold)
		assert.Equal(t, "test.go", issues[0].File)
		assert.Equal(t, 10, issues[0].Line)
	})

	t.Run("does not flag functions below threshold", func(t *testing.T) {
		cfg := config.Default()
		cfg.Analysis.MaxParameters = 5

		fileMetrics := &parserPkg.FileMetrics{FilePath: "test.go"}
		funcMetrics := &parserPkg.FunctionMetrics{
			Name:       "FewParams",
			Parameters: 3, // Below threshold
		}

		issues := detector.Detect(&cfg.Analysis, fileMetrics, funcMetrics, nil, nil)

		assert.Empty(t, issues, "should not flag functions below threshold")
	})
}

func TestIssueStruct(t *testing.T) {
	t.Run("creates issue with all fields", func(t *testing.T) {
		issue := &Issue{
			Severity:  "warning",
			Type:      "test_issue",
			File:      "test.go",
			Line:      42,
			Function:  "TestFunc",
			Message:   "This is a test issue",
			Value:     100,
			Threshold: 50,
		}

		assert.Equal(t, "warning", issue.Severity)
		assert.Equal(t, "test_issue", issue.Type)
		assert.Equal(t, "test.go", issue.File)
		assert.Equal(t, 42, issue.Line)
		assert.Equal(t, "TestFunc", issue.Function)
		assert.Equal(t, "This is a test issue", issue.Message)
		assert.Equal(t, 100, issue.Value)
		assert.Equal(t, 50, issue.Threshold)
	})
}
