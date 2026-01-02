# Gone - AI Agent Guidelines

## Project Overview

Gone is a fast, concurrent dead link detector for documentation files. It scans Markdown, JSON, YAML, TOML, and XML files for HTTP/HTTPS URLs and checks if they're still alive.

## Tech Stack

- **Go 1.23+** - Programming language
- **Cobra** - CLI framework for argument parsing and subcommands
- **Bubble Tea** - TUI framework (Model-View-Update architecture)
- **Lipgloss** - Terminal styling
- **Bubbles** - Pre-built TUI components
- **Goldmark** - Markdown parser
- **gopkg.in/yaml.v3** - YAML parser
- **BurntSushi/toml** - TOML parser
- **gobwas/glob** - Glob pattern matching

## Project Structure

```
gone/
├── main.go                       # Entry point
├── cmd/
│   ├── root.go                   # Cobra root command
│   ├── check.go                  # CLI check command
│   ├── check_output.go           # Output formatting for check
│   ├── check_print.go            # Print helpers for check
│   ├── fix.go                    # Auto-fix redirects command
│   ├── interactive.go            # TUI mode (Bubble Tea)
│   └── helpers.go                # Shared CLI helpers
├── internal/
│   ├── checker/                  # HTTP link validation with concurrency
│   │   ├── checker.go            # Main checker logic
│   │   ├── options.go            # Checker configuration
│   │   └── result.go             # Result types
│   ├── config/                   # Configuration file handling
│   │   └── config.go             # .gonerc.yaml parsing
│   ├── filter/                   # URL filtering (ignore rules)
│   │   └── filter.go             # Domain, pattern, regex filters
│   ├── fixer/                    # Auto-fix functionality
│   │   └── fixer.go              # Replace URLs in files
│   ├── helpers/                  # Shared utilities
│   │   └── helpers.go            # Common helper functions
│   ├── output/                   # Output formatting
│   │   ├── output.go             # Report structure
│   │   ├── json.go               # JSON output
│   │   ├── yaml.go               # YAML output
│   │   ├── xml.go                # XML output
│   │   ├── junit.go              # JUnit XML output
│   │   └── markdown.go           # Markdown output
│   ├── parser/                   # URL extraction from files
│   │   ├── parser.go             # Common parser utilities
│   │   ├── registry.go           # Parser registry
│   │   ├── json/                 # JSON parser
│   │   ├── markdown/             # Markdown parser
│   │   ├── toml/                 # TOML parser
│   │   ├── xml/                  # XML parser
│   │   └── yaml/                 # YAML parser
│   ├── scanner/                  # File discovery
│   │   └── scanner.go            # Find files by type
│   ├── stats/                    # Performance statistics
│   │   └── stats.go              # Timing and metrics
│   └── ui/                       # TUI components
│       ├── app.go                # Main TUI model
│       ├── commands.go           # TUI commands
│       ├── keys.go               # Key bindings
│       ├── messages.go           # TUI messages
│       └── styles.go             # TUI styling
```

## Commands

```bash
# Basic usage
gone check                        # Scan current dir for markdown files
gone check ./docs                 # Scan specific directory
gone check --types=md,json,yaml   # Scan multiple file types

# Output formats
gone check --format=json          # JSON output
gone check --format=yaml          # YAML output
gone check --output=report.xml    # Write to file (format inferred)

# Filtering
gone check --dead                 # Show only dead links
gone check --warnings             # Show only warnings
gone check --all                  # Show all including alive

# Performance
gone check --concurrency=100      # Increase concurrent workers
gone check --timeout=30           # Set timeout in seconds
gone check --retries=3            # Set retry attempts

# Interactive mode
gone interactive                  # Launch TUI
gone interactive --types=md,json  # TUI with multiple file types

# Auto-fix redirects
gone fix                          # Fix redirect URLs
gone fix --dry-run                # Preview changes
gone fix --yes                    # Apply without prompting
```

## Development Commands

```bash
# Build and run
go run . check                    # Run without building
go build .                        # Build binary
./gone check                      # Run built binary

# Testing
go test ./...                     # Run all tests
go test ./internal/parser/...     # Run parser tests
go test -race ./...               # Run with race detector
go test -bench=. ./...            # Run benchmarks

# Linting
golangci-lint run ./...           # Run linter

# All checks (before committing)
go build ./... && go test ./... && golangci-lint run ./...
```

## Configuration

Create `.gonerc.yaml` in project root:

```yaml
types:
  - md
  - json
  - yaml

scan:
  include:
    - "docs/**"
  exclude:
    - "node_modules/**"
    - "vendor/**"

check:
  concurrency: 50
  timeout: 10
  retries: 2
  strict: false

output:
  format: ""
  showAlive: false
  showWarnings: true
  showDead: true
  showStats: false

ignore:
  domains:
    - localhost
    - example.com
  patterns:
    - "*.local/*"
  regex:
    - "192\\.168\\..*"
```

## Conventions

### Code Style

- Keep packages in `internal/` for encapsulation
- Error handling: always check and return errors, don't panic
- Concurrency: use goroutines + channels for parallel work
- Interfaces: define in the package that uses them, not implements them
- Tests: place in same package with `_test.go` suffix

### Package Guidelines

- `checker` - HTTP validation, concurrent checking
- `parser` - URL extraction, file format handling
- `scanner` - File discovery, glob patterns
- `filter` - URL filtering, ignore rules
- `config` - Configuration file parsing
- `output` - Report formatting
- `cmd` - CLI commands (thin layer over internal packages)

## Git Commit Guidelines

**CRITICAL**: NEVER commit without explicit user permission.

### Rules

- DO NOT run `git commit` unless user explicitly says "commit"
- DO NOT run `git reset` on commits that have been pushed
- DO NOT undo commits without explicit instruction
- WAIT for user to ask before committing
- ASK if unsure about whether to commit

### Pre-Commit Checklist

1. Run tests: `go test ./...`
2. Run linter: `golangci-lint run ./...`
3. Stage changes: `git add -A`
4. Review staged: `git status` and `git diff --cached --stat`

### Commit Message Format

Follow Conventional Commits:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat` - New feature
- `fix` - Bug fix
- `refactor` - Code change that neither fixes bug nor adds feature
- `perf` - Performance improvement
- `test` - Adding or updating tests
- `docs` - Documentation only
- `chore` - Maintenance tasks
- `build` - Build system or dependencies
- `ci` - CI configuration

### Scopes

Packages: `checker`, `parser`, `scanner`, `filter`, `config`, `output`, `fixer`, `stats`, `ui`, `cmd`

Features: `check`, `fix`, `interactive`

### Writing Style

Use imperative mood ("add feature" not "added feature"):
- Think: "If applied, this commit will ___"
- Start with lowercase after colon
- No period at end
- Keep under 72 characters

### Examples

```
feat(parser): add TOML file support

fix(checker): handle timeout errors gracefully

refactor(scanner): simplify file filtering logic

Consolidate include/exclude pattern matching into single function.
Improves readability and reduces code duplication.

perf(checker): reduce memory allocations in URL validation

chore(deps): update goldmark to v1.7.0

test(parser): add edge case tests for malformed JSON
```

### Breaking Changes

Add `!` after type/scope:

```
feat(parser)!: change FileParser interface

BREAKING CHANGE: Removed Validate and Parse methods.
Use ValidateAndParse instead.
```

## Pull Request Guidelines

1. Create feature branch from `master`
2. Make changes with clear, focused commits
3. Ensure all tests pass
4. Ensure linter passes
5. Push branch and create PR
6. Include summary of changes in PR description

Never push directly to `master`.
