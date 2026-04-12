package cmd

import (
	"io"
	"os"
	"time"

	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/config"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/leonardomso/gone/internal/fixer"
	"github.com/leonardomso/gone/internal/output"
	"github.com/leonardomso/gone/internal/parser"
	"github.com/leonardomso/gone/internal/scanner"
	"github.com/leonardomso/gone/internal/stats"
)

// IOStreams groups the command IO streams to make runners testable.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

func defaultIOStreams() IOStreams {
	return IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// LinkChecker is the checker dependency used by command runners.
type LinkChecker interface {
	CheckAll([]checker.Link) []checker.Result
}

// RedirectFixer is the fixer dependency used by command runners.
type RedirectFixer interface {
	SetParserLinks([]parser.Link)
	FindFixes([]checker.Result) []fixer.FileChanges
	Preview([]fixer.FileChanges) string
	ApplyAll([]fixer.FileChanges) []fixer.FixResult
	ApplyToFile(fixer.FileChanges) (*fixer.FixResult, error)
}

// CommandEnv groups command-side dependencies so runners can be tested directly.
type CommandEnv struct {
	LoadConfig             func(bool) (*LoadedConfig, error)
	FindFiles              func(scanner.ScanOptions) ([]string, error)
	ExtractLinks           func([]string, bool) ([]parser.Link, error)
	CreateFilterWithConfig func(*config.Config, []string, []string, []string) (*filter.Filter, error)
	NewChecker             func(checker.Options) LinkChecker
	NewFixer               func() RedirectFixer
	FormatReport           func(*output.Report, output.Format) ([]byte, error)
	WriteToFile            func(*output.Report, string) error
	NewStats               func() *stats.Stats
	Now                    func() time.Time
}

func defaultCommandEnv() CommandEnv {
	return CommandEnv{
		LoadConfig:             LoadConfig,
		FindFiles:              scanner.FindFilesWithOptions,
		ExtractLinks:           parser.ExtractLinksFromMultipleFilesWithRegistry,
		CreateFilterWithConfig: CreateFilterWithConfig,
		NewChecker: func(opts checker.Options) LinkChecker {
			return checker.New(opts)
		},
		NewFixer: func() RedirectFixer {
			return fixer.New()
		},
		FormatReport: output.FormatReport,
		WriteToFile:  output.WriteToFile,
		NewStats:     stats.New,
		Now:          time.Now,
	}
}
