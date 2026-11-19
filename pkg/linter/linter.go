package linter

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing/fstest"

	"golang.org/x/sync/errgroup"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/profiler"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown/print"
	outil "github.com/open-policy-agent/opa/v1/util"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/funsafe"
	rio "github.com/open-policy-agent/regal/internal/io"
	regalmetrics "github.com/open-policy-agent/regal/internal/metrics"
	"github.com/open-policy-agent/regal/internal/ogre"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/linter/params"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
	"github.com/open-policy-agent/regal/pkg/rules"
	"github.com/open-policy-agent/regal/pkg/version"

	_ "github.com/open-policy-agent/regal/pkg/builtins"
)

type (
	// Linter stores data to use for linting.
	Linter struct {
		printHook         print.Hook
		metrics           metrics.Metrics
		inputModules      *rules.Input
		userConfig        *config.Config
		combinedCfg       *config.Config
		params            *params.Params
		pathPrefix        string
		customRuleError   error
		inputPaths        []string
		ruleBundles       []*bundle.Bundle
		customRuleModules []*ast.Module
		useCollectQuery   bool
		debugMode         bool
		exportAggregates  bool
		profiling         bool
		instrumentation   bool
		isPrepared        bool

		preparedQuery *ogre.Query
	}
	// ValidationError is the error type used for invalid CLI parameters, configuration or anything else that the user
	// provided as part of setting up the linter. These errors should normally not be wrapped but displayed exactly as
	// they are to the user invoking the linter.
	ValidationError struct {
		Messages []string
	}
	determineRulesHandler struct {
		ruleKeys []string
	}
	lintResultHandler struct {
		results []*report.Report
	}
	prepareResultHandler struct {
		stg storage.Store
		txn storage.Transaction
	}
	aggregateResultHandler struct {
		rep *report.Report
	}
)

var (
	eqVarValue      ast.Value = ast.Var("eq")
	lintVarValue    ast.Value = ast.Var("lint")
	enabledVarValue ast.Value = ast.Var("enabled")
	eqNameVarTerm   *ast.Term = ast.NewTerm(eqVarValue)
	lintVarTerm     *ast.Term = ast.NewTerm(lintVarValue)
	enabledVarTerm  *ast.Term = ast.NewTerm(enabledVarValue)
	eqRef                     = ast.RefTerm(eqNameVarTerm)

	lintQuery = []*ast.Expr{{ // lint = data.regal.main.lint.
		Terms: []*ast.Term{eqRef, lintVarTerm, {Value: ast.Ref{
			ast.DefaultRootDocument,
			ast.InternedTerm("regal"),
			ast.InternedTerm("main"),
			ast.InternedTerm("lint"),
		}}},
	}}
	enabledRulesQuery = []*ast.Expr{{ // enabled = data.regal.main.enabled_rules.
		Terms: []*ast.Term{eqRef, enabledVarTerm, {Value: ast.Ref{
			ast.DefaultRootDocument,
			ast.InternedTerm("regal"),
			ast.InternedTerm("main"),
			ast.InternedTerm("enabled_rules"),
		}}},
	}}

	preparedInput = ast.NewObject(
		rast.Item("regal", ast.ObjectTerm(
			rast.Item("operations", rast.ArrayTerm("prepare")),
			rast.Item("file", ast.ObjectTerm(
				rast.Item("name", ast.InternedTerm("__prepare__")),
				rast.Item("rego_version", ast.InternedTerm("v1")),
			)),
		)),
	)
	aggregateArray  = &ast.Term{Value: ast.NewArray(ast.InternedTerm("aggregate"))}
	aggregateObject = &ast.Term{Value: ast.NewObject(
		rast.Item("name", ast.InternedTerm("__aggregate_report__")),
		rast.Item("lines", ast.InternedEmptyArray),
	)}
	aggregateRegalObject = &ast.Term{Value: ast.NewObject(
		rast.Item("operations", aggregateArray),
		rast.Item("file", aggregateObject),
	)}

	internalPreparedNoticesPath = storage.Path{"internal", "prepared", "notices"}
	preparedPath                = internalPreparedNoticesPath[:2]

	errNoEnabledRules  = errors.New("expected enabled rules object, didn't get it")
	errNothingToLint   = errors.New("nothing provided to lint")
	errNoPreparedField = errors.New("expected 'prepared' field in result object")

	emptyCtx = context.TODO()
)

