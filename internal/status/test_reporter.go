package status

import "fmt"

// TestReporter implements Reporter for testing.
// Captures all status messages in a slice for verification.
type TestReporter struct {
	messages []string
	started  bool
	stopped  bool
}

// NewTestReporter creates a new TestReporter.
func NewTestReporter() *TestReporter {
	return &TestReporter{
		messages: make([]string, 0),
	}
}

// Start records that the reporter was started.
func (t *TestReporter) Start() {
	t.started = true
}

// Update captures a simple status message.
func (t *TestReporter) Update(message string) {
	t.messages = append(t.messages, message)
}

// UpdateProgress captures a progress message.
func (t *TestReporter) UpdateProgress(prefix string, current, total int, detail string) {
	message := fmt.Sprintf("%s... (%d/%d) %s", prefix, current, total, detail)
	t.messages = append(t.messages, message)
}

// Clear records a clear operation.
func (t *TestReporter) Clear() {
	t.messages = append(t.messages, "[CLEAR]")
}

// Stop records that the reporter was stopped.
func (t *TestReporter) Stop() {
	t.stopped = true
}

// GetMessages returns all captured messages.
func (t *TestReporter) GetMessages() []string {
	return t.messages
}

// WasStarted returns whether Start() was called.
func (t *TestReporter) WasStarted() bool {
	return t.started
}

// WasStopped returns whether Stop() was called.
func (t *TestReporter) WasStopped() bool {
	return t.stopped
}
