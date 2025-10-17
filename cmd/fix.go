package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/regal/internal/git"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/fixer"
	"github.com/open-policy-agent/regal/pkg/fixer/fileprovider"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/linter"
)

// fixParams is similar to the lint params, but with some fields such as profiling removed.
// It is intended that it is compatible with the same command line flags as the lint command to
// control the behavior of lint rules used.
type fixParams struct {
	lintAndFixParams

	conflictMode string
	dryRun       bool
	verbose      bool
	force        bool
}

func init() {
	params := &fixParams{}

	intro := strings.ReplaceAll(`Fix Rego source files with linter violations.
Note that this command is intended to help fix style-related issues,
and could be considered as a stricter opa fmt command.
Issues like bugs should be fixed manually,
as it is important to understand why the were flagged.`, "\n", " ")

	fixCommand := &cobra.Command{
		Use:   "fix <path> [path [...]]",
		Short: "Fix Rego source files",
		Long: func() string {
			fixableRules := util.Map(fixes.NewDefaultFixes(), fixes.Fix.Name)
			if len(fixableRules) == 0 {
				return intro
			}

			return fmt.Sprintf(`%s

The linter rules with automatic fixes available are currently:
- %s
`, intro, strings.Join(fixableRules, "\n- "))
		}(),
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("at least one file or directory must be provided for fixing")
			}

			return nil
		},

		RunE: wrapProfiling(func(args []string) error {
			if err := fix(args, params); err != nil {
				log.SetOutput(os.Stderr)
				log.Println(err)

				return exit(1)
			}

			return nil
		}),
	}

	setCommonFlags(fixCommand, &params.lintAndFixParams)

	fixCommand.Flags().BoolVarP(&params.dryRun, "dry-run", "", false,
		"run the fixer in dry-run mode, use with --verbose to see changes")
	fixCommand.Flags().BoolVarP(&params.verbose, "verbose", "", false, "show the full changes applied in the console")
	fixCommand.Flags().BoolVarP(&params.force, "force", "", false,
		"allow fixing of files that have uncommitted changes in git or when git is not being used")
	fixCommand.Flags().StringVarP(&params.conflictMode, "on-conflict", "", "error",
		"configure behavior when filename conflicts are detected. Options are 'error' (default) or 'rename'")

	addPprofFlag(fixCommand.Flags())

	RootCommand.AddCommand(fixCommand)
}

