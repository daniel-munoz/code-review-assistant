package javascript

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

func TestNewDependencyAnalyzer(t *testing.T) {
	// Test with a directory that doesn't have package.json
	analyzer, err := NewDependencyAnalyzer("/nonexistent")
	require.NoError(t, err, "should not error even without package.json")
	assert.NotNil(t, analyzer, "analyzer should not be nil")
	assert.Empty(t, analyzer.projectName, "project name should be empty without package.json")
}

func TestCategorizeImport(t *testing.T) {
	analyzer := &DependencyAnalyzer{
		projectName: "my-app",
		projectPath: "/project",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Relative imports are internal
		{"relative current dir", "./utils", "internal"},
		{"relative parent dir", "../components", "internal"},
		{"deep relative", "../../lib/helpers", "internal"},

		// Node.js built-ins are stdlib
		{"fs", "fs", "stdlib"},
		{"path", "path", "stdlib"},
		{"http", "http", "stdlib"},
		{"node:fs", "node:fs", "stdlib"},
		{"node:path", "node:path", "stdlib"},
		{"fs/promises", "fs/promises", "stdlib"},
		{"stream", "stream", "stdlib"},
		{"crypto", "crypto", "stdlib"},
		{"util", "util", "stdlib"},
		{"events", "events", "stdlib"},
		{"child_process", "child_process", "stdlib"},

		// npm packages are external
		{"react", "react", "external"},
		{"lodash", "lodash", "external"},
		{"express", "express", "external"},
		{"@types/node", "@types/node", "external"},
		{"@babel/core", "@babel/core", "external"},
		{"lodash/debounce", "lodash/debounce", "external"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.categorizeImport(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNodeBuiltin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"fs", "fs", true},
		{"path", "path", true},
		{"http", "http", true},
		{"https", "https", true},
		{"crypto", "crypto", true},
		{"node:fs", "node:fs", true},
		{"node:path", "node:path", true},
		{"fs/promises", "fs/promises", true},
		{"stream/web", "stream/web", true},
		{"react", "react", false},
		{"lodash", "lodash", false},
		{"express", "express", false},
		{"@types/node", "@types/node", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNodeBuiltin(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyze(t *testing.T) {
	analyzer := &DependencyAnalyzer{
		projectName: "test-project",
		projectPath: "/project",
	}

	files := []*parser.FileMetrics{
		{
			FilePath:    "/project/src/index.ts",
			PackageName: "index",
			Imports:     []string{"fs", "path", "./utils", "../lib", "react", "lodash"},
		},
		{
			FilePath:    "/project/src/utils.ts",
			PackageName: "utils",
			Imports:     []string{"crypto", "./helpers", "express"},
		},
	}

	results, err := analyzer.Analyze(files)
	require.NoError(t, err)
	require.Len(t, results, 1, "files in same directory should be grouped")

	// Find the src package
	var srcPkg *struct {
		Stdlib   []string
		Internal []string
		External []string
	}
	for _, r := range results {
		if r.PackageName == "src" {
			srcPkg = &struct {
				Stdlib   []string
				Internal []string
				External []string
			}{r.StdlibImports, r.InternalImports, r.ExternalImports}
			break
		}
	}

	require.NotNil(t, srcPkg, "should have src package")
	assert.ElementsMatch(t, []string{"fs", "path", "crypto"}, srcPkg.Stdlib, "should have Node.js stdlib imports")
	assert.ElementsMatch(t, []string{"./utils", "../lib", "./helpers"}, srcPkg.Internal, "should have internal imports")
	assert.ElementsMatch(t, []string{"react", "lodash", "express"}, srcPkg.External, "should have external imports")
}

func TestCycleDetection(t *testing.T) {
	detector := newJSCycleDetector(map[string][]string{
		"a.ts": {"b.ts"},
		"b.ts": {"c.ts"},
		"c.ts": {"a.ts"}, // Creates cycle: a -> b -> c -> a
	})

	cycles := detector.findCycles()
	require.Len(t, cycles, 1, "should detect one cycle")

	// The cycle should contain a, b, c
	assert.Len(t, cycles[0].Cycle, 4, "cycle should have 4 elements (including closing)")
}

func TestNoCycle(t *testing.T) {
	detector := newJSCycleDetector(map[string][]string{
		"a.ts": {"b.ts"},
		"b.ts": {"c.ts"},
		"c.ts": {}, // No cycle
	})

	cycles := detector.findCycles()
	assert.Empty(t, cycles, "should not detect any cycles")
}

func TestNormalizeCycle(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "already normalized",
			input:    []string{"a", "b", "c", "a"},
			expected: []string{"a", "b", "c", "a"},
		},
		{
			name:     "needs rotation",
			input:    []string{"c", "a", "b", "c"},
			expected: []string{"a", "b", "c", "a"}, // Rotates to start with 'a', closes with 'a'
		},
		{
			name:     "empty",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCycle(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAreCyclesEquivalent(t *testing.T) {
	tests := []struct {
		name     string
		cycle1   []string
		cycle2   []string
		expected bool
	}{
		{
			name:     "identical",
			cycle1:   []string{"a", "b", "c", "a"},
			cycle2:   []string{"a", "b", "c", "a"},
			expected: true,
		},
		{
			name:     "different",
			cycle1:   []string{"a", "b", "c", "a"},
			cycle2:   []string{"a", "b", "d", "a"},
			expected: false,
		},
		{
			name:     "different lengths",
			cycle1:   []string{"a", "b", "a"},
			cycle2:   []string{"a", "b", "c", "a"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areCyclesEquivalent(tt.cycle1, tt.cycle2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
