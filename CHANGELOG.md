# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0](https://github.com/leonardomso/gone/releases/tag/v0.1.0) - Initial Release

### Features

- **Fast concurrent link checking** with configurable worker pool
- **Interactive terminal UI** with vim-style navigation and real-time progress
- **Automatic redirect fixing** with `gone fix` command
- **Smart URL deduplication** - each unique URL checked once
- **Flexible ignore rules** - domains, glob patterns, and regex
- **Multiple output formats** - JSON, YAML, XML, JUnit, Markdown
- **Markdown-aware parsing** - inline links, reference links, autolinks, HTML anchors
- **Configuration file support** - `.gonerc.yaml` for project settings
- **Shell autocompletion** - bash, zsh, fish, powershell

### Commands

- `gone check` - Scan markdown files for dead links
- `gone fix` - Automatically fix redirect URLs
- `gone interactive` - Launch terminal UI
- `gone completion` - Generate shell autocompletion scripts
