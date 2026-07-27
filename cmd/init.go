package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Flags for init command
	initLocal bool
	initForce bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a configuration file with default settings",
	Long: `Create a configuration file with default settings for code-review-assistant.

By default, creates ~/.cra/config.yaml (global configuration).
Use --local to create ./config.yaml in the current directory (project-specific).

The generated file includes all available settings with documentation.
You can customize it to adjust thresholds, exclude patterns, and output preferences.

Example usage:
  code-review-assistant init              # Create ~/.cra/config.yaml
  code-review-assistant init --local      # Create ./config.yaml
  code-review-assistant init --force      # Overwrite existing config`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initLocal, "local", false, "create config.yaml in current directory instead of ~/.cra/")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
}

func runInit(cmd *cobra.Command, args []string) error {
	var configPath string

	if initLocal {
		configPath = "config.yaml"
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		craDir := filepath.Join(homeDir, ".cra")

		// Create ~/.cra directory if it doesn't exist
		if err := os.MkdirAll(craDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", craDir, err)
		}

		configPath = filepath.Join(craDir, "config.yaml")
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		if !initForce {
			return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
		}
	}

	// Write the default config
	if err := os.WriteFile(configPath, []byte(defaultConfigYAML), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created configuration file: %s\n", configPath)
	fmt.Println("\nYou can now customize the settings in this file.")
	fmt.Println("Run 'code-review-assistant analyze <path>' to analyze a project.")

	return nil
}

// defaultConfigYAML is the default configuration file content with documentation
const defaultConfigYAML = `# Code Review Assistant Configuration
# ===================================
# This file configures the behavior of code-review-assistant.
# All settings shown are the defaults - uncomment and modify as needed.

# Language to analyze
# Options: auto, go, python, javascript, kotlin
# "auto" will detect the language from project files (go.mod, package.json, etc.)
language: auto

# Analysis settings
analysis:
  # Patterns to exclude from analysis (glob patterns)
  # Language-specific patterns (node_modules, __pycache__, etc.) are added automatically
  exclude_patterns:
    - "vendor/**"
    - "**/*_test.go"
    - "**/testdata/**"
    - "**/*.pb.go"

  # Thresholds for identifying issues
  large_file_threshold: 500       # Lines - files larger than this trigger a warning
  long_function_threshold: 50     # Lines - functions longer than this trigger a warning
  complexity_threshold: 10        # Cyclomatic complexity threshold
  min_comment_ratio: 0.15         # Minimum comment ratio (0.15 = 15%)

  # Anti-pattern detection settings
  max_parameters: 5               # Max function parameters before warning
  max_nesting_depth: 4            # Max nesting depth before warning
  max_return_statements: 3        # Max return statements before warning (info level)
  detect_magic_numbers: true      # Flag numeric literals that should be constants
  detect_duplicate_errors: true   # Detect duplicated error handling
  detect_non_null_assertions: true # Detect !! usage (Kotlin)
  detect_run_blocking: true        # Detect runBlocking outside main (Kotlin)

  # Test coverage settings
  enable_coverage: true           # Run test coverage analysis
  min_coverage_threshold: 0       # Minimum required coverage percentage (0 = no minimum)
  coverage_timeout_seconds: 60    # coverage timeout in seconds (per package for Go; whole run for JS and Kotlin/Gradle)

  # Dependency analysis settings
  max_imports: 20                 # Max imports per file before warning
  max_external_dependencies: 50   # Max external dependencies before warning
  detect_circular_deps: true      # Detect circular import dependencies

# Output settings
output:
  format: console                 # Output format: console, markdown, json, html
  verbose: false                  # Show detailed per-file metrics
  output_file: ""                 # Write to file instead of stdout (empty = stdout)
  json_pretty: true               # Pretty-print JSON output
  quiet: false                    # Suppress progress output
  show_status: true               # Show live status during analysis

# Storage settings (for historical tracking)
storage:
  enabled: false                  # Enable report storage for historical comparison
  backend: file                   # Storage backend: file, sqlite
  path: ""                        # Storage path (empty = default ~/.cra or ./.cra)
  auto_save: false                # Automatically save reports after analysis
  project_mode: false             # Use project-local storage (./.cra) instead of global

# Comparison settings (for tracking changes over time)
comparison:
  enabled: false                  # Enable comparison with previous reports
  auto_compare: false             # Automatically compare with last report
  stable_threshold: 5.0           # Percentage change to consider "stable"
`
