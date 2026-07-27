// Package dependencies analyzes import dependencies and provides language-agnostic
// cycle detection infrastructure. Language-specific analyzers (e.g., Go, Kotlin)
// build their own dependency graphs and delegate circular dependency detection
// to the shared algorithms here.
//
// For Go projects, the dependency analyzer also categorizes imports into three types:
// - Standard library (e.g., fmt, os, strings)
// - Internal packages (within the same module)
// - External packages (third-party dependencies)
//
// # Import Categorization
//
// The analyzer uses the following heuristics:
// - Imports without a dot in the first path element are stdlib
// - golang.org/x/* imports are treated as stdlib
// - Imports matching the module path are internal
// - All other imports are external
//
// # Circular Dependency Detection
//
// Circular dependencies are detected using DFS traversal of the dependency graph.
// Only internal packages are considered for cycle detection. The algorithm:
// 1. Builds a graph of package -> imported packages
// 2. Runs DFS from each unvisited node
// 3. Detects back edges (cycles) during traversal
// 4. Normalizes and deduplicates cycles
//
// # Usage
//
// For Go analysis:
//
//	analyzer, err := dependencies.NewAnalyzer("/path/to/project")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Analyze imports
//	deps, err := analyzer.Analyze(fileMetrics)
//
//	// Detect circular dependencies
//	cycles, err := analyzer.DetectCircularDependencies(fileMetrics)
//
// For other languages, build a dependency graph and use the shared cycle detector:
//
//	// Language providers with their own graph can reuse cycle detection:
//	cycles := dependencies.FindCycles(graph)
package dependencies