func init() {
	ast.InternStringTerm("eval", "aggregates_internal", "__aggregate_report__")
}

// NewLinter creates a new Regal linter.
func NewLinter() Linter {
	return Linter{ruleBundles: []*bundle.Bundle{rbundle.Loaded()}, params: params.New()}
}

// NewEmptyLinter creates a linter with no rule bundles.
func NewEmptyLinter() Linter {
	return Linter{params: params.New()}
}

// WithInputPaths sets the inputPaths to lint. Note that these will be
// filtered according to the ignore options.
func (l Linter) WithInputPaths(paths []string) Linter {
	l.inputPaths = paths

	return l
}

// WithInputModules sets the input modules to lint. This is used for programmatic
// access, where you don't necessarily want to lint *files*.
func (l Linter) WithInputModules(input *rules.Input) Linter {
	l.inputModules = input

	return l
}

// WithAddedBundle adds a bundle of rules and data to include in evaluation.
func (l Linter) WithAddedBundle(b *bundle.Bundle) Linter {
	l.ruleBundles = append(l.ruleBundles, b)

	return l.notPrepared()
}

// WithCustomRules adds custom rules for evaluation, from the Rego (and data) files provided at paths.
//
// Deprecated: use WithCustomRulesPaths instead.
func (l Linter) WithCustomRules(paths []string) Linter {
	return l.WithCustomRulesPaths(paths...)
}

// WithCustomRulesPaths adds custom rules for evaluation, from the Rego (and data) files provided at paths.
// Empty strings are ignored.
func (l Linter) WithCustomRulesPaths(paths ...string) Linter {
	for _, path := range paths {
		if path == "" {
			continue
		}

		if rio.IsDir(path) {
			l = l.WithCustomRulesFromFS(os.DirFS(path), ".")
		} else {
			contents, err := os.ReadFile(path)
			if err != nil {
				l.customRuleError = fmt.Errorf("failed to read custom rule file %s: %w", path, err)

				return l.notPrepared()
			}

			l = l.WithCustomRulesFromFS(fstest.MapFS{filepath.Base(path): &fstest.MapFile{Data: contents}}, ".")
		}
	}

	return l.notPrepared()
}

// WithCustomRulesFromFS adds custom rules for evaluation from a filesystem implementing the [fs.FS] interface.
// A root path within the filesystem must also be specified. Note, _test.rego files will be ignored.
func (l Linter) WithCustomRulesFromFS(f fs.FS, rootPath string) Linter {
	if f != nil {
		if modules, err := rio.ModulesFromCustomRuleFS(f, rootPath); err != nil {
			l.customRuleError = err
		} else {
			l.customRuleModules = append(l.customRuleModules, outil.Values(modules)...)
		}
	}

	return l.notPrepared()
}

// WithDebugMode enables debug mode.
func (l Linter) WithDebugMode(debugMode bool) Linter {
	l.debugMode = debugMode

	return l
}

// WithUserConfig provides config overrides set by the user.
// Passing nil can be used to clear any previously set user config
// and revert to using only the embedded configuration.
func (l Linter) WithUserConfig(cfg *config.Config) Linter {
	l.userConfig = cfg

	return l.notPrepared()
}

// WithDisabledRules disables provided rules. This overrides configuration provided in file.
func (l Linter) WithDisabledRules(disable ...string) Linter {
	l.params = l.params.WithDisable(disable...)

	return l.notPrepared()
}

