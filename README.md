# Code Review Assistant

A comprehensive CLI tool that analyzes Go codebases to provide actionable insights about code quality, complexity, maintainability, test coverage, and dependencies. Track your code quality metrics over time with historical comparison and multiple output formats.

## Features

### 📊 Code Metrics & Analysis

- **Lines of Code Analysis**: Detailed breakdown of total, code, comment, and blank lines
- **Function Metrics**: Count, average length, and 95th percentile tracking
- **Cyclomatic Complexity**: McCabe's complexity calculation with configurable thresholds
- **Large File Detection**: Identify files exceeding size thresholds
- **Long Function Detection**: Flag functions that may need refactoring

### 🔍 Anti-Pattern Detection

Modular detector system identifies common code smells:
- **Too Many Parameters**: Flags functions with excessive parameters (default: >5)
- **Deep Nesting**: Detects excessive nesting depth (default: >4 levels)
- **Too Many Returns**: Identifies functions with multiple return statements (default: >3)
- **Magic Numbers**: Detects numeric literals that should be named constants
- **Duplicate Error Handling**: Identifies repetitive `if err != nil` patterns

### ✅ Test Coverage Integration

- Automatic `go test -cover` execution for all packages
- Coverage percentage parsing and reporting
- Per-package coverage breakdown in verbose mode
- Low coverage warnings with configurable thresholds
- Graceful handling of packages without tests
- Configurable timeout for test execution

### 🔗 Dependency Analysis

- Import categorization (stdlib, internal, external)
- Per-package dependency breakdown
- Circular dependency detection using DFS algorithm
- Too many imports warnings
- Too many external dependencies warnings
- Verbose mode shows full dependency lists

### 📄 Multiple Output Formats

- **Console**: Rich terminal output with tables and color-coded severity (default)
- **Markdown**: GitHub-flavored Markdown with tables and collapsible sections
- **JSON**: Structured JSON output for programmatic consumption and CI/CD integration
- File output capability for all formats
- Pretty-print option for JSON

### 💾 Persistent Storage & Historical Tracking

- **File Storage**: JSON-based storage in `~/.cra/history/` (default)
- **SQLite Storage**: Structured database storage with efficient querying
- Project isolation using path hashing
- Automatic Git metadata extraction (commit hash, branch)
- Configurable storage backend and path
- Project-specific or global storage modes

### 📈 Historical Comparison

- Compare current analysis with previous reports
- Metric deltas with percentage changes
- Trend detection (improving/worsening/stable)
- New and fixed issues tracking
- Configurable stable threshold (default: 5%)
- Visual indicators for trends (✅ improving, 📉 worsening, ➡️ stable)
- Detailed comparison reports in all output formats

### ⚙️ Flexible Configuration

- YAML configuration file support
- Environment variable overrides (CRA_ prefix)
- CLI flag overrides
- Exclude patterns with glob support
- Per-project or global configuration
- All thresholds and settings configurable

## Installation

```bash
go install github.com/daniel-munoz/code-review-assistant@latest
```

Or build from source:

```bash
git clone https://github.com/daniel-munoz/code-review-assistant.git
cd code-review-assistant
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

Generate a Markdown report:

```bash
code-review-assistant analyze . --format markdown --output-file report.md
```

Save and compare with previous analysis:

```bash
code-review-assistant analyze . --save-report --compare
```

## Configuration

Create a `config.yaml` file in your project root or `~/.cra/config.yaml` for global settings:

```yaml
analysis:
  # File patterns to exclude
  exclude_patterns:
    - "vendor/**"
    - "**/*_test.go"
    - "**/testdata/**"
    - "**/*.pb.go"        # Generated protobuf files
    - "**/*_gen.go"       # Generated Go files

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
  detect_duplicate_errors: true  # Detect repetitive error handling

  # Test coverage
  enable_coverage: true          # Run go test -cover
  min_coverage_threshold: 50.0   # Minimum coverage percentage (0-100)
  coverage_timeout_seconds: 30   # Timeout for test execution per package

  # Dependency analysis
  max_imports: 10                # Maximum imports per package
  max_external_dependencies: 10  # Maximum external dependencies per package
  detect_circular_deps: true     # Detect circular dependencies

output:
  format: "console"              # Output format: console, markdown, json
  verbose: false                 # Show detailed per-file metrics
  output_file: ""                # Write to file instead of stdout
  json_pretty: true              # Pretty-print JSON output

# Storage configuration
storage:
  enabled: false                 # Enable persistent storage
  backend: "file"                # Storage backend: file or sqlite
  path: ""                       # Custom path (default: ~/.cra)
  auto_save: false               # Automatically save after analysis
  project_mode: false            # Use ./.cra instead of ~/.cra

# Comparison configuration
comparison:
  enabled: false                 # Enable historical comparison
  auto_compare: false            # Auto-compare with latest report
  stable_threshold: 5.0          # % change for "stable" (default: 5.0)