// TODO: This function is too long and should be broken down
//
//nolint:maintidx
func fix(args []string, params *fixParams) (err error) {
	ctx, cancel := getLinterContext(params.lintAndFixParams)
	defer cancel()

	outputWriter, err := params.outputWriter()
	if err != nil {
		return err
	}

	var (
		regalDir         *os.File
		customRulesDir   string
		configSearchPath string
	)

	if len(args) == 1 {
		if configSearchPath = args[0]; !strings.HasPrefix(args[0], "/") {
			configSearchPath = filepath.Join(rio.Getwd(), args[0])
		}
	} else {
		configSearchPath = rio.Getwd()
	}

	if configSearchPath == "" {
		log.Println("failed to determine relevant directory for config file search - won't search for custom config or rules")
	} else {
		if regalDir, err = config.FindRegalDirectory(configSearchPath); err == nil {
			defer regalDir.Close()

			if customRulesPath := filepath.Join(regalDir.Name(), "rules"); rio.IsDir(customRulesPath) {
				customRulesDir = customRulesPath
			}
		}
	}

	l := linter.NewLinter().
		WithDisableAll(params.disableAll).
		WithDisabledCategories(params.disableCategory.v...).
		WithDisabledRules(params.disable.v...).
		WithEnableAll(params.enableAll).
		WithEnabledCategories(params.enableCategory.v...).
		WithEnabledRules(params.enable.v...).
		WithDebugMode(params.debug)

	if customRulesDir != "" {
		l = l.WithCustomRules([]string{customRulesDir})
	}

	if params.rules.isSet {
		l = l.WithCustomRules(params.rules.v)
	}

	if params.ignoreFiles.isSet {
		l = l.WithIgnore(params.ignoreFiles.v)
	}

	if regalDir != nil {
		l = l.WithPathPrefix(filepath.Dir(regalDir.Name()))
	}

	var userConfig config.Config

	userConfigFile, err := readUserConfig(params.lintAndFixParams, configSearchPath)
	switch {
	case err == nil:
		defer rio.CloseIgnore(userConfigFile)

		if params.debug {
			log.Printf("found user config file: %s", userConfigFile.Name())
		}

		if err := yaml.NewDecoder(userConfigFile).Decode(&userConfig); errors.Is(err, io.EOF) {
			log.Printf("user config file %q is empty, will use the default config", userConfigFile.Name())
		} else if err != nil {
			if regalDir != nil {
				return fmt.Errorf("failed to decode user config from %s: %w", regalDir.Name(), err)
			}

			return fmt.Errorf("failed to decode user config: %w", err)
		}

		l = l.WithUserConfig(userConfig)
	case params.configFile != "":
		return fmt.Errorf("user-provided config file not found: %w", err)
	case params.debug:
		log.Println("no user-provided config file found, will use the default config")
	}

	roots, err := config.GetPotentialRoots(args...)
	if err != nil {
		return fmt.Errorf("could not find potential roots: %w", err)
	}

	f := fixer.NewFixer().RegisterRoots(roots...).RegisterFixes(fixes.NewDefaultFixes()...)

	if userConfigFile != nil {
		versionsMap, err := config.AllRegoVersions(filepath.Dir(userConfigFile.Name()), &userConfig)
		if err != nil {
			return fmt.Errorf("failed to get all Rego versions: %w", err)
		}

		f.SetRegoVersionsMap(versionsMap)
	}

	if !slices.Contains([]string{"error", "rename"}, params.conflictMode) {
		return fmt.Errorf("invalid conflict mode: %s, expected 'error' or 'rename'", params.conflictMode)
	}

	// the default is error, so this is only set when it's rename
	if params.conflictMode == "rename" {
		f.SetOnConflictOperation(fixer.OnConflictRename)
	}

	ignore := userConfig.Ignore.Files
	if len(params.ignoreFiles.v) > 0 {
		ignore = params.ignoreFiles.v
	}

	filtered, err := config.FilterIgnoredPaths(args, ignore, true, "")
	if err != nil {
		return fmt.Errorf("failed to filter ignored paths: %w", err)
	}

	slices.Sort(filtered)
	// TODO: Figure out why filtered returns duplicates in the first place
	filtered = slices.Compact(filtered)

	// convert the filtered paths to absolute paths before fixing to
	// support accurate root matching.
	absFiltered := make([]string, len(filtered))

	for i, f := range filtered {
		if filepath.IsAbs(f) {
			absFiltered[i] = f

			continue
		}

		if absFiltered[i], err = filepath.Abs(f); err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", f, err)
		}
	}

	fileProvider, err := fileprovider.NewInMemoryFileProviderFromFS(absFiltered...)
	if err != nil {
		return fmt.Errorf("failed to create file provider: %w", err)
	}

	r, err := fixer.ReporterForFormat(params.format, outputWriter)
	if err != nil {
		return fmt.Errorf("failed to create reporter for format %s: %w", params.format, err)
	}

	r.SetDryRun(params.dryRun)

	fixReport, err := f.Fix(ctx, &l, fileProvider)
	if err != nil {
		return fmt.Errorf("failed to fix: %w", err)
	}

	if fixReport.HasConflicts() {
		if err = r.Report(fixReport); err != nil {
			return fmt.Errorf("failed to output fix report: %w", err)
		}

		return errors.New("fixing failed due to conflicts")
	}

	if !params.dryRun && !params.force {
		gitRepo, err := git.FindGitRepo(args...)
		if err != nil {
			return fmt.Errorf("failed to establish git repo (use --force to override): %w", err)
		}

		if gitRepo == "" {
			return errors.New("no git repo found to support undo (use --force to override)")
		}

		// if the fixer is being run in a git repo, we must not fix files that have changes
		cf, err := git.GetChangedFiles(gitRepo)
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}

		conflictingFiles := util.NewSet(fileProvider.ModifiedFiles()...).Intersect(util.NewSet(cf...))
		if conflictingFiles.Size() > 0 {
			return fmt.Errorf(
				`the following files have been changed since the fixer was run:
- %s
please run fix from a clean state to support the use of git to undo, or use --force to ignore`,
				strings.Join(conflictingFiles.Items(), "\n- "),
			)
		}
	}

	if params.verbose {
		if params.dryRun {
			fmt.Fprintln(outputWriter, "Dry run mode enabled, the following changes would be made:")
		}

		for _, file := range fileProvider.ModifiedFiles() {
			fc, err := fileProvider.Get(file)
			if err != nil {
				return fmt.Errorf("failed to get file %s: %w", file, err)
			}

			fmt.Fprintln(outputWriter, "Set:", file, "to:\n", fc, "\n----------")
		}

		for _, file := range fileProvider.DeletedFiles() {
			fmt.Fprintln(outputWriter, "Delete:", file)
		}
	}

	if !params.dryRun {
		for _, file := range fileProvider.DeletedFiles() {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", file, err)
			}

			dirs, err := rio.DirCleanUpPaths(file, roots)
			if err != nil {
				return fmt.Errorf("failed to delete empty directories: %w", err)
			}

			for _, dir := range dirs {
				if err := os.Remove(dir); err != nil {
					return fmt.Errorf("failed to delete directory %s: %w", dir, err)
				}
			}
		}

		for _, file := range fileProvider.ModifiedFiles() {
			if err = os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", file, err)
			}

			fc, err := fileProvider.Get(file)
			if err != nil {
				return fmt.Errorf("failed to get file %s: %w", file, err)
			}

			fileMode := fs.FileMode(0o600)
			if fileInfo, err := os.Stat(file); err == nil {
				fileMode = fileInfo.Mode()
			}

			if err = os.WriteFile(file, []byte(fc), fileMode); err != nil {
				return fmt.Errorf("failed to write file %s: %w", file, err)
			}
		}
	}

	return util.WrapErr(r.Report(fixReport), "failed to output fix report")
}
