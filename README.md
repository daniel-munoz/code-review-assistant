# Code Review Assistant

A CLI tool that analyzes Go codebases to provide actionable insights about code quality, complexity, maintainability, test coverage, and dependencies.

## Features

### Phase 1: Basic Metrics

- **Basic Metrics Collection**: LOC, code/comment/blank line counts, function counts
- **Code Quality Detection**: Large files, long functions, function length percentiles
- **Flexible Configuration**: YAML config, CLI overrides, pattern exclusions
- **Console Reporting**: Summary tables, aggregate metrics, issue warnings, largest files

### Phase 2: Advanced Analysis (✅ Complete)

#### 1. Enhanced Cyclomatic Complexity
- McCabe's complexity calculation for all functions
- Average complexity and 95th percentile tracking
- Top 10 most complex functions reporting
- Configurable complexity threshold (default: 10)
- High complexity warnings with function location

#### 2. Anti-Pattern Detection
- **Too Many Parameters**: Flags functions with >5 parameters (configurable)
- **Deep Nesting**: Detects nesting depth >4 levels (configurable)
- **Too Many Returns**: Flags functions with >3 return statements (configurable)
- **Magic Numbers**: Detects numeric literals (excludes 0, 1, -1)
- **Duplicate Error Handling**: Identifies repetitive `if err != nil` patterns (>5)
- Modular detector system with registry pattern for easy extension

#### 3. Test Coverage Integration
- Automatic `go test -cover` execution for all packages
- Coverage percentage parsing and reporting
- Low coverage warnings (default threshold: 50%)
- Per-package coverage breakdown in verbose mode
- Configurable timeout for test execution
- Graceful handling of packages without tests

#### 4. Dependency Analysis
- Import categorization (stdlib, internal, external)
- Per-package dependency breakdown
- Circular dependency detection using DFS algorithm
- Too many imports warnings (default: 10)
- Too many external dependencies warnings (default: 10)
- Verbose mode shows full dependency lists

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
  # File patterns to exclude
  exclude_patterns:
    - "vendor/**"
    - "**/*_test.go"
    - "**/testdata/**"
    - "**/*.pb.go"        # Generated protobuf files
    - "**/*_gen.go"       # Generated Go files

  # Phase 1: Basic thresholds
  large_file_threshold: 500      # Lines
  long_function_threshold: 50    # Lines
  min_comment_ratio: 0.15        # 15%

  # Phase 2.1: Complexity
  complexity_threshold: 10       # Cyclomatic complexity

  # Phase 2.2: Anti-pattern detection
  max_parameters: 5              # Maximum function parameters
  max_nesting_depth: 4           # Maximum nesting depth
  max_return_statements: 3       # Maximum return statements per function
  detect_magic_numbers: true     # Detect numeric literals
  detect_duplicate_errors: true  # Detect repetitive error handling

  # Phase 2.3: Test coverage
  enable_coverage: true          # Run go test -cover
  min_coverage_threshold: 50.0   # Minimum coverage percentage (0-100)
  coverage_timeout_seconds: 30   # Timeout for test execution per package

  # Phase 2.4: Dependency analysis
  max_imports: 10                # Maximum imports per package
  max_external_dependencies: 10  # Maximum external dependencies per package
  detect_circular_deps: true     # Detect circular dependencies

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

*General:*
- `--config, -c` - Path to config file (default: ./config.yaml)
- `--format, -f` - Output format: console (default: console)
- `--verbose, -v` - Show verbose output with per-file and per-package details
- `--exclude` - Additional exclude patterns (can be repeated)

*Phase 1 Thresholds:*
- `--large-file-threshold` - Override large file threshold (default: 500)
- `--long-function-threshold` - Override long function threshold (default: 50)

*Phase 2 Thresholds:*
- `--complexity-threshold` - Override cyclomatic complexity threshold (default: 10)
- `--enable-coverage` - Enable/disable test coverage analysis (default: true)
- `--min-coverage-threshold` - Minimum coverage percentage 0-100 (default: 50)
- `--coverage-timeout` - Test execution timeout per package in seconds (default: 30)
- `--max-imports` - Maximum imports per package (default: 10)
- `--max-external-dependencies` - Maximum external dependencies per package (default: 10)
- `--detect-circular-deps` - Enable/disable circular dependency detection (default: true)

**Examples:**
```bash
# Basic analysis with all Phase 2 features
code-review-assistant analyze .

# With custom Phase 1 thresholds
code-review-assistant analyze . --large-file-threshold 1000 --long-function-threshold 75

# With custom Phase 2 thresholds
code-review-assistant analyze . --complexity-threshold 15 --min-coverage-threshold 80

# Exclude additional patterns
code-review-assistant analyze . --exclude "generated/**" --exclude "**/*.pb.go"

# Verbose mode (shows per-file, per-package coverage, and dependency details)
code-review-assistant analyze . --verbose

# Disable coverage analysis for faster execution
code-review-assistant analyze . --enable-coverage=false

# Strict dependency checking
code-review-assistant analyze . --max-imports 5 --max-external-dependencies 3
```

