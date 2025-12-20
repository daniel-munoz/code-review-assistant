// Package parser extracts metrics from Go source code using the go/ast package.
//
// The parser analyzes Go source files to extract:
// - File-level metrics (total lines, code lines, comment lines, blank lines)
// - Function-level metrics (length, parameters, return statements, nesting depth)
// - Cyclomatic complexity using McCabe's algorithm
// - Import statements and package information
//
// # Usage
//
// Parse a single file:
//
//	fset := token.NewFileSet()
//	metrics, err := parser.ParseFile(fset, "main.go")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Parse an entire directory:
//
//	excludePatterns := []string{"vendor/**", "**/*_test.go"}
//	metrics, errs := parser.ParseDirectory("./src", excludePatterns)
//	for _, err := range errs {
//	    log.Printf("Parse error: %v", err)
//	}
//
// The parser uses glob patterns for exclusions and supports:
// - ** (match any number of directories)
// - * (match any characters in a single path component)
// - ? (match a single character)
package parser
