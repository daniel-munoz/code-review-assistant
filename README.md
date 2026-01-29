# Code Review Assistant

![Version](https://img.shields.io/badge/version-1.2.0-beta-blue)
![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8)
![License](https://img.shields.io/badge/license-MIT-green)

**Status: Production Ready (v1.2 beta)**

A comprehensive CLI tool that analyzes codebases to provide actionable insights about code quality, complexity, maintainability, test coverage, and dependencies. Supports **multiple programming languages** with automatic language detection. Track your code quality metrics over time with historical comparison and multiple output formats.

## Supported Languages

| Language | Parsing | Metrics | Anti-Patterns | Coverage | Dependencies |
|----------|---------|---------|---------------|----------|--------------|
| **Go** | Go AST | Full | Full | Yes | Yes |
| **Python** | tree-sitter | Full | Full | Planned | Planned |

More languages coming soon (TypeScript, JavaScript, Rust).

## Features

### Multi-Language Support

- **Automatic Detection**: Detects project language from manifest files (`go.mod`, `pyproject.toml`, `requirements.txt`, `setup.py`)
- **Explicit Selection**: Use `--language` flag to specify language manually
- **Extensible Architecture**: Language provider system makes adding new languages straightforward

### Code Metrics & Analysis

- **Lines of Code Analysis**: Detailed breakdown of total, code, comment, and blank lines
- **Function Metrics**: Count, average length, and 95th percentile tracking
- **Cyclomatic Complexity**: Complexity calculation with configurable thresholds
- **Large File Detection**: Identify files exceeding size thresholds
- **Long Function Detection**: Flag functions that may need refactoring

### Anti-Pattern Detection

Modular detector system identifies common code smells:
- **Too Many Parameters**: Flags functions with excessive parameters (default: >5)
- **Deep Nesting**: Detects excessive nesting depth (default: >4 levels)
- **Too Many Returns**: Identifies functions with multiple return statements (default: >3)
- **Magic Numbers**: Detects numeric literals that should be named constants
- **Duplicate Error Handling**: Identifies repetitive error patterns (Go)

### Test Coverage Integration (Go)

- Automatic `go test -cover` execution for all packages
- Coverage percentage parsing and reporting
- Per-package coverage breakdown in verbose mode
- Low coverage warnings with configurable thresholds
- Graceful handling of packages without tests
- Configurable timeout for test execution

### Dependency Analysis (Go)

- Import categorization (stdlib, internal, external)
- Per-package dependency breakdown
- Circular dependency detection using DFS algorithm
- Too many imports warnings
- Too many external dependencies warnings
- Verbose mode shows full dependency lists

### Multiple Output Formats

- **Console**: Rich terminal output with tables and color-coded severity (default)
- **Markdown**: GitHub-flavored Markdown with tables and collapsible sections
- **JSON**: Structured JSON output for programmatic consumption and CI/CD integration
- **HTML**: Interactive dashboard with charts, heatmaps, and dependency graphs
- File output capability for all formats
- Pretty-print option for JSON

### Persistent Storage & Historical Tracking

- **File Storage**: JSON-based storage in `~/.cra/history/` (default)
- **SQLite Storage**: Structured database storage with efficient querying
- Project isolation using path hashing
- Automatic Git metadata extraction (commit hash, branch)
- Configurable storage backend and path
- Project-specific or global storage modes

### Historical Comparison

- Compare current analysis with previous reports
- Metric deltas with percentage changes
- Trend detection (improving/worsening/stable)
- New and fixed issues tracking
- Configurable stable threshold (default: 5%)
- Visual indicators for trends

### Flexible Configuration

- YAML configuration file support
- Environment variable overrides (CRA_ prefix)
- CLI flag overrides
- Exclude patterns with glob support
- Per-project or global configuration
- All thresholds and settings configurable

## Installation

Install the latest stable release:

```bash
go install github.com/daniel-munoz/code-review-assistant@v1.2.0-beta
```

Or build from source:

```bash
git clone https://github.com/daniel-munoz/code-review-assistant.git
cd code-review-assistant
go build -o code-review-assistant
```

**Note**: Building requires CGO for tree-sitter support. Make sure you have a C compiler available.

## Quick Start

### Analyze with Auto-Detection

The tool automatically detects the project language:

```bash
# Go project (detected via go.mod)
code-review-assistant analyze /path/to/go/project

# Python project (detected via requirements.txt, pyproject.toml, or setup.py)
code-review-assistant analyze /path/to/python/project
```

### Explicit Language Selection

```bash
# Explicitly analyze as Go
code-review-assistant analyze . --language go

# Explicitly analyze as Python
code-review-assistant analyze . --language python
```

### Output Formats

```bash
# Console output (default)
code-review-assistant analyze .

# Markdown report
code-review-assistant analyze . --format markdown --output-file report.md

# JSON for CI/CD
code-review-assistant analyze . --format json --output-file report.json

# Interactive HTML dashboard
code-review-assistant analyze . --format html --output-file dashboard.html
```

### Historical Tracking

```bash
# Save and compare with previous analysis
code-review-assistant analyze . --save-report --compare
```

## Language-Specific Features

### Go

Full-featured analysis including:
- Go AST-based parsing
- Test coverage via `go test -cover`
- Dependency analysis with circular dependency detection
- Duplicate error handling detection (`if err != nil` patterns)

Default exclude patterns:
```
vendor/**
**/*_test.go
**/testdata/**
**/*.pb.go
```

### Python

Tree-sitter based parsing with:
- Function and class extraction
- Method detection with class association
- Docstring handling for comment counting
- Comprehension complexity (list, dict, set, generator)
- Match statement support (Python 3.10+)

Default exclude patterns:
```
**/__pycache__/**
**/venv/**
**/.venv/**
**/site-packages/**
**/test_*.py
**/*_test.py
**/conftest.py
**/tests/**
```

## Configuration

Create a `config.yaml` file in your project root or `~/.cra/config.yaml` for global settings:

```yaml
# Language selection (auto, go, python)
language: "auto"

analysis:
  # File patterns to exclude (merged with language defaults)
  exclude_patterns:
    - "generated/**"

  # Code quality thresholds
  large_file_threshold: 500      # Lines
  long_function_threshold: 50    # Lines
  min_comment_ratio: 0.15        # 15%
  complexity_threshold: 10       # Cyclomatic complexity

  # Anti-pattern detection
  max_parameters: 5              # Maximum function parameters
  max_nesting_depth: 4           # Maximum nesting depth
  max_return_statements: 3       # Maximum return statements per function
  detect_magic_numbers: true     # Detect numeric literals

  # Go-specific settings
  enable_coverage: true          # Run go test -cover
  min_coverage_threshold: 50.0   # Minimum coverage percentage (0-100)
  coverage_timeout_seconds: 30   # Timeout for test execution per package
  max_imports: 10                # Maximum imports per package
  max_external_dependencies: 10  # Maximum external dependencies per package
  detect_circular_deps: true     # Detect circular dependencies
  detect_duplicate_errors: true  # Detect repetitive error handling

output:
  format: "console"              # Output format: console, markdown, json, html
  verbose: false                 # Show detailed per-file metrics
  output_file: ""                # Write to file instead of stdout
  json_pretty: true              # Pretty-print JSON output

storage:
  enabled: false                 # Enable persistent storage
  backend: "file"                # Storage backend: file or sqlite
  path: ""                       # Custom path (default: ~/.cra)
  auto_save: false               # Automatically save after analysis
  project_mode: false            # Use ./.cra instead of ~/.cra

comparison:
  enabled: false                 # Enable historical comparison
  auto_compare: false            # Auto-compare with latest report
  stable_threshold: 5.0          # % change for "stable" (default: 5.0)
```

## CLI Reference

### analyze

Analyze a codebase and generate a report.

**Usage:**
```bash
code-review-assistant analyze [path] [flags]
```

**Language Flags:**
- `--language, -l` - Language to analyze: auto, go, python (default: auto)

**General Flags:**
- `--config, -c` - Path to config file (default: ./config.yaml or ~/.cra/config.yaml)
- `--verbose, -v` - Show verbose output with per-file and per-package details
- `--exclude` - Additional exclude patterns (can be repeated)
- `--quiet, -q` - Disable live status reporting

**Analysis Thresholds:**
- `--large-file-threshold` - Override large file threshold in lines (default: 500)
- `--long-function-threshold` - Override long function threshold in lines (default: 50)
- `--complexity-threshold` - Override cyclomatic complexity threshold (default: 10)

**Output & Storage Flags:**
- `--format, -f` - Output format: console, markdown, json, html (default: console)
- `--output-file, -o` - Write output to file instead of stdout
- `--json-pretty` - Pretty-print JSON output (default: true)
- `--save-report` - Save report to storage for historical tracking
- `--compare` - Compare with previous report from storage
- `--storage-backend` - Storage backend: file or sqlite (default: file)
- `--storage-path` - Custom storage path (default: ~/.cra)

## Example Output

### Console Output

```
Code Review Assistant - Analysis Report
============================================================

Project: /path/to/project
Analyzed: 2026-01-29 10:30:00

SUMMARY
------------------------------------------------------------
      METRIC          VALUE
------------------+--------------
  Total Files      42
  Total Lines      5,234
  Code Lines       3,845 (73.5%)
  Comment Lines    892 (17.0%)
  Blank Lines      497 (9.5%)
  Total Functions  156

AGGREGATE METRICS
------------------------------------------------------------
  Average Function Length       24.6 lines
  Function Length (95th %ile)   68 lines
  Comment Ratio                 17.0%
  Average Complexity            4.2
  Complexity (95th %ile)        12

ISSUES FOUND (15)
------------------------------------------------------------
⚠️  [WARNING] Function has high cyclomatic complexity
  File: internal/processor/transform.py:45
  Function: process_complex_data
  Complexity: 15 (threshold: 10)

⚠️  [WARNING] Function has too many parameters
  File: internal/server/handler.py:123
  Function: handle_request
  Parameters: 7 (threshold: 5)

ℹ️  [INFO] Magic number should be replaced with a named constant: 1000
  File: pkg/utils/helpers.py:42
  Function: calculate_limit

LARGEST FILES
------------------------------------------------------------
  1.  internal/server/handler.py         687 lines
  2.  pkg/database/migrations.py         542 lines
  3.  internal/api/routes.py             498 lines

MOST COMPLEX FUNCTIONS
------------------------------------------------------------
  1.  process_complex_data    CC=15  82 lines   internal/processor/transform.py
  2.  validate_input          CC=12  54 lines   pkg/validator/rules.py
  3.  handle_request          CC=11  95 lines   internal/server/handler.py

Analysis complete.
```

## Architecture

The tool follows a clean, extensible architecture with a language provider system:

```
CLI (Cobra) → Orchestrator → Language Provider → Parser → Analyzer → Reporter
                  ↓                  ↓               ↓         ↓          ↓
              Storage           Detector Runner  FileMetrics  Issues   Multiple
                  ↓                                  ↓                  Formats
              Comparison                         Coverage Runner
                                                 Dependency Analyzer
```

### Components

- **Language Provider**: Abstracts language-specific parsing and analysis
  - **Go Provider**: Uses Go's `go/ast` package
  - **Python Provider**: Uses tree-sitter for parsing
- **Parser**: Extracts metrics (LOC, functions, imports, complexity) per language
- **Analyzer**: Aggregates metrics, applies thresholds, coordinates sub-analyzers
  - **Detector Runner**: Language-specific anti-pattern detection
  - **Coverage Runner**: Language-specific test coverage (Go only currently)
  - **Dependency Analyzer**: Language-specific dependency analysis (Go only currently)
- **Reporter**: Formats and outputs results in console, Markdown, JSON, or HTML
- **Storage**: Persists reports for historical tracking (file or SQLite backend)
- **Comparator**: Compares current and previous reports, detects trends
- **Orchestrator**: Coordinates the complete analysis pipeline
- **Config**: Multi-level configuration (defaults → file → env → CLI flags)

All major components implement interfaces for easy testing and extensibility.

## Project Structure

```
code-review-assistant/
├── cmd/                          # CLI commands
│   ├── root.go
│   └── analyze.go
├── internal/
│   ├── language/                 # Language provider system
│   │   ├── language.go           # Core interfaces
│   │   ├── registry.go           # Language registration & detection
│   │   ├── golang/               # Go language support
│   │   │   ├── provider.go
│   │   │   └── detectors.go
│   │   └── python/               # Python language support
│   │       ├── provider.go
│   │       ├── parser.go
│   │       └── detectors.go
│   ├── parser/                   # Go AST parsing (legacy, used by Go provider)
│   ├── analyzer/                 # Analysis and aggregation logic
│   │   └── detectors/            # Anti-pattern detectors
│   ├── coverage/                 # Test coverage analysis (Go)
│   ├── dependencies/             # Dependency analysis (Go)
│   ├── comparison/               # Historical comparison logic
│   ├── reporter/                 # Report formatting
│   ├── storage/                  # Persistent storage
│   ├── git/                      # Git metadata extraction
│   ├── orchestrator/             # Pipeline coordination
│   ├── config/                   # Configuration management
│   ├── status/                   # Live status reporting
│   └── constants/                # Shared constants
├── testdata/
│   ├── sample/                   # Go test fixtures
│   └── python/                   # Python test fixtures
├── main.go
└── README.md
```

## Development

### Running Tests

```bash
# Run all tests (fast - skips slow integration tests)
go test -short ./...

# Run all tests including slow integration tests
go test ./...

# Run with coverage
go test -short -cover ./...

# Generate HTML coverage report
go test -short -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Adding a New Language

1. Create a new package under `internal/language/<lang>/`
2. Implement the `language.Language` interface:
   - `Name()` - Language identifier (e.g., "typescript")
   - `DisplayName()` - Human-readable name (e.g., "TypeScript")
   - `Extensions()` - File extensions (e.g., []string{".ts", ".tsx"})
   - `DefaultExcludePatterns()` - Language-specific excludes
   - `Parser()` - Returns a `parser.Parser` implementation
   - `DetectorRunner()` - Returns anti-pattern detector
   - `CoverageRunner()` - Returns coverage analyzer (or nil)
   - `DependencyAnalyzer()` - Returns dependency analyzer (or nil)
3. Register in `init()` via `language.Register(&YourLanguage{})`
4. Import the package in `internal/orchestrator/orchestrator.go`
5. Update language detection in `internal/language/registry.go`

## CI/CD Integration

The JSON output format makes it easy to integrate with CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Analyze Code Quality
  run: |
    code-review-assistant analyze . \
      --format json \
      --output-file report.json \
      --save-report \
      --compare
```

## Troubleshooting

### Language Detection Fails

If auto-detection doesn't find your project language:

```bash
# Use explicit language selection
code-review-assistant analyze . --language python
```

Or add a manifest file (`go.mod`, `requirements.txt`, etc.) to your project root.

### CGO Errors During Build

Python support requires tree-sitter which needs CGO:

```bash
# Ensure CGO is enabled
CGO_ENABLED=1 go build -o code-review-assistant

# On macOS, install Xcode command line tools
xcode-select --install

# On Linux, install build-essential
sudo apt-get install build-essential
```

### Exclude Generated Files

Add to config.yaml or use CLI:

```bash
code-review-assistant analyze . --exclude "generated/**" --exclude "**/*.pb.go"
```

### Storage Issues

If you encounter storage errors, try using a custom path:

```bash
code-review-assistant analyze . --save-report --storage-path /custom/path
```

## Roadmap

### v2.1 (Planned)
- Python test coverage via pytest-cov
- Python dependency analysis via requirements.txt/pyproject.toml

### v2.2 (Planned)
- TypeScript/JavaScript support
- Rust support

### Future
- Git history integration and code churn detection
- Multi-repository analysis
- Custom rules engine with plugin support

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE.md for details

## Author

Daniel Munoz
