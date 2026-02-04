package javascript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

func TestNewParser(t *testing.T) {
	p := NewParser()
	require.NotNil(t, p, "parser should not be nil")
	assert.NotNil(t, p.tsLanguage, "TypeScript language should be set")
	assert.NotNil(t, p.tsxLanguage, "TSX language should be set")
}

func TestParseFile_SampleTS(t *testing.T) {
	p := NewParser()

	testFile := filepath.Join("..", "..", "..", "testdata", "javascript", "sample.ts")
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
	assert.Equal(t, "sample", metrics.PackageName)
	assert.Equal(t, "javascript", metrics.Language)

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
	assert.True(t, functionNames["fetchUser"], "should have fetchUser async function")
	assert.True(t, functionNames["numberGenerator"], "should have numberGenerator generator function")
	assert.True(t, functionNames["main"], "should have main function")

	// Arrow functions
	assert.True(t, functionNames["processData"], "should have processData arrow function")
	assert.True(t, functionNames["formatCurrency"], "should have formatCurrency function expression")

	// Class methods
	assert.True(t, functionNames["constructor"], "should have constructor")
	assert.True(t, functionNames["addItem"], "should have addItem method")
	assert.True(t, functionNames["getItems"], "should have getItems method")
	assert.True(t, functionNames["getName"], "should have getName arrow function field")

	// Verify imports
	assert.Greater(t, len(metrics.Imports), 0, "should have imports")
}

func TestParseFile_ComplexTS(t *testing.T) {
	p := NewParser()

	testFile := filepath.Join("..", "..", "..", "testdata", "javascript", "complex.ts")
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
}

func TestParseFile_Antipatterns(t *testing.T) {
	p := NewParser()

	testFile := filepath.Join("..", "..", "..", "testdata", "javascript", "antipatterns.ts")
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
		if fn.Name == "constructor" && fn.ReceiverType == "AntiPatternClass" {
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

	// Find arrowWithTooManyParams
	var arrowFunc *struct {
		Name       string
		Parameters int
	}
	for _, fn := range metrics.Functions {
		if fn.Name == "arrowWithTooManyParams" {
			arrowFunc = &struct {
				Name       string
				Parameters int
			}{fn.Name, fn.Parameters}
			break
		}
	}

	require.NotNil(t, arrowFunc, "should find arrowWithTooManyParams")
	assert.Equal(t, 6, arrowFunc.Parameters, "arrow function should have 6 parameters")
}

func TestParseFile_JSX(t *testing.T) {
	p := NewParser()

	testFile := filepath.Join("..", "..", "..", "testdata", "javascript", "sample.jsx")
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data file not found:", absPath)
	}

	metrics, err := p.ParseFile(absPath)
	require.NoError(t, err, "should parse JSX file without error")
	require.NotNil(t, metrics, "metrics should not be nil")

	// Check for expected React components
	functionNames := make(map[string]bool)
	for _, fn := range metrics.Functions {
		functionNames[fn.Name] = true
	}

	assert.True(t, functionNames["Greeting"], "should have Greeting component")
	assert.True(t, functionNames["Counter"], "should have Counter component")
	assert.True(t, functionNames["UserCard"], "should have UserCard arrow component")
	assert.True(t, functionNames["DataFetcher"], "should have DataFetcher component")
	assert.True(t, functionNames["useLocalStorage"], "should have useLocalStorage custom hook")
}

func TestParseFile_NonJSFile(t *testing.T) {
	p := NewParser()

	_, err := p.ParseFile("/some/path/file.go")
	assert.Error(t, err, "should error on non-JS/TS file")
	assert.Contains(t, err.Error(), "not a JavaScript/TypeScript file")
}

func TestParseFile_NonExistent(t *testing.T) {
	p := NewParser()

	_, err := p.ParseFile("/nonexistent/path/file.ts")
	assert.Error(t, err, "should error on non-existent file")
}

func TestParseDirectory(t *testing.T) {
	p := NewParser()

	testDir := filepath.Join("..", "..", "..", "testdata", "javascript")
	absPath, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absPath)
	}

	reporter := status.NewSilentReporter()
	metrics, errs := p.ParseDirectory(absPath, nil, []string{".ts", ".tsx", ".js", ".jsx"}, reporter)

	assert.Empty(t, errs, "should have no parse errors")
	assert.Equal(t, 4, len(metrics), "should parse 4 JavaScript/TypeScript files")

	// Verify each file was parsed
	fileParsed := make(map[string]bool)
	for _, m := range metrics {
		fileParsed[m.PackageName] = true
	}

	assert.True(t, fileParsed["sample"], "should have parsed sample.ts")
	assert.True(t, fileParsed["complex"], "should have parsed complex.ts")
	assert.True(t, fileParsed["antipatterns"], "should have parsed antipatterns.ts")
}

