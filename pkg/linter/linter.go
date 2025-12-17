package linter

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing/fstest"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/profiler"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"
	"github.com/open-policy-agent/opa/v1/topdown/print"
	outil "github.com/open-policy-agent/opa/v1/util"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rio "github.com/open-policy-agent/regal/internal/io"
	regalmetrics "github.com/open-policy-agent/regal/internal/metrics"
	"github.com/open-policy-agent/regal/internal/ogre"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/intern"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
	"github.com/open-policy-agent/regal/pkg/rules"

	_ "github.com/open-policy-agent/regal/pkg/builtins"
)

// Linter stores data to use for linting.
type Linter struct {
	printHook            print.Hook
	metrics              metrics.Metrics
	inputModules         *rules.Input
	userConfig           *config.Config
	combinedCfg          *config.Config
	pathPrefix           string
	customRuleError      error
	inputPaths           []string
	ruleBundles          []*bundle.Bundle
	disable              []string
	disableCategory      []string
	enable               []string
	enableCategory       []string
	ignoreFiles          []string
	customRuleModules    []*ast.Module
	overriddenAggregates ast.Object
	useCollectQuery      bool
	debugMode            bool
	exportAggregates     bool
	disableAll           bool
	enableAll            bool
	profiling            bool
	instrumentation      bool
	isPrepared           bool

	preparedQuery *ogre.Query
}

var (
	eqRef     = ast.RefTerm(ast.VarTerm(ast.Equality.Name))
	lintQuery = []*ast.Expr{{ // lint = data.regal.main.lint.
		Terms: []*ast.Term{eqRef, ast.VarTerm("lint"), ast.RefTerm(
			ast.DefaultRootDocument,
			ast.InternedTerm("regal"),
			ast.InternedTerm("main"),
			ast.InternedTerm("lint"),
		)},
	}}
	enabledRulesQuery = []*ast.Expr{{ // enabled = data.regal.main.enabled_rules.
		Terms: []*ast.Term{eqRef, ast.VarTerm("enabled"), ast.RefTerm(
			ast.DefaultRootDocument,
			ast.InternedTerm("regal"),
			ast.InternedTerm("main"),
			ast.InternedTerm("enabled_rules"),
		)},
	}}

	aggregateRegalObject = ast.ObjectTerm(
		ast.Item(ast.InternedTerm("operations"), ast.ArrayTerm(ast.InternedTerm("aggregate"))),
		ast.Item(ast.InternedTerm("file"), ast.ObjectTerm(
			ast.Item(ast.InternedTerm("name"), ast.InternedTerm("__aggregate_report__")),
			ast.Item(ast.InternedTerm("lines"), ast.InternedEmptyArray),
		)),
	)

	preparedPath = storage.Path{"internal", "prepared"}
)

func init() {
	ast.InternStringTerm(
		"eval", "disable_all", "disable_category", "disable", "enable_all", "enable_category", "enable", "ignore_files",
	)
}

// NewLinter creates a new Regal linter.
func NewLinter() Linter {
	return Linter{
		ruleBundles: []*bundle.Bundle{rbundle.Loaded()},
	}
}