```

## CLI Reference

### analyze

Analyze a Go codebase and generate a report.

**Usage:**
```bash
code-review-assistant analyze [path] [flags]
```

**General Flags:**
- `--config, -c` - Path to config file (default: ./config.yaml or ~/.cra/config.yaml)
- `--verbose, -v` - Show verbose output with per-file and per-package details
- `--exclude` - Additional exclude patterns (can be repeated)

**Analysis Thresholds:**
- `--large-file-threshold` - Override large file threshold in lines (default: 500)
- `--long-function-threshold` - Override long function threshold in lines (default: 50)
- `--complexity-threshold` - Override cyclomatic complexity threshold (default: 10)
- `--max-parameters` - Maximum function parameters before flagging (default: 5)
- `--max-nesting-depth` - Maximum nesting depth before flagging (default: 4)
- `--max-return-statements` - Maximum return statements per function (default: 3)

**Coverage Flags:**
- `--enable-coverage` - Enable/disable test coverage analysis (default: true)
- `--min-coverage-threshold` - Minimum coverage percentage 0-100 (default: 50)
- `--coverage-timeout` - Test execution timeout per package in seconds (default: 30)

**Dependency Flags:**
- `--max-imports` - Maximum imports per package (default: 10)
- `--max-external-dependencies` - Maximum external dependencies per package (default: 10)
- `--detect-circular-deps` - Enable/disable circular dependency detection (default: true)

**Output & Storage Flags:**
- `--format, -f` - Output format: console, markdown, json (default: console)
- `--output-file, -o` - Write output to file instead of stdout
- `--json-pretty` - Pretty-print JSON output (default: true)
- `--save-report` - Save report to storage for historical tracking
- `--compare` - Compare with previous report from storage
- `--storage-backend` - Storage backend: file or sqlite (default: file)
- `--storage-path` - Custom storage path (default: ~/.cra)

## Usage Examples

### Basic Analysis

```bash
# Analyze current directory
code-review-assistant analyze .

# Analyze specific project
code-review-assistant analyze /path/to/project

# Verbose mode with detailed per-file metrics
code-review-assistant analyze . --verbose
```

### Custom Thresholds

```bash
# Strict thresholds for high-quality codebases
code-review-assistant analyze . \
  --large-file-threshold 300 \
  --long-function-threshold 30 \
  --complexity-threshold 8 \
  --min-coverage-threshold 80

# Relaxed thresholds for legacy codebases
code-review-assistant analyze . \
  --large-file-threshold 1000 \
  --complexity-threshold 15 \
  --min-coverage-threshold 30
```

### Output Formats

```bash
# Generate Markdown report
code-review-assistant analyze . --format markdown --output-file report.md

# Generate JSON report for CI/CD
code-review-assistant analyze . --format json --output-file report.json

# Compact JSON output
code-review-assistant analyze . --format json --json-pretty=false
```

### Historical Tracking

```bash
# Save report to file storage (default)
code-review-assistant analyze . --save-report

# Use SQLite for better querying
code-review-assistant analyze . --save-report --storage-backend sqlite

# Save and compare with previous report
code-review-assistant analyze . --save-report --compare

# Custom storage location
code-review-assistant analyze . --save-report --storage-path /custom/path
```

### Advanced Workflows

```bash
# Exclude additional patterns
code-review-assistant analyze . --exclude "generated/**" --exclude "**/*.pb.go"

# Disable coverage for faster analysis
code-review-assistant analyze . --enable-coverage=false

# Strict dependency checking
code-review-assistant analyze . --max-imports 5 --max-external-dependencies 3

# Complete workflow with Markdown output and comparison
code-review-assistant analyze . \
  --format markdown \
  --output-file report.md \
  --save-report \
  --compare \
  --verbose
```

## Example Output

### Console Output

```
Code Review Assistant - Analysis Report
============================================================

Project: /Users/you/myproject
Analyzed: 2025-12-20 16:30:00

COMPARISON WITH PREVIOUS REPORT
------------------------------------------------------------
Previous: 2025-12-19 10:15:00

Complexity:    ✅  IMPROVING
Coverage:      ➡️  STABLE
Issue Count:   📉  WORSENING

      METRIC     | PREVIOUS | CURRENT |   CHANGE
-----------------+----------+---------+--------------
  Files          |       40 |      42 | +2 (+5.0%)
  Lines          |    4,892 |   5,234 | +342 (+7.0%)
  Functions      |      148 |     156 | +8 (+5.4%)
  Avg Complexity |     4.50 |    4.20 | -0.30 (-6.7%)
  Avg Coverage   |   67.5%  |  68.5%  | +1.00 (+1.5%)
  Issues         |       12 |      15 | +3 (+25.0%)

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
⚠️  [WARNING] Package has low test coverage
  Package: myproject/internal/api
  Coverage: 35.0% (threshold: 50.0%)

