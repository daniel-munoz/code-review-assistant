package parser

// FileMetrics contains metrics extracted from a single Go file
type FileMetrics struct {
	FilePath     string             // Absolute or relative path to the file
	PackageName  string             // Package declaration
	TotalLines   int                // Total lines in the file
	CodeLines    int                // Lines with actual code (excluding blank and comments)
	CommentLines int                // Lines with comments
	BlankLines   int                // Blank lines
	Functions    []*FunctionMetrics // All functions/methods in the file
	Imports      []string           // Import paths
}

// FunctionMetrics contains metrics for a single function or method
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

// IsMethod returns true if this is a method (has a receiver)
func (fm *FunctionMetrics) IsMethod() bool {
	return fm.ReceiverType != ""
}

// FullName returns the full function name including receiver for methods
func (fm *FunctionMetrics) FullName() string {
	if fm.IsMethod() {
		return fm.ReceiverType + "." + fm.Name
	}
	return fm.Name
}

// CommentRatio calculates the ratio of comment lines to total lines
func (fm *FileMetrics) CommentRatio() float64 {
	if fm.TotalLines == 0 {
		return 0
	}
	return float64(fm.CommentLines) / float64(fm.TotalLines)
}

// CodeRatio calculates the ratio of code lines to total lines
func (fm *FileMetrics) CodeRatio() float64 {
	if fm.TotalLines == 0 {
		return 0
	}
	return float64(fm.CodeLines) / float64(fm.TotalLines)
}
