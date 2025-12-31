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

- Keep packages in `internal/` for encapsulation
- Error handling: always check and return errors, don't panic
- Concurrency: use goroutines + channels for parallel work

## Git Commit Guidelines

**CRITICAL - READ THIS FIRST**

NEVER commit without explicit user permission. NEVER use git reset on pushed commits.

Rules:

- DO NOT run git commit unless user explicitly says "commit" or "commit our changes"
- DO NOT run git reset on commits that have been pushed to remote
- DO NOT undo commits without explicit user instruction
- WAIT for user to ask before committing
- ASK user if unsure about whether to commit

IMPORTANT: ONLY commit changes when explicitly asked by the user.

### Pre-Commit Checklist

1. Stage changes: ALWAYS use `git add -A` (not `git add .`) to stage all changes including deletions
2. Verify staged files: Run `git status` to review what will be committed
3. Review changes: Run `git diff --cached --stat` to see a summary of changes
4. Check authorship: Run `git log -1 --format='%an %ae'` to verify git user configuration

### Commit Message Format

Follow the Conventional Commits specification strictly:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Structure Rules

- **Type**: Required (feat, fix, chore, docs, style, refactor, perf, test, build, ci, revert)
- **Scope**: Optional, use package or feature name
- **Description**: Required, concise summary in imperative mood
- **Body**: Optional, detailed explanation separated by blank line
- **Footer**: Optional, for references or breaking changes

### Writing Style

Use imperative mood - write as commands:

- "add feature" (correct)
- "added feature" or "adds feature" (incorrect)

Think: "If applied, this commit will ___"

Subject line format:

- Start with lowercase after colon
- No period at end
- Keep under 72 characters
- Be specific and descriptive

Body guidelines (when needed):

- Explain what and why, not how
- Wrap lines at 72 characters
- Use present tense
- Separate from subject with blank line
- Can include bullet points for multiple changes

Breaking changes:

- Add `!` after type/scope: `feat(parser)!: change URL extraction`
- OR include footer: `BREAKING CHANGE: description`

### Scope Names for Gone

Packages:

- `scanner` - markdown file discovery
- `parser` - URL extraction
- `checker` - HTTP link validation
- `cmd` - CLI commands

Features:

- `check` - CLI check command
- `interactive` - TUI mode
- `output` - text/JSON formatting

Infrastructure:

- `deps` - dependencies
- `ci` - continuous integration
- `docs` - documentation
- `config` - configuration files

### Examples

Simple feature:

```
feat(checker): add timeout configuration for HTTP requests
```

Bug fix:

```
fix(parser): handle URLs with trailing punctuation
```

With body:

```
feat(interactive): add progress bar during link checking

Replace spinner with progress bar showing checked/total count.
Updates in real-time as each link check completes.
```

Breaking change:

```
feat(checker)!: change Result struct fields

BREAKING CHANGE: Renamed StatusCode to Status and added
new fields for response headers.
```

Chore:

```
chore(deps): update bubble tea to v1.4
```

Documentation:

```
docs: add usage examples to README
```

## Future Improvements

- [ ] Recursive directory scanning flag
- [ ] Follow redirects option (treat 301/302 as alive)
- [ ] Timeout configuration
- [ ] Ignore patterns (skip certain domains)
- [ ] Unit tests for all packages
- [ ] Remove dead links from files (interactive mode feature)
