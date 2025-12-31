# gone

**Fast, concurrent dead link detector for markdown files.**

[![Test](https://github.com/leonardomso/gone/actions/workflows/test.yml/badge.svg)](https://github.com/leonardomso/gone/actions/workflows/test.yml)
[![Lint](https://github.com/leonardomso/gone/actions/workflows/lint.yml/badge.svg)](https://github.com/leonardomso/gone/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/leonardomso/gone)](https://goreportcard.com/report/github.com/leonardomso/gone)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Scan your markdown files for broken links. `gone` finds all HTTP/HTTPS URLs, checks if they're still alive, and helps you fix the ones that aren't.

<p align="center">
  <!-- TODO: Add logo here -->
</p>

## Here's a list of powerful features:

- **Fast concurrent checking.** Check hundreds of URLs at the same time. Configure the number of workers based on your network. Large documentation sites take seconds, not minutes.

- **Interactive terminal UI.** Watch progress in real-time. Navigate results with vim-style keys, filter by status, and explore issues without leaving your terminal. It's like `htop` for your links.

- **Automatic redirect fixing.** Found a redirect? `gone fix` updates your markdown to use the final URL. Preview changes first with `--dry-run`, confirm each file interactively, or let it fix everything with `--yes`.

- **Smart deduplication.** Same URL in 50 files? It gets checked once. Results map back to every occurrence. Less waiting, fewer rate limits.

- **Flexible ignore rules.** Skip localhost, staging environments, or that one URL that's always flaky. Use domains, glob patterns, or regex. Set them in `.gonerc.yaml` or pass them as flags.

- **Multiple output formats.** JSON, YAML, XML, JUnit, Markdown—pick your favorite. The format is auto-detected from the file extension, or set it explicitly with `--format`.

- **Markdown-aware parsing.** Finds links in `[text](url)`, reference-style `[text][ref]`, autolinks `<url>`, and HTML `<a>` tags. Knows to skip URLs inside code blocks.

- **CI/CD friendly.** Exit code 0 means all good. Exit code 1 means dead links. JUnit output works with GitHub Actions, GitLab CI, Jenkins, and everything else.

<p align="center">
  <!-- TODO: Add demo GIF here -->
</p>

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [gone check](#gone-check)
  - [gone interactive](#gone-interactive)
  - [gone fix](#gone-fix)
- [Configuration](#configuration)
- [Output Formats](#output-formats)
- [CI/CD Integration](#cicd-integration)
- [Exit Codes](#exit-codes)
- [License](#license)

## Installation

### Go Install

```bash
go install github.com/leonardomso/gone@latest
```

### Homebrew (coming soon)

```bash
brew install leonardomso/tap/gone
```

### Build from Source

```bash
git clone https://github.com/leonardomso/gone.git
cd gone
go build
```

### Download Binary

Download the latest release from the [GitHub Releases](https://github.com/leonardomso/gone/releases) page.

## Quick Start

```bash
# Scan current directory
gone check

# Scan specific directory
gone check ./docs

# Output as JSON
gone check --format=json

# Launch interactive TUI
gone interactive

# Auto-fix redirect URLs
gone fix
```

## Commands

### `gone check`

Scan markdown files for dead links.

```bash
gone check [path] [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--format` | `-f` | Output format: `json`, `yaml`, `xml`, `junit`, `markdown` |
| `--output` | `-o` | Write report to file (format inferred from extension) |
| `--all` | `-a` | Show all results including alive links |
| `--dead` | `-d` | Show only dead links and errors |
| `--warnings` | `-w` | Show only warnings (redirects, blocked) |
| `--concurrency` | `-c` | Number of concurrent workers (default: 10) |
| `--timeout` | `-t` | Timeout per request in seconds (default: 10) |
| `--retries` | `-r` | Number of retries for failed requests (default: 2) |
| `--ignore-domain` | | Domains to ignore (comma-separated or repeated) |
| `--ignore-pattern` | | Glob patterns to ignore |
| `--ignore-regex` | | Regex patterns to ignore |
| `--show-ignored` | | Show which URLs were ignored |
| `--no-config` | | Skip loading .gonerc.yaml |

**Examples:**

```bash
# Scan with JSON output to stdout
gone check --format=json

# Write JUnit report for CI
gone check --output=report.junit.xml

# Show only dead links
gone check --dead

# Increase concurrency for faster scanning
gone check --concurrency=20

# Ignore specific domains
gone check --ignore-domain=localhost,example.com
```

### `gone interactive`

Launch an interactive terminal UI with real-time progress.

```bash
gone interactive [path] [flags]
```

**Controls:**

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate through results |
| `f` | Cycle through filters (All → Warnings → Dead → Duplicates) |
| `?` | Toggle help |
| `q` | Quit |

### `gone fix`

Automatically fix redirect URLs in markdown files.

```bash
gone fix [path] [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--yes` | `-y` | Apply all fixes without prompting |
| `--dry-run` | `-n` | Preview changes without modifying files |

**Examples:**

```bash
# Interactive mode - prompt for each file
gone fix

# Preview what would be fixed
gone fix --dry-run

# Apply all fixes automatically
gone fix --yes
```

## Configuration

### Config File

Create a `.gonerc.yaml` file in your project root:

```yaml
ignore:
  # Ignore entire domains (includes subdomains)
  domains:
    - localhost
    - example.com
    - staging.myapp.com

  # Glob patterns
  patterns:
    - "*.local/*"
    - "*/internal/*"

  # Regular expressions
  regex:
    - ".*\\.(test|dev)$"
    - "192\\.168\\..*"
```

### CLI Flags

CLI flags are additive with config file settings:

```bash
# These are combined with .gonerc.yaml settings
gone check --ignore-domain=api.example.com --ignore-pattern="*.internal/*"
```

## Output Formats

### JSON

```bash
gone check --format=json
# or
gone check --output=report.json
```

### YAML

```bash
gone check --format=yaml
# or
gone check --output=report.yaml
```

### JUnit XML (for CI/CD)

```bash
gone check --output=report.junit.xml
```

### Markdown

```bash
gone check --format=markdown
# or
gone check --output=report.md
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Check Links

on:
  push:
    branches: [master]
  pull_request:
    branches: [master]
  schedule:
    - cron: '0 0 * * 0'  # Weekly

jobs:
  check-links:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install gone
        run: go install github.com/leonardomso/gone@latest

      - name: Check links
        run: gone check --output=report.junit.xml

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: link-report
          path: report.junit.xml
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All links are alive (or only warnings) |
| `1` | Dead links or errors found |
| `2` | User quit interactive fix mode |

## License

[MIT](LICENSE)
