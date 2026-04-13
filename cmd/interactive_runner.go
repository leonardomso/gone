package cmd

import "github.com/leonardomso/gone/internal/checker"

type interactiveOptions struct {
	FileTypes      []string
	StrictMode     bool
	IgnoreDomains  []string
	IgnorePatterns []string
	IgnoreRegex    []string
	NoConfig       bool
}

type interactiveRunner struct {
	opts interactiveOptions
	env  CommandEnv
	io   IOStreams
}

func newInteractiveRunner(
	opts interactiveOptions,
	env CommandEnv,
	streams IOStreams,
) *interactiveRunner {
	return &interactiveRunner{
		opts: opts,
		env:  env,
		io:   streams,
	}
}

func currentInteractiveOptions() interactiveOptions {
	return interactiveOptions{
		FileTypes:      append([]string{}, iFileTypes...),
		StrictMode:     iStrictMode,
		IgnoreDomains:  append([]string{}, iIgnoreDomains...),
		IgnorePatterns: append([]string{}, iIgnorePatterns...),
		IgnoreRegex:    append([]string{}, iIgnoreRegex...),
		NoConfig:       iNoConfig,
	}
}

func (r *interactiveRunner) Run(args []string) int {
	loadedCfg, err := r.env.LoadConfig(r.opts.NoConfig)
	if err != nil {
		writef(r.io.ErrOut, "Config error: %v\n", err)
		return 1
	}

	effectiveTypes := loadedCfg.GetTypes(r.opts.FileTypes, []string{"md"})
	if err := validateFileTypes(effectiveTypes); err != nil {
		writef(r.io.ErrOut, "Error: %v\n", err)
		return 1
	}

	urlFilter, err := r.env.CreateFilterWithConfig(
		loadedCfg.Config(),
		r.opts.IgnoreDomains,
		r.opts.IgnorePatterns,
		r.opts.IgnoreRegex,
	)
	if err != nil {
		writef(r.io.ErrOut, "Error creating filter: %v\n", err)
		return 1
	}

	scanInclude, scanExclude := loadedCfg.GetScanOptions()
	model := r.env.NewInteractiveModel(
		getPathArg(args),
		urlFilter,
		effectiveTypes,
		loadedCfg.GetStrict(r.opts.StrictMode),
		scanInclude,
		scanExclude,
		loadedCfg.BuildCheckerOptions(
			checker.DefaultConcurrency,
			int(checker.DefaultTimeout.Seconds()),
			checker.DefaultMaxRetries,
		),
	)

	if err := r.env.RunInteractiveProgram(model, r.io); err != nil {
		writef(r.io.ErrOut, "Error running interactive mode: %v\n", err)
		return 1
	}

	return 0
}