// WithDisableAll disables all rules when set to true. This overrides configuration provided in file.
func (l Linter) WithDisableAll(disableAll bool) Linter {
	l.params = l.params.WithDisableAll(disableAll)

	return l.notPrepared()
}

// WithDisabledCategories disables provided categories of rules. This overrides configuration provided in file.
func (l Linter) WithDisabledCategories(disableCategory ...string) Linter {
	l.params = l.params.WithDisableCategory(disableCategory...)

	return l.notPrepared()
}

// WithEnabledRules enables provided rules. This overrides configuration provided in file.
func (l Linter) WithEnabledRules(enable ...string) Linter {
	l.params = l.params.WithEnable(enable...)

	return l.notPrepared()
}

// WithEnableAll enables all rules when set to true. This overrides configuration provided in file.
func (l Linter) WithEnableAll(enableAll bool) Linter {
	l.params = l.params.WithEnableAll(enableAll)

	return l.notPrepared()
}

// WithEnabledCategories enables provided categories of rules. This overrides configuration provided in file.
func (l Linter) WithEnabledCategories(enableCategory ...string) Linter {
	l.params = l.params.WithEnableCategory(enableCategory...)

	return l.notPrepared()
}

// WithIgnore excludes files matching patterns. This overrides configuration provided in file.
func (l Linter) WithIgnore(ignore []string) Linter {
	if l.params == nil {
		l.params = params.New()
	}

	l.params = l.params.WithIgnoreFiles(ignore...)

	return l.notPrepared()
}

// WithMetrics enables metrics collection.
func (l Linter) WithMetrics(m metrics.Metrics) Linter {
	l.metrics = m

	return l
}

func (l Linter) WithPrintHook(printHook print.Hook) Linter {
	l.printHook = printHook

	return l
}

// WithProfiling enables profiling metrics.
func (l Linter) WithProfiling(enabled bool) Linter {
	l.profiling = enabled

	return l
}

// WithInstrumentation enables instrumentation metrics.
func (l Linter) WithInstrumentation(enabled bool) Linter {
	l.instrumentation = enabled

	return l
}

// WithPathPrefix sets the root path prefix for the linter.
// A root directory prefix can be used to resolve relative paths
// referenced in the linter configuration with absolute file paths or URIs.
func (l Linter) WithPathPrefix(pathPrefix string) Linter {
	l.pathPrefix = pathPrefix

	return l.notPrepared()
}

// WithExportAggregates enables the setting of intermediate aggregate data
// on the final report. This is useful when you want to collect and
// aggregate state from multiple different linting runs.
func (l Linter) WithExportAggregates(enabled bool) Linter {
	l.exportAggregates = enabled

	return l
}

// WithCollectQuery forcibly enables the collect query even when there is
// only one file to lint.
func (l Linter) WithCollectQuery(enabled bool) Linter {
	l.useCollectQuery = enabled

	return l
}

// Prepare stores linter preparation state, like the determined configuration,
// and the query perpared for linting.
// Experimental: while used internally, the details of what is prepared here
// are very likely to change in the future, and this method should not yet be
// relied on by external clients.
func (l Linter) Prepare(ctx context.Context) (_ Linter, err error) {
	l.startTimer(regalmetrics.RegalPrepare)
	defer l.stopTimer(regalmetrics.RegalPrepare)

	l.params = cmp.Or(l.params, params.None)

	l.combinedCfg, err = l.GetConfig()
	if err != nil {
		return l, fmt.Errorf("failed to merge config: %w", err)
	}

	if l.customRuleError != nil {
		return l, fmt.Errorf("failed to load custom rules: %w", l.customRuleError)
	}

	if l.debugMode && l.printHook == nil {
		l.printHook = util.StderrPrintHook
	}

	l.preparedQuery, err = ogre.NewWithBundle(lintQuery, rbundle.LoadedNoLSP()).
		WithModules(l.customModulesMap()).
		WithStore(ogre.NewStoreFromObject(l.prepareData(l.combinedCfg))).
		WithMetrics(l.metrics).
		WithPrintHook(l.printHook).
		WithInstrumentation(l.instrumentation).
		WithSkipSafetyChecks(l.canSkipCompilerStages()).
		Prepare(ctx)

	if err = wrapNonValidationError(err, "failed preparing query for linting"); err == nil {
		if err = wrapNonValidationError(l.regoPrepare(ctx), "failed preparing Rego"); err == nil {
			l.isPrepared = true
		}
	}

	return l, err
}

