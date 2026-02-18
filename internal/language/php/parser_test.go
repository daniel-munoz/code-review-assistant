package php

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
	assert.NotNil(t, p.language, "PHP language should be set")
}

func TestParseFile_SamplePHP(t *testing.T) {
	p := NewParser(1)

	testFile := filepath.Join("..", "..", "..", "testdata", "php", "sample.php")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")
	require.NotNil(t, metrics, "metrics should not be nil")

	// Verify basic metrics
	assert.Equal(t, absPath, metrics.FilePath)
	assert.Equal(t, "App\\Services", metrics.PackageName)
	assert.Equal(t, "php", metrics.Language)

	// Should have found functions and classes
	assert.Greater(t, len(metrics.Functions), 0, "should have functions")
	assert.Greater(t, metrics.TotalLines, 0, "should have lines")
	assert.Greater(t, metrics.CodeLines, 0, "should have code lines")

	// Check for expected functions
	functionNames := make(map[string]bool)
	for _, fn := range metrics.Functions {
		functionNames[fn.Name] = true
	}

	// Regular functions
	assert.True(t, functionNames["greet"], "should have greet function")
	assert.True(t, functionNames["calculateTotal"], "should have calculateTotal function")
	assert.True(t, functionNames["processData"], "should have processData function")
	assert.True(t, functionNames["main"], "should have main function")

	// Class methods
	assert.True(t, functionNames["__construct"], "should have __construct")
	assert.True(t, functionNames["addItem"], "should have addItem method")
	assert.True(t, functionNames["getItems"], "should have getItems method")
	assert.True(t, functionNames["clear"], "should have clear method")
	assert.True(t, functionNames["getLog"], "should have getLog method")

	// Verify imports (use statements)
	assert.Greater(t, len(metrics.Imports), 0, "should have imports")
}

func TestParseFile_ComplexPHP(t *testing.T) {
	p := NewParser(1)

	testFile := filepath.Join("..", "..", "..", "testdata", "php", "complex.php")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")

	// Find the highlyComplexFunction
	var complexFunc *struct {
		Name       string
		Complexity int
		Parameters int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "highlyComplexFunction" {
			complexFunc = &struct {
				Name       string
				Complexity int
				Parameters int
			}{fn.Name, fn.Complexity, fn.Parameters}
			break
		}
	}

	require.NotNil(t, complexFunc, "should find highlyComplexFunction")
	assert.Greater(t, complexFunc.Complexity, 10, "highlyComplexFunction should have high complexity")
	assert.Equal(t, 6, complexFunc.Parameters, "should have 6 parameters")

	// Find deeplyNestedFunction
	var nestedFunc *struct {
		Name       string
		Complexity int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "deeplyNestedFunction" {
			nestedFunc = &struct {
				Name       string
				Complexity int
			}{fn.Name, fn.Complexity}
			break
		}
	}

	require.NotNil(t, nestedFunc, "should find deeplyNestedFunction")
	assert.Greater(t, nestedFunc.Complexity, 5, "deeplyNestedFunction should have elevated complexity")

	// Check that interface methods, trait methods, and enum methods are extracted
	functionNames := make(map[string]bool)
	for _, fn := range metrics.Functions {
		functionNames[fn.Name] = true
	}

	assert.True(t, functionNames["process"], "should have interface method process")
	assert.True(t, functionNames["validate"], "should have interface method validate")
	assert.True(t, functionNames["getCached"], "should have trait method getCached")
	assert.True(t, functionNames["setCached"], "should have trait method setCached")
	assert.True(t, functionNames["label"], "should have enum method label")
}

func TestParseFile_Antipatterns(t *testing.T) {
	p := NewParser(1)

	testFile := filepath.Join("..", "..", "..", "testdata", "php", "antipatterns.php")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")

	// Find functionWithTooManyParams
	var manyParamsFunc *struct {
		Name       string
		Parameters int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "functionWithTooManyParams" {
			manyParamsFunc = &struct {
				Name       string
				Parameters int
			}{fn.Name, fn.Parameters}
			break
		}
	}

	require.NotNil(t, manyParamsFunc, "should find functionWithTooManyParams")
	assert.Equal(t, 7, manyParamsFunc.Parameters, "should have 7 parameters")

	// Find AntiPatternClass constructor
	var classConstructor *struct {
		Name         string
		Parameters   int
		ReceiverType string
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "__construct" && fn.ReceiverType == "AntiPatternClass" {
			classConstructor = &struct {
				Name         string
				Parameters   int
				ReceiverType string
			}{fn.Name, fn.Parameters, fn.ReceiverType}
			break
		}
	}

	require.NotNil(t, classConstructor, "should find AntiPatternClass constructor")
	assert.Equal(t, 7, classConstructor.Parameters, "class constructor should have 7 parameters")
	assert.Equal(t, "AntiPatternClass", classConstructor.ReceiverType, "should belong to AntiPatternClass")
}

func TestParseFile_NonPHPFile(t *testing.T) {
	p := NewParser(1)

	_, err := p.ParseFile("/some/path/file.go")
	assert.Error(t, err, "should error on non-PHP file")
	assert.Contains(t, err.Error(), "not a PHP file")
}

func TestParseFile_NonExistent(t *testing.T) {
	p := NewParser(1)

	_, err := p.ParseFile("/nonexistent/path/file.php")
	assert.Error(t, err, "should error on non-existent file")
}

func TestParseDirectory(t *testing.T) {
	p := NewParser(1)

	testDir := filepath.Join("..", "..", "..", "testdata", "php")
	absPath, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absPath)
	}

	reporter := status.NewSilentReporter()
	metrics, errs := p.ParseDirectory(absPath, nil, []string{".php"}, reporter)

	assert.Empty(t, errs, "should have no parse errors")
	assert.Equal(t, 3, len(metrics), "should parse 3 PHP files")
}