// NewEmptyLinter creates a linter with no rule bundles.
func NewEmptyLinter() Linter {
	return Linter{}
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
func (l Linter) WithCustomRules(paths []string) Linter {
	for _, path := range paths {
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

// WithCustomRulesFromFS adds custom rules for evaluation from a filesystem implementing the fs.FS interface.
// A root path within the filesystem must also be specified. Note, _test.rego files will be ignored.
func (l Linter) WithCustomRulesFromFS(f fs.FS, rootPath string) Linter {
	if f != nil {
		modules, err := rio.ModulesFromCustomRuleFS(f, rootPath)
		if err != nil {
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
func (l Linter) WithUserConfig(cfg config.Config) Linter {
	l.userConfig = &cfg

	return l.notPrepared()
}

// WithDisabledRules disables provided rules. This overrides configuration provided in file.
func (l Linter) WithDisabledRules(disable ...string) Linter {
	l.disable = disable

	return l.notPrepared()
}

// WithDisableAll disables all rules when set to true. This overrides configuration provided in file.
func (l Linter) WithDisableAll(disableAll bool) Linter {
	l.disableAll = disableAll

	return l.notPrepared()
}

// WithDisabledCategories disables provided categories of rules. This overrides configuration provided in file.
func (l Linter) WithDisabledCategories(disableCategory ...string) Linter {
	l.disableCategory = disableCategory

	return l.notPrepared()
}

// WithEnabledRules enables provided rules. This overrides configuration provided in file.
func (l Linter) WithEnabledRules(enable ...string) Linter {
	l.enable = enable

	return l.notPrepared()
}

// WithEnableAll enables all rules when set to true. This overrides configuration provided in file.
func (l Linter) WithEnableAll(enableAll bool) Linter {
	l.enableAll = enableAll

	return l.notPrepared()
}

// WithEnabledCategories enables provided categories of rules. This overrides configuration provided in file.
func (l Linter) WithEnabledCategories(enableCategory ...string) Linter {
	l.enableCategory = enableCategory

	return l.notPrepared()
}

// WithIgnore excludes files matching patterns. This overrides configuration provided in file.
func (l Linter) WithIgnore(ignore []string) Linter {
	l.ignoreFiles = ignore

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

// WithAggregates supplies aggregate data to a linter instance.
// Likely generated in a previous run, and used to provide a global context to
// a subsequent run of a single file lint.
func (l Linter) WithAggregates(aggs ast.Object) Linter {
	l.overriddenAggregates = aggs

	return l
}

// Prepare stores linter preparation state, like the determined configuration,
// and the query perpared for linting.
// Experimental: while used internally, the details of what is prepared here
// are very likely to change in the future, and this method should not yet be
// relied on by external clients.
func (l Linter) Prepare(ctx context.Context) (Linter, error) {
	l.startTimer(regalmetrics.RegalPrepare)
	defer l.stopTimer(regalmetrics.RegalPrepare)

	conf, err := l.GetConfig()
	if err != nil {
		return l, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := l.validate(conf); err != nil {
		return l, fmt.Errorf("validation failed: %w", err)
	}

	l.combinedCfg = conf

	if l.debugMode && l.printHook == nil {
		l.printHook = topdown.NewPrintHook(os.Stderr)
	}

	l.preparedQuery, err = ogre.New(lintQuery).
		WithModules(l.customModulesMap()).
		WithStore(ogre.NewStoreFromObject(ctx, l.prepareData(conf))).
		WithMetrics(l.metrics).
		WithPrintHook(l.printHook).
		WithInstrumentation(l.instrumentation).
		Prepare(ctx)
	if err != nil {
		return l, fmt.Errorf("failed preparing query for linting: %w", err)
	}

	// TODO: this offers a tremendous perf boost in projects, but may slow down
	// linting a single file. investigate further, and if we can skip this in that case.
	if err := l.regoPrepare(ctx); err != nil {
		return l, fmt.Errorf("failed to prepare Rego: %w", err)
	}

	l.isPrepared = true

	return l, nil
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
func (l Linter) Lint(ctx context.Context) (report.Report, error) {
	l.startTimer(regalmetrics.RegalLint)

	if !l.isPrepared {
		var err error
		if l, err = l.Prepare(ctx); err != nil {
			return report.Report{}, fmt.Errorf("failed to prepare linter: %w", err)
		}
	}

	ignore := l.combinedCfg.Ignore.Files

	if len(l.ignoreFiles) > 0 {
		ignore = l.ignoreFiles
	}

	l.startTimer(regalmetrics.RegalFilterIgnoredFiles)

	filtered, err := config.FilterIgnoredPaths(l.inputPaths, ignore, true, l.pathPrefix)
	if err != nil {
		return report.Report{}, fmt.Errorf("errors encountered when reading files to lint: %w", err)
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
		return report.Report{}, fmt.Errorf("errors encountered when reading files to lint: %w", err)
	}

	l.stopTimer(regalmetrics.RegalInputParse)

	input := inputFromPaths

	if l.inputModules != nil {
		l.startTimer(regalmetrics.RegalFilterIgnoredModules)

		filteredPaths, err := config.FilterIgnoredPaths(l.inputModules.FileNames, ignore, false, l.pathPrefix)
		if err != nil {
			return report.Report{}, fmt.Errorf("failed to filter paths: %w", err)
		}

		for _, filename := range filteredPaths {
			input.FileNames = append(input.FileNames, filename)
			input.Modules[filename] = l.inputModules.Modules[filename]
			input.FileContent[filename] = l.inputModules.FileContent[filename]
		}

		l.stopTimer(regalmetrics.RegalFilterIgnoredModules)
	}

	hasAggregatesOverride := l.overriddenAggregates != nil && l.overriddenAggregates.Len() > 0
	if len(l.inputPaths) == 0 && l.inputModules == nil && !hasAggregatesOverride {
		return report.Report{}, errors.New("nothing provided to lint")
	}

	regoReport, err := l.lint(ctx, input)
	if err != nil {
		return report.Report{}, fmt.Errorf("failed to lint using Rego rules: %w", err)
	}

	var allAggregates ast.Object
	if hasAggregatesOverride {
		allAggregates = l.overriddenAggregates
	} else if len(input.FileNames) > 1 {
		allAggregates = regoReport.Aggregates
	}

	if allAggregates != nil && allAggregates.Len() > 0 {
		aggregateReport, err := l.lintWithAggregateRules(ctx, allAggregates, regoReport.IgnoreDirectives)
		if err != nil {
			return report.Report{}, fmt.Errorf("failed to lint using Rego aggregate rules: %w", err)
		}

		regoReport.Violations = append(regoReport.Violations, aggregateReport.Violations...)

		if l.profiling {
			regoReport.AggregateProfile = aggregateReport.AggregateProfile
		}
	}

	regoReport, skippedCount := l.countSkippedFromNotices(ctx, regoReport)

	regoReport.Summary = report.Summary{
		FilesScanned:  len(input.FileNames),
		FilesFailed:   len(regoReport.ViolationsFileCount()),
		RulesSkipped:  skippedCount,
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

	return regoReport, nil
}

// DetermineEnabledRules returns the list of rules that are enabled based on
// the supplied configuration. This makes use of the linter rule settings
// to produce a single list of the rules that are to be run on this linter
// instance.
func (l Linter) DetermineEnabledRules(ctx context.Context) ([]string, []string, error) {
	conf, err := l.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge config: %w", err)
	}

	pq, err := ogre.New(enabledRulesQuery).
		WithStore(ogre.NewStoreFromObject(ctx, l.prepareData(conf))).
		Prepare(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed preparing query: %w", err)
	}

	var regular, aggregate []string

	input := ast.InternedEmptyObject.Value

	ex := pq.Evaluator().WithInput(input).WithResultHandler(func(result ast.Value) error {
		enabled, ok := result.(ast.Object)
		if !ok {
			return errors.New("expected enabled rules object, didn't get it")
		}

		// Since we currently have no reliable way to determine whether a rule is an aggregate
		// rule or not in Rego without actually evaluating it, we query the comiler for this
		// information for each rule in the result. Long term, we should figure out the best way
		// to do this in Rego only.
		ref := ast.DefaultRootRef.Extend(ast.MustParseRef("regal.rules.category.title.aggregate"))

		return enabled.Iter(func(category, rules *ast.Term) error {
			categoryRules, ok := rules.Value.(ast.Object)
			if !ok {
				return fmt.Errorf("expected list of enabled rules for category %s, didn't get it", category)
			}

			ref[3] = category

			for _, title := range categoryRules.Keys() {
				ref[4] = title
				titleStr, _ := title.Value.(ast.String)

				if rules := pq.Compiler().GetRulesExact(ref); len(rules) == 0 {
					regular = append(regular, string(titleStr))
				} else {
					aggregate = append(aggregate, string(titleStr))
				}
			}

			return nil
		})
	})

	if err = ex.Eval(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to evaluate enabled rules query: %w", err)
	}

	return util.Sorted(regular), util.Sorted(aggregate), nil
}

// GetConfig returns the final configuration for the linter, i.e. Regal's default
// configuration plus any user-provided configuration merged on top of it.
func (l Linter) GetConfig() (*config.Config, error) {
	if l.combinedCfg != nil {
		return l.combinedCfg, nil
	}

	mergedConf, err := config.WithDefaultsFromBundle(rbundle.Loaded(), l.userConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read provided config: %w", err)
	}

	if l.debugMode {
		bs, err := yaml.Marshal(mergedConf)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}

		log.Println("merged provided and user config:\n", outil.ByteSliceToString(bs))
	}

	return &mergedConf, nil
}

func (l Linter) countSkippedFromNotices(ctx context.Context, r report.Report) (report.Report, int) {
	s := l.preparedQuery.Store()
	p := storage.Path{"internal", "prepared", "notices"}

	// Nesting galore. Any better way? A Rego transform would be nice.
	if noticesAny, err := storage.ReadOne(ctx, s.Storage(), p); err == nil {
		if categoriesObj, ok := noticesAny.(ast.Object); ok {
			categoriesObj.Foreach(func(_, v *ast.Term) {
				if noticesObj, ok := v.Value.(ast.Object); ok {
					noticesObj.Foreach(func(_, vv *ast.Term) {
						if notices, ok := vv.Value.(ast.Set); ok {
							for v := range rast.ValuesOfType[ast.Object](notices.Slice()) {
								r.Notices = append(r.Notices, report.NoticeFromObject(v))
							}
						}
					})
				}
			})
		}
	}

	slices.SortFunc(r.Notices, func(a, b report.Notice) int {
		return strings.Compare(a.Title, b.Title)
	})
	r.Notices = slices.Compact(r.Notices)
	rulesSkippedCounter := 0

	for _, notice := range r.Notices {
		if notice.Severity != "none" {
			rulesSkippedCounter++
		}
	}

	return r, rulesSkippedCounter
}

func (l Linter) regoPrepare(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	inputValue := ast.NewObject(
		rast.Item("regal", ast.ObjectTerm(
			rast.Item("operations", ast.ArrayTerm(ast.InternedTerm("prepare"))),
			rast.Item("file", ast.ObjectTerm(
				rast.Item("name", ast.InternedTerm("__prepare__")),
				rast.Item("rego_version", ast.InternedTerm("v1")),
			)),
		)),
	)

	stg := l.preparedQuery.Store().Storage()
	txn := storage.NewTransactionOrDie(ctx, stg, storage.WriteParams)

	// TODO: profiling, instrumentation, metrics
	ev := l.preparedQuery.Evaluator().
		WithTransaction(txn).
		WithInput(inputValue).
		WithResultHandler(func(result ast.Value) error {
			obj, ok := result.(ast.Object)
			if !ok {
				return fmt.Errorf("expected prepared result object, got: %T", result)
			}

			prep, ok := rast.GetValue[ast.Object](obj, "prepared")
			if !ok {
				return errors.New("expected 'prepared' field in result object")
			}

			return util.WrapErr(
				stg.Write(ctx, txn, storage.ReplaceOp, preparedPath, prep),
				"failed to write prepared data to store",
			)
		})

	if err := ev.Eval(ctx); err != nil {
		stg.Abort(ctx, txn)

		return fmt.Errorf("failed to evaluate prepare query: %w", err)
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
	userConf := ast.InternedEmptyObject.Value
	if l.userConfig != nil {
		userConf = l.userConfig.ToValue()
	}

	return ast.NewObject(
		rast.Item("eval", ast.ObjectTerm(
			rast.Item("params", ast.ObjectTerm(
				rast.Item("disable_all", ast.InternedTerm(l.disableAll)),
				rast.Item("disable_category", rast.ArrayTerm(l.disableCategory)),
				rast.Item("disable", rast.ArrayTerm(l.disable)),
				rast.Item("enable_all", ast.InternedTerm(l.enableAll)),
				rast.Item("enable_category", rast.ArrayTerm(l.enableCategory)),
				rast.Item("enable", rast.ArrayTerm(l.enable)),
				rast.Item("ignore_files", rast.ArrayTerm(l.ignoreFiles)),
			)),
		)),
		rast.Item("internal", ast.ObjectTerm(
			rast.Item("combined_config", ast.NewTerm(conf.ToValue())),
			rast.Item("user_config", ast.NewTerm(userConf)),
			rast.Item("capabilities", ast.NewTerm(rast.StructToValue(config.CapabilitiesForThisVersion()))),
			rast.Item("path_prefix", ast.InternedTerm(l.pathPrefix)),
			rast.Item("prepared", ast.InternedNullTerm),
		)),
	)
}

func (l Linter) validate(conf *config.Config) error {
	if l.customRuleError != nil {
		return fmt.Errorf("failed to load custom rules: %w", l.customRuleError)
	}

	validCategories := util.NewSet[string]()
	validRules := util.NewSet[string]()

	// Add all built-in rules
	for _, b := range l.ruleBundles {
		for _, module := range b.Modules {
			parts, _ := storage.NewPathForRef(module.Parsed.Package.Path)
			// 1     2     3   4
			// regal.rules.cat.rule
			if len(parts) != 4 {
				continue
			}

			validCategories.Add(parts[2])
			validRules.Add(parts[3])
		}
	}

	// Add any custom rules
	for _, module := range l.customRuleModules {
		parts, _ := storage.NewPathForRef(module.Package.Path)
		// 1      2     3     4   5
		// custom.regal.rules.cat.rule
		if len(parts) != 5 {
			continue
		}

		validCategories.Add(parts[3])
		validRules.Add(parts[4])
	}

	configuredCategories := util.NewSet(outil.Keys(conf.Rules)...)
	configuredRules := util.NewSet[string]()

	for _, cat := range conf.Rules {
		configuredRules.Add(outil.Keys(cat)...)
	}

	configuredRules.Add(l.enable...)
	configuredRules.Add(l.disable...)
	configuredCategories.Add(l.enableCategory...)
	configuredCategories.Add(l.disableCategory...)

	invalidCategories := configuredCategories.Diff(validCategories)
	invalidRules := configuredRules.Diff(validRules)

	switch {
	case invalidCategories.Size() > 0 && invalidRules.Size() > 0:
		return fmt.Errorf("unknown categories: %v, unknown rules: %v", invalidCategories, invalidRules)
	case invalidCategories.Size() > 0:
		return fmt.Errorf("unknown categories: %v", invalidCategories)
	case invalidRules.Size() > 0:
		return fmt.Errorf("unknown rules: %v", invalidRules)
	}

	return nil
}

func (l Linter) lint(ctx context.Context, input rules.Input) (report.Report, error) {
	l.startTimer(regalmetrics.RegalLintRego)
	defer l.stopTimer(regalmetrics.RegalLintRego)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numFiles := len(input.FileNames)
	operationCollect := numFiles > 1 || l.useCollectQuery

	// NB(sr): We benchmarked using `wg.SetLimit(runtime.GOMAXPROCS(-1))` here, but performance
	// got a little worse. So let's not bother.
	wg, ctx := errgroup.WithContext(ctx)
	results := make([]report.Report, numFiles)

	l.preparedQuery.StartReadTransaction(ctx)
	defer l.preparedQuery.EndReadTransaction(ctx)

	for i, name := range input.FileNames {
		wg.Go(func() error {
			inputValue, err := transform.ToAST(name, input.FileContent[name], input.Modules[name], operationCollect)
			if err != nil {
				return fmt.Errorf("failed to transform input value: %w", err)
			} else {
				ex := l.preparedQuery.Evaluator().WithInput(inputValue)

				if l.profiling {
					ex = ex.WithProfiler(profiler.New())
				}

				ex = ex.WithResultHandler(func(result ast.Value) error {
					r, err := report.FromQueryResult(result, false)
					if err != nil {
						return fmt.Errorf("failed to convert query result to report: %w", err)
					}

					if l.profiling {
						// Perhaps we'll want to make this number configurable later, but do note that
						// this is only the top 10 locations for a *single* file, not the final report.
						profRep := ex.Profiler().ReportTopNResults(10, []string{"total_time_ns"})

						r.AggregateProfile = make(map[string]report.ProfileEntry, len(profRep))
						for _, rs := range profRep {
							r.AggregateProfile[rs.Location.String()] = regalmetrics.FromExprStats(rs)
						}
					}

					results[i] = r

					return nil
				})

				if err := ex.Eval(ctx); err != nil {
					return fmt.Errorf("error evaluating file %s: %w", name, err)
				}
			}

			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return report.Report{}, fmt.Errorf("context cancelled: %w", err)
		}

		return report.Report{}, fmt.Errorf("error encountered in rule evaluation %w", err)
	}

	var regoReport report.Report

	if len(results) == 0 {
		return regoReport, nil
	}

	l.startTimer(regalmetrics.RegalMergeReport)
	defer l.stopTimer(regalmetrics.RegalMergeReport)

	regoReport = results[0]

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

func (l Linter) lintWithAggregateRules(
	ctx context.Context,
	aggregates ast.Object,
	ignoreDirectives ast.Object,
) (report.Report, error) {
	l.startTimer(regalmetrics.RegalLintRegoAggregate)
	defer l.stopTimer(regalmetrics.RegalLintRegoAggregate)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	inputValue := ast.NewObject(
		rast.Item("aggregates_internal", ast.NewTerm(cmp.Or(aggregates, intern.EmptyObject))),
		rast.Item("ignore_directives", ast.NewTerm(cmp.Or(ignoreDirectives, intern.EmptyObject))),
		rast.Item("regal", aggregateRegalObject),
	)

	var rep report.Report

	ex := l.preparedQuery.Evaluator().WithInput(inputValue)
	if l.profiling {
		ex = ex.WithProfiler(profiler.New())
	}

	ex = ex.WithResultHandler(func(result ast.Value) (err error) {
		rep, err = report.FromQueryResult(result, true)
		if err != nil {
			return fmt.Errorf("failed to convert query result to report: %w", err)
		}

		for i := range rep.Violations {
			rep.Violations[i].IsAggregate = true
		}

		if l.profiling {
			profRep := ex.Profiler().ReportTopNResults(10, []string{"total_time_ns"})

			rep.AggregateProfile = make(map[string]report.ProfileEntry, len(profRep))
			for _, rs := range profRep {
				rep.AggregateProfile[rs.Location.String()] = regalmetrics.FromExprStats(rs)
			}
		}

		return nil
	})

	if err := ex.Eval(ctx); err != nil {
		return report.Report{}, fmt.Errorf("error evaluating aggregate rules: %w", err)
	}

	return rep, nil
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
