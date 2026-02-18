// Package php provides PHP language support for code analysis.
package php

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// DependencyAnalyzer analyzes PHP package dependencies via Composer and use statements.
type DependencyAnalyzer struct {
	projectPath    string
	psr4Mappings   map[string]string // Namespace prefix -> directory path
	externalPkgs   map[string]bool   // External packages from composer.json require
}

// NewDependencyAnalyzer creates a new PHP dependency analyzer.
func NewDependencyAnalyzer(projectPath string) (*DependencyAnalyzer, error) {
	cleanPath := filepath.Clean(projectPath)

	analyzer := &DependencyAnalyzer{
		projectPath:  cleanPath,
		psr4Mappings: make(map[string]string),
		externalPkgs: make(map[string]bool),
	}

	// Read composer.json for autoload mappings and dependencies
	if err := analyzer.loadComposerConfig(); err != nil {
		// If no composer.json, return analyzer with empty mappings (still functional)
		return analyzer, nil
	}

	return analyzer, nil
}

// composerJSON represents the relevant parts of composer.json.
type composerJSON struct {
	Name    string `json:"name"`
	Require map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
	Autoload struct {
		PSR4 map[string]interface{} `json:"psr-4"`
	} `json:"autoload"`
	AutoloadDev struct {
		PSR4 map[string]interface{} `json:"psr-4"`
	} `json:"autoload-dev"`
}

// loadComposerConfig reads composer.json and extracts PSR-4 mappings and dependencies.
func (a *DependencyAnalyzer) loadComposerConfig() error {
	composerPath := filepath.Join(a.projectPath, "composer.json")

	data, err := os.ReadFile(composerPath)
	if err != nil {
		return err
	}

	var composer composerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return err
	}

	// Extract PSR-4 autoload mappings (both main and dev)
	for prefix, paths := range composer.Autoload.PSR4 {
		a.addPSR4Mapping(prefix, paths)
	}
	for prefix, paths := range composer.AutoloadDev.PSR4 {
		a.addPSR4Mapping(prefix, paths)
	}

	// Extract external package names from require sections
	for pkg := range composer.Require {
		// Skip PHP platform requirements (php, ext-*)
		if pkg == "php" || strings.HasPrefix(pkg, "ext-") {
			continue
		}
		a.externalPkgs[pkg] = true
	}
	for pkg := range composer.RequireDev {
		if pkg == "php" || strings.HasPrefix(pkg, "ext-") {
			continue
		}
		a.externalPkgs[pkg] = true
	}

	return nil
}

// addPSR4Mapping adds a PSR-4 namespace prefix to directory mapping.
func (a *DependencyAnalyzer) addPSR4Mapping(prefix string, paths interface{}) {
	// PSR-4 values can be a string or an array of strings
	switch v := paths.(type) {
	case string:
		a.psr4Mappings[prefix] = v
	case []interface{}:
		for _, p := range v {
			if s, ok := p.(string); ok {
				a.psr4Mappings[prefix] = s
				break // Use first path
			}
		}
	}
}

// Analyze analyzes dependencies across all parsed PHP files.
func (a *DependencyAnalyzer) Analyze(files []*parser.FileMetrics) ([]*dependencies.PackageDependencies, error) {
	// Group files by namespace (as "package" in PHP)
	packageFiles := make(map[string][]*parser.FileMetrics)
	for _, file := range files {
		ns := file.PackageName
		if ns == "" {
			ns = "(global)"
		}
		packageFiles[ns] = append(packageFiles[ns], file)
	}

	var results []*dependencies.PackageDependencies

	for ns, pkgFiles := range packageFiles {
		// Collect unique imports across all files in this namespace
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

		results = append(results, &dependencies.PackageDependencies{
			PackageName:         ns,
			StdlibImports:       stdlibImports,
			InternalImports:     internalImports,
			ExternalImports:     externalImports,
			TotalImports:        len(importSet),
			ExternalImportCount: len(externalImports),
		})
	}

	return results, nil
}

// DetectCircularDependencies finds circular import chains in PHP code.
func (a *DependencyAnalyzer) DetectCircularDependencies(files []*parser.FileMetrics) ([]*dependencies.CircularDependency, error) {
	graph := a.buildDependencyGraph(files)
	detector := newPHPCycleDetector(graph)
	return detector.findCycles(), nil
}

// categorizeImport determines if an import is stdlib (PHP built-in), internal, or external.
func (a *DependencyAnalyzer) categorizeImport(importPath string) string {
	// Check if the import matches a PSR-4 mapping from the project's autoload
	for prefix := range a.psr4Mappings {
		nsPrefix := strings.TrimSuffix(prefix, "\\")
		if strings.HasPrefix(importPath, nsPrefix) {
			return "internal"
		}
	}

	// Check if it's a PHP built-in class/interface
	if isPHPBuiltin(importPath) {
		return "stdlib"
	}

	// Everything else is external
	return "external"
}