⚠️  [WARNING] Function has high cyclomatic complexity
  File: internal/processor/transform.go:45
  Function: ProcessComplexData
  Complexity: 15 (threshold: 10)

⚠️  [WARNING] Function has too many parameters
  File: internal/server/handler.go:123
  Function: HandleRequest
  Parameters: 7 (threshold: 5)

ℹ️  [INFO] Magic number should be replaced with a named constant: 1000
  File: pkg/utils/helpers.go:42
  Function: CalculateLimit

LARGEST FILES
------------------------------------------------------------
  1.  internal/server/handler.go         687 lines
  2.  pkg/database/migrations.go         542 lines
  3.  internal/api/routes.go             498 lines

MOST COMPLEX FUNCTIONS
------------------------------------------------------------
  1.  ProcessComplexData    CC=15  82 lines   internal/processor/transform.go
  2.  ValidateInput         CC=12  54 lines   pkg/validator/rules.go
  3.  HandleRequest         CC=11  95 lines   internal/server/handler.go

TEST COVERAGE
------------------------------------------------------------
  Average Coverage          68.5%
  Total Packages            8
  Packages Below Threshold  2

DEPENDENCIES
------------------------------------------------------------
  Total Packages                      8
  Packages with High Imports          1
  Packages with High External Deps    0

Analysis complete.
```

## Architecture

The tool follows a clean, extensible architecture:

```
CLI (Cobra) → Orchestrator → Parser → Analyzer → Reporter
                  ↓             ↓         ↓          ↓
              Storage      FileMetrics  Issues   Multiple
                  ↓                       ↓       Formats
              Comparison              Detectors
                                      Coverage
                                      Dependencies
```

### Components

- **Parser**: Walks Go AST to extract metrics (LOC, functions, imports, complexity)
- **Analyzer**: Aggregates metrics, applies thresholds, coordinates sub-analyzers
  - **Detectors Registry**: Modular anti-pattern detection system
  - **Coverage Runner**: Executes `go test -cover` and parses results
  - **Dependency Analyzer**: Categorizes imports and detects circular dependencies
- **Reporter**: Formats and outputs results in console, Markdown, or JSON
- **Storage**: Persists reports for historical tracking (file or SQLite backend)
- **Comparator**: Compares current and previous reports, detects trends
- **Orchestrator**: Coordinates the complete analysis pipeline
- **Config**: Multi-level configuration (defaults → file → env → CLI flags)

All major components implement interfaces for easy testing and extensibility.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# View coverage by package
go test -cover ./... | grep coverage

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Project Structure

```
code-review-assistant/
├── cmd/                      # CLI commands
│   ├── root.go
│   └── analyze.go
├── internal/
│   ├── parser/               # AST parsing and metrics extraction
│   ├── analyzer/             # Analysis and aggregation logic
│   │   └── detectors/        # Anti-pattern detectors
│   ├── coverage/             # Test coverage analysis
│   ├── dependencies/         # Dependency analysis
│   ├── comparison/           # Historical comparison logic
│   ├── reporter/             # Report formatting (console, markdown, JSON)
│   ├── storage/              # Persistent storage (file, SQLite)
│   ├── git/                  # Git metadata extraction
│   ├── orchestrator/         # Pipeline coordination
│   ├── config/               # Configuration management
│   └── constants/            # Shared constants
├── config/                   # Default configuration
│   └── config.yaml
├── examples/                 # Sample output files
│   ├── sample-report.md
│   └── sample-report.json
├── testdata/                 # Test fixtures
│   ├── sample/               # Sample code for testing
│   └── coverage/             # Test coverage samples
├── main.go                   # Entry point
└── README.md
```

### Test Coverage

Current test coverage by package:
- Comparison: **100%**
- Config: **100%**
- Git: **100%**
- Dependencies: **93.8%**
- Parser: **86.9%**
- Storage: **75.1%**
- Reporter: **66.0%**
- Analyzer: **66.7%**

## CI/CD Integration

The JSON output format makes it easy to integrate with CI/CD pipelines:

```bash
# GitHub Actions example
- name: Analyze Code Quality
  run: |
    code-review-assistant analyze . \
      --format json \
      --output-file report.json \
      --save-report \
      --compare

    # Parse JSON and fail if metrics worsen
    # (Add custom script to check trends)
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

### Adding New Detectors

The detector system is modular and easy to extend:

```go
// internal/analyzer/detectors/your_detector.go
func init() {
    Register("your_detector", &YourDetector{})
}

type YourDetector struct{}

func (d *YourDetector) Detect(file *parser.FileMetrics, cfg *config.AnalysisConfig) []*analyzer.Issue {
    // Your detection logic
    return issues
}
```

## License

MIT License - see LICENSE.md for details

## Author

Daniel Munoz
