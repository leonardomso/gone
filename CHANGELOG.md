# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1](https://github.com/leonardomso/gone/compare/v0.1.0...v0.1.1) (2026-01-01)


### Features

* performance optimizations and test coverage improvements ([a33c733](https://github.com/leonardomso/gone/commit/a33c7333b8eaecf691b89f1c016f0132faa14e30))


### Performance Improvements

* optimize allocations and reduce hot path overhead ([b874fe1](https://github.com/leonardomso/gone/commit/b874fe1ba283dc35ee51e797aecd072bb30cc698))
* optimize performance with new defaults and stats tracking ([815db4d](https://github.com/leonardomso/gone/commit/815db4de9efc5118cbc06522ca8cd32e544abb13))
* optimize struct field alignment for better memory usage ([cf51648](https://github.com/leonardomso/gone/commit/cf5164880409801ac05c8eaea195c7b838d2dcdc))


### Code Refactoring

* reorganize code for better maintainability ([5fd8eb7](https://github.com/leonardomso/gone/commit/5fd8eb7ecce40a8d2aa7b6a5fe71e6720f6dac34))


### Documentation

* update README with correct defaults and Homebrew availability ([8793e45](https://github.com/leonardomso/gone/commit/8793e450fbe5089988f76953e8dffab394c29adb))


### Tests

* add comprehensive tests for stats and helpers packages ([4f4b3e7](https://github.com/leonardomso/gone/commit/4f4b3e7680ec5a988464434e8bd7638d47307e86))
* improve parser package coverage from 79.9% to 95.1% ([0c6fe5f](https://github.com/leonardomso/gone/commit/0c6fe5f920b590744093ce4d03070e5ada0c4297))


### Miscellaneous

* add gone-test to gitignore ([385c26a](https://github.com/leonardomso/gone/commit/385c26a5eb04d673043cb77624653ce04b11eede))

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
