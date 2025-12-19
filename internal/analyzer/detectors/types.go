package detectors

// Issue represents a code quality issue found during analysis
type Issue struct {
	Severity  string `json:"severity"`  // "warning", "info", "error"
	Type      string `json:"type"`      // "large_file", "long_function", etc.
	File      string `json:"file"`      // File path
	Line      int    `json:"line"`      // Line number (0 if not applicable)
	Function  string `json:"function"`  // Function name (empty if not applicable)
	Message   string `json:"message"`   // Human-readable message
	Value     int    `json:"value"`     // Actual value (e.g., line count)
	Threshold int    `json:"threshold"` // Threshold value
}
