# Gone - Dead Link Detector

## Project Overview

Gone is a CLI tool that scans markdown files for dead links. It extracts all HTTP/HTTPS URLs and checks if they're still alive (return 200 status code).

## Goals

1. **Learn Go** - This project is a learning exercise for Go programming
2. **Build a useful tool** - A practical dead link detector for markdown documentation
3. **Explore Go patterns** - Concurrency (goroutines/channels), CLI frameworks, TUI development

## Tech Stack

- **Go** - Programming language
- **Cobra** - CLI framework for argument parsing and subcommands
- **Bubble Tea** - TUI framework (Model-View-Update architecture)
- **Lipgloss** - Terminal styling (colors, borders, etc.)
- **Bubbles** - Pre-built TUI components (spinners, etc.)

## Commands

```bash
gone check                    # Scan current dir, text output
gone check --format=json      # JSON output for CI/scripts
gone check ./docs             # Scan specific directory
gone interactive              # Launch interactive TUI
```

## Project Structure

```
gone/
├── main.go                   # Entry point
├── cmd/
│   ├── root.go               # Cobra root command
│   ├── check.go              # CLI mode (text/JSON output)
│   └── interactive.go        # TUI mode (Bubble Tea)
├── internal/
│   ├── scanner/              # Find .md files in directories
│   ├── parser/               # Extract URLs from content
│   └── checker/              # HTTP checking with concurrency
```

## Development Commands

```bash
go run . check                # Run without building
go run . interactive          # Run TUI mode
go build .                    # Build binary
go test ./...                 # Run tests (when added)
```

## Conventions

- Use conventional commits (feat:, fix:, chore:, etc.)
- Keep packages in `internal/` for encapsulation
- Error handling: always check and return errors, don't panic
- Concurrency: use goroutines + channels for parallel work

## Future Improvements

- [ ] Recursive directory scanning flag
- [ ] Follow redirects option (treat 301/302 as alive)
- [ ] Timeout configuration
- [ ] Ignore patterns (skip certain domains)
- [ ] Unit tests for all packages
- [ ] Remove dead links from files (interactive mode feature)
