# gone

**Fast, concurrent dead link detector for documentation files.**

[![Test](https://github.com/leonardomso/gone/actions/workflows/test.yml/badge.svg)](https://github.com/leonardomso/gone/actions/workflows/test.yml)
[![Lint](https://github.com/leonardomso/gone/actions/workflows/lint.yml/badge.svg)](https://github.com/leonardomso/gone/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/leonardomso/gone)](https://goreportcard.com/report/github.com/leonardomso/gone)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Scan your documentation files for broken links. `gone` finds all HTTP/HTTPS URLs in Markdown, JSON, YAML, TOML, and XML files, checks if they're still alive, and helps you fix the ones that aren't.

<p align="center">
  <img src="./github-image.png" alt="gone" width="100%">
</p>

## Here's a list of powerful features:

- **Fast concurrent checking.** Check hundreds of URLs at the same time. Configure the number of workers based on your network. Large documentation sites take seconds, not minutes.

- **Interactive terminal UI.** Watch progress in real-time. Navigate results with vim-style keys, filter by status, and explore issues without leaving your terminal. It's like `htop` for your links.

- **Automatic redirect fixing.** Found a redirect? `gone fix` updates your markdown to use the final URL. Preview changes first with `--dry-run`, confirm each file interactively, or let it fix everything with `--yes`.

- **Smart deduplication.** Same URL in 50 files? It gets checked once. Results map back to every occurrence. Less waiting, fewer rate limits.

- **Flexible ignore rules.** Skip localhost, staging environments, or that one URL that's always flaky. Use domains, glob patterns, or regex. Set them in `.gonerc.yaml` or pass them as flags.

- **Multiple output formats.** JSON, YAML, XML, JUnit, Markdown—pick your favorite. The format is auto-detected from the file extension, or set it explicitly with `--format`.

- **Multi-format support.** Scan Markdown, JSON, YAML, TOML, and XML files. Markdown parsing is format-aware: finds links in `[text](url)`, reference-style `[text][ref]`, autolinks `<url>`, and HTML `<a>` tags while skipping code blocks.

- **CI/CD friendly.** Exit code 0 means all good. Exit code 1 means dead links. JUnit output works with GitHub Actions, GitLab CI, Jenkins, and everything else.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [gone check](#gone-check)
  - [gone interactive](#gone-interactive)
  - [gone fix](#gone-fix)
  - [gone completion](#gone-completion)
- [Configuration](#configuration)
- [Output Formats](#output-formats)
- [CI/CD Integration](#cicd-integration)
- [Exit Codes](#exit-codes)
- [Reference](#reference)
  - [Commands Overview](#commands-overview)
  - [Flags Reference](#flags-reference)
- [License](#license)

## Installation

### Go Install

```bash
go install github.com/leonardomso/gone@latest
```

### Homebrew

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
# Scan current directory (markdown files by default)
gone check

# Scan specific directory
gone check ./docs

# Scan multiple file types
gone check --types=md,json,yaml

# Output as JSON
gone check --format=json

# Launch interactive TUI
gone interactive

# Auto-fix redirect URLs
gone fix
```

## Commands

### `gone check`

Scan files for dead links.

```bash
gone check [path] [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--types` | `-T` | `md` | File types to scan (comma-separated): md, json, yaml, toml, xml |
| `--strict` | — | `false` | Fail on malformed files instead of skipping them |
| `--format` | `-f` | — | Output format: `json`, `yaml`, `xml`, `junit`, `markdown` |
| `--output` | `-o` | — | Write report to file (format inferred from extension) |
| `--all` | `-a` | `false` | Show all results including alive links |
| `--dead` | `-d` | `false` | Show only dead links and errors |
| `--warnings` | `-w` | `false` | Show only warnings (redirects, blocked) |
| `--alive` | — | `false` | Show only alive links |
| `--concurrency` | `-c` | `50` | Number of concurrent workers |
| `--timeout` | `-t` | `5` | Timeout per request in seconds |
| `--retries` | `-r` | `1` | Number of retries for failed requests |
| `--ignore-domain` | — | — | Domains to ignore (comma-separated or repeated) |
| `--ignore-pattern` | — | — | Glob patterns to ignore |
| `--ignore-regex` | — | — | Regex patterns to ignore |
| `--show-ignored` | — | `false` | Show which URLs were ignored |
| `--no-config` | — | `false` | Skip loading .gonerc.yaml |
| `--stats` | — | `false` | Show performance statistics |

**Link Status Types:**

| Status | Description |
|--------|-------------|
| `Alive` | Link returned a 2xx response. All good. |
| `Redirect` | Link redirected (301, 302, 307, 308) but final destination is alive. Consider updating to the final URL. |
| `Blocked` | Server returned 403. Might be bot detection. Link may still work in a browser. |
| `Dead` | Link is broken (4xx, 5xx, or redirect chain ends in error). |
| `Error` | Network error: timeout, DNS failure, or connection refused. |
| `Duplicate` | Same URL appears multiple times. Checked once, result shared. |

**Examples:**

```bash
# Scan markdown, JSON, and YAML files
gone check --types=md,json,yaml

# Scan with strict mode (fail on malformed files)
gone check --types=json --strict

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

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--types` | `-T` | `md` | File types to scan (comma-separated): md, json, yaml, toml, xml |
| `--strict` | — | `false` | Fail on malformed files instead of skipping them |
| `--ignore-domain` | — | — | Domains to ignore |
| `--ignore-pattern` | — | — | Glob patterns to ignore |
| `--ignore-regex` | — | — | Regex patterns to ignore |
| `--no-config` | — | `false` | Skip loading .gonerc.yaml |

**Controls:**

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `PgUp` / `Ctrl+U` | Page up |
| `PgDn` / `Ctrl+D` | Page down |
| `g` / `Home` | Go to start |
| `G` / `End` | Go to end |
| `f` | Cycle filter (All → Warnings → Dead → Duplicates) |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

### `gone fix`

Automatically fix redirect URLs in files.

```bash
gone fix [path] [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--types` | `-T` | `md` | File types to scan (comma-separated): md, json, yaml, toml, xml |
| `--strict` | — | `false` | Fail on malformed files instead of skipping them |
| `--yes` | `-y` | `false` | Apply all fixes without prompting |
| `--dry-run` | `-n` | `false` | Preview changes without modifying files |
| `--concurrency` | `-c` | `50` | Number of concurrent workers |
| `--timeout` | `-t` | `5` | Timeout per request in seconds |
| `--retries` | `-r` | `1` | Number of retries for failed requests |
| `--ignore-domain` | — | — | Domains to ignore |
| `--ignore-pattern` | — | — | Glob patterns to ignore |
| `--ignore-regex` | — | — | Regex patterns to ignore |
| `--no-config` | — | `false` | Skip loading .gonerc.yaml |
| `--stats` | — | `false` | Show performance statistics |

**Examples:**

```bash
# Interactive mode - prompt for each file
gone fix

# Preview what would be fixed
gone fix --dry-run

# Apply all fixes automatically
gone fix --yes
```

### `gone completion`

Generate shell autocompletion scripts.

```bash
gone completion [shell]
```

**Supported Shells:**

| Shell | Command |
|-------|---------|
| Bash | `gone completion bash` |
| Zsh | `gone completion zsh` |
| Fish | `gone completion fish` |
| PowerShell | `gone completion powershell` |

**Setup Examples:**

```bash
# Bash (add to ~/.bashrc)
source <(gone completion bash)

# Zsh (add to ~/.zshrc)
source <(gone completion zsh)

# Fish
gone completion fish | source
```

## Configuration

### Config File

Create a `.gonerc.yaml` file in your project root:

```yaml
# File types to scan (default: md)
types:
  - md
  - json
  - yaml
  - toml
  - xml

# Scanner settings
scan:
  # Include only files matching these glob patterns
  include:
    - "docs/**"
    - "README.md"
  # Exclude files matching these glob patterns
  exclude:
    - "node_modules/**"
    - "vendor/**"
    - ".git/**"

# Checker settings
check:
  concurrency: 50  # Number of concurrent workers
  timeout: 10      # Request timeout in seconds
  retries: 2       # Retry attempts for failed requests
  strict: false    # Fail on malformed files

# Output preferences
output:
  format: ""         # Default format (json, yaml, xml, junit, markdown)
  showAlive: false   # Show alive links in output
  showWarnings: true # Show warnings (redirects, blocked)
  showDead: true     # Show dead links
  showStats: false   # Show performance statistics

# Ignore rules
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

### Supported File Types

| Type | Extensions | Description |
|------|------------|-------------|
| `md` | `.md`, `.mdx`, `.markdown` | Markdown files |
| `json` | `.json` | JSON files |
| `yaml` | `.yaml`, `.yml` | YAML files |
| `toml` | `.toml` | TOML files |
| `xml` | `.xml` | XML files |

### CLI Flags

CLI flags override config file settings:

```bash
# Override types from config
gone check --types=md,json

# Override concurrency
gone check --concurrency=100

# These ignore rules are combined with .gonerc.yaml settings
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
          go-version: '1.24'

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

## Reference

### Commands Overview

| Command | Description |
|---------|-------------|
| `gone check [path]` | Scan files and report dead links |
| `gone fix [path]` | Find redirects and update URLs to final destinations |
| `gone interactive [path]` | Launch terminal UI for interactive exploration |
| `gone completion [shell]` | Generate shell autocompletion (bash, zsh, fish, powershell) |
| `gone help [command]` | Show help for any command |

### Flags Reference

| Flag | Commands | Default | Description |
|------|----------|---------|-------------|
| `--types` | check, fix, interactive | `md` | File types to scan (comma-separated) |
| `--strict` | check, fix, interactive | `false` | Fail on malformed files |
| `-f, --format` | check | — | Output format (json, yaml, xml, junit, markdown) |
| `-o, --output` | check | — | Write report to file |
| `-a, --all` | check | `false` | Show all results including alive |
| `-d, --dead` | check | `false` | Show only dead links |
| `-w, --warnings` | check | `false` | Show only warnings |
| `--alive` | check | `false` | Show only alive links |
| `--show-ignored` | check | `false` | Show ignored URLs |
| `--stats` | check | `false` | Show performance statistics |
| `-y, --yes` | fix | `false` | Apply fixes without prompting |
| `-n, --dry-run` | fix | `false` | Preview changes only |
| `-c, --concurrency` | check, fix | `50` | Concurrent workers |
| `-t, --timeout` | check, fix | `5` | Request timeout (seconds) |
| `-r, --retries` | check, fix | `1` | Retry attempts |
| `--ignore-domain` | all | — | Domains to ignore |
| `--ignore-pattern` | all | — | Glob patterns to ignore |
| `--ignore-regex` | all | — | Regex patterns to ignore |
| `--no-config` | all | `false` | Skip .gonerc.yaml |
| `-h, --help` | all | — | Show help |
| `-v, --version` | root | — | Show version |

## License

[MIT](LICENSE)