// MustPrepare prepares the linter and panics on errors. Mostly used for tests.
// Experimental: see description of Prepare.
func (l Linter) MustPrepare(ctx context.Context) Linter {
	l, err := l.Prepare(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to prepare linter: %v", err))
	}

	return l
}

// Lint runs the linter on provided policies.
func (l Linter) Lint(ctx context.Context) (r report.Report, err error) {
	l.startTimer(regalmetrics.RegalLint)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !l.isPrepared {
		if l, err = l.Prepare(ctx); err != nil {
			return r, wrapNonValidationError(err, "failed to prepare linter")
		}
	}

	ignore := l.combinedCfg.Ignore.Files
	if len(l.params.IgnoreFiles) > 0 {
		ignore = l.params.IgnoreFiles
	}

	l.startTimer(regalmetrics.RegalFilterIgnoredFiles)

	filtered, err := config.FilterIgnoredPaths(l.inputPaths, ignore, true, l.pathPrefix)
	if err != nil {
		return r, fmt.Errorf("errors encountered when reading files to lint: %w", err)
	}

	l.stopTimer(regalmetrics.RegalFilterIgnoredFiles)
	l.startTimer(regalmetrics.RegalInputParse)

	var versionsMap map[string]ast.RegoVersion

	if l.pathPrefix != "" && !strings.HasPrefix(l.pathPrefix, "file://") {
		versionsMap, err = config.AllRegoVersions(l.pathPrefix, l.combinedCfg)
		if err != nil && l.debugMode {
			log.Printf("failed to get configured Rego versions: %v", err)
		}
	}

	inputFromPaths, err := rules.InputFromPaths(filtered, l.pathPrefix, versionsMap)
	if err != nil {
		return r, fmt.Errorf("errors encountered when reading files to lint: %w", err)
	}

	l.stopTimer(regalmetrics.RegalInputParse)

	input := inputFromPaths

	if l.inputModules != nil {
		l.startTimer(regalmetrics.RegalFilterIgnoredModules)

		filteredPaths, err := config.FilterIgnoredPaths(l.inputModules.FileNames, ignore, false, l.pathPrefix)
		if err != nil {
			return r, fmt.Errorf("failed to filter paths: %w", err)
		}

		for _, filename := range filteredPaths {
			input.FileNames = append(input.FileNames, filename)
			input.Modules[filename] = l.inputModules.Modules[filename]
			input.FileContent[filename] = l.inputModules.FileContent[filename]
		}

		l.stopTimer(regalmetrics.RegalFilterIgnoredModules)
	}

	if len(l.inputPaths) == 0 && l.inputModules == nil {
		return r, errNothingToLint
	}

	regoReport, err := l.lint(ctx, input)
	if err != nil {
		return r, fmt.Errorf("failed to lint using Rego rules: %w", err)
	}

	if len(input.FileNames) > 1 {
		allAggregates := regoReport.Aggregates

		if allAggregates != nil && allAggregates.Len() > 0 {
			aggregateReport, err := l.lintWithAggregateRules(ctx, allAggregates, regoReport.IgnoreDirectives)
			if err != nil {
				return r, fmt.Errorf("failed to lint using Rego aggregate rules: %w", err)
			}

			regoReport.Violations = append(regoReport.Violations, aggregateReport.Violations...)

			if l.profiling {
				regoReport.AggregateProfile = aggregateReport.AggregateProfile
			}
		}
	}

	regoReport.Summary = report.Summary{
		FilesScanned:  len(input.FileNames),
		FilesFailed:   len(regoReport.ViolationsFileCount()),
		RulesSkipped:  l.countSkippedFromNotices(ctx, regoReport),
		NumViolations: len(regoReport.Violations),
	}

	if !l.exportAggregates {
		regoReport.Aggregates = nil
	}

	l.stopTimer(regalmetrics.RegalLint)

	if l.metrics != nil {
		regoReport.Metrics = l.metrics.All()
	}

	if l.profiling && regoReport.AggregateProfile != nil {
		regoReport.AggregateProfileToSortedProfile(10)
		regoReport.AggregateProfile = nil
	}

	return *regoReport, nil
}

