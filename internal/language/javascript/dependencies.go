// Package javascript provides JavaScript/TypeScript language support for code analysis.
package javascript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// DependencyAnalyzer analyzes JavaScript/TypeScript package dependencies.
type DependencyAnalyzer struct {
	projectName string
	projectPath string
}

// NewDependencyAnalyzer creates a new JavaScript/TypeScript dependency analyzer.
func NewDependencyAnalyzer(projectPath string) (*DependencyAnalyzer, error) {
	projectName, err := getProjectName(projectPath)
	if err != nil {
		// If no package.json, use empty project name
		projectName = ""
	}

	cleanPath := filepath.Clean(projectPath)

	return &DependencyAnalyzer{
		projectName: projectName,
		projectPath: cleanPath,
	}, nil
}

// Analyze analyzes dependencies across all files.
func (a *DependencyAnalyzer) Analyze(files []*parser.FileMetrics) ([]*dependencies.PackageDependencies, error) {
	// Group files by directory (as "package" in JS/TS)
	packageFiles := make(map[string][]*parser.FileMetrics)
	for _, file := range files {
		dir := filepath.Dir(file.FilePath)
		relDir, err := filepath.Rel(a.projectPath, dir)
		if err != nil {
			relDir = dir
		}
		packageFiles[relDir] = append(packageFiles[relDir], file)
	}

	var results []*dependencies.PackageDependencies

	for pkgDir, pkgFiles := range packageFiles {
		// Collect unique imports across all files in directory
		importSet := make(map[string]bool)
		for _, file := range pkgFiles {
			for _, imp := range file.Imports {
				importSet[imp] = true
			}
		}

		// Categorize imports
		var stdlibImports []string
		var internalImports []string
		var externalImports []string

		for imp := range importSet {
			category := a.categorizeImport(imp)
			switch category {
			case "stdlib":
				stdlibImports = append(stdlibImports, imp)
			case "internal":
				internalImports = append(internalImports, imp)
			case "external":
				externalImports = append(externalImports, imp)
			}
		}

		pkgName := pkgDir
		if pkgName == "." {
			pkgName = "(root)"
		}

		results = append(results, &dependencies.PackageDependencies{
			PackageName:         pkgName,
			StdlibImports:       stdlibImports,
			InternalImports:     internalImports,
			ExternalImports:     externalImports,
			TotalImports:        len(importSet),
			ExternalImportCount: len(externalImports),
		})
	}

	return results, nil
}

// DetectCircularDependencies finds circular import chains in JavaScript/TypeScript code.
func (a *DependencyAnalyzer) DetectCircularDependencies(files []*parser.FileMetrics) ([]*dependencies.CircularDependency, error) {
	graph := a.buildDependencyGraph(files)
	detector := newJSCycleDetector(graph)
	return detector.findCycles(), nil
}

// categorizeImport determines if an import is stdlib (Node.js built-in), internal, or external.
func (a *DependencyAnalyzer) categorizeImport(importPath string) string {
	// Relative imports are internal
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		return "internal"
	}

	// Node.js built-in modules
	if isNodeBuiltin(importPath) {
		return "stdlib"
	}

	// Scoped packages or regular npm packages are external
	return "external"
}

// nodeBuiltins is a set of Node.js built-in module names.
var nodeBuiltins = map[string]bool{
	// Core modules
	"assert":              true,
	"async_hooks":         true,
	"buffer":              true,
	"child_process":       true,
	"cluster":             true,
	"console":             true,
	"constants":           true,
	"crypto":              true,
	"dgram":               true,
	"diagnostics_channel": true,
	"dns":                 true,
	"domain":              true,
	"events":              true,
	"fs":                  true,
	"http":                true,
	"http2":               true,
	"https":               true,
	"inspector":           true,
	"module":              true,
	"net":                 true,
	"os":                  true,
	"path":                true,
	"perf_hooks":          true,
	"process":             true,
	"punycode":            true,
	"querystring":         true,
	"readline":            true,
	"repl":                true,
	"stream":              true,
	"string_decoder":      true,
	"sys":                 true,
	"timers":              true,
	"tls":                 true,
	"trace_events":        true,
	"tty":                 true,
	"url":                 true,
	"util":                true,
	"v8":                  true,
	"vm":                  true,
	"wasi":                true,
	"worker_threads":      true,
	"zlib":                true,
}

// isNodeBuiltin checks if an import path is a Node.js built-in module.
func isNodeBuiltin(importPath string) bool {
	// Handle node: prefix (Node.js 16+)
	if strings.HasPrefix(importPath, "node:") {
		importPath = strings.TrimPrefix(importPath, "node:")
	}

	// Get the base module name (before any path separator)
	baseName := importPath
	if idx := strings.Index(importPath, "/"); idx != -1 {
		baseName = importPath[:idx]
	}

	return nodeBuiltins[baseName]
}

// getProjectName reads the project name from package.json.
func getProjectName(projectPath string) (string, error) {
	pkgPath := filepath.Join(projectPath, "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}

	var pkg struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}

	return pkg.Name, nil
}

