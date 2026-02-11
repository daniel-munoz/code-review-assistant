// Package parallel provides utilities for parallel processing with worker pools.
//
// This package implements a generic worker pool pattern that can be used to
// parallelize CPU-bound tasks like file parsing and analysis. The pool manages
// a configurable number of worker goroutines that process work items concurrently.
package parallel

import (
	"runtime"
	"sync"
)

// WorkerPool manages concurrent task execution with a fixed number of workers.
//
// Type parameters:
//   - T: the type of work items to process
//   - R: the type of results produced
//
// The pool uses channels for communication:
//   - Work items are submitted via Submit()
//   - Results are collected via the Results() channel
//
// Example usage:
//
//	pool := NewWorkerPool(4, func(path string) *FileMetrics {
//	    return parseFile(path)
//	})
//	pool.Start()
//	for _, path := range paths {
//	    pool.Submit(path)
//	}
//	pool.Close()
//	for result := range pool.Results() {
//	    // process result
//	}
type WorkerPool[T any, R any] struct {
	workers   int
	workCh    chan T
	resultCh  chan R
	wg        sync.WaitGroup
	processor func(T) R
	closed    bool
	mu        sync.Mutex
}

// NewWorkerPool creates a new worker pool.
//
// Parameters:
//   - workers: number of worker goroutines (0 = runtime.NumCPU())
//   - processor: function that processes a work item and returns a result
//
// The pool is not started until Start() is called.
func NewWorkerPool[T any, R any](workers int, processor func(T) R) *WorkerPool[T, R] {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &WorkerPool[T, R]{
		workers:   workers,
		workCh:    make(chan T, workers*2), // Buffered channel for work items
		resultCh:  make(chan R, workers*2), // Buffered channel for results
		processor: processor,
	}
}

// Start spawns worker goroutines that begin processing work items.
//
// Workers will continue processing until Close() is called and all
// work items have been processed. Each worker reads from the work
// channel, processes items using the processor function, and writes
// results to the result channel.
func (p *WorkerPool[T, R]) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	// Start a goroutine to close resultCh when all workers are done
	go func() {
		p.wg.Wait()
		close(p.resultCh)
	}()
}

// worker is the main worker goroutine loop.
func (p *WorkerPool[T, R]) worker() {
	defer p.wg.Done()
	for item := range p.workCh {
		result := p.processor(item)
		p.resultCh <- result
	}
}

// Submit adds a work item to the pool for processing.
//
// This method blocks if the work channel buffer is full.
// Panics if called after Close().
func (p *WorkerPool[T, R]) Submit(item T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		panic("Submit called on closed pool")
	}
	p.workCh <- item
}

// Close signals that no more work items will be submitted.
//
// This method closes the work channel, causing workers to finish
// processing remaining items and then exit. The result channel
// will be closed after all workers have finished.
//
// Close should only be called once. It is safe to call from any goroutine.
func (p *WorkerPool[T, R]) Close() {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.workCh)
	}
	p.mu.Unlock()
}

// Results returns the channel of results from processed work items.
//
// The channel will be closed after Close() is called and all workers
// have finished processing. Callers should range over this channel
// to collect all results.
func (p *WorkerPool[T, R]) Results() <-chan R {
	return p.resultCh
}

// Workers returns the number of worker goroutines.
func (p *WorkerPool[T, R]) Workers() int {
	return p.workers
}

// ProcessAll is a convenience method that processes all items and returns results as a slice.
//
// This method:
// 1. Starts the pool
// 2. Submits all items
// 3. Closes the pool
// 4. Collects all results
//
// It's useful for simple batch processing where you have all items upfront.
func ProcessAll[T any, R any](workers int, items []T, processor func(T) R) []R {
	if len(items) == 0 {
		return nil
	}

	pool := NewWorkerPool(workers, processor)
	pool.Start()

	// Submit all items in a goroutine to avoid deadlock
	go func() {
		for _, item := range items {
			pool.Submit(item)
		}
		pool.Close()
	}()

	// Collect results
	results := make([]R, 0, len(items))
	for result := range pool.Results() {
		results = append(results, result)
	}

	return results
}
