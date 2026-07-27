package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func TestNewParser(t *testing.T) {
	p := NewParser(1)
	require.NotNil(t, p, "parser should not be nil")
	assert.NotNil(t, p.language, "language should be set")
}

func TestParseFile_SamplePython(t *testing.T) {
	p := NewParser(1)

	// Get path to test data
	testFile := filepath.Join("..", "..", "..", "testdata", "python", "sample.py")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	// Skip if test file doesn't exist
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")
	require.NotNil(t, metrics, "metrics should not be nil")

	// Verify basic metrics
	assert.Equal(t, absPath, metrics.FilePath)
	assert.Equal(t, "sample", metrics.PackageName)
	assert.Equal(t, "python", metrics.Language)

	// Should have found functions and classes
	assert.Greater(t, len(metrics.Functions), 0, "should have functions")
	assert.Greater(t, metrics.TotalLines, 0, "should have lines")
	assert.Greater(t, metrics.CodeLines, 0, "should have code lines")

	// Check for expected functions
	functionNames := make(map[string]bool)
	for _, fn := range metrics.Functions {
		functionNames[fn.Name] = true
	}

	assert.True(t, functionNames["greet"], "should have greet function")
	assert.True(t, functionNames["calculate_total"], "should have calculate_total function")
	assert.True(t, functionNames["process_data"], "should have process_data function")
	assert.True(t, functionNames["main"], "should have main function")

	// Check for methods in classes
	assert.True(t, functionNames["__init__"], "should have __init__ method")
	assert.True(t, functionNames["add_item"], "should have add_item method")

	// Verify imports
	assert.Greater(t, len(metrics.Imports), 0, "should have imports")
}

func TestParseFile_ComplexPython(t *testing.T) {
	p := NewParser(1)

	testFile := filepath.Join("..", "..", "..", "testdata", "python", "complex.py")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")

	// Find the highly_complex_function
	var complexFunc *struct {
		Name       string
		Complexity int
		Parameters int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "highly_complex_function" {
			complexFunc = &struct {
				Name       string
				Complexity int
				Parameters int
			}{fn.Name, fn.Complexity, fn.Parameters}
			break
		}
	}

	require.NotNil(t, complexFunc, "should find highly_complex_function")
	assert.Greater(t, complexFunc.Complexity, 10, "highly_complex_function should have high complexity")
	assert.Equal(t, 6, complexFunc.Parameters, "should have 6 parameters (excluding self)")

	// Find deeply_nested_function
	var nestedFunc *struct {
		Name       string
		Complexity int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "deeply_nested_function" {
			nestedFunc = &struct {
				Name       string
				Complexity int
			}{fn.Name, fn.Complexity}
			break
		}
	}

	require.NotNil(t, nestedFunc, "should find deeply_nested_function")
	assert.Greater(t, nestedFunc.Complexity, 5, "deeply_nested_function should have elevated complexity")
}

func TestParseFile_Antipatterns(t *testing.T) {
	p := NewParser(1)

	testFile := filepath.Join("..", "..", "..", "testdata", "python", "antipatterns.py")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")

	// Find function_with_too_many_params
	var manyParamsFunc *struct {
		Name       string
		Parameters int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "function_with_too_many_params" {
			manyParamsFunc = &struct {
				Name       string
				Parameters int
			}{fn.Name, fn.Parameters}
			break
		}
	}

	require.NotNil(t, manyParamsFunc, "should find function_with_too_many_params")
	assert.Equal(t, 7, manyParamsFunc.Parameters, "should have 7 parameters")

	// Find AntiPatternClass.__init__
	var classInit *struct {
		Name         string
		Parameters   int
		ReceiverType string
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "__init__" && fn.ReceiverType == "AntiPatternClass" {
			classInit = &struct {
				Name         string
				Parameters   int
				ReceiverType string
			}{fn.Name, fn.Parameters, fn.ReceiverType}
			break
		}
	}

	require.NotNil(t, classInit, "should find AntiPatternClass.__init__")
	assert.Equal(t, 7, classInit.Parameters, "class __init__ should have 7 parameters (excluding self)")
	assert.Equal(t, "AntiPatternClass", classInit.ReceiverType, "should belong to AntiPatternClass")
}

func TestParseFile_NonPythonFile(t *testing.T) {
	p := NewParser(1)

	_, err := p.ParseFile("/some/path/file.go")
	assert.Error(t, err, "should error on non-Python file")
	assert.Contains(t, err.Error(), "not a Python file")
}

func TestParseFile_NonExistent(t *testing.T) {
	p := NewParser(1)

	_, err := p.ParseFile("/nonexistent/path/file.py")
	assert.Error(t, err, "should error on non-existent file")
}

