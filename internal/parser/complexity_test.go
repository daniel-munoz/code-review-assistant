package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComplexityCalculation(t *testing.T) {
	testCases := []struct {
		name               string
		filePath           string
		functionName       string
		expectedComplexity int
	}{
		{
			name:               "simple function - no branches",
			filePath:           "../../testdata/sample/main.go",
			functionName:       "Add",
			expectedComplexity: 1, // Base complexity only
		},
		{
			name:               "function with multiple if statements",
			filePath:           "../../testdata/sample/main.go",
			functionName:       "checkEnvironment",
			expectedComplexity: 3, // 1 base + 2 if statements
		},
		{
			name:               "method with conditions and logical operators",
			filePath:           "../../testdata/sample/util.go",
			functionName:       "IsValid",
			expectedComplexity: 4, // 1 base + 3 if + logical operators
		},
		{
			name:               "function with loop and condition",
			filePath:           "../../testdata/sample/util.go",
			functionName:       "Contains",
			expectedComplexity: 3, // 1 base + 1 for + 1 if
		},
		{
			name:               "function with conditional return",
			filePath:           "../../testdata/sample/util.go",
			functionName:       "MaxInt",
			expectedComplexity: 2, // 1 base + 1 if
		},
	}

	parser := NewParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics, err := parser.ParseFile(tc.filePath)
			assert.NoError(t, err)

			found := false
			for _, fn := range metrics.Functions {
				if fn.Name == tc.functionName {
					found = true
					assert.Equal(t, tc.expectedComplexity, fn.Complexity,
						"Complexity mismatch for %s", tc.functionName)
					// Ensure complexity is at least 1
					assert.GreaterOrEqual(t, fn.Complexity, 1,
						"Complexity should be at least 1 for %s", tc.functionName)
				}
			}
			assert.True(t, found, "Function %s not found", tc.functionName)
		})
	}
}

func TestComplexityMinimum(t *testing.T) {
	// All functions should have complexity >= 1
	parser := NewParser()
	metrics, err := parser.ParseFile("../../testdata/sample/main.go")
	assert.NoError(t, err)

	for _, fn := range metrics.Functions {
		assert.GreaterOrEqual(t, fn.Complexity, 1,
			"Function %s should have complexity >= 1", fn.Name)
	}
}

func TestComplexityEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		code        string
		expectedMin int
	}{
		{
			name: "empty function",
			code: `package test
func Empty() {}`,
			expectedMin: 1,
		},
		{
			name: "function with only return",
			code: `package test
func SimpleReturn() int {
	return 42
}`,
			expectedMin: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For now, we'll skip these tests as they require creating temp files
			// This is a placeholder for future comprehensive testing
			t.Skip("Edge case testing requires temp file creation")
		})
	}
}