func TestParseDirectory_WithExcludes(t *testing.T) {
	p := NewParser(1)

	testDir := filepath.Join("..", "..", "..", "testdata", "php")
	absPath, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absPath)
	}

	reporter := status.NewSilentReporter()
	excludes := []string{"**/antipatterns.php"}
	metrics, errs := p.ParseDirectory(absPath, excludes, []string{".php"}, reporter)

	assert.Empty(t, errs, "should have no parse errors")
	assert.Equal(t, 2, len(metrics), "should parse 2 files (1 excluded)")
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
			content: "$x = 1;\n$y = 2;\n",
			total:   3,
			code:    2,
			comment: 0,
			blank:   1,
		},
		{
			name:    "with single line comments",
			content: "// comment\n$x = 1;\n// another comment\n",
			total:   4,
			code:    1,
			comment: 2,
			blank:   1,
		},
		{
			name:    "with hash comments",
			content: "# comment\n$x = 1;\n# another comment\n",
			total:   4,
			code:    1,
			comment: 2,
			blank:   1,
		},
		{
			name:    "with multiline comment",
			content: "/* multiline\ncomment */\n$x = 1;\n",
			total:   4,
			code:    1,
			comment: 2,
			blank:   1,
		},
		{
			name:    "with docblock",
			content: "/**\n * Docblock\n */\nfunction foo() {}\n",
			total:   5,
			code:    1,
			comment: 3,
			blank:   1,
		},
		{
			name:    "blank lines",
			content: "$x = 1;\n\n\n$y = 2;\n",
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

	content := `<?php
function testFunction(int $x): int {
    if ($x > 0) {
        for ($i = 0; $i < $x; $i++) {
            if ($i % 2 === 0) {
                return $i;
            }
        }
    }
    return 0;
}
`
	tmpFile := filepath.Join(t.TempDir(), "test.php")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base complexity (1) + if (1) + for (1) + if (1) = 4
	assert.Equal(t, 4, metrics.Functions[0].Complexity, "complexity should be 4")
}

func TestComplexityWithLogicalOperators(t *testing.T) {
	p := NewParser(1)

	content := `<?php
function testLogicalOps(bool $a, bool $b, bool $c): bool {
    if ($a && $b) {
        return true;
    }
    if ($a || $b || $c) {
        return true;
    }
    return $a && $b && $c;
}
`
	tmpFile := filepath.Join(t.TempDir(), "logical.php")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base (1) + if (1) + && (1) + if (1) + || (1) + || (1) + && (1) + && (1) = 8
	assert.Equal(t, 8, metrics.Functions[0].Complexity, "complexity should include logical operators")
}

func TestComplexityWithForeach(t *testing.T) {
	p := NewParser(1)

	content := `<?php
function testForeach(array $items): void {
    foreach ($items as $item) {
        echo $item;
    }
    foreach ($items as $key => $value) {
        echo "$key: $value";
    }
}
`
	tmpFile := filepath.Join(t.TempDir(), "foreach.php")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base (1) + foreach (1) + foreach (1) = 3
	assert.Equal(t, 3, metrics.Functions[0].Complexity, "complexity should include foreach loops")
}

func TestParameterCounting(t *testing.T) {
	p := NewParser(1)

	content := `<?php
function simple(int $a, string $b, bool $c): void {}

function withDefaults(int $a, string $b = 'default', bool $c = true): void {}

function withVariadic(int $a, string ...$rest): void {}

class MyClass {
    public function method(int $a, string $b): void {}

    public function __construct(
        private string $name,
        private int $value = 0
    ) {}
}
`
	tmpFile := filepath.Join(t.TempDir(), "params.php")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)

	paramCounts := make(map[string]int)
	for _, fn := range metrics.Functions {
		paramCounts[fn.Name] = fn.Parameters
	}

	assert.Equal(t, 3, paramCounts["simple"], "simple should have 3 params")
	assert.Equal(t, 3, paramCounts["withDefaults"], "withDefaults should have 3 params")
	assert.Equal(t, 2, paramCounts["withVariadic"], "withVariadic should have 2 params (a + rest)")
	assert.Equal(t, 2, paramCounts["method"], "method should have 2 params")
	assert.Equal(t, 2, paramCounts["__construct"], "constructor should have 2 promoted params")
}

func TestExtensions(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"/path/to/file.php", true},
		{"/path/to/file.PHP", true},
		{"/path/to/file.go", false},
		{"/path/to/file.py", false},
		{"/path/to/file.txt", false},
		{"/path/to/file.phtml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidExtension(tt.path))
		})
	}
}

func TestNamespaceExtraction(t *testing.T) {
	p := NewParser(1)

	content := `<?php

namespace App\Services\Payment;

use App\Models\Order;

function processPayment(): void {}
`
	tmpFile := filepath.Join(t.TempDir(), "namespaced.php")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "App\\Services\\Payment", metrics.PackageName, "should extract namespace")
}

func TestImportExtraction(t *testing.T) {
	p := NewParser(1)

	content := `<?php

namespace App\Services;

use App\Models\User;
use App\Contracts\UserRepositoryInterface;
use RuntimeException;

function dummy(): void {}
`
	tmpFile := filepath.Join(t.TempDir(), "imports.php")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(metrics.Imports), 3, "should have at least 3 imports")
}

// TestParseDirectory_ParallelVsSequential verifies that parallel parsing produces
// the same results as sequential parsing (order-independent).
func TestParseDirectory_ParallelVsSequential(t *testing.T) {
	testDir := filepath.Join("..", "..", "..", "testdata", "php")
	absDir, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absDir)
	}

	extensions := []string{".php"}

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
