package constants

// Default analysis thresholds for Phase 1 (basic metrics)
const (
	DefaultLargeFileThreshold    = 500  // Lines beyond which a file is considered large
	DefaultLongFunctionThreshold = 50   // Lines beyond which a function is considered long
	DefaultMinCommentRatio       = 0.15 // Minimum recommended comment to code ratio (15%)
	DefaultComplexityThreshold   = 10   // McCabe cyclomatic complexity threshold
)

// Default anti-pattern detection thresholds for Phase 2.2
const (
	DefaultMaxParameters         = 5 // Maximum recommended function parameters
	DefaultMaxNestingDepth       = 4 // Maximum recommended nesting depth
	DefaultMaxReturnStatements   = 3 // Maximum recommended return statements per function
	DefaultDetectMagicNumbers    = true
	DefaultDetectDuplicateErrors = true
)

// Default test coverage settings for Phase 2.3
const (
	DefaultEnableCoverage       = true
	DefaultMinCoverageThreshold = 50.0 // Minimum test coverage percentage (50%)
	DefaultCoverageTimeout      = 30   // Timeout in seconds for coverage analysis per package
)

// Default dependency analysis settings for Phase 2.4
const (
	DefaultMaxImports              = 10 // Maximum imports per package before flagging
	DefaultMaxExternalDependencies = 10 // Maximum external dependencies before flagging
	DefaultDetectCircularDeps      = true
)

// Reporting and calculation constants
const (
	TopFilesLimit            = 10   // Number of largest files to report
	TopComplexFunctionsLimit = 10   // Number of most complex functions to report
	Percentile95             = 0.95 // 95th percentile for statistical calculations
)

// Duplicate error detection threshold
const (
	MinDuplicateErrorOccurrences = 5 // Minimum occurrences before flagging duplicate errors
)

// Default Kotlin-specific detector settings
const (
	DefaultDetectNonNullAssertions = true // Flag !! non-null assertion usage (Kotlin)
	DefaultDetectRunBlocking       = true // Flag runBlocking usage outside fun main (Kotlin)
)