func TestParseDirectory_WithExcludes(t *testing.T) {
	p := NewParser()

	testDir := filepath.Join("..", "..", "..", "testdata", "javascript")
	absPath, err := filepath.Abs(testDir)
	require.NoError(t, err)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("test data directory not found:", absPath)
	}

	reporter := status.NewSilentReporter()
	excludes := []string{"**/antipatterns.ts"}
	metrics, errs := p.ParseDirectory(absPath, excludes, []string{".ts", ".tsx", ".js", ".jsx"}, reporter)

	assert.Empty(t, errs, "should have no parse errors")
	assert.Equal(t, 3, len(metrics), "should parse 3 files (1 excluded)")
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
			content: "const x = 1;\nconst y = 2;\n",
			total:   3,
			code:    2,
			comment: 0,
			blank:   1,
		},
		{
			name:    "with single line comments",
			content: "// comment\nconst x = 1;\n// another comment\n",
			total:   4,
			code:    1,
			comment: 2,
			blank:   1,
		},
		{
			name:    "with multiline comment",
			content: "/* multiline\ncomment */\nconst x = 1;\n",
			total:   4,
			code:    1,
			comment: 2,
			blank:   1,
		},
		{
			name:    "blank lines",
			content: "const x = 1;\n\n\nconst y = 2;\n",
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
	p := NewParser()

	// Create a temp file with known complexity
	content := `function testFunction(x: number): number {
    if (x > 0) {
        for (let i = 0; i < x; i++) {
            if (i % 2 === 0) {
                return i;
            }
        }
    }
    return 0;
}
`
	tmpFile := filepath.Join(t.TempDir(), "test.ts")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base complexity (1) + if (1) + for (1) + if (1) = 4
	assert.Equal(t, 4, metrics.Functions[0].Complexity, "complexity should be 4")
}

func TestComplexityWithLogicalOperators(t *testing.T) {
	p := NewParser()

	// Test that && and || operators contribute to complexity
	content := `function testLogicalOps(a: boolean, b: boolean, c: boolean): boolean {
    if (a && b) {
        return true;
    }
    if (a || b || c) {
        return true;
    }
    return a && b && c;
}
`
	tmpFile := filepath.Join(t.TempDir(), "logical.ts")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base (1) + if (1) + && (1) + if (1) + || (1) + || (1) + && (1) + && (1) = 8
	assert.Equal(t, 8, metrics.Functions[0].Complexity, "complexity should include logical operators")
}

func TestComplexityWithForOfLoop(t *testing.T) {
	p := NewParser()

	// Test that for...of loops contribute to complexity
	content := `function testForOf(items: string[]): void {
    for (const item of items) {
        console.log(item);
    }
    for (const [key, value] of Object.entries({ a: 1 })) {
        console.log(key, value);
    }
}
`
	tmpFile := filepath.Join(t.TempDir(), "forof.ts")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	metrics, err := p.ParseFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, metrics.Functions, 1)

	// Base (1) + for-of (1) + for-of (1) = 3
	assert.Equal(t, 3, metrics.Functions[0].Complexity, "complexity should include for...of loops")
}

func TestParameterCounting(t *testing.T) {
	p := NewParser()

	content := `function simple(a: number, b: string, c: boolean): void {}

function withDefaults(a: number, b: string = 'default', c: boolean = true): void {}

function withRest(a: number, ...rest: number[]): void {}

function withDestructuring({ x, y }: { x: number; y: number }, [a, b]: number[]): void {}

const arrowSimple = (x: number, y: number): number => x + y;

const arrowSingleParam = x => x + 1;

class MyClass {
    method(a: number, b: string): void {}
}
`
	tmpFile := filepath.Join(t.TempDir(), "params.ts")
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
	assert.Equal(t, 2, paramCounts["withRest"], "withRest should have 2 params (a + rest)")
	assert.Equal(t, 2, paramCounts["withDestructuring"], "withDestructuring should have 2 params")
	assert.Equal(t, 2, paramCounts["arrowSimple"], "arrowSimple should have 2 params")
	assert.Equal(t, 1, paramCounts["arrowSingleParam"], "arrowSingleParam should have 1 param")
	assert.Equal(t, 2, paramCounts["method"], "method should have 2 params")
}

func TestExtensions(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"/path/to/file.ts", true},
		{"/path/to/file.tsx", true},
		{"/path/to/file.js", true},
		{"/path/to/file.jsx", true},
		{"/path/to/file.mjs", true},
		{"/path/to/file.cjs", true},
		{"/path/to/file.go", false},
		{"/path/to/file.py", false},
		{"/path/to/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidExtension(tt.path))
		})
	}
}

func TestMatchDoubleStarPattern(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		// Basic patterns
		{"vendor all", "vendor/pkg/file.go", "vendor/**", true},
		{"vendor nested", "vendor/a/b/c/file.go", "vendor/**", true},
		{"not vendor", "src/vendor/file.go", "vendor/**", false},

		// node_modules pattern (the bug we're fixing)
		{"node_modules root", "node_modules/react/index.js", "**/node_modules/**", true},
		{"node_modules nested", "src/node_modules/lodash/index.js", "**/node_modules/**", true},
		{"node_modules deep", "a/b/c/node_modules/pkg/file.js", "**/node_modules/**", true},
		{"not node_modules", "src/modules/file.js", "**/node_modules/**", false},
		{"node_modules exact", "node_modules", "**/node_modules/**", true},

		// Other common patterns
		{"pycache", "__pycache__/file.pyc", "**/__pycache__/**", true},
		{"pycache nested", "src/__pycache__/module.pyc", "**/__pycache__/**", true},
		{"dist folder", "dist/bundle.js", "**/dist/**", true},
		{"build folder", "build/output.js", "**/build/**", true},

		// File extension patterns
		{"test file", "src/utils.test.ts", "**/*.test.ts", true},
		{"spec file", "src/utils.spec.js", "**/*.spec.js", true},
		{"not test", "src/utils.ts", "**/*.test.ts", false},

		// testdata pattern
		{"testdata", "testdata/sample.json", "**/testdata/**", true},
		{"testdata nested", "pkg/testdata/fixtures/data.json", "**/testdata/**", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.path, tt.pattern)
			assert.Equal(t, tt.want, got, "matchPattern(%q, %q)", tt.path, tt.pattern)
		})
	}
}
