package status

// SilentReporter implements Reporter with no-op methods.
// Used when output is piped, redirected, or quiet mode is enabled.
type SilentReporter struct{}

// NewSilentReporter creates a new SilentReporter.
func NewSilentReporter() *SilentReporter {
	return &SilentReporter{}
}

// Start is a no-op.
func (s *SilentReporter) Start() {}

// Update is a no-op.
func (s *SilentReporter) Update(message string) {}

// UpdateProgress is a no-op.
func (s *SilentReporter) UpdateProgress(prefix string, current, total int, detail string) {}

// Clear is a no-op.
func (s *SilentReporter) Clear() {}

// Stop is a no-op.
func (s *SilentReporter) Stop() {}
