// Copyright 2017 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/cover"
	"github.com/open-policy-agent/opa/v1/loader"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"
	"github.com/open-policy-agent/opa/v1/topdown"
	outil "github.com/open-policy-agent/opa/v1/util"
	"github.com/open-policy-agent/opa/v1/version"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/compile"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/builtins"
	"github.com/open-policy-agent/regal/pkg/config"
)

const benchmarkGoBenchOutput = "gobench"

type testCommandParams struct {
	outputFormat *outil.EnumFlag
	runRegex     string
	ignore       []string
	threshold    float64
	timeout      time.Duration
	count        int
	verbose      bool
	coverage     bool
	bundleMode   bool
	benchmark    bool
	benchMem     bool
	skipExitZero bool
	varValues    bool
}

type loaderFilter struct {
	Ignore []string
}

func newTestCommandParams() *testCommandParams {
	return &testCommandParams{
		outputFormat: outil.NewEnumFlag(formatPretty, []string{
			formatPretty,
			formatJSON,
			benchmarkGoBenchOutput,
		}),
	}
}

func (f loaderFilter) Apply(abspath string, info os.FileInfo, depth int) bool {
	return slices.ContainsFunc(f.Ignore, func(s string) bool {
		return loader.GlobExcludeName(s, 1)(abspath, info, depth)
	})
}

var testParams = newTestCommandParams()

var testCommand = &cobra.Command{
	Hidden: true,
	Use:    "test <path> [path [...]]",
	Short:  "Execute Rego test cases for Regal",
	Long: `Execute Rego test cases for Regal rules.

The 'test' command works mostly like OPA's test command, but with all Regal-specific built-ins included. Note that this
command is only meant to be used for testing of Regal rules, and should only be relevant to users authoring their own
rules.

`,
	PreRunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("specify at least one file")
		}

		return nil
	},

	RunE: wrapProfiling(func(args []string) error {
		if c := opaTest(args); c != 0 {
			return ExitError{code: c}
		}

		return nil
	}),
}

func opaTest(args []string) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if testParams.outputFormat.String() == benchmarkGoBenchOutput && !testParams.benchmark {
		fmt.Fprintf(
			os.Stderr,
			"cannot use output format %s without running benchmarks (--bench)\n",
			benchmarkGoBenchOutput,
		)

		return 0
	}

	if !isThresholdValid(testParams.threshold) {
		fmt.Fprintln(os.Stderr, "Code coverage threshold must be between 0 and 100")

		return 1
	}

	filter := loaderFilter{
		Ignore: testParams.ignore,
	}

	var (
		modules map[string]*ast.Module
		bundles map[string]*bundle.Bundle
		store   storage.Store
		err     error
	)

	if testParams.bundleMode {
		bundles, err = tester.LoadBundles(args, filter.Apply)
		store = inmem.NewWithOpts(inmem.OptRoundTripOnWrite(false))
	} else {
		modules, store, err = tester.Load(args, filter.Apply)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	txn, err := store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	regalBundle := rbundle.LoadedBundle()

	if err := store.Write(
		ctx,
		txn,
		storage.AddOp,
		storage.MustParsePath("/regal"),
		regalBundle.Data["regal"],
	); err != nil {
		panic(err)
	}

	if err := store.Write(ctx, txn, storage.AddOp, []string{"internal"}, map[string]any{}); err != nil {
		panic(err)
	}

	if err := store.Write(ctx, txn, storage.AddOp,
		[]string{"internal", "capabilities"},
		rio.ToMap(config.CapabilitiesForThisVersion())); err != nil {
		panic(err)
	}

	defer store.Abort(ctx, txn)

	compiler := compile.NewCompilerWithRegalBuiltins().
		WithPathConflictsCheck(storage.NonEmpty(ctx, store, txn)).
		WithEnablePrintStatements(!testParams.benchmark).
		WithSchemas(compile.RegalSchemaSet()).
		WithUseTypeCheckAnnotations(true).
		WithModuleLoader(moduleLoader(regalBundle)).
		WithRewriteTestRules(testParams.varValues)

	if testParams.threshold > 0 && !testParams.coverage {
		testParams.coverage = true
	}

	var cov *cover.Cover

	var coverTracer topdown.QueryTracer

	if testParams.coverage {
		if testParams.benchmark {
			fmt.Fprintln(os.Stderr, "coverage reporting is not supported when benchmarking tests")

			return 1
		}

		cov = cover.New()
		coverTracer = cov
	}

	runner := tester.NewRunner().
		SetCompiler(compiler).
		SetStore(store).
		CapturePrintOutput(true).
		EnableTracing(testParams.verbose || testParams.varValues).
		SetCoverageQueryTracer(coverTracer).
		SetRuntime(Runtime()).
		SetModules(modules).
		SetBundles(bundles).
		SetTimeout(getTimeout()).
		AddCustomBuiltins(builtins.TestContextBuiltins()).
		Filter(testParams.runRegex)

	for range testParams.count {
		exitCode := runTests(ctx, txn, runner, testReporter(cov, modules))
		if exitCode != 0 {
			return exitCode
		}
	}

	return 0
}

