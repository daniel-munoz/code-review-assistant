package parser

// FileMetrics contains comprehensive metrics extracted from a single Go source file.
//
// This type aggregates both file-level and function-level metrics obtained through
// AST parsing. It provides a complete picture of file structure including:
//   - Line counts (total, code, comments, blank)
//   - Package information
//   - Import dependencies
//   - All functions and methods defined in the file
//
// FileMetrics is the primary data structure returned by ParseFile and used
// throughout the analysis pipeline.
type FileMetrics struct {
	FilePath     string             // Absolute or relative path to the file
	PackageName  string             // Package declaration (Go), module name (Python), module path (TypeScript)
	Language     string             // Language identifier: "go", "python", "typescript", etc.
	TotalLines   int                // Total lines in the file
	CodeLines    int                // Lines with actual code (excluding blank and comments)
	CommentLines int                // Lines with comments
	BlankLines   int                // Blank lines
	Functions    []*FunctionMetrics // All functions/methods in the file
	Imports      []string           // Import paths
}

// FunctionMetrics contains comprehensive metrics for a single function or method.
//
// This type captures both structural and complexity metrics:
//   - Identity: name, receiver type (for methods), location
//   - Size: line count and location within file
//   - Signature: parameter count, return value count
//   - Complexity: cyclomatic complexity using McCabe's method
//
// Methods can be identified by checking IsMethod() or examining ReceiverType.
// The FullName() method provides a consistent naming scheme for both functions
// and methods.
type FunctionMetrics struct {
	Name         string // Function name
	ReceiverType string // Empty if not a method, otherwise the receiver type
	StartLine    int    // Line number where function starts
	EndLine      int    // Line number where function ends
	Lines        int    // Total lines in function body
	Parameters   int    // Number of parameters
	ReturnValues int    // Number of return values
	Complexity   int    // Simple cyclomatic complexity (Phase 1: basic count)
}

// IsMethod returns true if this function is a method (has a receiver).
//
// Methods in Go are functions with a receiver type. This helper makes it easy
// to distinguish between standalone functions and methods when analyzing code.
func (fm *FunctionMetrics) IsMethod() bool {
	return fm.ReceiverType != ""
}

// FullName returns the fully qualified function name including receiver for methods.
//
// For methods, returns "ReceiverType.FunctionName" (e.g., "*Parser.ParseFile").
// For standalone functions, returns just the function name (e.g., "main").
//
// This provides a consistent naming scheme for reporting and issue tracking.
func (fm *FunctionMetrics) FullName() string {
	if fm.IsMethod() {
		return fm.ReceiverType + "." + fm.Name
	}
	return fm.Name
}

// CommentRatio calculates the ratio of comment lines to total lines.
//
// Returns a value between 0.0 and 1.0 representing the proportion of the file
// that consists of comments. A higher ratio indicates better documentation.
//
// Recommended minimum ratio is 0.15 (15%). Returns 0.0 for empty files.
func (fm *FileMetrics) CommentRatio() float64 {
	if fm.TotalLines == 0 {
		return 0
	}
	return float64(fm.CommentLines) / float64(fm.TotalLines)
}

// CodeRatio calculates the ratio of code lines to total lines.
//
// Returns a value between 0.0 and 1.0 representing the proportion of the file
// that consists of actual code (excluding comments and blank lines).
//
// Returns 0.0 for empty files.
func (fm *FileMetrics) CodeRatio() float64 {
	if fm.TotalLines == 0 {
		return 0
	}
	return float64(fm.CodeLines) / float64(fm.TotalLines)
}
