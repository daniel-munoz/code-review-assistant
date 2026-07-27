package kotlin

import (
	"sort"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/dependencies"
	"github.com/daniel-munoz/code-review-assistant/internal/parser"
)

// rootPackage groups files that have no package declaration (valid but
// unusual Kotlin, e.g. scratch files and simple mains).
const rootPackage = "(root)"

// stdlibPrefixes identify Kotlin/JVM standard library imports. kotlinx.* is
// deliberately NOT here: kotlinx libraries (coroutines, serialization, ...)
// are separate artifacts added via Gradle, so they count as external.
var stdlibPrefixes = []string{"kotlin.", "java.", "javax."}

// DependencyAnalyzer analyzes Kotlin package dependencies.
//
// Unlike the Go and JS analyzers it reads no manifest: the set of internal
// packages is derived from the `package` declarations of the analyzed files
// themselves, and the unit of analysis is the declared Kotlin package (not
// the directory).
type DependencyAnalyzer struct{}

// NewDependencyAnalyzer creates a new Kotlin dependency analyzer. The
// projectPath argument is required by the language.Language interface but
// unused: internal packages come from the files' own package declarations.
func NewDependencyAnalyzer(projectPath string) (*DependencyAnalyzer, error) {
	return &DependencyAnalyzer{}, nil
}

// Analyze categorizes imports for every declared package.
func (a *DependencyAnalyzer) Analyze(files []*parser.FileMetrics) ([]*dependencies.PackageDependencies, error) {
	declared := declaredPackages(files)

	var results []*dependencies.PackageDependencies
	for pkgName, pkgFiles := range groupByPackage(files) {
		importSet := make(map[string]bool)
		for _, file := range pkgFiles {
			for _, imp := range file.Imports {
				importSet[imp] = true
			}
		}

		var stdlibImports, internalImports, externalImports []string
		for imp := range importSet {
			switch categorizeImport(imp, declared) {
			case "stdlib":
				stdlibImports = append(stdlibImports, imp)
			case "internal":
				internalImports = append(internalImports, imp)
			case "external":
				externalImports = append(externalImports, imp)
			}
		}
		sort.Strings(stdlibImports)
		sort.Strings(internalImports)
		sort.Strings(externalImports)

		results = append(results, &dependencies.PackageDependencies{
			PackageName:         pkgName,
			StdlibImports:       stdlibImports,
			InternalImports:     internalImports,
			ExternalImports:     externalImports,
			TotalImports:        len(importSet),
			ExternalImportCount: len(externalImports),
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].PackageName < results[j].PackageName })
	return results, nil
}

// declaredPackages collects the set of packages declared in the project.
func declaredPackages(files []*parser.FileMetrics) map[string]bool {
	declared := make(map[string]bool)
	for _, file := range files {
		if file.PackageName != "" {
			declared[file.PackageName] = true
		}
	}
	return declared
}

// groupByPackage groups files by declared package, with undeclared files
// under rootPackage.
func groupByPackage(files []*parser.FileMetrics) map[string][]*parser.FileMetrics {
	groups := make(map[string][]*parser.FileMetrics)
	for _, file := range files {
		name := file.PackageName
		if name == "" {
			name = rootPackage
		}
		groups[name] = append(groups[name], file)
	}
	return groups
}

// internalTarget returns the declared package an import resolves to, or ""
// when the import is not internal to the project. It walks dot-segments from
// the right so imports of nested classes, companion objects, and enum entries
// (e.g. com.example.alpha.Alpha.Companion) resolve to their declared package.
func internalTarget(imp string, declared map[string]bool) string {
	for c := strings.TrimSuffix(imp, ".*"); c != ""; {
		if declared[c] {
			return c
		}
		i := strings.LastIndex(c, ".")
		if i < 0 {
			return ""
		}
		c = c[:i]
	}
	return ""
}

// categorizeImport classifies an import as internal, stdlib, or external.
// Internal is checked first (mirroring the Go analyzer): a project could in
// principle declare a package under a stdlib-looking prefix, and the
// project's own declaration wins.
func categorizeImport(imp string, declared map[string]bool) string {
	if internalTarget(imp, declared) != "" {
		return "internal"
	}
	for _, prefix := range stdlibPrefixes {
		if strings.HasPrefix(imp, prefix) {
			return "stdlib"
		}
	}
	return "external"
}