## Example Output

```
Code Review Assistant - Analysis Report
============================================================

Project: /Users/you/myproject
Analyzed: 2025-12-19 13:28:56

SUMMARY
------------------------------------------------------------
Total Files:          42
Total Lines:          5,234
Code Lines:           3,845 (73.5%)
Comment Lines:        892 (17.0%)
Blank Lines:          497 (9.5%)
Total Functions:      156

AGGREGATE METRICS
------------------------------------------------------------
Average Function Length       24.6 lines
Function Length (95th %%ile)  68 lines
Comment Ratio                 17.0%
Average Complexity            4.2
Complexity (95th %%ile)       12

ISSUES FOUND (15)
------------------------------------------------------------
⚠️  [WARNING] Package myproject/internal/api has low test coverage
  Coverage: 35 (threshold: 50)

⚠️  [WARNING] Function has high cyclomatic complexity
  File: internal/processor/transform.go:45
  Function: ProcessComplexData
  Complexity: 15 (threshold: 10)

⚠️  [WARNING] Function has too many parameters
  File: internal/server/handler.go:123
  Function: HandleRequest
  Parameters: 7 (threshold: 5)

⚠️  [WARNING] Function has deep nesting (depth: 5)
  File: pkg/validator/rules.go:67
  Function: ValidateComplex
  Nesting Depth: 5 (threshold: 4)

ℹ️  [INFO] Magic number should be replaced with a named constant: 1000
  File: pkg/utils/helpers.go:42
  Function: CalculateLimit

LARGEST FILES
------------------------------------------------------------
1.   internal/server/handler.go         687 lines
2.   pkg/database/migrations.go         542 lines
3.   internal/api/routes.go             498 lines

MOST COMPLEX FUNCTIONS
------------------------------------------------------------
1.   ProcessComplexData    CC=15  82 lines   internal/processor/transform.go
2.   ValidateInput         CC=12  54 lines   pkg/validator/rules.go
3.   HandleRequest          CC=11  95 lines   internal/server/handler.go

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
                               ↓         ↓          ↓
                          FileMetrics  Analysis  Console Output
                                         ↓
                               ┌─────────┼─────────┐
                               ↓         ↓         ↓
                          Detectors  Coverage  Dependencies
```

### Components

- **Parser**: Walks Go AST to extract metrics (LOC, functions, imports, complexity)
- **Analyzer**: Aggregates metrics, applies thresholds, coordinates sub-analyzers
  - **Detectors Registry**: Modular anti-pattern detection system
  - **Coverage Runner**: Executes `go test -cover` and parses results
  - **Dependency Analyzer**: Categorizes imports and detects circular dependencies
- **Reporter**: Formats and outputs comprehensive results
- **Orchestrator**: Coordinates the analysis pipeline
- **Config**: Multi-level configuration (file → env → CLI flags)

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
├── cmd/                      # CLI commands
│   ├── root.go
│   └── analyze.go
├── internal/
│   ├── parser/               # AST parsing and metrics extraction
│   ├── analyzer/             # Analysis and aggregation logic
│   │   └── detectors/        # Anti-pattern detectors (Phase 2.2)
│   ├── coverage/             # Test coverage analysis (Phase 2.3)
│   ├── dependencies/         # Dependency analysis (Phase 2.4)
│   ├── reporter/             # Report formatting and output
│   ├── orchestrator/         # Pipeline coordination
│   └── config/               # Configuration management
├── config/                   # Default configuration
│   └── config.yaml
├── testdata/                 # Test fixtures
│   ├── sample/               # Sample code for testing
│   └── coverage/             # Test coverage samples
├── main.go                   # Entry point
├── CLAUDE.md                 # Implementation progress tracking
└── README.md
```

## Roadmap

### ✅ Phase 1: Basic Metrics (Complete)
- Lines of code analysis
- Function metrics
- Comment ratio calculation
- Large file and long function detection

### ✅ Phase 2: Advanced Analysis (Complete - 2025-12-19)
- Enhanced cyclomatic complexity calculation
- Anti-pattern detection (5 detectors)
- Test coverage integration with `go test -cover`
- Dependency analysis and circular dependency detection

### Phase 3: Advanced Reporting (Planned)
- Markdown report generation
- JSON export for CI/CD integration
- HTML dashboard with charts and visualizations
- Historical comparison and trend analysis
- GitHub Actions integration
- Custom report templates

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE.md for details

## Author

Daniel Munoz