func moduleLoader(regalRules *bundle.Bundle) ast.ModuleLoader {
	// We use the package declarations to know which modules we still need, and return
	// those from the embedded regal bundle.
	extra := util.NewSet[string]()
	for _, mod := range regalRules.Modules {
		extra.Add(mod.Parsed.Package.Path.String())
	}

	return func(present map[string]*ast.Module) (map[string]*ast.Module, error) {
		for _, mod := range present {
			extra.Remove(mod.Package.Path.String())
		}

		extraMods := map[string]*ast.Module{}

		for id, mod := range regalRules.ParsedModules("bundle") {
			if extra.Contains(mod.Package.Path.String()) {
				extraMods[id] = mod
			}
		}

		return extraMods, nil
	}
}

func testReporter(cov *cover.Cover, modules map[string]*ast.Module) tester.Reporter {
	var reporter tester.Reporter

	goBench := false

	if !testParams.coverage {
		switch testParams.outputFormat.String() {
		case formatJSON:
			reporter = tester.JSONReporter{Output: os.Stdout}
		case benchmarkGoBenchOutput:
			goBench = true

			fallthrough
		default:
			reporter = tester.PrettyReporter{
				Verbose:                  testParams.verbose,
				Output:                   os.Stdout,
				BenchmarkResults:         testParams.benchmark,
				BenchMarkShowAllocations: testParams.benchMem,
				BenchMarkGoBenchFormat:   goBench,
				FailureLine:              testParams.varValues,
				LocalVars:                testParams.varValues,
			}
		}
	} else {
		reporter = tester.JSONCoverageReporter{
			Cover:     cov,
			Modules:   modules,
			Output:    os.Stdout,
			Threshold: testParams.threshold,
		}
	}

	return reporter
}

func getTimeout() time.Duration {
	timeout := testParams.timeout
	if timeout == 0 { // unset
		timeout = 5 * time.Second
		if testParams.benchmark {
			timeout = 30 * time.Second
		}
	}

	return timeout
}