func (h *determineRulesHandler) Handle(result ogre.Result) error {
	enabled, ok := result.Value.(ast.Object)
	if !ok {
		return errNoEnabledRules
	}

	return enabled.Iter(func(category, rules *ast.Term) error {
		categoryRules, ok := rules.Value.(ast.Object)
		if !ok {
			return fmt.Errorf("expected list of enabled rules for category %s, didn't get it", category)
		}

		titles := categoryRules.Keys()
		h.ruleKeys = slices.Grow(h.ruleKeys, len(titles))

		for _, title := range titles {
			titleStr, _ := title.Value.(ast.String)

			h.ruleKeys = append(h.ruleKeys, string(titleStr))
		}

		return nil
	})
}

// DetermineEnabledRules returns the list of rules that are enabled based on
// the supplied configuration. This makes use of the linter rule settings
// to produce a single list of the rules that are to be run on this linter
// instance.
func (l Linter) DetermineEnabledRules(ctx context.Context) ([]string, error) {
	conf, err := l.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	pq, err := ogre.New(enabledRulesQuery).
		WithStore(ogre.NewStoreFromObject(l.prepareData(conf))).
		Prepare(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed preparing query: %w", err)
	}

	handler := &determineRulesHandler{}

	input := ast.InternedEmptyObjectValue

	ex := pq.Evaluator().WithInput(input).WithResultHandler(handler)

	if err = ex.Eval(ctx); err != nil {
		return nil, fmt.Errorf("failed to evaluate enabled rules query: %w", err)
	}

	return outil.Sorted(handler.ruleKeys), nil
}

// GetConfig returns the final configuration for the linter, i.e. Regal's default
// configuration plus any user-provided configuration merged on top of it.
func (l Linter) GetConfig() (*config.Config, error) {
	if l.combinedCfg != nil {
		return l.combinedCfg, nil
	}

	merged := l.userConfig.WithRegalDefaults()
	if l.debugMode {
		_, _ = io.WriteString(log.Writer(), "merged provided and user config:\n"+merged.String())
	}

	return merged, nil
}

func (l Linter) countSkippedFromNotices(ctx context.Context, r *report.Report) int {
	s := l.preparedQuery.Store()

	// Nesting galore. Any better way? A Rego transform would be nice.
	if noticesAny, err := storage.ReadOne(ctx, s.Storage(), internalPreparedNoticesPath); err == nil {
		if categoriesObj, ok := noticesAny.(ast.Object); ok {
			for _, v := range funsafe.ObjectElems(categoriesObj) {
				if noticesObj, ok := v.Value().Value.(ast.Object); ok {
					for _, vv := range funsafe.ObjectElems(noticesObj) {
						if notices, ok := vv.Value().Value.(ast.Set); ok {
							for v := range rast.ValuesOfType[ast.Object](notices.Slice()) {
								r.Notices = append(r.Notices, report.NoticeFromObject(v))
							}
						}
					}
				}
			}
		}
	}

	r.Notices = slices.Compact(outil.SortedFunc(r.Notices, report.Notice.ByTitle))

	return outil.Count(severityNotNone, r.Notices...)
}

