package kotlin

import (
	"encoding/xml"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/daniel-munoz/code-review-assistant/internal/coverage"
)

// jacocoReport models the subset of the JaCoCo XML format we need. Kover's
// XML report uses the same format, so one parser serves both tools.
// Only counters that are DIRECT children of <package> are package totals;
// <class>/<sourcefile> children carry their own counters and are ignored.
type jacocoReport struct {
	Packages []jacocoPackage `xml:"package"`
}

type jacocoPackage struct {
	Name     string          `xml:"name,attr"`
	Counters []jacocoCounter `xml:"counter"`
}

type jacocoCounter struct {
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

// lineTally accumulates LINE counters for one package across reports
// (multi-module builds produce one report per module).
type lineTally struct {
	covered int
	missed  int
	hasLine bool
}

// parseReports reads JaCoCo/Kover XML reports and produces per-package
// coverage. Duplicate packages across reports are merged by summing their
// LINE counters before computing the percentage.
func parseReports(paths []string) ([]*coverage.PackageCoverage, error) {
	tallies := make(map[string]*lineTally)

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read coverage report %s: %w", path, err)
		}

		var report jacocoReport
		if err := xml.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("failed to parse coverage report %s: %w", path, err)
		}

		for _, pkg := range report.Packages {
			// JaCoCo names packages with slashes: com/example/alpha
			name := strings.ReplaceAll(pkg.Name, "/", ".")
			tally, ok := tallies[name]
			if !ok {
				tally = &lineTally{}
				tallies[name] = tally
			}
			for _, counter := range pkg.Counters {
				if counter.Type == "LINE" {
					tally.covered += counter.Covered
					tally.missed += counter.Missed
					tally.hasLine = true
				}
			}
		}
	}

	var results []*coverage.PackageCoverage
	for name, tally := range tallies {
		total := tally.covered + tally.missed
		if !tally.hasLine || total == 0 {
			results = append(results, &coverage.PackageCoverage{
				PackagePath: name,
				Skipped:     true,
			})
			continue
		}
		results = append(results, &coverage.PackageCoverage{
			PackagePath: name,
			Coverage:    float64(tally.covered) / float64(total) * 100,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].PackagePath < results[j].PackagePath })
	return results, nil
}
