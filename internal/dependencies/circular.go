package dependencies

import (
	"path/filepath"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// CircularDependency represents a circular dependency between packages
type CircularDependency struct {
	Cycle []string `json:"cycle"` // The circular dependency chain
}

// DetectCircularDependencies finds circular dependencies using DFS
func (a *Analyzer) DetectCircularDependencies(files []*parser.FileMetrics) ([]*CircularDependency, error) {
	graph := a.buildDependencyGraph(files)
	detector := newCycleDetector(graph)
	return detector.findCycles(), nil
}

// buildDependencyGraph creates a graph of internal package dependencies
func (a *Analyzer) buildDependencyGraph(files []*parser.FileMetrics) map[string][]string {
	// Group files by package import path (not package name!)
	packageFiles := a.groupFilesByImportPath(files)

	// Build graph of internal dependencies only
	graph := make(map[string][]string)
	for importPath, pkgFiles := range packageFiles {
		imports := a.extractInternalImports(pkgFiles)
		graph[importPath] = imports
	}

	return graph
}

// groupFilesByImportPath groups files by their import path
// This is critical - we must use import paths, not package names,
// because multiple directories can have the same package name
func (a *Analyzer) groupFilesByImportPath(files []*parser.FileMetrics) map[string][]*parser.FileMetrics {
	packageFiles := make(map[string][]*parser.FileMetrics)
	for _, file := range files {
		importPath := a.getImportPathForFile(file)
		if importPath != "" {
			packageFiles[importPath] = append(packageFiles[importPath], file)
		}
	}
	return packageFiles
}

// getImportPathForFile derives the import path from a file's path
func (a *Analyzer) getImportPathForFile(file *parser.FileMetrics) string {
	// If no module name, we can't determine import paths
	if a.moduleName == "" {
		return ""
	}

	// Get the directory containing the file
	cleanPath := filepath.Clean(file.FilePath)
	fileDir := filepath.Dir(cleanPath)

	// Get the relative path from project root to the file's directory
	relPath, err := filepath.Rel(a.projectPath, fileDir)
	if err != nil {
		// If we can't get a relative path, skip this file to avoid incorrect import paths
		return ""
	}

	// If the file is in the project root, use just the module name
	if relPath == "." {
		return a.moduleName
	}

	// Otherwise, combine module name with the relative path
	// Convert path separators to forward slashes for import paths
	relPath = filepath.ToSlash(relPath)
	return a.moduleName + "/" + relPath
}

// extractInternalImports extracts unique internal package imports from files
// Returns the full import paths, not just the last component
func (a *Analyzer) extractInternalImports(files []*parser.FileMetrics) []string {
	importSet := make(map[string]bool)
	for _, file := range files {
		for _, imp := range file.Imports {
			if a.categorizeImport(imp) == "internal" {
				// Use the full import path, not just the last component
				importSet[imp] = true
			}
		}
	}

	// Convert set to slice
	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	return imports
}

// cycleDetector encapsulates DFS state for cycle detection
type cycleDetector struct {
	graph    map[string][]string
	visited  map[string]bool
	recStack map[string]bool
	path     []string
	cycles   []*CircularDependency
}

// newCycleDetector creates a new cycle detector
func newCycleDetector(graph map[string][]string) *cycleDetector {
	return &cycleDetector{
		graph:    graph,
		visited:  make(map[string]bool),
		recStack: make(map[string]bool),
		path:     []string{},
		cycles:   []*CircularDependency{},
	}
}

// findCycles finds all cycles in the graph using DFS
func (d *cycleDetector) findCycles() []*CircularDependency {
	for node := range d.graph {
		if !d.visited[node] {
			d.dfs(node)
		}
	}
	return d.cycles
}

// dfs performs depth-first search to detect cycles
func (d *cycleDetector) dfs(node string) bool {
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

// addCycleIfNew extracts and adds a cycle if it's not a duplicate
func (d *cycleDetector) addCycleIfNew(neighbor string) {
	cycle := d.extractCycle(neighbor)
	if cycle != nil && !isDuplicateCycle(d.cycles, cycle) {
		d.cycles = append(d.cycles, &CircularDependency{Cycle: cycle})
	}
}

// extractCycle extracts the cycle from the current path
func (d *cycleDetector) extractCycle(neighbor string) []string {
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

// isDuplicateCycle checks if a cycle already exists in the list
func isDuplicateCycle(cycles []*CircularDependency, newCycle []string) bool {
	for _, existing := range cycles {
		if areCyclesEquivalent(existing.Cycle, newCycle) {
			return true
		}
	}
	return false
}

// areCyclesEquivalent checks if two cycles are the same (accounting for rotation)
func areCyclesEquivalent(cycle1, cycle2 []string) bool {
	if len(cycle1) != len(cycle2) {
		return false
	}

	// Normalize cycles by starting from the lexicographically smallest element
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

// normalizeCycle rotates the cycle to start with the smallest element
func normalizeCycle(cycle []string) []string {
	if len(cycle) == 0 {
		return cycle
	}

	minIdx := 0
	for i := 1; i < len(cycle); i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}

	normalized := make([]string, len(cycle))
	for i := range cycle {
		normalized[i] = cycle[(minIdx+i)%len(cycle)]
	}

	return normalized
}

// FormatCycle returns a human-readable representation of a circular dependency
func (cd *CircularDependency) FormatCycle() string {
	if len(cd.Cycle) == 0 {
		return ""
	}
	return strings.Join(cd.Cycle, " -> ")
}
