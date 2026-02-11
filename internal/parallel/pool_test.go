package parallel

import (
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool_DefaultWorkers(t *testing.T) {
	pool := NewWorkerPool(0, func(x int) int { return x * 2 })
	if pool.Workers() != runtime.NumCPU() {
		t.Errorf("expected %d workers for workers=0, got %d", runtime.NumCPU(), pool.Workers())
	}
}

func TestNewWorkerPool_ExplicitWorkers(t *testing.T) {
	pool := NewWorkerPool(4, func(x int) int { return x * 2 })
	if pool.Workers() != 4 {
		t.Errorf("expected 4 workers, got %d", pool.Workers())
	}
}

func TestWorkerPool_BasicProcessing(t *testing.T) {
	pool := NewWorkerPool(2, func(x int) int { return x * 2 })
	pool.Start()

	// Submit items
	for i := 1; i <= 5; i++ {
		pool.Submit(i)
	}
	pool.Close()

	// Collect results
	results := make([]int, 0, 5)
	for result := range pool.Results() {
		results = append(results, result)
	}

	// Sort for comparison (order is not guaranteed)
	sort.Ints(results)

	expected := []int{2, 4, 6, 8, 10}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("result[%d] = %d, expected %d", i, v, expected[i])
		}
	}
}

func TestWorkerPool_SequentialMode(t *testing.T) {
	// workers=1 should process items sequentially
	pool := NewWorkerPool(1, func(x int) int { return x * 2 })
	pool.Start()

	for i := 1; i <= 3; i++ {
		pool.Submit(i)
	}
	pool.Close()

	results := make([]int, 0, 3)
	for result := range pool.Results() {
		results = append(results, result)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestWorkerPool_ConcurrentResults(t *testing.T) {
	// Test that multiple workers process items concurrently
	var concurrentCount int64
	var maxConcurrent int64

	pool := NewWorkerPool(4, func(x int) int {
		current := atomic.AddInt64(&concurrentCount, 1)

		// Track max concurrent
		for {
			max := atomic.LoadInt64(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt64(&maxConcurrent, max, current) {
				break
			}
		}

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		atomic.AddInt64(&concurrentCount, -1)
		return x * 2
	})

	pool.Start()

	// Submit enough items to saturate workers
	for i := 0; i < 20; i++ {
		pool.Submit(i)
	}
	pool.Close()

	// Drain results
	for range pool.Results() {
	}

	// With 4 workers and blocking tasks, we should see >1 concurrent executions
	if atomic.LoadInt64(&maxConcurrent) <= 1 {
		t.Errorf("expected concurrent execution, max concurrent was %d", maxConcurrent)
	}
}

func TestWorkerPool_EmptyInput(t *testing.T) {
	pool := NewWorkerPool(2, func(x int) int { return x })
	pool.Start()
	pool.Close()

	count := 0
	for range pool.Results() {
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 results for empty input, got %d", count)
	}
}

func TestProcessAll_Basic(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	results := ProcessAll(2, items, func(x int) int { return x * 2 })

	// Sort for comparison
	sort.Ints(results)

	expected := []int{2, 4, 6, 8, 10}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("result[%d] = %d, expected %d", i, v, expected[i])
		}
	}
}

func TestProcessAll_Empty(t *testing.T) {
	results := ProcessAll(2, []int{}, func(x int) int { return x * 2 })
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
}

func TestProcessAll_SingleWorker(t *testing.T) {
	items := []int{1, 2, 3}
	results := ProcessAll(1, items, func(x int) int { return x + 1 })

	sort.Ints(results)
	expected := []int{2, 3, 4}

	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("result[%d] = %d, expected %d", i, v, expected[i])
		}
	}
}

func TestWorkerPool_SubmitAfterClose_Panics(t *testing.T) {
	pool := NewWorkerPool(2, func(x int) int { return x })
	pool.Start()
	pool.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when submitting to closed pool")
		}
	}()

	pool.Submit(1)
}

func TestWorkerPool_DoubleClose_Safe(t *testing.T) {
	pool := NewWorkerPool(2, func(x int) int { return x })
	pool.Start()

	// Double close should not panic
	pool.Close()
	pool.Close()

	// Drain results
	for range pool.Results() {
	}
}

func TestWorkerPool_CloseWithoutStart(t *testing.T) {
	pool := NewWorkerPool(2, func(x int) int { return x })

	// Close without Start should not block and should close resultCh
	pool.Close()

	// Ranging over Results should complete immediately (channel closed)
	count := 0
	for range pool.Results() {
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 results, got %d", count)
	}
}

// TestWorkerPool_Race tests for data races using concurrent operations.
// Run with: go test -race ./internal/parallel/
func TestWorkerPool_Race(t *testing.T) {
	pool := NewWorkerPool(4, func(x int) int {
		return x * 2
	})
	pool.Start()

	// Start collecting results in a separate goroutine to prevent deadlock
	resultCount := make(chan int)
	go func() {
		count := 0
		for range pool.Results() {
			count++
		}
		resultCount <- count
	}()

	// Concurrent submits
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			pool.Submit(i)
		}
		done <- true
	}()

	go func() {
		for i := 100; i < 200; i++ {
			pool.Submit(i)
		}
		done <- true
	}()

	<-done
	<-done
	pool.Close()

	// Wait for all results
	count := <-resultCount

	if count != 200 {
		t.Errorf("expected 200 results, got %d", count)
	}
}

// BenchmarkWorkerPool measures pool performance with different worker counts.
func BenchmarkWorkerPool(b *testing.B) {
	for _, workers := range []int{1, 2, 4, 8, 0} {
		name := "workers="
		if workers == 0 {
			name += "auto"
		} else {
			name += strconv.Itoa(workers)
		}

		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				items := make([]int, 1000)
				for j := range items {
					items[j] = j
				}
				ProcessAll(workers, items, func(x int) int {
					// Simulate light work
					return x * x
				})
			}
		})
	}
}