func (h *prepareResultHandler) Handle(result ogre.Result) error {
	obj, ok := result.Value.(ast.Object)
	if !ok {
		return fmt.Errorf("expected prepared result object, got: %T", result.Value)
	}

	prep, ok := rast.GetValue[ast.Object](obj, "prepared")
	if !ok {
		return errNoPreparedField
	}

	if validate, ok := rast.GetValue[ast.Object](prep, "validate"); ok {
		if unknown, ok := rast.GetValue[ast.Set](validate, "unknown"); ok {
			if messages := outil.Map(unknown.Slice(), rast.Text); len(messages) > 0 {
				return &ValidationError{Messages: messages}
			}
		}
	}

	ctx := emptyCtx

	// Ensure all intermediate path segments exist before writing the leaf.
	for i := 1; i < len(preparedPath); i++ {
		prefix := preparedPath[:i]
		if _, err := h.stg.Read(ctx, h.txn, prefix); err != nil {
			stErr, ok := errors.AsType[*storage.Error](err)
			if !ok || stErr.Code != storage.NotFoundErr {
				return util.WrapErr(err, "failed to read intermediate path")
			}

			if err := h.stg.Write(ctx, h.txn, storage.AddOp, prefix, ast.ObjectTerm()); err != nil {
				return util.WrapErr(err, "failed to create intermediate path")
			}
		}
	}

	// Replace if the leaf path exists, otherwise add it.
	if _, err := h.stg.Read(ctx, h.txn, preparedPath); err == nil {
		if err := h.stg.Write(ctx, h.txn, storage.ReplaceOp, preparedPath, prep); err != nil {
			return util.WrapErr(err, "failed to replace prepared data in store")
		}
	} else {
		stErr, ok := errors.AsType[*storage.Error](err)
		if !ok || stErr.Code != storage.NotFoundErr {
			return util.WrapErr(err, "failed to read prepared path")
		}

		if err := h.stg.Write(ctx, h.txn, storage.AddOp, preparedPath, prep); err != nil {
			return util.WrapErr(err, "failed to add prepared data to store")
		}
	}

	return nil
}

func (l Linter) regoPrepare(ctx context.Context) error {
	l.startTimer(regalmetrics.RegalPrepareRego)
	defer l.stopTimer(regalmetrics.RegalPrepareRego)

	stg := l.preparedQuery.Store().Storage()
	txn := storage.NewTransactionOrDie(ctx, stg, storage.WriteParams)

	err := l.preparedQuery.Evaluator().
		WithTransaction(txn).
		WithInput(preparedInput).
		WithResultHandler(&prepareResultHandler{stg: stg, txn: txn}).
		Eval(ctx)
	if err != nil {
		stg.Abort(ctx, txn)

		return wrapNonValidationError(err, "failed to evaluate prepare query")
	}

	return stg.Commit(ctx, txn)
}

func (l Linter) notPrepared() Linter {
	l.isPrepared = false

	return l
}

// Same logic as from rego.ParsedModule.
func (l Linter) customModulesMap() map[string]*ast.Module {
	numModules := len(l.customRuleModules)
	if numModules == 0 {
		return nil
	}

	m := make(map[string]*ast.Module, numModules)

	for _, module := range l.customRuleModules {
		if module.Package.Location != nil {
			m[module.Package.Location.File] = module
		} else {
			m[fmt.Sprintf("module_%p.rego", module)] = module
		}
	}

	return m
}

