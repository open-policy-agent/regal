package linter

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing/fstest"

	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/profiler"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"
	"github.com/open-policy-agent/opa/v1/topdown/print"
	outil "github.com/open-policy-agent/opa/v1/util"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rio "github.com/open-policy-agent/regal/internal/io"
	regalmetrics "github.com/open-policy-agent/regal/internal/metrics"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/builtins"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
	"github.com/open-policy-agent/regal/pkg/rules"
)

// Linter stores data to use for linting.
type Linter struct {
	printHook            print.Hook
	metrics              metrics.Metrics
	baseCache            topdown.BaseCache
	inputModules         *rules.Input
	userConfig           *config.Config
	combinedCfg          *config.Config
	dataBundle           *bundle.Bundle
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
	overriddenAggregates map[string][]report.Aggregate
	useCollectQuery      bool
	debugMode            bool
	exportAggregates     bool
	disableAll           bool
	enableAll            bool
	profiling            bool
	instrumentation      bool
	isPrepared           bool

	preparedQuery *rego.PreparedEvalQuery
}

var (
	lintQueryStr         = "lint := data.regal.main.lint"
	enabledRulesQueryStr = "enabled := data.regal.main.enabled_rules"
	lintQuery            = ast.MustParseBody(lintQueryStr)
	enabledRulesQuery    = ast.MustParseBody(enabledRulesQueryStr)
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
func (l Linter) WithAggregates(aggregates map[string][]report.Aggregate) Linter {
	l.overriddenAggregates = aggregates

	return l
}

// WithBaseCache sets the base cache (cache for "JSON" documents) to use for evaluation.
// This feature is **experimental** and should not be relied on by external clients for
// the time being.
func (l Linter) WithBaseCache(baseCache topdown.BaseCache) Linter {
	l.baseCache = baseCache

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
	l.dataBundle = l.createDataBundle(*conf)

	if l.preparedQuery, err = l.prepareQuery(ctx); err != nil {
		return l, fmt.Errorf("failed to prepare query: %w", err)
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

	if len(l.inputPaths) == 0 && l.inputModules == nil && len(l.overriddenAggregates) == 0 {
		return report.Report{}, errors.New("nothing provided to lint")
	}

	regoReport, err := l.lint(ctx, input)
	if err != nil {
		return report.Report{}, fmt.Errorf("failed to lint using Rego rules: %w", err)
	}

	slices.SortFunc(regoReport.Notices, func(a, b report.Notice) int {
		return strings.Compare(a.Title, b.Title)
	})
	regoReport.Notices = slices.Compact(regoReport.Notices)
	rulesSkippedCounter := 0

	for _, notice := range regoReport.Notices {
		if notice.Severity != "none" {
			rulesSkippedCounter++
		}
	}

	allAggregates := make(map[string][]report.Aggregate, len(l.overriddenAggregates)+len(regoReport.Aggregates))

	if len(l.overriddenAggregates) > 0 {
		for k, aggregates := range l.overriddenAggregates {
			allAggregates[k] = append(allAggregates[k], aggregates...)
		}
	} else if len(input.FileNames) > 1 {
		for k, aggregates := range regoReport.Aggregates {
			allAggregates[k] = append(allAggregates[k], aggregates...)
		}
	}

	if len(allAggregates) > 0 {
		aggregateReport, err := l.lintWithAggregateRules(ctx, allAggregates, regoReport.IgnoreDirectives)
		if err != nil {
			return report.Report{}, fmt.Errorf("failed to lint using Rego aggregate rules: %w", err)
		}

		regoReport.Violations = append(regoReport.Violations, aggregateReport.Violations...)

		if l.profiling {
			regoReport.AggregateProfile = aggregateReport.AggregateProfile
		}
	}

	regoReport.Summary = report.Summary{
		FilesScanned:  len(input.FileNames),
		FilesFailed:   len(regoReport.ViolationsFileCount()),
		RulesSkipped:  rulesSkippedCounter,
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

	l.dataBundle = l.createDataBundle(*conf)

	regoArgs, err := l.prepareRegoArgs(enabledRulesQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("failed preparing query %s: %w", enabledRulesQueryStr, err)
	}

	var compiler *ast.Compiler

	regoArgs = append(regoArgs, rego.CompilerHook(func(c *ast.Compiler) {
		compiler = c
	}))

	rs, err := rego.New(regoArgs...).Eval(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed evaluating query %s: %w", enabledRulesQueryStr, err)
	}

	return getEnabledRules(rs, compiler)
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

func (l Linter) prepareQuery(ctx context.Context) (*rego.PreparedEvalQuery, error) {
	regoArgs, err := l.prepareRegoArgs(lintQuery)
	if err != nil {
		return nil, fmt.Errorf("failed preparing query for linting: %w", err)
	}

	pq, err := rego.New(regoArgs...).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed preparing query for linting: %w", err)
	}

	return &pq, nil
}

func (l Linter) notPrepared() Linter {
	l.isPrepared = false

	return l
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

func getEnabledRules(rs rego.ResultSet, c *ast.Compiler) (regular, aggregate []string, err error) {
	if len(rs) == 0 {
		return nil, nil, errors.New("expected result set of at least one result, got none")
	}

	enabled, ok := rs[0].Bindings["enabled"].(map[string]any)
	if !ok {
		return nil, nil, errors.New("expected list of enabled rules, didn't get it")
	}

	// Since we currently have no reliable way to determine whether a rule is an aggregate
	// rule or not in Rego without actually evaluating it, we query the comiler for this
	// information for each rule in the result. Long term, we should figure out the best way
	// to do this in Rego only.
	ref := ast.DefaultRootRef.Extend(ast.MustParseRef("regal.rules.category.title.aggregate"))

	for category, rules := range enabled {
		categoryRules, ok := rules.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("expected list of enabled rules for category %s, didn't get it", category)
		}

		ref[3] = ast.InternedTerm(category)

		for title := range categoryRules {
			ref[4] = ast.InternedTerm(title)

			if rules := c.GetRulesExact(ref); len(rules) == 0 {
				regular = append(regular, title)
			} else {
				aggregate = append(aggregate, title)
			}
		}
	}

	return util.Sorted(regular), util.Sorted(aggregate), nil
}

func (l Linter) createDataBundle(conf config.Config) *bundle.Bundle {
	params := map[string]any{
		"disable_all":      l.disableAll,
		"disable_category": util.NilSliceToEmpty(l.disableCategory),
		"disable":          util.NilSliceToEmpty(l.disable),
		"enable_all":       l.enableAll,
		"enable_category":  util.NilSliceToEmpty(l.enableCategory),
		"enable":           util.NilSliceToEmpty(l.enable),
		"ignore_files":     util.NilSliceToEmpty(l.ignoreFiles),
	}

	return &bundle.Bundle{
		Manifest: bundle.Manifest{
			Roots:    &[]string{"internal", "eval"},
			Metadata: map[string]any{"name": "internal"},
		},
		Data: map[string]any{
			"eval": map[string]any{
				"params": params,
			},
			"internal": map[string]any{
				"combined_config": config.ToMap(conf),
				"capabilities":    rio.ToMap(config.CapabilitiesForThisVersion()),
				"path_prefix":     l.pathPrefix,
			},
		},
	}
}

func (l Linter) prepareRegoArgs(query ast.Body) ([]func(*rego.Rego), error) {
	if l.customRuleError != nil {
		return nil, fmt.Errorf("failed to load custom rules: %w", l.customRuleError)
	}

	regoArgs := append([]func(*rego.Rego){
		rego.StoreReadAST(true),
		rego.Metrics(l.metrics),
		rego.ParsedQuery(query),
		rego.Instrument(l.instrumentation),
	}, builtins.RegalBuiltinRegoFuncs...)

	if l.debugMode && l.printHook == nil {
		l.printHook = topdown.NewPrintHook(os.Stderr)
	}

	if l.printHook != nil {
		regoArgs = append(regoArgs, rego.EnablePrintStatements(true), rego.PrintHook(l.printHook))
	}

	if l.dataBundle != nil {
		regoArgs = append(regoArgs, rego.ParsedBundle("internal", l.dataBundle))
	}

	for _, m := range l.customRuleModules {
		regoArgs = append(regoArgs, rego.ParsedModule(m))
	}

	if l.ruleBundles != nil {
		for _, ruleBundle := range l.ruleBundles {
			var bundleName string
			if metadataName, ok := ruleBundle.Manifest.Metadata["name"].(string); ok {
				bundleName = metadataName
			}

			regoArgs = append(regoArgs, rego.ParsedBundle(bundleName, ruleBundle))
		}
	}

	return regoArgs, nil
}

func (l Linter) lint(ctx context.Context, input rules.Input) (report.Report, error) {
	l.startTimer(regalmetrics.RegalLintRego)
	defer l.stopTimer(regalmetrics.RegalLintRego)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numFiles := len(input.FileNames)
	operationCollect := numFiles > 1 || l.useCollectQuery

	var wg sync.WaitGroup

	// the error channel is buffered to prevent blocking
	// caused by the context cancellation happening before
	// errors are sent and the per-file goroutines can exit.
	errCh := make(chan error, numFiles)
	doneCh := make(chan bool)

	results := make([]report.Report, numFiles)

	for i, name := range input.FileNames {
		wg.Add(1)

		go func(name string) {
			defer wg.Done()

			inputValue, err := transform.ToAST(name, input.FileContent[name], input.Modules[name], operationCollect)
			if err != nil {
				errCh <- fmt.Errorf("failed to transform input value: %w", err)

				return
			}

			evalArgs := []rego.EvalOption{rego.EvalParsedInput(inputValue), rego.EvalInstrument(l.instrumentation)}

			if l.baseCache != nil {
				evalArgs = append(evalArgs, rego.EvalBaseCache(l.baseCache))
			}

			if l.metrics != nil {
				evalArgs = append(evalArgs, rego.EvalMetrics(l.metrics))
			}

			var prof *profiler.Profiler
			if l.profiling {
				prof = profiler.New()
				evalArgs = append(evalArgs, rego.EvalQueryTracer(prof))
			}

			resultSet, err := l.preparedQuery.Eval(ctx, evalArgs...)
			if err != nil {
				errCh <- fmt.Errorf("error encountered in query evaluation %w", err)

				return
			}

			result, err := report.FromResultSet(resultSet, false)
			if err != nil {
				errCh <- fmt.Errorf("failed to convert result set to report: %w", err)

				return
			}

			if l.profiling {
				// Perhaps we'll want to make this number configurable later, but do note that
				// this is only the top 10 locations for a *single* file, not the final report.
				profRep := prof.ReportTopNResults(10, []string{"total_time_ns"})

				result.AggregateProfile = make(map[string]report.ProfileEntry, len(profRep))

				for _, rs := range profRep {
					result.AggregateProfile[rs.Location.String()] = regalmetrics.FromExprStats(rs)
				}
			}

			results[i] = result
		}(name)
	}

	go func() {
		wg.Wait()

		doneCh <- true
	}()

	select {
	case <-ctx.Done():
		return report.Report{}, fmt.Errorf("context cancelled: %w", ctx.Err())
	case err := <-errCh:
		return report.Report{}, fmt.Errorf("error encountered in rule evaluation %w", err)
	case <-doneCh:
		l.startTimer(regalmetrics.RegalMergeReport)
		defer l.stopTimer(regalmetrics.RegalMergeReport)

		regoReport := report.Report{
			Aggregates:       make(map[string][]report.Aggregate, numFiles),
			IgnoreDirectives: make(map[string]map[string][]string, numFiles),
		}

		for i := range results {
			regoReport.Violations = append(regoReport.Violations, results[i].Violations...)
			regoReport.Notices = append(regoReport.Notices, results[i].Notices...)

			for k := range results[i].Aggregates {
				// Custom aggregate rules that have been invoked but not returned any data
				// will return an empty map to signal that they have been called, and that
				// the aggregate report for this rule should be invoked even when no data
				// was aggregated. This because the absence of data is exactly what some rules
				// will want to report on.
				for _, agg := range results[i].Aggregates[k] {
					if len(agg) == 0 {
						if _, ok := regoReport.Aggregates[k]; !ok {
							regoReport.Aggregates[k] = make([]report.Aggregate, 0)
						}
					} else {
						regoReport.Aggregates[k] = append(regoReport.Aggregates[k], agg)
					}
				}
			}

			for k := range results[i].IgnoreDirectives {
				regoReport.IgnoreDirectives[k] = results[i].IgnoreDirectives[k]
			}

			if l.profiling {
				regoReport.AddProfileEntries(results[i].AggregateProfile)
			}
		}

		return regoReport, nil
	}
}

func (l Linter) lintWithAggregateRules(
	ctx context.Context,
	aggregates map[string][]report.Aggregate,
	ignoreDirectives map[string]map[string][]string,
) (report.Report, error) {
	l.startTimer(regalmetrics.RegalLintRegoAggregate)
	defer l.stopTimer(regalmetrics.RegalLintRegoAggregate)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	regal := ast.ObjectTerm(
		ast.Item(ast.InternedTerm("operations"), ast.ArrayTerm(ast.InternedTerm("aggregate"))),
		ast.Item(ast.InternedTerm("file"), ast.ObjectTerm(
			ast.Item(ast.InternedTerm("name"), ast.InternedTerm("__aggregate_report__")),
			ast.Item(ast.InternedTerm("lines"), ast.InternedEmptyArray),
		)),
	)

	aggParsed, _ := transform.ToOPAInputValue(aggregates)
	dirParsed, _ := transform.ToOPAInputValue(ignoreDirectives)

	inputValue := ast.NewObject(
		ast.Item(ast.InternedTerm("aggregates_internal"), ast.NewTerm(aggParsed)),
		ast.Item(ast.InternedTerm("ignore_directives"), ast.NewTerm(dirParsed)),
		ast.Item(ast.InternedTerm("regal"), regal),
	)

	evalArgs := []rego.EvalOption{
		rego.EvalParsedInput(inputValue),
		rego.EvalInstrument(l.instrumentation),
	}

	if l.metrics != nil {
		evalArgs = append(evalArgs, rego.EvalMetrics(l.metrics))
	}

	var prof *profiler.Profiler
	if l.profiling {
		prof = profiler.New()
		evalArgs = append(evalArgs, rego.EvalQueryTracer(prof))
	}

	resultSet, err := l.preparedQuery.Eval(ctx, evalArgs...)
	if err != nil {
		return report.Report{}, fmt.Errorf("error encountered in query evaluation %w", err)
	}

	result, err := report.FromResultSet(resultSet, true)
	if err != nil {
		return report.Report{}, fmt.Errorf("failed to convert result set to report: %w", err)
	}

	for i := range result.Violations {
		result.Violations[i].IsAggregate = true
	}

	if l.profiling {
		profRep := prof.ReportTopNResults(10, []string{"total_time_ns"})

		result.AggregateProfile = make(map[string]report.ProfileEntry, len(profRep))

		for _, rs := range profRep {
			result.AggregateProfile[rs.Location.String()] = regalmetrics.FromExprStats(rs)
		}
	}

	return result, nil
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