func runTests(ctx context.Context, txn storage.Transaction, runner *tester.Runner, reporter tester.Reporter) int {
	var err error

	var ch chan *tester.Result

	if testParams.benchmark {
		benchOpts := tester.BenchmarkOptions{
			ReportAllocations: testParams.benchMem,
		}
		ch, err = runner.RunBenchmarks(ctx, txn, benchOpts)
	} else {
		ch, err = runner.RunTests(ctx, txn)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	exitCode := 0
	dup := make(chan *tester.Result)

	go func() {
		defer close(dup)

		for tr := range ch {
			if !tr.Pass() && !testParams.skipExitZero {
				exitCode = 2
			}

			tr.Trace = nil

			if tr.Skip && exitCode == 0 && testParams.skipExitZero {
				// there is a skipped test, adding the flag -z exits 0 if there are no failures
				exitCode = 0
			}

			dup <- tr
		}
	}()

	if err := reporter.Report(dup); err != nil {
		fmt.Fprintln(os.Stderr, err)

		if !testParams.benchmark {
			if _, ok := err.(*cover.CoverageThresholdError); ok { //nolint: errorlint
				return 2
			}
		}

		return 1
	}

	return exitCode
}

func isThresholdValid(t float64) bool {
	return 0 <= t && t <= 100
}

func init() {
	testCommand.Flags().BoolVarP(&testParams.skipExitZero, "exit-zero-on-skipped", "z", false,
		"skipped tests return status 0")
	testCommand.Flags().BoolVarP(&testParams.verbose, "verbose", "v", false,
		"set verbose reporting mode")
	testCommand.Flags().DurationVar(&testParams.timeout, "timeout", 0,
		"set test timeout (default 5s, 30s when benchmarking)")
	testCommand.Flags().VarP(testParams.outputFormat, "format", "f",
		"set output format")
	testCommand.Flags().BoolVarP(&testParams.coverage, "coverage", "c", false,
		"report coverage (overrides debug tracing)")
	testCommand.Flags().Float64VarP(&testParams.threshold, "threshold", "", 0,
		"set coverage threshold and exit with non-zero status if coverage is less than threshold %")
	testCommand.Flags().BoolVar(&testParams.benchmark, "bench", false,
		"benchmark the unit tests")
	testCommand.Flags().StringVarP(&testParams.runRegex, "run", "r", "",
		"run only test cases matching the regular expression.")
	testCommand.Flags().BoolVar(&testParams.varValues,
		"var-values", false, "show local variable values in test output")

	addPprofFlag(testCommand.Flags())
	addBundleModeFlag(testCommand.Flags(), &testParams.bundleMode, false)
	addBenchmemFlag(testCommand.Flags(), &testParams.benchMem, true)
	addCountFlag(testCommand.Flags(), &testParams.count, "test")
	addIgnoreFlag(testCommand.Flags(), &testParams.ignore)

	RootCommand.AddCommand(testCommand)
}

func Runtime() *ast.Term {
	obj := ast.NewObject()
	env := ast.NewObject()

	for _, s := range os.Environ() {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) == 1 {
			env.Insert(ast.StringTerm(parts[0]), ast.InternedNullTerm)
		} else if len(parts) > 1 {
			env.Insert(ast.StringTerm(parts[0]), ast.StringTerm(parts[1]))
		}
	}

	obj.Insert(ast.InternedTerm("env"), ast.NewTerm(env))
	obj.Insert(ast.InternedTerm("version"), ast.StringTerm(version.Version))
	obj.Insert(ast.InternedTerm("commit"), ast.StringTerm(version.Vcs))

	return ast.NewTerm(obj)
}

func addBundleModeFlag(fs *pflag.FlagSet, bundle *bool, value bool) {
	fs.BoolVarP(bundle, "bundle", "b", value, "load paths as bundle files or root directories")
}

func addBenchmemFlag(fs *pflag.FlagSet, benchMem *bool, value bool) {
	fs.BoolVar(benchMem, "benchmem", value, "report memory allocations with benchmark results")
}

func addCountFlag(fs *pflag.FlagSet, count *int, cmdType string) {
	fs.IntVar(count, "count", 1, "number of times to repeat each "+cmdType)
}

func addIgnoreFlag(fs *pflag.FlagSet, ignoreNames *[]string) {
	fs.StringSliceVarP(ignoreNames, "ignore", "", []string{},
		"set file and directory names to ignore during loading (e.g., '.*' excludes hidden files)")
}

func addPprofFlag(fs *pflag.FlagSet) {
	fs.String("pprof", "",
		"enable profiling (must be one of cpu, clock, mem_heap, mem_allocs, trace, goroutine, mutex, block, thread_creation)")
}