func (l Linter) prepareData(conf *config.Config) ast.Object {
	userConf := ast.InternedEmptyObject
	if l.userConfig != nil {
		userConf = ast.NewTerm(l.userConfig.ToValue())
	}

	rules := map[string][]*ast.Term{}

	for _, b := range l.ruleBundles { // Add all built-in rules
		for _, module := range b.Modules {
			if ref := module.Parsed.Package.Path[1:]; len(ref) == 4 && rast.Text(ref[1]) == "rules" {
				category := rast.Text(ref[2])

				rules[category] = append(rules[category], ref[3])
			}
		}
	}

	for _, module := range l.customRuleModules { // Add any custom rules
		if ref := module.Package.Path[1:]; len(ref) == 5 && rast.Text(ref[2]) == "rules" {
			category := rast.Text(ref[3])
			rules[category] = append(rules[category], ref[4])
		}
	}

	rulesObj := ast.NewObjectWithCapacity(len(rules))
	for cat, rules := range rules {
		rast.Insert(rulesObj, cat, ast.SetTerm(rules...))
	}

	return ast.NewObject(
		rast.Item("eval", ast.ObjectTerm(rast.Item("params", l.params.ToTerm()))),
		rast.Item("internal", ast.ObjectTerm(
			rast.Item("combined_config", ast.NewTerm(conf.ToValue())),
			rast.Item("user_config", userConf),
			rast.Item("capabilities", config.OpaCapsForThisVersionTerm()),
			rast.Item("path_prefix", ast.InternedTerm(l.pathPrefix)),
			rast.Item("prepared", ast.InternedNullTerm),
			rast.Item("rules", ast.NewTerm(rulesObj)),
		)),
	)
}

func (h *lintResultHandler) Handle(result ogre.Result) error {
	r, err := report.FromQueryResult(result.Value, false)
	if err != nil {
		return fmt.Errorf("failed to convert query result to report: %w", err)
	}

	if profiler := result.Evaluator.Profiler(); profiler.Enabled() {
		// Perhaps we'll want to make this number configurable later, but do note that
		// this is only the top 10 locations for a *single* file, not the final report.
		profRep := profiler.ReportTopNResults(10, []string{"total_time_ns"})

		r.AggregateProfile = make(map[string]report.ProfileEntry, len(profRep))
		for _, rs := range profRep {
			r.AggregateProfile[rs.Location.String()] = regalmetrics.FromExprStats(rs)
		}
	}

	h.results[result.Evaluator.ID()] = r

	return nil
}

