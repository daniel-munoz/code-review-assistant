package parallel

import (
	"sync"
	"sync/atomic"

	"github.com/daniel-munoz/code-review-assistant/internal/status"
)

// ProgressReporter is a thread-safe wrapper around status.Reporter.
//
// It provides atomic progress counting and synchronized updates for
// use in parallel processing scenarios where multiple goroutines need
// to report progress.
type ProgressReporter struct {
	inner     status.Reporter
	processed atomic.Int64
	total     int
	prefix    string
	mu        sync.Mutex
}

// NewProgressReporter creates a new thread-safe progress reporter.
//
// Parameters:
//   - inner: the underlying status reporter to delegate to
//   - prefix: the status prefix (e.g., "[PARSE]")
//   - total: the total number of items to process (can be updated later)
func NewProgressReporter(inner status.Reporter, prefix string, total int) *ProgressReporter {
	return &ProgressReporter{
		inner:  inner,
		total:  total,
		prefix: prefix,
	}
}

// SetTotal updates the total count of items to process.
// This is useful when the total is not known at creation time.
func (r *ProgressReporter) SetTotal(total int) {
	r.mu.Lock()
	r.total = total
	r.mu.Unlock()
}

// RecordProgress atomically increments the progress counter and updates the status.
//
// This method is safe to call from multiple goroutines concurrently.
// The detail string provides context about the current item being processed.
func (r *ProgressReporter) RecordProgress(detail string) {
	count := int(r.processed.Add(1))
	r.mu.Lock()
	r.inner.UpdateProgress(r.prefix, count, r.total, detail)
	r.mu.Unlock()
}

// Update shows a simple status message through the underlying reporter.
// This method is synchronized to prevent interleaving with progress updates.
func (r *ProgressReporter) Update(message string) {
	r.mu.Lock()
	r.inner.Update(message)
	r.mu.Unlock()
}

// Processed returns the current count of processed items.
func (r *ProgressReporter) Processed() int {
	return int(r.processed.Load())
}

// Reset resets the progress counter to zero.
func (r *ProgressReporter) Reset() {
	r.processed.Store(0)
}
