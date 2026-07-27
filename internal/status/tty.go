package status

import (
	"os"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"golang.org/x/term"
)

// IsTTY checks if the given file descriptor is a terminal.
func IsTTY(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// NewReporter creates an appropriate Reporter based on configuration and TTY detection.
//
// Returns:
//   - TTYReporter: If stderr is a TTY and status is enabled
//   - SilentReporter: If stderr is not a TTY, quiet mode is on, or output is redirected
//
// Logic:
//  1. If QuietMode is true: return SilentReporter
//  2. If ShowStatus is true: return TTYReporter (force enable)
//  3. If OutputFile is set: return SilentReporter (output redirected)
//  4. If stderr is a TTY: return TTYReporter
//  5. Otherwise: return SilentReporter (piped/redirected)
func NewReporter(cfg *config.OutputConfig) Reporter {
	// Explicit disable via --quiet flag
	if cfg.QuietMode {
		return NewSilentReporter()
	}

	// Explicit enable via --show-status flag
	if cfg.ShowStatus {
		return NewTTYReporter()
	}

	// Disable if output is redirected to a file
	if cfg.OutputFile != "" {
		return NewSilentReporter()
	}

	// Auto-detect: enable only if stderr is a TTY
	if IsTTY(os.Stderr.Fd()) {
		return NewTTYReporter()
	}

	// Default: silent (piped or redirected)
	return NewSilentReporter()
}
