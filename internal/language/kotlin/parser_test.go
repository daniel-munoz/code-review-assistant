package kotlin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// fixturePath returns the absolute path to a testdata/kotlin fixture,
// skipping the test if it does not exist.
func fixturePath(t *testing.T, name string) string {
	t.Helper()
	testFile := filepath.Join("..", "..", "..", "testdata", "kotlin", name)
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}
	return absPath
}

// findFunction returns the named function's metrics, or nil.
func findFunction(metrics *parser.FileMetrics, name string) *parser.FunctionMetrics {
	for _, fn := range metrics.Functions {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}

func TestNewParser(t *testing.T) {
	p := NewParser(1)
	require.NotNil(t, p, "parser should not be nil")
	assert.NotNil(t, p.language, "language should be set")
}

func TestParseFile_RejectsNonKotlinFile(t *testing.T) {
	p := NewParser(1)
	_, err := p.ParseFile("main.go")
	assert.Error(t, err)
}

func TestParseFile_SampleKotlin(t *testing.T) {
	p := NewParser(1)
	absPath := fixturePath(t, "sample.kt")

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse without error")
	require.NotNil(t, metrics)

	assert.Equal(t, absPath, metrics.FilePath)
	assert.Equal(t, "com.example.sample", metrics.PackageName)
	assert.Equal(t, "kotlin", metrics.Language)

	assert.Greater(t, metrics.TotalLines, 0)
	assert.Greater(t, metrics.CodeLines, 0)
	assert.Greater(t, metrics.CommentLines, 0, "KDoc and line comments should count")

	// Top-level functions
	require.NotNil(t, findFunction(metrics, "greet"), "should have greet function")
	require.NotNil(t, findFunction(metrics, "main"), "should have main function")
	assert.Equal(t, "", findFunction(metrics, "greet").ReceiverType)

	// Class methods carry the enclosing type name
	add := findFunction(metrics, "add")
	require.NotNil(t, add, "should have add method")
	assert.Equal(t, "Calculator", add.ReceiverType)
	assert.Equal(t, 2, add.Parameters)

	// Object members and interface methods
	logFn := findFunction(metrics, "log")
	require.NotNil(t, logFn, "should have log method from object")
	assert.Equal(t, "Logger", logFn.ReceiverType)
	save := findFunction(metrics, "save")
	require.NotNil(t, save, "should have save from interface")
	assert.Equal(t, "Repository", save.ReceiverType)
	assert.Equal(t, 1, save.Complexity, "bodyless interface method has base complexity")

	// Enum class methods are collected too
	describe := findFunction(metrics, "describe")
	require.NotNil(t, describe, "should have describe method from enum class")
	assert.Equal(t, "Status", describe.ReceiverType)

	// Imports
	assert.Contains(t, metrics.Imports, "java.time.Instant")
	assert.Contains(t, metrics.Imports, "kotlin.math.max")
}

func TestParseFile_ComplexKotlin(t *testing.T) {
	p := NewParser(1)
	absPath := fixturePath(t, "complex.kt")

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err)

	complexFn := findFunction(metrics, "highlyComplexFunction")
	require.NotNil(t, complexFn)
	assert.Equal(t, 6, complexFn.Parameters)
	assert.Greater(t, complexFn.Complexity, 10,
		"if/&&/||/elvis/when-entries/for/while/catch should each add complexity")

	nested := findFunction(metrics, "deeplyNestedFunction")
	require.NotNil(t, nested)
	assert.Greater(t, nested.Complexity, 5)
}

func TestParseDirectory_Kotlin(t *testing.T) {
	p := NewParser(1)
	dir := filepath.Dir(fixturePath(t, "sample.kt"))

	metrics, errs := p.ParseDirectory(dir, nil, []string{".kt"}, status.NewSilentReporter())
	assert.Empty(t, errs)
	assert.Len(t, metrics, 7, "sample.kt, complex.kt, antipatterns.kt, plus 4 dependency fixtures under deps/")
}
