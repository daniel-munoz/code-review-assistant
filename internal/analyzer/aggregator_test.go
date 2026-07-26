package analyzer

import (
	"testing"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

func TestCalculateMedian_OddCount(t *testing.T) {
	functions := []*parser.FunctionMetrics{
		{Name: "a", Lines: 5, Complexity: 1},
		{Name: "b", Lines: 10, Complexity: 3},
		{Name: "c", Lines: 25, Complexity: 8},
	}

	median := calculateMedian(functions, func(fn *parser.FunctionMetrics) int {
		return fn.Lines
	})

	if median != 10 {
		t.Errorf("expected median function length 10, got %d", median)
	}
}

func TestCalculateMedian_EvenCount(t *testing.T) {
	functions := []*parser.FunctionMetrics{
		{Name: "a", Lines: 4, Complexity: 1},
		{Name: "b", Lines: 8, Complexity: 2},
		{Name: "c", Lines: 12, Complexity: 5},
		{Name: "d", Lines: 30, Complexity: 9},
	}

	median := calculateMedian(functions, func(fn *parser.FunctionMetrics) int {
		return fn.Lines
	})

	if median != 10 {
		t.Errorf("expected median function length 10, got %d", median)
	}
}

func TestCalculateMedians_Empty(t *testing.T) {
	metrics := &AggregateMetrics{}
	calculateMedians(metrics, nil)

	if metrics.MedianFunctionLength != 0 || metrics.MedianComplexity != 0 {
		t.Errorf("expected zero medians for empty input, got length=%d complexity=%d",
			metrics.MedianFunctionLength, metrics.MedianComplexity)
	}
}