func (l Linter) lint(ctx context.Context, input rules.Input) (*report.Report, error) {
	l.startTimer(regalmetrics.RegalLintRego)
	defer l.stopTimer(regalmetrics.RegalLintRego)

	numFiles := len(input.FileNames)
	operationCollect := numFiles > 1 || l.useCollectQuery

	// NB(sr): We benchmarked using `wg.SetLimit(runtime.GOMAXPROCS(-1))` here, but performance
	// got a little worse. So let's not bother.
	wg, ctx := errgroup.WithContext(ctx)
	results := make([]*report.Report, numFiles)

	l.preparedQuery.StartReadTransaction(ctx)
	defer l.preparedQuery.EndReadTransaction(ctx)

	prepQuery := l.preparedQuery
	profiling := l.profiling
	handler := &lintResultHandler{results: results}

	for i, name := range input.FileNames {
		ctxNoEscape := ctx

		wg.Go(func() error {
			inputValue, err := transform.ToAST(name, input.FileContent[name], input.Modules[name], operationCollect)
			if err != nil {
				return fmt.Errorf("failed to transform input value: %w", err)
			} else {
				ex := prepQuery.Evaluator().WithInput(inputValue)
				if profiling {
					ex = ex.WithProfiler(profiler.New())
				}

				if err := ex.WithID(int64(i)).WithResultHandler(handler).Eval(ctxNoEscape); err != nil {
					return fmt.Errorf("error evaluating file %s: %w", name, err)
				}
			}

			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context cancelled: %w", err)
		}

		return nil, fmt.Errorf("error encountered in rule evaluation %w", err)
	}

	if len(results) == 0 {
		return &report.Report{}, nil
	}

	l.startTimer(regalmetrics.RegalMergeReport)
	defer l.stopTimer(regalmetrics.RegalMergeReport)

	regoReport := results[0]

	for i := range results[1:] {
		i++
		regoReport.Violations = append(regoReport.Violations, results[i].Violations...)
		regoReport.Notices = append(regoReport.Notices, results[i].Notices...)

		// Since the "primary key" is the file name, there is no need to handle collisions here.
		results[i].Aggregates.Foreach(regoReport.Aggregates.Insert)

		if results[i].IgnoreDirectives != nil {
			regoReport.IgnoreDirectives, _ = regoReport.IgnoreDirectives.Merge(results[i].IgnoreDirectives)
		}

		if l.profiling {
			regoReport.AddProfileEntries(results[i].AggregateProfile)
		}
	}

	return regoReport, nil
}

func (h *aggregateResultHandler) Handle(result ogre.Result) (err error) {
	h.rep, err = report.FromQueryResult(result.Value, true)
	if err != nil {
		return fmt.Errorf("failed to convert query result to report: %w", err)
	}

	if profiler := result.Evaluator.Profiler(); profiler.Enabled() {
		profRep := profiler.ReportTopNResults(10, []string{"total_time_ns"})

		h.rep.AggregateProfile = make(map[string]report.ProfileEntry, len(profRep))
		for _, rs := range profRep {
			h.rep.AggregateProfile[rs.Location.String()] = regalmetrics.FromExprStats(rs)
		}
	}

	return nil
}

func (l Linter) lintWithAggregateRules(
	ctx context.Context,
	aggregates ast.Object,
	ignoreDirectives ast.Object,
) (*report.Report, error) {
	l.startTimer(regalmetrics.RegalLintRegoAggregate)
	defer l.stopTimer(regalmetrics.RegalLintRegoAggregate)

	inputValue := ast.NewObject( //nolint:forcetypeassert
		rast.Item("ignore_directives", ast.NewTerm(cmp.Or(ignoreDirectives, ast.InternedEmptyObjectValue.(ast.Object)))),
		rast.Item("regal", aggregateRegalObject),
	)

	if aggregates != nil && aggregates.Len() > 0 {
		rast.Insert(inputValue, "aggregates_internal", ast.NewTerm(aggregates))
	}

	hnd := &aggregateResultHandler{rep: new(report.Report)}

	ex := l.preparedQuery.Evaluator().WithInput(inputValue).WithResultHandler(hnd)
	if l.profiling {
		ex = ex.WithProfiler(profiler.New())
	}

	if err := ex.Eval(ctx); err != nil {
		return nil, fmt.Errorf("error evaluating aggregate rules: %w", err)
	}

	return hnd.rep, nil
}

func (l Linter) startTimer(name string) {
	if l.metrics != nil {
		l.metrics.Timer(name).Start()
	}
}

func (l Linter) stopTimer(name string) {
	if l.metrics != nil {
		l.metrics.Timer(name).Stop()
	}
}

// canSkipCompilerStages returns true if some compiler stages safely can be skipped.
// The requirements for this is that a Regal version is set, which it typically only will be
// when running as a "standalone" Regal binary built for a release. In that case, AND we are
// able to assert that we aren't compiling any other code than our own embedded Rego, we can
// skip stages we know the bundle compiles successfully with, when enabled.
func (l Linter) canSkipCompilerStages() bool {
	return version.Version != "" && len(l.ruleBundles) == 1 && len(l.customRuleModules) == 0
}

func (e *ValidationError) Error() string {
	sb := new(strings.Builder)
	if len(e.Messages) > 0 {
		rio.WriteStrings(sb, "  - ", e.Messages[0])

		for _, msg := range e.Messages[1:] {
			rio.WriteStrings(sb, "\n  - ", msg)
		}
	}

	return sb.String()
}

// wrapNonValidationError wraps the error with the provided message if it is not a ValidationError.
// ValidationErrors should not be wrapped as we want the message to be displayed to the user without
// including additional context that isn't helpful to the user.
func wrapNonValidationError(err error, msg string) error {
	if err != nil {
		if _, ok := errors.AsType[*ValidationError](err); !ok {
			err = fmt.Errorf("%s: %w", msg, err)
		}
	}

	return err
}

func severityNotNone(n report.Notice) bool {
	return n.Severity != "none"
}
