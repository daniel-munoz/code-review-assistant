// Package orchestrator coordinates the code analysis pipeline.
//
// The orchestrator integrates the parser, analyzer, and reporter to provide
// a complete code analysis workflow. It handles the execution flow and error
// propagation between components.
//
// # Pipeline
//
// The orchestrator executes the following pipeline:
// 1. Parse - Extract metrics from Go source files
// 2. Analyze - Apply thresholds and detect issues
// 3. Report - Format and output results
//
// Each stage can fail independently, and errors are propagated up.
//
// # Usage
//
// Create and run the orchestrator:
//
//	cfg := config.Default()
//	projectPath := "/path/to/project"
//	orch, err := orchestrator.New(cfg, projectPath)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = orch.Run(projectPath)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The orchestrator automatically creates the parser, analyzer, and reporter
// based on the configuration and executes the full analysis pipeline.
package orchestrator
