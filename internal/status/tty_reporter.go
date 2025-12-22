package status

import (
	"fmt"
	"os"
	"strings"
)

// TTYReporter implements Reporter for terminal output.
// Writes live status updates to stderr using single-line overwrites.
type TTYReporter struct {
	lastMessageLen int // Track length of last message for clearing
}

// NewTTYReporter creates a new TTYReporter.
func NewTTYReporter() *TTYReporter {
	return &TTYReporter{}
}

// Start initializes the reporter (no-op for TTY).
func (t *TTYReporter) Start() {}

// Update writes a simple status message to stderr.
// Uses \r to overwrite the current line.
func (t *TTYReporter) Update(message string) {
	t.write(message)
}

// UpdateProgress writes a status message with progress counter.
// Format: "{prefix}... ({current}/{total}) {detail}"
func (t *TTYReporter) UpdateProgress(prefix string, current, total int, detail string) {
	message := fmt.Sprintf("%s... (%d/%d) %s", prefix, current, total, detail)
	t.write(message)
}

// write outputs a message to stderr with proper clearing of previous content.
func (t *TTYReporter) write(message string) {
	// Clear previous message by writing spaces
	if t.lastMessageLen > 0 {
		clearLine := "\r" + strings.Repeat(" ", t.lastMessageLen) + "\r"
		fmt.Fprint(os.Stderr, clearLine)
	}

	// Write new message
	fmt.Fprintf(os.Stderr, "\r%s", message)

	// Track length for next clear
	t.lastMessageLen = len(message)
}

// Clear removes the current status line from stderr.
func (t *TTYReporter) Clear() {
	if t.lastMessageLen > 0 {
		// Write spaces to clear, then carriage return
		clearLine := "\r" + strings.Repeat(" ", t.lastMessageLen) + "\r"
		fmt.Fprint(os.Stderr, clearLine)
		t.lastMessageLen = 0
	}
}

// Stop performs cleanup (clears the line and prints newline).
func (t *TTYReporter) Stop() {
	t.Clear()
}
