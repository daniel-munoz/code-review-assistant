package status

// Reporter is the interface for progress reporting during analysis.
//
// Components can use this interface to report their status without
// coupling to specific output implementations. The orchestrator creates
// an appropriate implementation based on configuration and TTY detection.
//
// Implementations:
//   - TTYReporter: Shows live updates to stderr when running in a terminal
//   - SilentReporter: No-op implementation for non-TTY or quiet mode
//   - TestReporter: Captures messages for testing
type Reporter interface {
	// Start initializes the reporter.
	// Called at the beginning of the analysis pipeline.
	Start()

	// Update shows a simple status message.
	// Example: status.Update("[PARSE] Discovering Go files...")
	Update(message string)

	// UpdateProgress shows a status message with progress counter.
	// Example: status.UpdateProgress("[COVERAGE] Running tests", 5, 23, "pkg/name")
	// Displays as: "[COVERAGE] Running tests... (5/23) pkg/name"
	UpdateProgress(prefix string, current, total int, detail string)

	// Clear removes the current status line from the display.
	// Called before final output to ensure clean output.
	Clear()

	// Stop performs cleanup and finalizes the reporter.
	// Called at the end of the analysis pipeline.
	Stop()
}
