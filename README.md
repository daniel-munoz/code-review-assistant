# Code Review Assistant

A CLI tool that analyzes Go codebases to provide actionable insights about code quality, complexity, and maintainability.

## Phase 1 Features

### Basic Metrics Collection
- Lines of code (LOC) per package and total
- Code, comment, and blank line counts
- Function count and average function length
- Comment ratio calculation

### Code Quality Detection
- Identify large files (>500 lines, configurable)
- Flag long functions (>50 lines, configurable)
- Calculate function length percentiles
- Generate actionable warnings

### Flexible Configuration
- YAML configuration file support
- Command-line flag overrides
- Configurable thresholds for all metrics
- Pattern-based file exclusion

### Console Reporting
- Clear, formatted summary output
- Aggregate metrics table
- Issue warnings with file locations
- Largest files listing

## Installation

```bash
go install github.com/daniel-munoz/code-review-assistant@latest
```

Or build from source:

```bash
git clone https://github.com/daniel-munoz/code-review-assistant.git
cd code-review-assistant/phase1
go build -o code-review-assistant
```

## Quick Start

Analyze your current directory:

```bash
code-review-assistant analyze .
```

Analyze a specific project:

```bash
code-review-assistant analyze /path/to/your/go/project
```

Use a custom configuration:

```bash
code-review-assistant analyze . --config custom-config.yaml
```

## Configuration

Create a `config.yaml` file in your project root or `~/.cra/config.yaml` for global settings:

```yaml
analysis:
  exclude_patterns:
    - "vendor/**"
    - "**/*_test.go"
    - "**/testdata/**"
  large_file_threshold: 500
  long_function_threshold: 50
  min_comment_ratio: 0.15

output:
  format: "console"
  verbose: false
```

## CLI Reference

### Commands

#### `analyze`
Analyze a Go codebase and generate a report.

**Usage:**
```bash
code-review-assistant analyze [path] [flags]
```

**Flags:**
- `--config, -c` - Path to config file (default: ./config.yaml)
- `--format, -f` - Output format: console (default: console)
- `--verbose, -v` - Show verbose output with per-file details
- `--exclude` - Additional exclude patterns (can be repeated)
- `--large-file-threshold` - Override large file threshold (default: 500)
- `--long-function-threshold` - Override long function threshold (default: 50)

**Examples:**
```bash
# Basic analysis
code-review-assistant analyze .

# With custom thresholds
code-review-assistant analyze . --large-file-threshold 1000 --long-function-threshold 75

# Exclude additional patterns
code-review-assistant analyze . --exclude "generated/**" --exclude "**/*.pb.go"

# Verbose mode
code-review-assistant analyze . --verbose
```

## Example Output

```
Code Review Assistant - Analysis Report
========================================

Project: /Users/you/myproject
Analyzed: 2025-12-18 15:30:45

SUMMARY
-------
Total Files:          42
Total Lines:          5,234
Code Lines:           3,845 (73.5%)
Comment Lines:        892 (17.0%)
Blank Lines:          497 (9.5%)
Total Functions:      156

AGGREGATE METRICS
-----------------
Average Function Length:      24.6 lines
Function Length (95th %ile):  68 lines
Comment Ratio:                17.0%

ISSUES FOUND (3)
----------------
[WARNING] Large file detected
  File: internal/server/handler.go
  Lines: 687
  Threshold: 500 lines

[WARNING] Long function detected
  File: pkg/processor/transform.go
  Function: ProcessComplexData
  Lines: 82
  Threshold: 50 lines

LARGEST FILES
-------------
1. internal/server/handler.go         687 lines
2. pkg/database/migrations.go         542 lines
3. internal/api/routes.go             498 lines
4. pkg/models/user.go                 456 lines
5. internal/service/auth.go           423 lines

Analysis complete.
```

## Architecture

The tool follows a clean, extensible architecture:

```
CLI (Cobra) → Orchestrator → Parser → Analyzer → Reporter
                               ↓         ↓          ↓
                          FileMetrics  Analysis  Console Output
```

### Components

- **Parser**: Walks Go AST to extract metrics (LOC, functions, imports)
- **Analyzer**: Aggregates metrics and applies thresholds
- **Reporter**: Formats and outputs results
- **Orchestrator**: Coordinates the pipeline

All major components implement interfaces for easy testing and extensibility.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Project Structure

```
code-review-assistant/
├── cmd/                    # CLI commands
│   ├── root.go
│   └── analyze.go
├── internal/
│   ├── parser/             # AST parsing and metrics extraction
│   ├── analyzer/           # Analysis and aggregation logic
│   ├── reporter/           # Report formatting and output
│   ├── orchestrator/       # Pipeline coordination
│   └── config/             # Configuration management
├── config/                 # Default configuration
├── testdata/               # Test fixtures
├── main.go                 # Entry point
└── README.md
```

## Roadmap

### Phase 2: Core Analysis (Future)
- Cyclomatic complexity calculation
- Anti-pattern detection (too many parameters, deep nesting, magic numbers)
- Dependency analysis and circular dependency detection

### Phase 3: Advanced Reporting (Future)
- Markdown report generation
- JSON export for CI/CD integration
- HTML dashboard with charts
- Historical comparison and trend analysis

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE.md for details

## Author

Daniel Munoz
