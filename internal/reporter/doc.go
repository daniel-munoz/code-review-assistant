// Package reporter handles formatting and outputting analysis results.
//
// The reporter package provides a flexible reporting interface that can output
// analysis results in different formats. Currently supports console output with
// formatted tables and sections.
//
// # Output Format
//
// The console reporter outputs:
// - Summary statistics (files, lines, functions)
// - Aggregate metrics (averages, percentiles, comment ratio)
// - Issues grouped by severity (warnings, info)
// - Test coverage report (if enabled)
// - Dependency analysis (if enabled)
// - Top N largest files and most complex functions
//
// # Usage
//
// Create a console reporter:
//
//	cfg := &config.OutputConfig{
//	    Format: "console",
//	    Verbose: true,
//	}
//	reporter, err := reporter.NewReporter(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Output analysis results:
//
//	err = reporter.Report(analysisResult)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Verbose mode includes additional details like full file paths and
// detailed issue descriptions.
package reporter
