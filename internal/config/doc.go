// Package config handles configuration management for the code review assistant.
//
// Configuration is loaded with the following priority (highest to lowest):
// 1. CLI flags (passed via command-line arguments)
// 2. Environment variables
// 3. Configuration file (YAML format)
// 4. Default values
//
// # Configuration File
//
// The configuration file uses YAML format and supports the following sections:
//
//	analysis:
//	  large_file_threshold: 500
//	  long_function_threshold: 50
//	  complexity_threshold: 10
//	  max_parameters: 5
//	  enable_coverage: true
//	  min_coverage_threshold: 50.0
//
//	output:
//	  format: console
//	  verbose: false
//
// # Usage
//
// Load configuration from file:
//
//	cfg, err := config.LoadConfig("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Apply CLI flag overrides:
//
//	overrides := map[string]interface{}{
//	    "complexity_threshold": 15,
//	    "verbose": true,
//	}
//	cfg.Merge(overrides)
package config