// phpBuiltins contains PHP built-in classes, interfaces, and exceptions.
var phpBuiltins = map[string]bool{
	// SPL exceptions
	"Exception": true, "RuntimeException": true, "LogicException": true,
	"InvalidArgumentException": true, "BadMethodCallException": true,
	"BadFunctionCallException": true, "DomainException": true,
	"LengthException": true, "OutOfBoundsException": true,
	"OutOfRangeException": true, "OverflowException": true,
	"RangeException": true, "UnderflowException": true,
	"UnexpectedValueException": true, "TypeError": true,
	"ValueError": true, "Error": true, "ArithmeticError": true,
	"DivisionByZeroError": true,

	// SPL interfaces
	"Iterator": true, "IteratorAggregate": true, "ArrayAccess": true,
	"Serializable": true, "Countable": true, "Stringable": true,
	"JsonSerializable": true, "Throwable": true, "Traversable": true,

	// SPL classes
	"ArrayObject": true, "SplStack": true, "SplQueue": true,
	"SplPriorityQueue": true, "SplFixedArray": true,
	"SplDoublyLinkedList": true, "SplHeap": true,
	"SplMinHeap": true, "SplMaxHeap": true,
	"SplFileInfo": true, "SplFileObject": true,
	"DirectoryIterator": true, "RecursiveDirectoryIterator": true,
	"RecursiveIteratorIterator": true, "FilterIterator": true,
	"RegexIterator": true, "AppendIterator": true,
	"CachingIterator": true, "LimitIterator": true,

	// Core classes
	"stdClass": true, "Closure": true, "Generator": true,
	"Fiber": true, "WeakReference": true, "WeakMap": true,

	// Date/Time
	"DateTime": true, "DateTimeImmutable": true, "DateTimeInterface": true,
	"DateInterval": true, "DatePeriod": true, "DateTimeZone": true,

	// Database
	"PDO": true, "PDOStatement": true, "PDOException": true,
	"mysqli": true, "mysqli_result": true, "mysqli_stmt": true,

	// Reflection
	"ReflectionClass": true, "ReflectionMethod": true,
	"ReflectionFunction": true, "ReflectionProperty": true,
	"ReflectionParameter": true, "ReflectionType": true,

	// Other common built-ins
	"SplAutoloadRegister": true, "GlobIterator": true,
	"SimpleXMLElement": true, "DOMDocument": true,
	"DOMElement": true, "DOMNode": true,
	"CurlHandle": true, "GdImage": true,
	"Attribute": true, "BackedEnum": true, "UnitEnum": true,
}

// isPHPBuiltin checks if an import path is a PHP built-in class.
func isPHPBuiltin(importPath string) bool {
	// Get the class name (last part of the namespace)
	parts := strings.Split(importPath, "\\")
	className := parts[len(parts)-1]

	// Check against known built-ins
	if phpBuiltins[className] {
		// Only count as stdlib if it's a simple name (no namespace prefix)
		// or if it's not matching any external package pattern
		if len(parts) == 1 {
			return true
		}
	}

	return false
}

// buildDependencyGraph creates a graph of internal namespace dependencies.
func (a *DependencyAnalyzer) buildDependencyGraph(files []*parser.FileMetrics) map[string][]string {
	graph := make(map[string][]string)

	for _, file := range files {
		ns := file.PackageName
		if ns == "" {
			ns = "(global)"
		}

		var internalDeps []string
		for _, imp := range file.Imports {
			if a.categorizeImport(imp) == "internal" {
				// Extract the namespace portion of the import
				impNS := extractNamespaceFromImport(imp)
				if impNS != "" && impNS != ns {
					internalDeps = append(internalDeps, impNS)
				}
			}
		}

		if existing, ok := graph[ns]; ok {
			graph[ns] = append(existing, internalDeps...)
		} else {
			graph[ns] = internalDeps
		}
	}

	// Deduplicate edges
	for ns, deps := range graph {
		graph[ns] = dedup(deps)
	}

	return graph
}

// extractNamespaceFromImport extracts the namespace from a fully-qualified import.
// e.g., "App\Services\UserService" -> "App\Services"
func extractNamespaceFromImport(imp string) string {
	parts := strings.Split(imp, "\\")
	if len(parts) <= 1 {
		return imp
	}
	return strings.Join(parts[:len(parts)-1], "\\")
}

// dedup removes duplicate strings from a slice.
func dedup(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// phpCycleDetector encapsulates DFS state for cycle detection.
type phpCycleDetector struct {
	graph    map[string][]string
	visited  map[string]bool
	recStack map[string]bool
	path     []string
	cycles   []*dependencies.CircularDependency
}

// newPHPCycleDetector creates a new cycle detector.
func newPHPCycleDetector(graph map[string][]string) *phpCycleDetector {
	return &phpCycleDetector{
		graph:    graph,
		visited:  make(map[string]bool),
		recStack: make(map[string]bool),
		path:     []string{},
		cycles:   []*dependencies.CircularDependency{},
	}
}

// findCycles finds all cycles in the graph using DFS.
func (d *phpCycleDetector) findCycles() []*dependencies.CircularDependency {
	for node := range d.graph {
		if !d.visited[node] {
			d.dfs(node)
		}
	}
	return d.cycles
}

// dfs performs depth-first search to detect cycles.
func (d *phpCycleDetector) dfs(node string) bool {
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
func (d *phpCycleDetector) addCycleIfNew(neighbor string) {
	cycle := d.extractCycle(neighbor)
	if cycle != nil && !d.isDuplicateCycle(cycle) {
		d.cycles = append(d.cycles, &dependencies.CircularDependency{Cycle: cycle})
	}
}

// extractCycle extracts the cycle from the current path.
func (d *phpCycleDetector) extractCycle(neighbor string) []string {
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
func (d *phpCycleDetector) isDuplicateCycle(newCycle []string) bool {
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
func normalizeCycle(cycle []string) []string {
	if len(cycle) <= 1 {
		return cycle
	}

	n := len(cycle) - 1

	minIdx := 0
	for i := 1; i < n; i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}

	normalized := make([]string, len(cycle))
	for i := 0; i < n; i++ {
		normalized[i] = cycle[(minIdx+i)%n]
	}
	normalized[n] = normalized[0]

	return normalized
}
