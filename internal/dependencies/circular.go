package dependencies

import (
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// CircularDependency represents a circular dependency between packages
type CircularDependency struct {
	Cycle []string `json:"cycle"` // The circular dependency chain
}

// DetectCircularDependencies finds circular dependencies using DFS
func (a *Analyzer) DetectCircularDependencies(files []*parser.FileMetrics) ([]*CircularDependency, error) {
	// Build dependency graph: package -> packages it imports
	graph := make(map[string][]string)

	// Group files by package and collect internal imports
	packageFiles := make(map[string][]*parser.FileMetrics)
	for _, file := range files {
		packageFiles[file.PackageName] = append(packageFiles[file.PackageName], file)
	}

	// Build graph of internal dependencies only
	for pkgName, pkgFiles := range packageFiles {
		importSet := make(map[string]bool)
		for _, file := range pkgFiles {
			for _, imp := range file.Imports {
				// Only track internal imports for circular dependency detection
				if a.categorizeImport(imp) == "internal" {
					// Extract package name from import path
					// e.g., "github.com/user/project/internal/parser" -> "parser"
					parts := strings.Split(imp, "/")
					if len(parts) > 0 {
						importedPkg := parts[len(parts)-1]
						importSet[importedPkg] = true
					}
				}
			}
		}

		// Convert set to slice
		for imp := range importSet {
			graph[pkgName] = append(graph[pkgName], imp)
		}
	}

	// Find cycles using DFS
	var cycles []*CircularDependency
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle - extract the cycle from path
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycle = append(cycle, neighbor) // Close the cycle

					// Check if this cycle is new (not a duplicate)
					if !isDuplicateCycle(cycles, cycle) {
						cycles = append(cycles, &CircularDependency{
							Cycle: cycle,
						})
					}
				}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	// Run DFS from each unvisited node
	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles, nil
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
