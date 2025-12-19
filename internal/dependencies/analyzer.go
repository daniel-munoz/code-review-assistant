package dependencies

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// PackageDependencies represents dependency information for a single package
type PackageDependencies struct {
	PackageName        string   `json:"package_name"`
	StdlibImports      []string `json:"stdlib_imports"`
	InternalImports    []string `json:"internal_imports"`
	ExternalImports    []string `json:"external_imports"`
	TotalImports       int      `json:"total_imports"`
	ExternalImportCount int     `json:"external_import_count"`
}

// Analyzer analyzes package dependencies
type Analyzer struct {
	moduleName string
}

// NewAnalyzer creates a new dependency analyzer
func NewAnalyzer(projectPath string) (*Analyzer, error) {
	moduleName, err := getModuleName(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get module name: %w", err)
	}

	return &Analyzer{
		moduleName: moduleName,
	}, nil
}

// Analyze analyzes dependencies across all files
func (a *Analyzer) Analyze(files []*parser.FileMetrics) ([]*PackageDependencies, error) {
	// Group files by package
	packageFiles := make(map[string][]*parser.FileMetrics)
	for _, file := range files {
		packageFiles[file.PackageName] = append(packageFiles[file.PackageName], file)
	}

	var results []*PackageDependencies

	// Analyze each package
	for pkgName, pkgFiles := range packageFiles {
		// Collect unique imports across all files in package
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

		results = append(results, &PackageDependencies{
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

// categorizeImport determines if an import is stdlib, internal, or external
func (a *Analyzer) categorizeImport(importPath string) string {
	// Stdlib packages don't have dots in the first path component
	// and are in the standard library
	if isStdlib(importPath) {
		return "stdlib"
	}

	// Internal packages start with the module name
	if strings.HasPrefix(importPath, a.moduleName) {
		return "internal"
	}

	return "external"
}

// isStdlib checks if a package is in the Go standard library
func isStdlib(importPath string) bool {
	// Standard library packages are in the default GOROOT
	pkg, err := build.Default.Import(importPath, "", build.FindOnly)
	if err != nil {
		return false
	}

	// Check if package is in GOROOT
	return pkg.Goroot
}

// getModuleName reads the module name from go.mod
func getModuleName(projectPath string) (string, error) {
	goModPath := filepath.Join(projectPath, "go.mod")

	data, err := os.ReadFile(goModPath)
	if err != nil {
		// If no go.mod, return empty string (will treat all imports as external)
		return "", nil
	}

	// Parse module name from first line: "module <name>"
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("no module declaration found in go.mod")
}
