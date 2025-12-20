package constants

// Console output formatting constants
const (
	SectionSeparatorLength = 60  // Length of separator lines in console output
	TableColumnPadding     = 2   // Padding between table columns
	MinColumnWidth         = 10  // Minimum column width for tables
)

// Number formatting constants
const (
	PercentageMultiplier = 100.0 // Multiplier to convert ratio to percentage
	DecimalPlaces        = 1     // Number of decimal places for percentages
)

// Coverage parsing constants
const (
	CoverageRegexCaptureGroup = 2 // Index of the capture group containing coverage percentage
)

// Package exclusion patterns
var DefaultExcludePatterns = []string{
	"vendor/**",
	"**/*_test.go",
	"**/testdata/**",
	"**/*.pb.go",
}