func TestParseDirectory(t *testing.T) {
	p := NewParser(1)

	testDir := filepath.Join("..", "..", "..", "testdata", "python")
	absPath, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absPath)
	}

	reporter := status.NewSilentReporter()
	metrics, errs := p.ParseDirectory(absPath, nil, []string{".py"}, reporter)

	assert.Empty(t, errs, "should have no parse errors")
	assert.Equal(t, 3, len(metrics), "should parse 3 Python files")

	// Verify each file was parsed
	fileParsed := make(map[string]bool)
	for _, m := range metrics {
		fileParsed[m.PackageName] = true
	}

	assert.True(t, fileParsed["sample"], "should have parsed sample.py")
	assert.True(t, fileParsed["complex"], "should have parsed complex.py")
	assert.True(t, fileParsed["antipatterns"], "should have parsed antipatterns.py")
}

func TestParseDirectory_WithExcludes(t *testing.T) {
	p := NewParser(1)

	testDir := filepath.Join("..", "..", "..", "testdata", "python")
	absPath, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absPath)
	}

	reporter := status.NewSilentReporter()
	excludes := []string{"**/antipatterns.py"}
	metrics, errs := p.ParseDirectory(absPath, excludes, []string{".py"}, reporter)

	assert.Empty(t, errs, "should have no parse errors")
	assert.Equal(t, 2, len(metrics), "should parse 2 Python files (1 excluded)")
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		total   int
		code    int
		comment int
		blank   int
	}{
		{
			name:    "simple code",
			content: "x = 1\ny = 2\n",
			total:   3,
			code:    2,
			comment: 0,
			blank:   1,
		},
		{
			name:    "with comments",
			content: "# comment\nx = 1\n# another comment\n",
			total:   4,
			code:    1,
			comment: 2,
			blank:   1,
		},
		{
			name:    "with docstring",
			content: "\"\"\"docstring\"\"\"\nx = 1\n",
			total:   3,
			code:    1,
			comment: 1,
			blank:   1,
		},
		{
			name:    "multiline docstring",
			content: "\"\"\"line1\nline2\nline3\"\"\"\nx = 1\n",
			total:   5,
			code:    1,
			comment: 3,
			blank:   1,
		},
		{
			name:    "blank lines",
			content: "x = 1\n\n\ny = 2\n",
			total:   5,
			code:    2,
			comment: 0,
			blank:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, code, comment, blank := countLines([]byte(tt.content))
			assert.Equal(t, tt.total, total, "total lines")
			assert.Equal(t, tt.code, code, "code lines")
			assert.Equal(t, tt.comment, comment, "comment lines")
			assert.Equal(t, tt.blank, blank, "blank lines")
		})
	}
}

func TestComplexityCalculation(t *testing.T) {
	p := NewParser(1)

	// Create a temp file with known complexity
	content := `def test_function(x):
    if x > 0:
        for i in range(x):
            if i % 2 == 0:
                return i
    return 0
`
	tmpFile := filepath.Join(t.TempDir(), "test.py")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base complexity (1) + if (1) + for (1) + if (1) = 4
	assert.Equal(t, 4, metrics.Functions[0].Complexity, "complexity should be 4")
}

func TestParameterCounting(t *testing.T) {
	p := NewParser(1)

	// Test with various parameter styles
	content := `def simple(a, b, c):
    pass

def with_defaults(a, b=1, c=2):
    pass

def with_types(a: int, b: str, c: float = 1.0):
    pass

def with_args(*args, **kwargs):
    pass

class MyClass:
    def method(self, x, y):
        pass
`
	tmpFile := filepath.Join(t.TempDir(), "params.py")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)

	paramCounts := make(map[string]int)
	for _, fn := range metrics.Functions {
		paramCounts[fn.Name] = fn.Parameters
	}

	assert.Equal(t, 3, paramCounts["simple"], "simple should have 3 params")
	assert.Equal(t, 3, paramCounts["with_defaults"], "with_defaults should have 3 params")
	assert.Equal(t, 3, paramCounts["with_types"], "with_types should have 3 params")
	assert.Equal(t, 2, paramCounts["with_args"], "with_args should have 2 params (*args, **kwargs)")
	assert.Equal(t, 2, paramCounts["method"], "method should have 2 params (self not counted)")
}

// TestParseDirectory_ParallelVsSequential verifies that parallel parsing produces
// the same results as sequential parsing (order-independent).
func TestParseDirectory_ParallelVsSequential(t *testing.T) {
	testDir := filepath.Join("..", "..", "..", "testdata", "python")
	absDir, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absDir)
	}

	extensions := []string{".py"}

	// Parse sequentially (workers=1)
	seqParser := NewParser(1)
	seqMetrics, seqErrors := seqParser.ParseDirectory(absDir, []string{}, extensions, status.NewSilentReporter())

	// Parse in parallel (workers=4)
	parParser := NewParser(4)
	parMetrics, parErrors := parParser.ParseDirectory(absDir, []string{}, extensions, status.NewSilentReporter())

	// Should have same number of files
	assert.Equal(t, len(seqMetrics), len(parMetrics), "parallel and sequential should find same number of files")

	// Should have same number of errors
	assert.Equal(t, len(seqErrors), len(parErrors), "parallel and sequential should have same number of errors")

	// Build maps for order-independent comparison
	seqFileMap := make(map[string]*parser.FileMetrics)
	for _, m := range seqMetrics {
		seqFileMap[m.FilePath] = m
	}

	parFileMap := make(map[string]*parser.FileMetrics)
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