// buildDependencyGraph creates a graph of internal file dependencies.
func (a *DependencyAnalyzer) buildDependencyGraph(files []*parser.FileMetrics) map[string][]string {
	// Create a map of file paths for quick lookup
	filePathSet := make(map[string]bool)
	for _, file := range files {
		relPath, err := filepath.Rel(a.projectPath, file.FilePath)
		if err != nil {
			relPath = file.FilePath
		}
		filePathSet[relPath] = true
	}

	graph := make(map[string][]string)

	for _, file := range files {
		relPath, err := filepath.Rel(a.projectPath, file.FilePath)
		if err != nil {
			relPath = file.FilePath
		}

		var deps []string
		fileDir := filepath.Dir(file.FilePath)

		for _, imp := range file.Imports {
			// Only consider relative imports for circular dependency detection
			if !strings.HasPrefix(imp, "./") && !strings.HasPrefix(imp, "../") {
				continue
			}

			// Resolve the import to a file path
			resolved := a.resolveImport(fileDir, imp)
			if resolved != "" {
				relResolved, err := filepath.Rel(a.projectPath, resolved)
				if err != nil {
					relResolved = resolved
				}
				// Only add if it's a file we're analyzing
				if filePathSet[relResolved] {
					deps = append(deps, relResolved)
				}
			}
		}

		graph[relPath] = deps
	}

	return graph
}

// resolveImport resolves a relative import to an absolute file path.
func (a *DependencyAnalyzer) resolveImport(fromDir, importPath string) string {
	// Resolve relative path
	resolved := filepath.Join(fromDir, importPath)

	// Try with various extensions
	extensions := []string{"", ".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"}

	for _, ext := range extensions {
		candidate := resolved + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// jsCycleDetector encapsulates DFS state for cycle detection.
type jsCycleDetector struct {
	graph    map[string][]string
	visited  map[string]bool
	recStack map[string]bool
	path     []string
	cycles   []*dependencies.CircularDependency
}

// newJSCycleDetector creates a new cycle detector.
func newJSCycleDetector(graph map[string][]string) *jsCycleDetector {
	return &jsCycleDetector{
		graph:    graph,
		visited:  make(map[string]bool),
		recStack: make(map[string]bool),
		path:     []string{},
		cycles:   []*dependencies.CircularDependency{},
	}
}

// findCycles finds all cycles in the graph using DFS.
func (d *jsCycleDetector) findCycles() []*dependencies.CircularDependency {
	for node := range d.graph {
		if !d.visited[node] {
			d.dfs(node)
		}
	}
	return d.cycles
}

// dfs performs depth-first search to detect cycles.
func (d *jsCycleDetector) dfs(node string) bool {
	d.visited[node] = true
	d.recStack[node] = true
	d.path = append(d.path, node)

	for _, neighbor := range d.graph[node] {
		if !d.visited[neighbor] {
			if d.dfs(neighbor) {
				return true
			}
		} else if d.recStack[neighbor] {
			d.addCycleIfNew(neighbor)
		}
	}

	d.path = d.path[:len(d.path)-1]
	d.recStack[node] = false
	return false
}

// addCycleIfNew extracts and adds a cycle if it's not a duplicate.
func (d *jsCycleDetector) addCycleIfNew(neighbor string) {
	cycle := d.extractCycle(neighbor)
	if cycle != nil && !d.isDuplicateCycle(cycle) {
		d.cycles = append(d.cycles, &dependencies.CircularDependency{Cycle: cycle})
	}
}

// extractCycle extracts the cycle from the current path.
func (d *jsCycleDetector) extractCycle(neighbor string) []string {
	cycleStart := -1
	for i, p := range d.path {
		if p == neighbor {
			cycleStart = i
			break
		}
	}

	if cycleStart < 0 {
		return nil
	}

	cycle := make([]string, len(d.path)-cycleStart)
	copy(cycle, d.path[cycleStart:])
	cycle = append(cycle, neighbor) // Close the cycle
	return cycle
}

// isDuplicateCycle checks if a cycle already exists in the list.
func (d *jsCycleDetector) isDuplicateCycle(newCycle []string) bool {
	for _, existing := range d.cycles {
		if areCyclesEquivalent(existing.Cycle, newCycle) {
			return true
		}
	}
	return false
}

// areCyclesEquivalent checks if two cycles are the same (accounting for rotation).
func areCyclesEquivalent(cycle1, cycle2 []string) bool {
	if len(cycle1) != len(cycle2) {
		return false
	}

	norm1 := normalizeCycle(cycle1)
	norm2 := normalizeCycle(cycle2)

	if len(norm1) != len(norm2) {
		return false
	}

	for i := range norm1 {
		if norm1[i] != norm2[i] {
			return false
		}
	}

	return true
}

// normalizeCycle rotates the cycle to start with the smallest element.
// Cycles are represented as [a, b, c, a] where the last element closes the cycle.
func normalizeCycle(cycle []string) []string {
	if len(cycle) <= 1 {
		return cycle
	}

	// The last element is the closing element (same as first), so we only
	// consider the first n-1 elements when finding the minimum
	n := len(cycle) - 1

	minIdx := 0
	for i := 1; i < n; i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}

	// Build normalized cycle: rotate to start with min element, then close with it
	normalized := make([]string, len(cycle))
	for i := 0; i < n; i++ {
		normalized[i] = cycle[(minIdx+i)%n]
	}
	// Close the cycle with the starting element
	normalized[n] = normalized[0]

	return normalized
}
