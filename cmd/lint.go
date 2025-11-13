package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/jstemmer/go-junit-report/v2/junit"
	"github.com/spf13/cobra"

	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/topdown"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/cache"
	rio "github.com/open-policy-agent/regal/internal/io"
	regalmetrics "github.com/open-policy-agent/regal/internal/metrics"
	"github.com/open-policy-agent/regal/internal/update"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/reporter"
	"github.com/open-policy-agent/regal/pkg/version"
)

type lintAndFixParams struct {
	configFile      string
	format          string
	outputFile      string
	rules           repeatedStringFlag
	disable         repeatedStringFlag
	disableCategory repeatedStringFlag
	enable          repeatedStringFlag
	enableCategory  repeatedStringFlag
	ignoreFiles     repeatedStringFlag
	timeout         time.Duration
	debug           bool
	disableAll      bool
	enableAll       bool
}

type lintParams struct {
	lintAndFixParams

	failLevel   string
	enablePrint bool
	metrics     bool
	profile     bool
	instrument  bool
}

func (params *lintAndFixParams) outputWriter() (io.Writer, error) {
	if params.outputFile == "" {
		return os.Stdout, nil
	}

	f, err := os.OpenFile(params.outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create or open output file %w", err)
	}

	return f, nil
}

func setCommonFlags(cmd *cobra.Command, params *lintAndFixParams) {
	flags := cmd.Flags()
	flags.StringVarP(&params.configFile, "config-file", "c", "", "set path of configuration file")
	flags.StringVarP(&params.format, "format", "f", formatPretty,
		"set output format (pretty, compact, json, github, sarif)")
	flags.StringVarP(&params.outputFile, "output-file", "o", "",
		"set file to use for linting output, defaults to stdout")
	flags.BoolVar(&color.NoColor, "no-color", false, "disable color output")
	flags.VarP(&params.rules, "rules", "r", "set custom rules file(s). This flag can be repeated.")
	flags.DurationVar(&params.timeout, "timeout", 0, "set timeout for linting (default no timeout)")
	flags.BoolVar(&params.debug, "debug", false,
		"enable debug logging (including print output from custom policy)")
	flags.VarP(&params.disable, "disable", "d", "disable specific rule(s). This flag can be repeated.")
	flags.BoolVarP(&params.disableAll, "disable-all", "D", false, "disable all rules")
	flags.VarP(&params.disableCategory, "disable-category", "",
		"disable all rules in a category. This flag can be repeated.")
	flags.VarP(&params.enable, "enable", "e", "enable specific rule(s). This flag can be repeated.")
	flags.BoolVarP(&params.enableAll, "enable-all", "E", false, "enable all rules")
	flags.VarP(&params.enableCategory, "enable-category", "",
		"enable all rules in a category. This flag can be repeated.")
	flags.VarP(&params.ignoreFiles, "ignore-files", "",
		"ignore all files matching a glob-pattern. This flag can be repeated.")

	// Allow setting debug mode via GitHub UI for failing actions
	if os.Getenv("RUNNER_DEBUG") != "" {
		params.debug = true
	}
}

func init() {
	params := &lintParams{}

	lintCommand := &cobra.Command{
		Use:   "lint <path> [path [...]]",
		Short: "Lint Rego source files",
		Long:  `Lint Rego source files for linter rule violations.`,
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("at least one file or directory must be provided for linting")
			}

			return nil
		},
		RunE: wrapProfiling(func(args []string) error {
			rep, err := lint(args, params)
			if err != nil {
				log.SetOutput(os.Stderr)
				log.Println(err)

				return exit(1)
			}

			errorsFound, warningsFound := 0, 0

			for i := range rep.Violations {
				switch rep.Violations[i].Level {
				case "error":
					errorsFound++
				case "warning":
					warningsFound++
				}
			}

			exitCode := 0
			if params.failLevel == "error" && errorsFound > 0 {
				exitCode = 3
			}

			if params.failLevel == "warning" {
				if errorsFound > 0 {
					exitCode = 3
				} else if warningsFound > 0 {
					exitCode = 2
				}
			}

			if exitCode != 0 {
				return exit(exitCode)
			}

			return nil
		}),
	}

	setCommonFlags(lintCommand, &params.lintAndFixParams)

	lintCommand.Flags().StringVarP(&params.failLevel, "fail-level", "l", "error",
		"set level at which to fail with a non-zero exit code (error, warning)")
	lintCommand.Flags().BoolVar(&params.enablePrint, "enable-print", false, "enable print output from policy")
	lintCommand.Flags().BoolVar(&params.metrics, "metrics", false,
		"enable metrics reporting (currently supported only for JSON output format)")
	lintCommand.Flags().BoolVar(&params.profile, "profile", false,
		"enable profiling metrics to be added to reporting (currently supported only for JSON output format)")
	lintCommand.Flags().BoolVar(&params.instrument, "instrument", false,
		"enable instrumentation metrics to be added to reporting (currently supported only for JSON output format)")

	addPprofFlag(lintCommand.Flags())

	RootCommand.AddCommand(lintCommand)
}

func lint(args []string, params *lintParams) (result report.Report, err error) {
	ctx, cancel := getLinterContext(params.lintAndFixParams)
	defer cancel()

	outputWriter, err := params.outputWriter()
	if err != nil {
		return report.Report{}, err
	}

	regal := linter.NewLinter().
		WithDisableAll(params.disableAll).
		WithDisabledCategories(params.disableCategory.v...).
		WithDisabledRules(params.disable.v...).
		WithEnableAll(params.enableAll).
		WithEnabledCategories(params.enableCategory.v...).
		WithEnabledRules(params.enable.v...).
		WithDebugMode(params.debug).
		WithProfiling(params.profile).
		WithInstrumentation(params.instrument).
		WithInputPaths(args).
		WithBaseCache(cache.NewBaseCache())

	if params.enablePrint {
		regal = regal.WithPrintHook(topdown.NewPrintHook(os.Stderr))
	}

	if params.rules.isSet {
		regal = regal.WithCustomRules(params.rules.v)
	}

	if params.ignoreFiles.isSet {
		regal = regal.WithIgnore(params.ignoreFiles.v)
	}

	m := metrics.New()
	if params.metrics {
		regal = regal.WithMetrics(m)
		m.Timer(regalmetrics.RegalConfigSearch).Start()
	}

	var regalPath string

	searchPath := getSearchPath(args)
	if searchPath != "" {
		if regalPath, err = config.FindRegalDirectoryPath(searchPath); err == nil {
			regal = regal.WithPathPrefix(regalPath)

			if params.configFile == "" {
				if regalConf := filepath.Join(regalPath, "config.yaml"); rio.IsFile(regalConf) {
					// override param in case it's unset, so that the readUserConfig function
					// called below can skip searching for the config file
					params.configFile = regalConf
				}
			}

			if rulesDir := filepath.Join(regalPath, "rules"); !params.rules.isSet && rio.IsDir(rulesDir) {
				regal = regal.WithCustomRules([]string{rulesDir})
			}
		}
	}

	if params.metrics {
		m.Timer(regalmetrics.RegalConfigSearch).Stop()
		m.Timer(regalmetrics.RegalConfigParse).Start()
	}

	userConfig, path, err := loadUserConfig(params.lintAndFixParams, searchPath)
	if err != nil {
		return report.Report{}, fmt.Errorf("failed to read user-provided config in %s: %w", path, err)
	}

	if params.metrics {
		m.Timer(regalmetrics.RegalConfigParse).Stop()
	}

	regal = regal.WithUserConfig(userConfig)

	go updateCheckAndWarn(params, rbundle.Loaded(), &userConfig)

	regal, err = regal.Prepare(ctx)
	if err != nil {
		return report.Report{}, fmt.Errorf("failed to prepare for linting: %w", err)
	}

	result, err = regal.Lint(ctx)
	if err != nil {
		return report.Report{}, formatError(params.format, fmt.Errorf("error(s) encountered while linting: %w", err))
	}

	rep, err := getReporter(params.format, outputWriter)
	if err != nil {
		return report.Report{}, fmt.Errorf("failed to get reporter: %w", err)
	}

	return result, rep.Publish(ctx, result)
}

func updateCheckAndWarn(params *lintParams, regalRules *bundle.Bundle, userConfig *config.Config) {
	mergedConfig, err := config.WithDefaultsFromBundle(regalRules, userConfig)
	if err != nil {
		if params.debug {
			log.Printf("failed to merge user config with default config when checking version: %v", err)
		}

		return
	}

	if mergedConfig.Features.Remote.CheckVersion &&
		os.Getenv(update.CheckVersionDisableEnvVar) == "" {
		update.CheckAndWarn(update.Options{
			CurrentVersion: version.Version,
			CurrentTime:    time.Now().UTC(),
			Debug:          params.debug,
			StateDir:       config.GlobalConfigDir(true),
		}, os.Stderr)
	}
}

func getReporter(format string, outputWriter io.Writer) (reporter.Reporter, error) {
	switch format {
	case formatPretty:
		return reporter.NewPrettyReporter(outputWriter), nil
	case formatCompact:
		return reporter.NewCompactReporter(outputWriter), nil
	case formatJSON:
		return reporter.NewJSONReporter(outputWriter), nil
	case formatGitHub:
		return reporter.NewGitHubReporter(outputWriter), nil
	case formatFestive:
		return reporter.NewFestiveReporter(outputWriter), nil
	case formatSarif:
		return reporter.NewSarifReporter(outputWriter), nil
	case formatJunit:
		return reporter.NewJUnitReporter(outputWriter), nil
	default:
		return nil, fmt.Errorf("unknown format %s", format)
	}
}

func formatError(format string, err error) error {
	// currently, JSON and SARIF will get the same generic JSON error format
	switch format {
	case formatJSON, formatSarif:
		bs, err := json.MarshalIndent(map[string]any{
			"errors": []string{err.Error()},
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format errors for output: %w", err)
		}

		return fmt.Errorf("%s", string(bs))
	case formatJunit:
		testSuites := junit.Testsuites{
			Name: "regal",
		}
		testsuite := junit.Testsuite{
			Name: "lint",
		}
		testsuite.AddTestcase(junit.Testcase{
			Name: "Command execution failed",
			Error: &junit.Result{
				Message: err.Error(),
			},
		})
		testSuites.AddSuite(testsuite)

		buf := &bytes.Buffer{}

		if err := testSuites.WriteXML(buf); err != nil {
			return fmt.Errorf("failed to format errors for output: %w", err)
		}

		return fmt.Errorf("%s", buf.String())
	}

	return err
}
