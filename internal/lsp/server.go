//nolint:nilnil
package lsp

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/ast/oracle"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/open-policy-agent/opa/v1/storage"
	outil "github.com/open-policy-agent/opa/v1/util"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/capabilities"
	"github.com/open-policy-agent/regal/internal/compile"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/lsp/bundles"
	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/completions"
	"github.com/open-policy-agent/regal/internal/lsp/completions/providers"
	lsconfig "github.com/open-policy-agent/regal/internal/lsp/config"
	"github.com/open-policy-agent/regal/internal/lsp/documentsymbol"
	"github.com/open-policy-agent/regal/internal/lsp/examples"
	"github.com/open-policy-agent/regal/internal/lsp/foldingrange"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/hover"
	"github.com/open-policy-agent/regal/internal/lsp/inlayhint"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/update"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/internal/web"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/fixer"
	"github.com/open-policy-agent/regal/pkg/fixer/fileprovider"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
	"github.com/open-policy-agent/regal/pkg/rules"
	"github.com/open-policy-agent/regal/pkg/version"
)

const (
	methodTdPublishDiagnostics = "textDocument/publishDiagnostics"
	methodWsApplyEdit          = "workspace/applyEdit"

	ruleNameOPAFmt                = "opa-fmt"
	ruleNameUseRegoV1             = "use-rego-v1"
	ruleNameUseAssignmentOperator = "use-assignment-operator"
)

var (
	noDocumentSymbols = make([]types.DocumentSymbol, 0)
	noCompletionItems = make([]types.CompletionItem, 0)
	noFoldingRanges   = make([]types.FoldingRange, 0)
	noDiagnostics     = make([]types.Diagnostic, 0)

	orc = oracle.New()

	regalEvalUseAsInputComment = regexp.MustCompile(`^\s*regal eval:\s*use-as-input`)
	validPathComponentPattern  = regexp.MustCompile(`^\w+[\w\-]*\w+$`)
)

type LanguageServerOptions struct {
	// Logger is the logger to use for the language server.
	Logger *log.Logger

	// WorkspaceDiagnosticsPoll, if set > 0 will cause a full workspace lint
	// to run on this interval. This is intended to be used where eventing
	// is not working, as expected. E.g. with a client that does not send
	// changes or when running in extremely slow environments like GHA with
	// the go race detector on. TODO, work out why this is required.
	WorkspaceDiagnosticsPoll time.Duration
}

type LanguageServer struct {
	log *log.Logger

	regoStore storage.Store
	conn      *jsonrpc2.Conn

	configWatcher *lsconfig.Watcher
	loadedConfig  *config.Config
	// this is also used to lock the updates to the cache of enabled rules
	loadedConfigLock                     sync.RWMutex
	loadedConfigEnabledNonAggregateRules []string
	loadedConfigEnabledAggregateRules    []string
	loadedConfigAllRegoVersions          *concurrent.Map[string, ast.RegoVersion]
	loadedBuiltins                       *concurrent.Map[string, map[string]*ast.Builtin]

	client types.Client

	cache       *cache.Cache
	bundleCache *bundles.Cache
	queryCache  *query.Cache

	completionsManager *completions.Manager
	regoRouter         *rego.RegoRouter

	commandRequest       chan types.ExecuteCommandParams
	lintWorkspaceJobs    chan lintWorkspaceJob
	lintFileJobs         chan lintFileJob
	builtinsPositionJobs chan lintFileJob
	templateFileJobs     chan lintFileJob

	// templatingFiles tracks files currently being templated to ensure
	// other updates are not processed while the file is being updated.
	templatingFiles *concurrent.Map[string, bool]

	webServer *web.Server

	workspaceRootURI         string
	workspaceDiagnosticsPoll time.Duration
}

// lintFileJob is sent to the lintFileJobs channel to trigger a
// diagnostic update for a file.
type lintFileJob struct {
	Reason string
	URI    string
}

// lintWorkspaceJob is sent to lintWorkspaceJobs when a full workspace
// diagnostic update is needed.
type lintWorkspaceJob struct {
	Reason string
	// OverwriteAggregates for a workspace is only run once at start up. All
	// later updates to aggregate state is made as files are changed.
	OverwriteAggregates bool
	AggregateReportOnly bool
}

func NewLanguageServer(ctx context.Context, opts *LanguageServerOptions) *LanguageServer {
	ls := NewLanguageServerMinimal(ctx, opts, nil)
	ls.configWatcher = lsconfig.NewWatcher(&lsconfig.WatcherOpts{Logger: ls.log})

	return ls
}

// NewLanguageServerMinimal starts a language server that doesn't assume a shared filesystem with the editor
// instance. It's used from pkg/lsp for Websocket connectivity from web editors (playground, build/ws).
func NewLanguageServerMinimal(ctx context.Context, opts *LanguageServerOptions, cfg *config.Config) *LanguageServer {
	c := cache.NewCache()
	qc := query.NewCache()
	store := NewRegalStore()

	ls := &LanguageServer{
		cache:                       c,
		queryCache:                  qc,
		loadedConfig:                cfg,
		regoStore:                   store,
		log:                         opts.Logger,
		lintFileJobs:                make(chan lintFileJob, 10),
		lintWorkspaceJobs:           make(chan lintWorkspaceJob, 10),
		builtinsPositionJobs:        make(chan lintFileJob, 10),
		commandRequest:              make(chan types.ExecuteCommandParams, 10),
		templateFileJobs:            make(chan lintFileJob, 10),
		templatingFiles:             concurrent.MapOf(make(map[string]bool)),
		completionsManager:          completions.NewDefaultManager(ctx, c, store, qc),
		webServer:                   web.NewServer(c, opts.Logger),
		loadedBuiltins:              concurrent.MapOf(make(map[string]map[string]*ast.Builtin)),
		workspaceDiagnosticsPoll:    opts.WorkspaceDiagnosticsPoll,
		loadedConfigAllRegoVersions: concurrent.MapOf(make(map[string]ast.RegoVersion)),
	}

	ls.regoRouter = rego.NewRegoRouter(ctx, store, qc, rego.Providers{
		ContextProvider:              ls.regalContext,
		IgnoredProvider:              ls.ignoreURI,
		ContentProvider:              ls.cache.GetFileContents,
		ParseErrorsProvider:          ls.cache.GetParseErrors,
		SuccessfulParseCountProvider: ls.cache.GetSuccessfulParseLineCount,
	})

	merged, _ := config.WithDefaultsFromBundle(rbundle.EmbeddedBundle(), cfg)

	ls.loadConfig(ctx, merged)

	return ls
}

func (l *LanguageServer) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	l.log.Debug("received request: %s", req.Method)

	// null params are allowed, but only for certain methods
	if req.Params == nil && req.Method != "shutdown" && req.Method != "exit" {
		return nil, handler.ErrInvalidParams
	}

	switch req.Method {
	case "initialize":
		return handler.WithContextAndParams(ctx, req, l.handleInitialize)
	case "initialized":
		return l.handleInitialized()
	case "textDocument/definition":
		return handler.WithParams(req, l.handleTextDocumentDefinition)
	case "textDocument/diagnostic":
		return l.handleTextDocumentDiagnostic()
	case "textDocument/didOpen":
		return handler.WithParams(req, l.handleTextDocumentDidOpen)
	case "textDocument/didClose":
		return handler.WithParams(req, l.handleTextDocumentDidClose)
	case "textDocument/didSave":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentDidSave)
	case "textDocument/documentSymbol":
		return handler.WithParams(req, l.handleTextDocumentDocumentSymbol)
	case "textDocument/didChange":
		return handler.WithParams(req, l.handleTextDocumentDidChange)
	case "textDocument/foldingRange":
		return handler.WithParams(req, l.handleTextDocumentFoldingRange)
	case "textDocument/formatting":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentFormatting)
	case "textDocument/hover":
		return handler.WithParams(req, l.handleTextDocumentHover)
	case "textDocument/inlayHint":
		return handler.WithParams(req, l.handleTextDocumentInlayHint)
	case "textDocument/completion":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentCompletion)
	case "workspace/didChangeWatchedFiles":
		return handler.WithParams(req, l.handleWorkspaceDidChangeWatchedFiles)
	case "workspace/diagnostic":
		return l.handleWorkspaceDiagnostic()
	case "workspace/didRenameFiles":
		return handler.WithContextAndParams(ctx, req, l.handleWorkspaceDidRenameFiles)
	case "workspace/didDeleteFiles":
		return handler.WithContextAndParams(ctx, req, l.handleWorkspaceDidDeleteFiles)
	case "workspace/didCreateFiles":
		return handler.WithParams(req, l.handleWorkspaceDidCreateFiles)
	case "workspace/executeCommand":
		return handler.WithParams(req, l.handleWorkspaceExecuteCommand)
	case "workspace/symbol":
		return l.handleWorkspaceSymbol()
	case "shutdown":
		// no-op as we wait for the exit signal before closing channel
		return struct{}{}, nil
	case "exit":
		// close the channel, cancel the context for all workers, and exit
		if err := l.conn.Close(); err != nil {
			return nil, fmt.Errorf("failed to close connection: %w", err)
		}

		return struct{}{}, nil
	case "$/setTrace":
		return handler.WithParams(req, func(params types.TraceParams) (any, error) {
			if level, err := log.TraceValueToLevel(params.Value); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			} else {
				l.log.SetLevel(level)
			}

			return struct{}{}, nil
		})
	case "$/cancelRequest":
		// TODO: this is a no-op, but is something that we should implement
		// if we want to support longer running, client-triggered operations
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#dollarRequests
		return struct{}{}, nil
	}

	// Handles:
	// - textDocument/codeAction
	// - textDocument/codeLens
	// - textDocument/documentLink
	// - textDocument/documentHighlight
	// - textDocument/signatureHelp
	//
	// returns jsonrpc2.Error with code jsonrpc2.CodeMethodNotFound if provided unknown method.
	return l.regoRouter.Handle(ctx, l.conn, req)
}

func (l *LanguageServer) SetConn(conn *jsonrpc2.Conn) {
	l.conn = conn
}

func (l *LanguageServer) StartDiagnosticsWorker(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-l.lintFileJobs:
				l.log.Debug("linting file %s (%s)", job.URI, job.Reason)

				// updateParse will not return an error when the parsing failed,
				// but only when it was impossible to parse the file.
				if _, err := updateParse(ctx, l.parseOpts(job.URI, l.builtinsForCurrentCapabilities())); err != nil {
					l.log.Message("failed to update module for %s: %s", job.URI, err)

					continue
				}

				// lint the file and send the diagnostics
				if err := updateFileDiagnostics(ctx, diagnosticsRunOpts{
					Cache:            l.cache,
					RegalConfig:      l.getLoadedConfig(),
					FileURI:          job.URI,
					WorkspaceRootURI: l.workspaceRootURI,
					// updateFileDiagnostics only ever updates the diagnostics
					// of non aggregate rules
					UpdateForRules:  l.getEnabledNonAggregateRules(),
					CustomRulesPath: l.getCustomRulesPath(),
				}); err != nil {
					l.log.Message("failed to update file diagnostics: %s", err)

					continue
				}

				if err := l.sendFileDiagnostics(ctx, job.URI); err != nil {
					l.log.Message("failed to send diagnostic: %s", err)

					continue
				}

				l.lintWorkspaceJobs <- lintWorkspaceJob{
					Reason: fmt.Sprintf("file %s %s", job.URI, job.Reason),
					// this run is expected to used the cached aggregate state
					// for other files.
					// The aggregate state for this file will still be updated.
					OverwriteAggregates: false,
					// when a file has changed, then there is no need to run
					// any other rules globally other than aggregate rules.
					AggregateReportOnly: true,
				}

				l.log.Debug("linting file %s done", job.URI)
			}
		}
	})

	wg.Add(1)

	workspaceLintRunBufferSize := 10
	workspaceLintRuns := make(chan lintWorkspaceJob, workspaceLintRunBufferSize)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case job := <-l.lintWorkspaceJobs:
				// AggregateReportOnly is set when updating aggregate
				// violations on character changes. Since these happen so
				// frequently, we stop adding to the channel if there already
				// jobs set to preserve performance
				if job.AggregateReportOnly && len(workspaceLintRuns) > workspaceLintRunBufferSize/2 {
					l.log.Debug("rate limiting aggregate reports")

					continue
				}

				workspaceLintRuns <- job
			}
		}
	}()

	if l.workspaceDiagnosticsPoll > 0 {
		wg.Add(1)

		ticker := time.NewTicker(l.workspaceDiagnosticsPoll)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					workspaceLintRuns <- lintWorkspaceJob{Reason: "poll ticker", OverwriteAggregates: true}
				}
			}
		}()
	}

	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-workspaceLintRuns:
				l.log.Debug("linting workspace: %#v", job)

				// if there are no parsed modules in the cache, then there is
				// no need to run the aggregate report. This can happen if the
				// server is very slow to start up.
				if len(l.cache.GetAllModules()) == 0 {
					continue
				}

				targetRules := l.getEnabledAggregateRules()
				if !job.AggregateReportOnly {
					targetRules = append(targetRules, l.getEnabledNonAggregateRules()...)
				}

				err := updateWorkspaceDiagnostics(ctx, diagnosticsRunOpts{
					Cache:            l.cache,
					RegalConfig:      l.getLoadedConfig(),
					WorkspaceRootURI: l.workspaceRootURI,
					// this is intended to only be set to true once at start up,
					// on following runs, cached aggregate data is used.
					OverwriteAggregates: job.OverwriteAggregates,
					AggregateReportOnly: job.AggregateReportOnly,
					UpdateForRules:      targetRules,
					CustomRulesPath:     l.getCustomRulesPath(),
				})
				if err != nil {
					l.log.Message("failed to update all diagnostics: %s", err)
				}

				for fileURI := range l.cache.GetAllFiles() {
					if err := l.sendFileDiagnostics(ctx, fileURI); err != nil {
						l.log.Message("failed to send diagnostic: %s", err)
					}
				}

				l.log.Debug("linting workspace done")
			}
		}
	})

	<-ctx.Done()
	wg.Wait()
}

func (l *LanguageServer) StartHoverWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-l.builtinsPositionJobs:
			if err := l.processHoverContentUpdate(ctx, job.URI); err != nil {
				l.log.Message(err.Error())
			}
		}
	}
}

func (l *LanguageServer) StartConfigWorker(ctx context.Context) {
	if err := l.configWatcher.Start(ctx); err != nil {
		l.log.Message("failed to start config watcher: %s", err)

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case path := <-l.configWatcher.Reload:
			userConfig, err := config.FromPath(path)
			if err != nil && !errors.Is(err, io.EOF) {
				l.log.Message("failed to reload config: %s", err)

				continue
			}

			mergedConfig, err := config.WithDefaultsFromBundle(rbundle.LoadedBundle(), &userConfig)
			if err != nil {
				l.log.Message("failed to load config: %s", err)

				continue
			}

			l.loadConfig(ctx, mergedConfig)

			//nolint:contextcheck
			go func() {
				if l.getLoadedConfig().Features.Remote.CheckVersion &&
					os.Getenv(update.CheckVersionDisableEnvVar) != "" {
					update.CheckAndWarn(update.Options{
						CurrentVersion: version.Version,
						CurrentTime:    time.Now().UTC(),
						Debug:          false,
						StateDir:       config.GlobalConfigDir(true),
					}, os.Stderr)
				}
			}()

			l.lintWorkspaceJobs <- lintWorkspaceJob{Reason: "config file changed"}
		case <-l.configWatcher.Drop:
			l.loadedConfigLock.Lock()

			defaultConfig, _ := config.WithDefaultsFromBundle(rbundle.LoadedBundle(), nil)
			l.loadedConfig = &defaultConfig
			l.loadedConfigLock.Unlock()

			l.lintWorkspaceJobs <- lintWorkspaceJob{Reason: "config file dropped"}
		}
	}
}

func (l *LanguageServer) StartCommandWorker(ctx context.Context) { //nolint:maintidx
	// note, in this function conn.Call is used as the workspace/applyEdit message is a request, not a notification
	// as per the spec. In order to be 'routed' to the correct handler on the client it must have an ID
	// receive responses too.
	// Note however that the responses from the client are not needed by the server.
	for {
		select {
		case <-ctx.Done():
			return
		case params := <-l.commandRequest:
			var (
				editParams *types.ApplyWorkspaceEditParams
				args       types.CommandArgs
				fixed      bool
				err        error
			)

			if len(params.Arguments) != 1 {
				l.log.Message("expected one argument, got %d", len(params.Arguments))

				continue
			}

			jsonData, ok := params.Arguments[0].(string)
			if !ok {
				l.log.Message("expected argument to be a json.RawMessage, got %T", params.Arguments[0])

				continue
			}

			if err = encoding.JSON().Unmarshal([]byte(jsonData), &args); err != nil {
				l.log.Message("failed to unmarshal command arguments: %s", err)

				continue
			}

			switch params.Command {
			case "regal.fix.opa-fmt":
				fixed, editParams, err = l.fixEditParams(
					"Format using opa fmt",
					&fixes.Fmt{OPAFmtOpts: format.Opts{}},
					args,
				)
			case "regal.fix.use-rego-v1":
				fixed, editParams, err = l.fixEditParams(
					"Format for Rego v1 using opa-fmt",
					&fixes.Fmt{OPAFmtOpts: format.Opts{RegoVersion: ast.RegoV0CompatV1}},
					args,
				)
			case "regal.fix.use-assignment-operator":
				fixed, editParams, err = l.fixEditParams(
					"Replace = with := in assignment",
					&fixes.UseAssignmentOperator{},
					args,
				)
			case "regal.fix.no-whitespace-comment":
				fixed, editParams, err = l.fixEditParams(
					"Format comment to have leading whitespace",
					&fixes.NoWhitespaceComment{},
					args,
				)
			case "regal.fix.non-raw-regex-pattern":
				fixed, editParams, err = l.fixEditParams(
					"Replace \" with ` in regex pattern",
					&fixes.NonRawRegexPattern{},
					args,
				)
			case "regal.fix.directory-package-mismatch":
				params, err := l.fixRenameParams("Rename file to match package path", args.Target)
				if err != nil {
					l.log.Message("failed to fix directory package mismatch: %s", err)

					break
				}

				if err := l.conn.Call(ctx, methodWsApplyEdit, params, nil); err != nil {
					l.log.Message("failed %s notify: %v", methodWsApplyEdit, err.Error())
				}

				// handle this ourselves as it's a rename and not a content edit
				fixed = false
			case "regal.debug":
				if args.Target == "" {
					l.log.Message("expected command target to be set, got %q", args.Target)

					break
				}

				if args.Query == "" {
					l.log.Message("expected command query path to be set, got %q", args.Query)

					break
				}

				inputPath := rio.FindInputPath(l.toPath(args.Target), l.workspacePath())

				responseParams := map[string]any{
					"type":        "opa-debug",
					"name":        args.Query,
					"request":     "launch",
					"command":     "eval",
					"query":       args.Query,
					"enablePrint": true,
					"stopOnEntry": true,
					"inputPath":   inputPath,
				}

				responseResult := map[string]any{}

				if err = l.conn.Call(ctx, "regal/startDebugging", responseParams, &responseResult); err != nil {
					l.log.Message("regal/startDebugging failed: %v", err.Error())
				}
			case "regal.eval":
				fileURI := args.Target
				if fileURI == "" {
					l.log.Message("expected command target to be set, got %q", fileURI)

					break
				}

				if args.Query == "" {
					l.log.Message("expected command query to be set, got %q", args.Query)

					break
				}

				contents, module, ok := l.cache.GetContentAndModule(fileURI)
				if !ok {
					l.log.Message("failed to get content or module for file %q", fileURI)

					break
				}

				var pq *query.Prepared

				pq, err = l.queryCache.GetOrSet(ctx, l.regoStore, query.RuleHeadLocations)
				if err != nil {
					l.log.Message("failed to prepare query %s", query.RuleHeadLocations, err)

					break
				}

				var allRuleHeadLocations rego.RuleHeads

				allRuleHeadLocations, err = rego.AllRuleHeadLocations(
					ctx, pq, filepath.Base(l.toPath(fileURI)), contents, module,
				)
				if err != nil {
					l.log.Message("failed to get rule head locations: %s", err)

					break
				}

				// if there are none, then it's a package evaluation
				ruleHeadLocations := allRuleHeadLocations[args.Query]

				var inputMap map[string]any

				// When the first comment in the file is `regal eval: use-as-input`, the AST of that module is
				// used as the input rather than the contents of input.json/yaml. This is a development feature for
				// working on rules (built-in or custom), allowing querying the AST of the module directly.
				if len(module.Comments) > 0 && regalEvalUseAsInputComment.Match(module.Comments[0].Text) {
					inputMap, err = rparse.PrepareAST(l.toRelativePath(fileURI), contents, module)
					if err != nil {
						l.log.Message("failed to prepare module: %s", err)

						break
					}
				} else {
					// Normal mode — try to find the input.json/yaml file in the workspace and use as input
					// NOTE that we don't break on missing input, as some rules don't depend on that, and should
					// still be evaluable. We may consider returning some notice to the user though.
					_, inputMap = rio.FindInput(l.toPath(fileURI), l.workspacePath())
				}

				var result EvalResult

				if result, err = l.EvalInWorkspace(ctx, args.Query, inputMap); err != nil {
					fmt.Fprintf(os.Stderr, "failed to evaluate workspace path: %v\n", err)

					break
				}

				target := "package"
				if len(ruleHeadLocations) > 0 {
					target = strings.TrimPrefix(args.Query, module.Package.Path.String()+".")
				}

				if l.client.InitOptions.EvalCodelensDisplayInline != nil &&
					*l.client.InitOptions.EvalCodelensDisplayInline {
					responseParams := map[string]any{
						"result": result,
						"line":   args.Row,
						"target": target,
						// only used when the target is 'package'
						"package": strings.TrimPrefix(module.Package.Path.String(), "data."),
						// only used when the target is a rule
						"rule_head_locations": ruleHeadLocations,
					}

					responseResult := map[string]any{}

					if err = l.conn.Call(ctx, "regal/showEvalResult", responseParams, &responseResult); err != nil {
						l.log.Message("regal/showEvalResult failed: %v", err.Error())
					}
				} else {
					output := filepath.Join(l.workspacePath(), "output.json")

					var f *os.File

					f, err = os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
					if err == nil {
						var jsonVal []byte

						value := result.Value
						if result.IsUndefined {
							// Display undefined as an empty object
							value = emptyStringAnyMap
						}

						if jsonVal, err = encoding.JSON().MarshalIndent(value, "", "  "); err == nil {
							_, err = f.Write(jsonVal)
						}

						f.Close()
					}
				}
			}

			if err != nil {
				l.log.Message("command failed: %s", err)

				if err := l.conn.Notify(ctx, "window/showMessage", types.ShowMessageParams{
					Type:    1, // error
					Message: err.Error(),
				}); err != nil {
					l.log.Message("failed to notify client of command error: %s", err)
				}

				break
			}

			if fixed {
				if err = l.conn.Call(ctx, methodWsApplyEdit, editParams, nil); err != nil {
					l.log.Message("failed %s notify: %v", methodWsApplyEdit, err.Error())
				}
			}
		}
	}
}

// StartWorkspaceStateWorker will poll for changes to the workspaces state that
// are not sent from the client. For example, when a file a is removed from the
// workspace after changing branch.
func (l *LanguageServer) StartWorkspaceStateWorker(ctx context.Context) {
	timer := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// first clear files that are missing from the workspaceDir
			for fileURI := range l.cache.GetAllFiles() {
				if _, err := os.Stat(l.toPath(fileURI)); os.IsNotExist(err) {
					// clear the cache first,
					l.cache.Delete(fileURI)

					// then send the diagnostics message based on the cleared cache
					if err := l.sendFileDiagnostics(ctx, fileURI); err != nil {
						l.log.Message("failed to send diagnostic: %s", err)
					}
				}
			}

			// for this next operation, the workspace root must be set as it's
			// used to scan for new files.
			if l.workspaceRootURI == "" {
				continue
			}

			// next, check if there are any new files that are not ignored and
			// need to be loaded. We get new only so that files being worked
			// on are not loaded from disk during editing.
			newURIs, failed, err := l.loadWorkspaceContents(ctx, true)
			for _, f := range failed {
				l.log.Message("failed to load file %s: %s", f.URI, f.Error)
			}

			if err != nil {
				l.log.Message("failed to refresh workspace contents: %s", err)

				continue
			}

			for _, cnURI := range newURIs {
				l.lintFileJobs <- lintFileJob{URI: cnURI, Reason: "internal/workspaceStateWorker/changedOrNewFile"}
			}
		}
	}
}

// StartWebServer starts the web server that serves explorer.
func (l *LanguageServer) StartWebServer(ctx context.Context) {
	l.webServer.Start(ctx)
}

// StartTemplateWorker runs the process of the server that templates newly
// created Rego files.
func (l *LanguageServer) StartTemplateWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-l.templateFileJobs:
			l.processTemplateJob(ctx, job)
		}
	}
}

func (l *LanguageServer) getLoadedConfig() *config.Config {
	l.loadedConfigLock.RLock()
	defer l.loadedConfigLock.RUnlock()

	return l.loadedConfig
}

func (l *LanguageServer) getEnabledNonAggregateRules() []string {
	l.loadedConfigLock.RLock()
	defer l.loadedConfigLock.RUnlock()

	return l.loadedConfigEnabledNonAggregateRules
}

func (l *LanguageServer) getEnabledAggregateRules() []string {
	l.loadedConfigLock.RLock()
	defer l.loadedConfigLock.RUnlock()

	return l.loadedConfigEnabledAggregateRules
}

func (l *LanguageServer) getCustomRulesPath() string {
	if l.workspaceRootURI != "" {
		if customRulesPath := filepath.Join(l.workspacePath(), ".regal", "rules"); rio.IsDir(customRulesPath) {
			return customRulesPath
		}
	}

	return ""
}

func (l *LanguageServer) loadConfig(ctx context.Context, conf config.Config) {
	l.loadedConfigLock.Lock()
	l.loadedConfig = &conf
	l.loadedConfigLock.Unlock()

	if err := PutConfig(ctx, l.regoStore, &conf); err != nil {
		l.log.Message("failed to update config in storage: %v", err)
	}

	// Rego versions may have changed, so reload them.
	allRegoVersions, err := config.AllRegoVersions(l.workspacePath(), &conf)
	if err != nil {
		l.log.Debug("failed to reload rego versions: %s", err)
	} else {
		l.loadedConfigAllRegoVersions.Clear()

		for k, v := range allRegoVersions {
			l.loadedConfigAllRegoVersions.Set(k, v)
		}
	}

	// Enabled rules might have changed with the new config, so reload.
	if err = l.loadEnabledRulesFromConfig(ctx, conf); err != nil {
		l.log.Message("failed to cache enabled rules: %s", err)
	}

	// Capabilities URL may have changed, so we should reload it.
	capsURL := cmp.Or(conf.CapabilitiesURL, capabilities.DefaultURL)

	caps, err := capabilities.Lookup(ctx, capsURL)
	if err != nil {
		l.log.Message("failed to load capabilities for URL %q: %s", capsURL, err)

		return
	}

	bis := rego.BuiltinsForCapabilities(caps)

	l.loadedBuiltins.Set(capsURL, bis)

	if err := PutBuiltins(ctx, l.regoStore, bis); err != nil {
		l.log.Message("failed to update builtins in storage: %v", err)
	}

	// the config may now ignore files that existed in the cache before,
	// in which case we need to remove them to stop their contents being
	// used in other ls functions.
	for k := range l.cache.GetAllFiles() {
		if !l.ignoreURI(k) {
			continue
		}

		// move the contents to the ignored part of the cache
		contents, ok := l.cache.GetFileContents(k)
		if ok {
			l.cache.Delete(k)
			l.cache.SetIgnoredFileContents(k, contents)
		}

		if err := RemoveFileMod(ctx, l.regoStore, k); err != nil {
			l.log.Message("failed to remove mod from store: %s", err)
		}
	}

	// when a file is 'unignored', we move its contents to the
	// standard file list if missing
	for k, v := range l.cache.GetAllIgnoredFiles() {
		if l.ignoreURI(k) {
			continue
		}

		// ignored contents will only be used when there is no existing content
		_, ok := l.cache.GetFileContents(k)
		if !ok {
			l.cache.SetFileContents(k, v)

			// updating the parse here will enable things like go-to definition
			// to start working right away without the need for a file content
			// update to run updateParse.
			if _, err = updateParse(ctx, l.parseOpts(k, bis)); err != nil {
				l.log.Message("failed to update parse for previously ignored file %q: %s", k, err)
			}
		}

		l.cache.ClearIgnoredFileContents(k)
	}
}

// loadEnabledRulesFromConfig is used to cache the enabled rules for the current
// config. These take some time to compute and only change when config changes,
// so we can store them on the server to speed up diagnostic runs.
func (l *LanguageServer) loadEnabledRulesFromConfig(ctx context.Context, cfg config.Config) error {
	lint := linter.NewLinter().WithUserConfig(cfg)

	if customRulesPath := l.getCustomRulesPath(); customRulesPath != "" {
		lint = lint.WithCustomRules([]string{customRulesPath})
	}

	enabledRules, err := lint.DetermineEnabledRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine enabled rules: %w", err)
	}

	enabledAggregateRules, err := lint.DetermineEnabledAggregateRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine enabled aggregate rules: %w", err)
	}

	l.loadedConfigLock.Lock()
	defer l.loadedConfigLock.Unlock()

	l.loadedConfigEnabledNonAggregateRules = []string{}

	for _, r := range enabledRules {
		if !slices.Contains(enabledAggregateRules, r) {
			l.loadedConfigEnabledNonAggregateRules = append(l.loadedConfigEnabledNonAggregateRules, r)
		}
	}

	l.loadedConfigEnabledAggregateRules = enabledAggregateRules

	return nil
}

// processTemplateJob handles the templating of a newly created Rego file.
func (l *LanguageServer) processTemplateJob(ctx context.Context, job lintFileJob) {
	l.log.Debug("template worker received job: %s (reason: %s)", job.URI, job.Reason)

	// mark file as being templated to prevent race conditions
	l.templatingFiles.Set(job.URI, true)
	defer l.templatingFiles.Delete(job.URI)

	// disable the templating feature for files in the workspace root.
	if filepath.Dir(l.toPath(job.URI)) == l.workspacePath() {
		return
	}

	// determine the new contents for the file, if permitted
	newContents, err := l.templateContentsForFile(job.URI)
	if err != nil {
		l.log.Message("failed to template new file: %s", err)

		return
	}

	// set the contents of the new file in the cache immediately as
	// these must be update to date in order for fixRenameParams
	// to work
	l.cache.SetFileContents(job.URI, newContents)

	edits := []any{types.TextDocumentEdit{
		TextDocument: types.OptionalVersionedTextDocumentIdentifier{URI: job.URI},
		Edits:        ComputeEdits("", newContents),
	}}

	// determine if a rename is needed based on the new file package.
	// edits will be empty if no file rename is needed.
	additionalRenameEdits, err := l.fixRenameParams("Template new Rego file", job.URI)
	if err != nil {
		l.log.Message("failed to get rename params: %s", err)

		return
	}

	// combine content edits with any additional rename edits
	edits = append(edits, additionalRenameEdits.Edit.DocumentChanges...)

	// send the edit back to the editor so it appears in the open buffer.
	if err = l.conn.Call(ctx, methodWsApplyEdit, types.ApplyWorkspaceAnyEditParams{
		Label: "Template new Rego file",
		Edit:  types.WorkspaceAnyEdit{DocumentChanges: edits},
	}, nil); err != nil {
		l.log.Message("failed %s notify: %v", methodWsApplyEdit, err.Error())
	}

	// finally, trigger a diagnostics run for the new contents
	l.lintFileJobs <- lintFileJob{Reason: "internal/templateNewFile", URI: job.URI}
}

func (l *LanguageServer) templateContentsForFile(fileURI string) (string, error) {
	path := l.toPath(fileURI)

	// this function should not be called with files in the root, but if it is,
	// then it is an error to prevent unwanted behavior.
	if filepath.Dir(path) == l.workspacePath() {
		return "", errors.New("this function does not template files in the workspace root")
	}

	content, ok := l.cache.GetFileContents(fileURI)
	if !ok {
		return "", fmt.Errorf("failed to get file contents for URI %q", fileURI)
	}

	if content != "" {
		return "", errors.New("file already has contents, templating not allowed")
	}

	if diskContent, err := os.ReadFile(path); err == nil && len(diskContent) > 0 {
		// then we found the file on disk
		return "", errors.New("file on disk already has contents, templating not allowed")
	}

	roots, err := config.GetPotentialRoots(path)
	if err != nil {
		return "", fmt.Errorf("failed to get potential roots during templating of new file: %w", err)
	}

	dir := filepath.Dir(path)

	// handle the case where the root is unknown by providing the server's root
	// dir as a defacto root. This allows templating of files when there is no
	// known root, but the package could be determined based on the file path
	// relative to the server's workspace root
	if len(roots) == 1 && roots[0] == dir {
		roots = []string{l.workspacePath()}
	} else {
		roots = append(roots, l.workspacePath())
	}

	longestPrefixRoot := ""

	for _, root := range roots {
		if strings.HasPrefix(dir, root) && len(root) > len(longestPrefixRoot) {
			longestPrefixRoot = root
		}
	}

	if longestPrefixRoot == "" {
		return "", fmt.Errorf("failed to find longest prefix root for templating of new file: %s", path)
	}

	parts := slices.Compact(strings.Split(strings.TrimPrefix(dir, longestPrefixRoot), string(os.PathSeparator)))

	var pkg string

	for _, part := range parts {
		if part == "" {
			continue
		}

		if !validPathComponentPattern.MatchString(part) {
			return "", fmt.Errorf("failed to template new file as package path contained invalid part: %s", part)
		}

		switch {
		case strings.Contains(part, "-"):
			pkg += fmt.Sprintf(`["%s"]`, part)
		case pkg == "":
			pkg += part
		default:
			pkg += "." + part
		}
	}

	// if we are in the root, then we can use main as a default
	pkg = cmp.Or(pkg, "main")

	if strings.HasSuffix(fileURI, "_test.rego") {
		pkg += "_test"
	}

	if l.regoVersionForURI(fileURI) == ast.RegoV0 {
		return fmt.Sprintf("package %s\n\nimport rego.v1\n", pkg), nil
	}

	return fmt.Sprintf("package %s\n\n", pkg), nil
}

func (l *LanguageServer) fixEditParams(
	label string,
	fix fixes.Fix,
	args types.CommandArgs,
) (bool, *types.ApplyWorkspaceEditParams, error) {
	oldContent, ok := l.cache.GetFileContents(args.Target)
	if !ok {
		return false, nil, fmt.Errorf("could not get file contents for uri %q", args.Target)
	}

	rto := &fixes.RuntimeOptions{BaseDir: l.workspacePath()}
	if args.Diagnostic != nil {
		rto.Locations = []report.Location{{
			Row:    util.SafeUintToInt(args.Diagnostic.Range.Start.Line + 1),
			Column: util.SafeUintToInt(args.Diagnostic.Range.Start.Character + 1),
			End: &report.Position{
				Row:    util.SafeUintToInt(args.Diagnostic.Range.End.Line + 1),
				Column: util.SafeUintToInt(args.Diagnostic.Range.End.Character + 1),
			},
		}}
	}

	res, err := fix.Fix(&fixes.FixCandidate{Filename: filepath.Base(l.toPath(args.Target)), Contents: oldContent}, rto)
	if err != nil {
		return false, nil, fmt.Errorf("failed to fix: %w", err)
	}

	if len(res) == 0 {
		return false, &types.ApplyWorkspaceEditParams{}, nil
	}

	var edits []types.TextEdit

	if l.client.Identifier == clients.IdentifierIntelliJ {
		// IntelliJ clients need a single edit that replaces the entire file
		lines := strings.Split(oldContent, "\n")
		endLine := len(lines) - 1
		endChar := 0

		if endLine >= 0 {
			endChar = len(lines[endLine])
		}

		edits = []types.TextEdit{{Range: types.RangeBetween(0, 0, endLine, endChar), NewText: res[0].Contents}}
	} else {
		// Other clients use the standard diff-based edits
		edits = ComputeEdits(oldContent, res[0].Contents)
	}

	editParams := &types.ApplyWorkspaceEditParams{
		Label: label,
		Edit: types.WorkspaceEdit{DocumentChanges: []types.TextDocumentEdit{{
			TextDocument: types.OptionalVersionedTextDocumentIdentifier{URI: args.Target},
			Edits:        edits,
		}}},
	}

	return true, editParams, nil
}

func (l *LanguageServer) fixRenameParams(label, fileURI string) (types.ApplyWorkspaceAnyEditParams, error) {
	roots, err := config.GetPotentialRoots(l.workspacePath())
	if err != nil {
		return types.ApplyWorkspaceAnyEditParams{}, fmt.Errorf("failed to get potential roots: %w", err)
	}

	fix := &fixes.DirectoryPackageMismatch{}

	// the default for the LSP is to rename on conflict
	f := fixer.NewFixer().RegisterRoots(roots...).RegisterFixes(fix).SetOnConflictOperation(fixer.OnConflictRename)

	violations := []report.Violation{{Title: fix.Name(), Location: report.Location{File: l.toPath(fileURI)}}}
	cfprovider := fileprovider.NewCacheFileProvider(l.cache, l.client.Identifier)

	fixReport, err := f.FixViolations(violations, cfprovider, l.getLoadedConfig())
	if err != nil {
		return types.ApplyWorkspaceAnyEditParams{}, fmt.Errorf("failed to fix violations: %w", err)
	}

	ff := fixReport.FixedFiles()
	if len(ff) == 0 {
		return types.ApplyWorkspaceAnyEditParams{Label: label, Edit: types.WorkspaceAnyEdit{}}, nil
	}

	// find the new file and the old location
	var fixedFile, oldFile string

	var found bool

	for _, f := range ff {
		if oldFile, found = fixReport.OldPathForFile(f); found {
			fixedFile = f

			break
		}
	}

	if !found {
		params := types.ApplyWorkspaceAnyEditParams{Label: label, Edit: types.WorkspaceAnyEdit{}}

		return params, errors.New("failed to find fixed file's old location")
	}

	oldURI := l.fromPath(oldFile)
	newURI := l.fromPath(fixedFile)

	// is the newURI still in the root?
	if !strings.HasPrefix(newURI, l.workspaceRootURI) {
		return types.ApplyWorkspaceAnyEditParams{
			Label: label,
			Edit:  types.WorkspaceAnyEdit{},
		}, errors.New("cannot move file out of workspace root, consider using a workspace config or manually setting roots")
	}

	// are there old dirs?
	dirs, err := rio.DirCleanUpPaths(l.toPath(oldURI), []string{
		l.workspacePath(), // stop at the root
		l.toPath(newURI),  // also preserve any dirs needed for the new file
	})
	if err != nil {
		return types.ApplyWorkspaceAnyEditParams{}, fmt.Errorf("failed to determine empty directories post rename: %w", err)
	}

	renopts := &types.RenameFileOptions{Overwrite: false, IgnoreIfExists: false}
	changes := append(make([]any, 0, len(dirs)+1),
		types.RenameFile{Kind: "rename", OldURI: oldURI, NewURI: newURI, Options: renopts},
	)

	delopts := &types.DeleteFileOptions{Recursive: true, IgnoreIfNotExists: true}
	for _, dir := range dirs {
		changes = append(changes, types.DeleteFile{Kind: "delete", URI: l.fromPath(dir), Options: delopts})
	}

	l.cache.Delete(oldURI)

	return types.ApplyWorkspaceAnyEditParams{Label: label, Edit: types.WorkspaceAnyEdit{DocumentChanges: changes}}, nil
}

// processHoverContentUpdate updates information about built in, and keyword
// positions in the cache for use when handling hover requests.
func (l *LanguageServer) processHoverContentUpdate(ctx context.Context, fileURI string) error {
	if l.ignoreURI(fileURI) {
		return nil
	}

	if _, ok := l.cache.GetFileContents(fileURI); !ok {
		// If the file is not in the cache, exit early or else
		// we might accidentally put it in the cache after it's been
		// deleted: https://github.com/open-policy-agent/regal/issues/679
		return nil
	}

	bis := l.builtinsForCurrentCapabilities()

	if success, err := updateParse(ctx, l.parseOpts(fileURI, bis)); err != nil {
		return fmt.Errorf("failed to update parse: %w", err)
	} else if !success {
		return nil
	}

	if err := hover.UpdateBuiltinPositions(l.cache, fileURI, bis); err != nil {
		return fmt.Errorf("failed to update builtin positions: %w", err)
	}

	pq, err := l.queryCache.GetOrSet(ctx, l.regoStore, query.Keywords)
	if err != nil {
		return fmt.Errorf("failed to prepare query %s, %w", query.Keywords, err)
	}

	if err := hover.UpdateKeywordLocations(ctx, pq, l.cache, fileURI); err != nil {
		return fmt.Errorf("failed to update keyword locations: %w", err)
	}

	return nil
}

func (l *LanguageServer) handleTextDocumentHover(params types.TextDocumentHoverParams) (any, error) {
	if l.ignoreURI(params.TextDocument.URI) {
		return nil, nil
	}

	// The Zed editor doesn't show CodeDescription.Href in diagnostic messages.
	// Instead, we hijack the hover request to show the documentation links
	// when there are violations present.
	violations, ok := l.cache.GetFileDiagnostics(params.TextDocument.URI)
	if l.client.Identifier == clients.IdentifierZed && ok && len(violations) > 0 {
		var docSnippets []string

		var sharedRange types.Range

		for _, v := range violations {
			if v.Range.Start.Line == params.Position.Line &&
				v.Range.Start.Character <= params.Position.Character &&
				v.Range.End.Character >= params.Position.Character {
				// this is an approximation, if there are multiple violations on the same line
				// where hover loc is in their range, then they all just share a range as a
				// single range is needed in the hover response.
				source := ""
				if v.Source != nil {
					source = *v.Source
				}

				sharedRange = v.Range
				docSnippets = append(docSnippets, fmt.Sprintf("[%s/%s](%s)", source, v.Code, v.CodeDescription.Href))
			}
		}

		hov := types.Hover{Range: sharedRange}
		if len(docSnippets) > 1 {
			hov.Contents = *types.Markdown("Documentation links:\n\n* " + strings.Join(docSnippets, "\n* "))
		} else if len(docSnippets) == 1 {
			hov.Contents = *types.Markdown("Documentation: " + docSnippets[0])
		}

		return hov, nil
	}

	builtinsOnLine, ok := l.cache.GetBuiltinPositions(params.TextDocument.URI)
	// when no builtins are found, we can't return a useful hover response.
	// log the error, but return an empty struct to avoid an error being shown in the client.
	if !ok {
		l.log.Message("could not get builtins for uri %q", params.TextDocument.URI)

		// return "null" as per the spec
		return nil, nil
	}

	for _, bp := range builtinsOnLine[params.Position.Line+1] {
		if params.Position.Character >= bp.Start-1 && params.Position.Character <= bp.End-1 {
			return types.Hover{
				Contents: *types.Markdown(hover.CreateHoverContent(bp.Builtin)),
				Range:    types.RangeBetween(bp.Line-1, bp.Start-1, bp.Line-1, bp.End-1),
			}, nil
		}
	}

	keywordsOnLine, ok := l.cache.GetKeywordLocations(params.TextDocument.URI)
	if !ok {
		// when no keywords are found, we can't return a useful hover response.
		// return "null" as per the spec
		return nil, nil
	}

	for _, kp := range keywordsOnLine[params.Position.Line+1] {
		if params.Position.Character >= kp.Start-1 && params.Position.Character <= kp.End-1 {
			link, ok := examples.GetKeywordLink(kp.Name)
			if !ok {
				continue
			}

			return types.Hover{
				Contents: *types.Markdown(fmt.Sprintf(
					"### %s\n\n[View examples](%s) for the '%s' keyword.", kp.Name, link, kp.Name)),
				Range: types.RangeBetween(kp.Line-1, kp.Start-1, kp.Line-1, kp.End-1),
			}, nil
		}
	}

	// return "null" as per the spec
	return nil, nil
}

func (l *LanguageServer) handleWorkspaceExecuteCommand(params types.ExecuteCommandParams) (any, error) {
	// this must not block, so we send the request to the worker on a buffered channel.
	// the response to the workspace/executeCommand request must be sent before the command is executed
	// so that the client can complete the request and be ready to receive the follow-on request for
	// workspace/applyEdit.
	l.commandRequest <- params

	// however, the contents of the response is not important
	return struct{}{}, nil
}

func (l *LanguageServer) handleTextDocumentInlayHint(params types.InlayHintParams) (any, error) {
	if l.ignoreURI(params.TextDocument.URI) {
		return []types.InlayHint{}, nil
	}

	bis := l.builtinsForCurrentCapabilities()

	// when a file cannot be parsed, we do a best effort attempt to provide inlay hints
	// by finding the location of the first parse error and attempting to parse up to that point
	if parseErrors, ok := l.cache.GetParseErrors(params.TextDocument.URI); ok && len(parseErrors) > 0 {
		contents, ok := l.cache.GetFileContents(params.TextDocument.URI)
		if !ok {
			// if there is no content, we can't even do a partial parse
			return []types.InlayHint{}, nil
		}

		return inlayhint.Partial(parseErrors, contents, params.TextDocument.URI, bis), nil
	}

	if contents, ok := l.cache.GetFileContents(params.TextDocument.URI); ok && contents == "" {
		return []types.InlayHint{}, nil
	}

	module, ok := l.cache.GetModule(params.TextDocument.URI)
	if !ok {
		l.log.Message("failed to get inlay hint: no parsed module for uri %q", params.TextDocument.URI)

		return []types.InlayHint{}, nil
	}

	return inlayhint.FromModule(module, bis), nil
}

func (l *LanguageServer) handleTextDocumentCompletion(ctx context.Context, params types.CompletionParams) (any, error) {
	// when config ignores a file, then we return an empty completion list as a no-op.
	if l.ignoreURI(params.TextDocument.URI) {
		return types.CompletionList{IsIncomplete: false, Items: []types.CompletionItem{}}, nil
	}

	// items is allocated here so that the return value is always a non-nil CompletionList
	items, err := l.completionsManager.Run(ctx, params, &providers.Options{
		Client:      l.client,
		RootURI:     l.workspaceRootURI,
		Builtins:    l.builtinsForCurrentCapabilities(),
		RegoVersion: l.regoVersionForURI(params.TextDocument.URI),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find completions: %w", err)
	}

	if items == nil {
		// make sure the items is always [] instead of null as is required by the spec
		items = noCompletionItems
	}

	return types.CompletionList{IsIncomplete: items != nil, Items: items}, nil
}

// Note: currently ignoring params.Query, as the client seems to do a good
// job of filtering anyway, and that would merely be an optimization here.
// But perhaps a good one to do at some point, and I'm not sure all clients
// do this filtering.
func (l *LanguageServer) handleWorkspaceSymbol() (any, error) {
	contents := l.cache.GetAllFiles()
	symbols := make([]types.WorkspaceSymbol, 0, len(contents)*10)
	bis := l.builtinsForCurrentCapabilities()

	for moduleURL, module := range l.cache.GetAllModules() {
		wrkSyms := make([]types.WorkspaceSymbol, 0)

		documentsymbol.ToWorkspaceSymbols(documentsymbol.All(contents[moduleURL], module, bis), moduleURL, &wrkSyms)

		symbols = append(symbols, wrkSyms...)
	}

	return symbols, nil
}

func (l *LanguageServer) handleTextDocumentDefinition(params types.DefinitionParams) (any, error) {
	if l.ignoreURI(params.TextDocument.URI) {
		return nil, nil
	}

	contents, ok := l.cache.GetFileContents(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("failed to get file contents for uri %q", params.TextDocument.URI)
	}

	// modules are loaded from the cache and keyed by their URI.
	modules, err := l.getFilteredModules()
	if err != nil {
		return nil, fmt.Errorf("failed to filter ignored paths: %w", err)
	}

	query := oracle.DefinitionQuery{
		// The value of Filename is used if the defn in the current buffer.
		Filename: params.TextDocument.URI,
		Pos:      positionToOffset(contents, params.Position),
		Modules:  modules,
		Buffer:   outil.StringToByteSlice(contents),
	}

	definition, err := orc.WithCompiler(compile.NewCompilerWithRegalBuiltins()).FindDefinition(query)
	if err != nil {
		if !util.IsAnyError(err, oracle.ErrNoDefinitionFound, oracle.ErrNoMatchFound) {
			l.log.Message("failed to find definition: %s", err)
		}

		// else fail silently — the user could have clicked anywhere. return "null" as per the spec
		return nil, nil
	}

	res := definition.Result

	return types.Location{
		// res.File will be relative to the workspace root. The response here needs
		// a URI for the client to be able to navigate correctly.
		URI:   util.EnsureSuffix(l.workspaceRootURI, "/") + res.File,
		Range: types.RangeBetween(res.Row-1, res.Col-1, res.Row-1, res.Col-1),
	}, nil
}

func (l *LanguageServer) handleTextDocumentDidOpen(params types.DidOpenTextDocumentParams) (any, error) {
	// if the opened file is ignored, we only store the contents for file level operations like formatting
	if l.ignoreURI(params.TextDocument.URI) {
		l.cache.SetIgnoredFileContents(params.TextDocument.URI, params.TextDocument.Text)
	} else {
		// check if file is currently being templated
		if _, isTemplating := l.templatingFiles.Get(params.TextDocument.URI); isTemplating {
			l.log.Message("%s is being templated, skipping didOpen update", params.TextDocument.URI)
		} else {
			l.cache.SetFileContents(params.TextDocument.URI, params.TextDocument.Text)
		}

		sendToAll(lintFileJob{Reason: "textDocument/didOpen", URI: params.TextDocument.URI},
			l.lintFileJobs, l.builtinsPositionJobs,
		)
	}

	return struct{}{}, nil
}

func (l *LanguageServer) handleTextDocumentDidClose(params types.DidCloseTextDocumentParams) (any, error) {
	// if the file being closed is ignored, we clear it from the ignored state in the cache.
	if l.ignoreURI(params.TextDocument.URI) {
		l.cache.Delete(params.TextDocument.URI)
	}

	return struct{}{}, nil
}

func (l *LanguageServer) handleTextDocumentDidChange(params types.DidChangeTextDocumentParams) (any, error) {
	if len(params.ContentChanges) == 0 {
		return struct{}{}, nil
	}

	var contents string

	for _, change := range params.ContentChanges {
		if change.Range == nil {
			// If no range is specified, the whole document is replaced.
			contents = change.Text
		} else {
			if contents == "" {
				var ok bool
				// If a range is specified, we patch the existing content.
				if contents, ok = l.maybeIgnoredContents(params.TextDocument.URI); !ok {
					return nil, fmt.Errorf("failed to get file contents for uri %q", params.TextDocument.URI)
				}
			}

			contents = patch(contents, change.Text, *change.Range)
		}
	}

	if ignored := l.setMaybeIgnoredContents(params.TextDocument.URI, contents); !ignored {
		sendToAll(lintFileJob{Reason: "textDocument/didChange", URI: params.TextDocument.URI},
			l.lintFileJobs,
			l.builtinsPositionJobs,
		)
	}

	return struct{}{}, nil
}

func (l *LanguageServer) maybeIgnoredContents(uri string) (string, bool) {
	if l.ignoreURI(uri) {
		return l.cache.GetIgnoredFileContents(uri)
	}

	return l.cache.GetFileContents(uri)
}

func (l *LanguageServer) setMaybeIgnoredContents(uri, contents string) bool {
	ignored := l.ignoreURI(uri)

	if l.ignoreURI(uri) {
		l.cache.SetIgnoredFileContents(uri, contents)
	} else {
		l.cache.SetFileContents(uri, contents)
	}

	return ignored
}

func patch(doc, text string, rang types.Range) string {
	start := positionToOffset(doc, types.Position{Line: rang.Start.Line, Character: rang.Start.Character})
	end := positionToOffset(doc, types.Position{Line: rang.End.Line, Character: rang.End.Character})

	if start < 0 || end < 0 || start > len(doc) || end > len(doc) || start > end {
		return doc // invalid range
	}

	return doc[:start] + text + doc[end:]
}

func (l *LanguageServer) handleTextDocumentDidSave(
	ctx context.Context,
	params types.DidSaveTextDocumentParams,
) (any, error) {
	if params.Text == nil || !strings.Contains(*params.Text, "\r\n") {
		return struct{}{}, nil
	}

	enabled, err := linter.NewLinter().WithUserConfig(*l.getLoadedConfig()).DetermineEnabledRules(ctx)
	if err != nil {
		l.log.Message("failed to determine enabled rules: %s", err)

		return struct{}{}, nil
	}

	if slices.ContainsFunc(enabled, util.EqualsAny(ruleNameOPAFmt, ruleNameUseRegoV1)) {
		resp := types.ShowMessageParams{
			Type:    2, // warning
			Message: "CRLF line ending detected. Please change editor setting to use LF for line endings.",
		}

		if err := l.conn.Notify(ctx, "window/showMessage", resp); err != nil {
			l.log.Message("failed to notify: %s", err)
		}
	}

	return struct{}{}, nil
}

func (l *LanguageServer) handleTextDocumentDocumentSymbol(params types.DocumentSymbolParams) (any, error) {
	if l.ignoreURI(params.TextDocument.URI) {
		return noDocumentSymbols, nil
	}

	contents, module, ok := l.cache.GetContentAndModule(params.TextDocument.URI)
	if !ok {
		l.log.Message("failed to get file contents for uri %q", params.TextDocument.URI)

		return noDocumentSymbols, nil
	}

	bis := l.builtinsForCurrentCapabilities()

	return documentsymbol.All(contents, module, bis), nil
}

func (l *LanguageServer) handleTextDocumentFoldingRange(params types.FoldingRangeParams) (any, error) {
	text, module, ok := l.cache.GetContentAndModule(params.TextDocument.URI)
	if !ok {
		return noFoldingRanges, nil
	}

	return foldingrange.FindAll(text, module), nil
}

func (l *LanguageServer) handleTextDocumentFormatting(
	ctx context.Context,
	params types.DocumentFormattingParams,
) (any, error) {
	// Fetch the contents used for formatting from the appropriate cache location.
	oldContent, _ := l.maybeIgnoredContents(params.TextDocument.URI)

	// if the file is empty, then the formatters will fail, so we template instead
	if oldContent == "" {
		// disable the templating feature for files in the workspace root.
		if filepath.Dir(l.toPath(params.TextDocument.URI)) == l.workspacePath() {
			return []types.TextEdit{}, nil
		}

		newContent, err := l.templateContentsForFile(params.TextDocument.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to template contents as a templating fallback: %w", err)
		}

		l.cache.ClearFileDiagnostics()
		l.cache.SetFileContents(params.TextDocument.URI, newContent)

		l.lintFileJobs <- lintFileJob{Reason: "internal/templateFormattingFallback", URI: params.TextDocument.URI}

		return ComputeEdits(oldContent, newContent), nil
	}

	// opa-fmt is the default formatter if not set in the client options
	formatter := "opa-fmt"

	if l.client.InitOptions != nil && l.client.InitOptions.Formatter != nil {
		formatter = *l.client.InitOptions.Formatter
	}

	var newContent string

	switch formatter {
	case "opa-fmt", "opa-fmt-rego-v1":
		opts := format.Opts{RegoVersion: l.regoVersionForURI(params.TextDocument.URI)}
		if formatter == "opa-fmt-rego-v1" {
			opts.RegoVersion = ast.RegoV0CompatV1
		}

		f := &fixes.Fmt{OPAFmtOpts: opts}

		fixResults, err := f.Fix(
			&fixes.FixCandidate{Filename: filepath.Base(l.toPath(params.TextDocument.URI)), Contents: oldContent},
			&fixes.RuntimeOptions{BaseDir: l.workspacePath()},
		)
		if err != nil {
			l.log.Message("failed to format file: %s", err)

			return nil, nil // return "null" as per the spec
		}

		if len(fixResults) == 0 {
			return []types.TextEdit{}, nil
		}

		newContent = fixResults[0].Contents
	case "regal-fix":
		// set up an in-memory file provider to pass to the fixer for this one file
		memfp := fileprovider.NewInMemoryFileProvider(map[string]string{params.TextDocument.URI: oldContent})

		input, err := memfp.ToInput(l.loadedConfigAllRegoVersions.Clone())
		if err != nil {
			return nil, fmt.Errorf("failed to create fixer input: %w", err)
		}

		roots, err := config.GetPotentialRoots(l.workspacePath(), l.toPath(params.TextDocument.URI))
		if err != nil {
			return nil, fmt.Errorf("could not find potential roots: %w", err)
		}

		fi := fixer.NewFixer().RegisterFixes(fixes.NewDefaultFormatterFixes()...).RegisterRoots(roots...)
		li := linter.NewLinter().WithInputModules(&input)

		if cfg := l.getLoadedConfig(); cfg != nil {
			li = li.WithUserConfig(*cfg)
		}

		fixReport, err := fi.Fix(ctx, &li, memfp)
		if err != nil {
			return nil, fmt.Errorf("failed to format: %w", err)
		}

		if fixReport.TotalFixes() == 0 {
			return []types.TextEdit{}, nil
		}

		if newContent, err = memfp.Get(params.TextDocument.URI); err != nil {
			return nil, fmt.Errorf("failed to get formatted contents: %w", err)
		}
	default:
		return nil, fmt.Errorf("unrecognized formatter %q", formatter)
	}

	return ComputeEdits(oldContent, newContent), nil
}

func (l *LanguageServer) handleWorkspaceDidCreateFiles(params types.CreateFilesParams) (any, error) {
	if l.ignoreURI(params.Files[0].URI) {
		return struct{}{}, nil
	}

	for _, createOp := range params.Files {
		if _, _, err := l.cache.UpdateForURIFromDisk(l.fromPath(createOp.URI), l.toPath(createOp.URI)); err != nil {
			return nil, fmt.Errorf("failed to update cache for uri %q: %w", createOp.URI, err)
		}

		sendToAll(lintFileJob{Reason: "textDocument/didCreate", URI: createOp.URI},
			l.lintFileJobs,
			l.builtinsPositionJobs,
			l.templateFileJobs,
		)
	}

	return struct{}{}, nil
}

func (l *LanguageServer) handleWorkspaceDidDeleteFiles(
	ctx context.Context,
	params types.DeleteFilesParams,
) (any, error) {
	if l.ignoreURI(params.Files[0].URI) {
		return struct{}{}, nil
	}

	for _, deleteOp := range params.Files {
		l.cache.Delete(deleteOp.URI)

		if err := l.sendFileDiagnostics(ctx, deleteOp.URI); err != nil {
			l.log.Message("failed to send diagnostic: %s", err)
		}
	}

	return struct{}{}, nil
}

func (l *LanguageServer) handleWorkspaceDidRenameFiles(
	ctx context.Context,
	params types.RenameFilesParams,
) (any, error) {
	for _, renameOp := range params.Files {
		if l.ignoreURI(renameOp.OldURI) && l.ignoreURI(renameOp.NewURI) {
			continue
		}

		var err error

		content, ok := l.cache.GetFileContents(renameOp.OldURI)
		// if the content is not in the cache then we can attempt to load from
		// the disk instead.
		if !ok || content == "" {
			_, content, err = l.cache.UpdateForURIFromDisk(l.fromPath(renameOp.NewURI), l.toPath(renameOp.NewURI))
			if err != nil {
				return nil, fmt.Errorf("failed to update cache for uri %q: %w", renameOp.NewURI, err)
			}
		}

		// clear the cache and send diagnostics for the old URI to clear the client
		l.cache.Delete(renameOp.OldURI)

		if err = l.sendFileDiagnostics(ctx, renameOp.OldURI); err != nil {
			l.log.Message("failed to send diagnostic: %s", err)
		}

		if l.ignoreURI(renameOp.NewURI) {
			continue
		}

		l.cache.SetFileContents(renameOp.NewURI, content)

		sendToAll(lintFileJob{Reason: "textDocument/didRename", URI: renameOp.NewURI},
			l.lintFileJobs,
			l.builtinsPositionJobs,
			l.templateFileJobs, // if the file being moved is empty, we template it too (if empty)
		)
	}

	return struct{}{}, nil
}

func (l *LanguageServer) handleWorkspaceDiagnostic() (any, error) {
	workspaceReport := types.WorkspaceDiagnosticReport{Items: make([]types.WorkspaceFullDocumentDiagnosticReport, 0)}

	// if the workspace root is not set, then we return an empty report
	// since we can't provide workspace diagnostics without a workspace root
	// being set. This is unset when the client in is in single file mode.
	if l.workspaceRootURI == "" {
		return workspaceReport, nil
	}

	wkspceDiags, ok := l.cache.GetFileDiagnostics(l.workspaceRootURI)
	if !ok {
		wkspceDiags = noDiagnostics
	}

	workspaceReport.Items = append(workspaceReport.Items, types.WorkspaceFullDocumentDiagnosticReport{
		URI:   l.workspaceRootURI,
		Kind:  "full",
		Items: wkspceDiags,
	})

	return workspaceReport, nil
}

func (l *LanguageServer) handleInitialize(ctx context.Context, params types.InitializeParams) (any, error) {
	// Allow the Regal bundle to be read from path instead of the one embedded in the binary.
	// This allows an extremely fast feedback loop when working on language server policies.
	// TODO: Get this from init options or config instead.
	if devPath := os.Getenv("REGAL_BUNDLE_PATH"); devPath != "" && !testing.Testing() {
		fmt.Fprintln(os.Stderr, "REGAL_BUNDLE_PATH set. Will attempt to use development bundle from:", devPath)

		rbundle.Dev.SetBundlePath(devPath)
	}

	if os.Getenv("REGAL_DEBUG") != "" {
		fmt.Fprintln(os.Stderr, "Debug mode enabled")
		l.log.SetLevel(log.LevelDebug)
	}

	l.client = types.Client{
		Identifier:   clients.DetermineIdentifier(params.ClientInfo.Name),
		InitOptions:  params.InitializationOptions,
		Capabilities: params.Capabilities,
	}

	// params.RootURI not expected to have a trailing slash, remove if present for consistency
	rootURI := strings.TrimSuffix(params.RootURI, string(os.PathSeparator))
	if rootURI == "" {
		return nil, errors.New("rootURI was not set by the client but is required")
	}

	workspaceRootPath := uri.ToPath(l.client.Identifier, rootURI)

	configRoots, err := lsconfig.FindConfigRoots(workspaceRootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find config roots: %w", err)
	}

	l.workspaceRootURI = rootURI

	switch {
	case len(configRoots) > 1:
		l.log.Message("warning: multiple configuration root directories found in workspace:"+
			"\n%s\nusing %q as workspace root directory",
			strings.Join(configRoots, "\n"), configRoots[0],
		)

		l.workspaceRootURI = uri.FromPath(l.client.Identifier, configRoots[0])
	case len(configRoots) == 1:
		l.log.Message("using %q as workspace root directory", configRoots[0])

		l.workspaceRootURI = uri.FromPath(l.client.Identifier, configRoots[0])
	default:
		l.log.Message(
			"using workspace root directory: %q, custom config not found — may be inherited from parent directory",
			workspaceRootPath,
		)
	}

	if l.client.Identifier == clients.IdentifierGeneric {
		l.log.Message(
			"unable to match client identifier for initializing client, using generic functionality: %s",
			params.ClientInfo.Name,
		)
	}

	l.webServer.SetClient(l.client.Identifier)

	regoFilter := types.FileOperationFilter{Scheme: "file", Pattern: types.FileOperationPattern{Glob: "**/*.rego"}}
	fileOpOpts := types.FileOperationRegistrationOptions{Filters: []types.FileOperationFilter{regoFilter}}

	initializeResult := types.InitializeResult{
		Capabilities: types.ServerCapabilities{
			TextDocumentSyncOptions: types.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // we support incremental updates
				Save:      types.SaveOptions{IncludeText: true},
			},
			DiagnosticProvider: types.DiagnosticOptions{
				Identifier:            "rego",
				InterFileDependencies: true,
				WorkspaceDiagnostics:  true,
			},
			Workspace: types.WorkspaceOptions{
				FileOperations: types.FileOperationsServerCapabilities{
					DidCreate: fileOpOpts,
					DidRename: fileOpOpts,
					DidDelete: fileOpOpts,
				},
				WorkspaceFolders: types.WorkspaceFoldersServerCapabilities{
					// NOTE(anders): The language server protocol doesn't go into detail about what this is meant to
					// entail, and there's nothing else in the request/response payloads that carry workspace folder
					// information. The best source I've found on the this topic is this example repo from VS Code,
					// where they have the client start one instance of the server per workspace folder:
					// https://github.com/microsoft/vscode-extension-samples/tree/main/lsp-multi-server-sample
					// That seems like a reasonable approach to take, and means we won't have to deal with workspace
					// folders throughout the rest of the codebase. But the question then is — what is the point of
					// this capability, and what does it mean to say we support it? Clearly we don't in the server as
					// *there is no way* to support it here.
					Supported: true,
				},
			},
			InlayHintProvider: types.ResolveProviderOption{},
			HoverProvider:     true,
			SignatureHelpProvider: types.SignatureHelpOptions{
				TriggerCharacters: []string{"(", ","},
			},
			CodeActionProvider: types.CodeActionOptions{CodeActionKinds: []string{"quickfix", "source.explore"}},
			ExecuteCommandProvider: types.ExecuteCommandOptions{
				Commands: []string{
					"regal.debug",
					"regal.eval",
					"regal.fix.opa-fmt",
					"regal.fix.use-rego-v1",
					"regal.fix.use-assignment-operator",
					"regal.fix.no-whitespace-comment",
					"regal.fix.directory-package-mismatch",
					"regal.fix.non-raw-regex-pattern",
				},
			},
			DocumentFormattingProvider: true,
			FoldingRangeProvider:       true,
			DefinitionProvider:         true,
			DocumentSymbolProvider:     true,
			WorkspaceSymbolProvider:    true,
			CompletionProvider: types.CompletionOptions{
				CompletionItem: types.CompletionItemOptions{LabelDetailsSupport: true},
				// Note: these are characters that trigger completions *in addition to* the client's default characters.
				TriggerCharacters: []string{
					":", // to suggest :=
					".", // for refs
				},
			},
			CodeLensProvider:          types.ResolveProviderOption{},
			DocumentLinkProvider:      types.ResolveProviderOption{},
			DocumentHighlightProvider: true,
		},
	}

	defaultConfig, _ := config.WithDefaultsFromBundle(rbundle.LoadedBundle(), nil)

	l.loadedConfigLock.Lock()
	l.loadedConfig = &defaultConfig
	l.loadedConfigLock.Unlock()

	if err := l.loadEnabledRulesFromConfig(ctx, defaultConfig); err != nil {
		l.log.Message("failed to cache enabled rules: %s", err)
	}

	if l.workspaceRootURI != "" {
		workspaceRootPath := l.workspacePath()

		l.bundleCache = bundles.NewCache(workspaceRootPath, l.log)

		var configFilePath string
		if configFile, err := config.FindConfig(workspaceRootPath); err == nil {
			configFilePath = configFile.Name()
		} else if globalConfigDir := config.GlobalConfigDir(false); globalConfigDir != "" {
			// the file might not exist and we only want to log we're using the global file if it does.
			if globalConfigFile := filepath.Join(globalConfigDir, "config.yaml"); rio.IsFile(globalConfigFile) {
				configFilePath = globalConfigFile
			}
		}

		if configFilePath != "" {
			l.log.Message("using config file: %s", configFilePath)
			l.configWatcher.Watch(configFilePath)
		} else {
			l.log.Message("no config file found for workspace")
		}

		_, failed, err := l.loadWorkspaceContents(ctx, false)
		for _, f := range failed {
			l.log.Message("failed to load file %s: %s", f.URI, f.Error)
		}

		if err != nil {
			l.log.Message("failed to load workspace contents: %s", err)
		}

		l.webServer.SetWorkspaceURI(l.workspaceRootURI)

		// 'OverwriteAggregates' is set to populate the cache's initial aggregate state.
		// Subsequent runs of lintWorkspaceJobs will not set this and use the cached state.
		l.lintWorkspaceJobs <- lintWorkspaceJob{Reason: "server initialize", OverwriteAggregates: true}
	}

	return initializeResult, nil
}

type fileLoadFailure struct {
	URI   string
	Error error
}

func (l *LanguageServer) loadWorkspaceContents(ctx context.Context, newOnly bool) (
	[]string, []fileLoadFailure, error,
) {
	changedOrNewURIs := make([]string, 0)
	failed := make([]fileLoadFailure, 0)

	if err := files.DefaultWalker(l.workspacePath()).Walk(func(path string) error {
		fileURI := uri.FromPath(l.client.Identifier, path)
		if l.ignoreURI(fileURI) {
			return nil
		}

		// if the caller has requested only new files, then we can exit early
		// if the file is already in the cache.
		if _, ok := l.cache.GetFileContents(fileURI); newOnly && ok {
			return nil
		}

		changed, _, err := l.cache.UpdateForURIFromDisk(fileURI, path)
		if err != nil {
			failed = append(failed,
				fileLoadFailure{URI: fileURI, Error: fmt.Errorf("failed to update cache for uri %q: %w", path, err)},
			)

			return nil // continue processing other files
		}

		// there is no need to update the parse if the file contents
		// was not changed in the above operation.
		if !changed {
			return nil
		}

		if _, err = updateParse(ctx, l.parseOpts(fileURI, l.builtinsForCurrentCapabilities())); err != nil {
			failed = append(failed, fileLoadFailure{URI: fileURI, Error: fmt.Errorf("failed to update parse: %w", err)})

			return nil // continue processing other files
		}

		changedOrNewURIs = append(changedOrNewURIs, fileURI)

		return nil
	}); err != nil {
		return nil, nil, fmt.Errorf("failed to walk workspace dir %q: %w", l.workspacePath(), err)
	}

	if l.bundleCache != nil {
		if _, err := l.bundleCache.Refresh(); err != nil {
			return nil, nil, fmt.Errorf("failed to refresh the bundle cache: %w", err)
		}
	}

	return changedOrNewURIs, failed, nil
}

func (l *LanguageServer) handleInitialized() (any, error) {
	// if running without config, then we should send the diagnostic request now
	// otherwise it'll happen when the config is loaded
	if !l.configWatcher.IsWatching() {
		l.lintWorkspaceJobs <- lintWorkspaceJob{Reason: "server initialized"}
	}

	return struct{}{}, nil
}

func (*LanguageServer) handleTextDocumentDiagnostic() (any, error) {
	// this is a no-op. Because we accept the textDocument/didChange event, which contains the new content,
	// we don't need to do anything here as once the new content has been parsed, the diagnostics will be sent
	// on the channel regardless of this request.
	return nil, nil
}

func (l *LanguageServer) handleWorkspaceDidChangeWatchedFiles(
	params types.WorkspaceDidChangeWatchedFilesParams,
) (any, error) {
	// when a file is changed (saved), then we trigger a full workspace lint
	regoFiles := make([]string, 0, len(params.Changes))

	for _, change := range params.Changes {
		// this handles the case of a new config file being created when one did not exist before
		if util.HasAnySuffix(change.URI, filepath.Join(".regal", "config.yaml"), ".regal.yaml") {
			if configFile, err := config.FindConfig(l.workspacePath()); err == nil {
				l.configWatcher.Watch(configFile.Name())
				configFile.Close()
			}
		}

		if change.URI == "" || l.ignoreURI(change.URI) {
			continue
		}

		regoFiles = append(regoFiles, change.URI)
	}

	if len(regoFiles) > 0 {
		l.lintWorkspaceJobs <- lintWorkspaceJob{
			Reason: fmt.Sprintf("workspace/didChangeWatchedFiles (%s)", strings.Join(regoFiles, ", ")),
		}
	}

	return struct{}{}, nil
}

func (l *LanguageServer) sendFileDiagnostics(ctx context.Context, fileURI string) error {
	// first, set the diagnostics for the file to the current parse errors
	fileDiags, _ := l.cache.GetParseErrors(fileURI)

	// if there are no parse errors, then we can check for lint errors
	if len(fileDiags) == 0 {
		fileDiags, _ = l.cache.GetFileDiagnostics(fileURI)
	}

	// must be a non-nil slice, otherwise diagnostics may not be cleared by the client.
	if fileDiags == nil {
		fileDiags = noDiagnostics
	}

	err := l.conn.Notify(ctx, methodTdPublishDiagnostics, types.FileDiagnostics{URI: fileURI, Items: fileDiags})

	return util.WrapErr(err, "failed to notify")
}

func (l *LanguageServer) getFilteredModules() (map[string]*ast.Module, error) {
	allModules := l.cache.GetAllModules()
	ignore := l.getLoadedConfig().Ignore.Files

	filtered, err := config.FilterIgnoredPaths(outil.Keys(allModules), ignore, false, l.workspaceRootURI)
	if err != nil {
		return nil, fmt.Errorf("failed to filter ignored paths: %w", err)
	}

	modules := make(map[string]*ast.Module, len(filtered))
	for _, path := range filtered {
		modules[path] = allModules[path]
	}

	return modules, nil
}

func (l *LanguageServer) ignoreURI(fileURI string) bool {
	// TODO(charlieegan3): make this configurable for things like .rq etc?
	if !strings.HasSuffix(fileURI, ".rego") {
		return true
	}

	cfg := l.getLoadedConfig()
	paths, err := config.FilterIgnoredPaths([]string{l.toPath(fileURI)}, cfg.Ignore.Files, false, l.workspacePath())

	return err != nil || len(paths) == 0
}

func (l *LanguageServer) workspacePath() string {
	return uri.ToPath(l.client.Identifier, l.workspaceRootURI)
}

func (l *LanguageServer) toPath(fileURI string) string {
	return uri.ToPath(l.client.Identifier, fileURI)
}

func (l *LanguageServer) toRelativePath(fileURI string) string {
	return strings.TrimPrefix(l.toPath(fileURI), l.workspacePath()+string(os.PathSeparator))
}

func (l *LanguageServer) fromPath(filePath string) string {
	return uri.FromPath(l.client.Identifier, filePath)
}

func (l *LanguageServer) regoVersionForURI(fileURI string) ast.RegoVersion {
	if l.loadedConfigAllRegoVersions != nil {
		return rules.RegoVersionFromMap(
			l.loadedConfigAllRegoVersions.Clone(),
			strings.TrimPrefix(l.toPath(fileURI), l.workspacePath()),
			ast.RegoUndefined,
		)
	}

	return ast.RegoUndefined
}

// builtinsForCurrentCapabilities returns the map of builtins for use
// in the server based on the currently loaded capabilities. If there is no
// config, then the default for the Regal OPA version is used.
func (l *LanguageServer) builtinsForCurrentCapabilities() map[string]*ast.Builtin {
	capsURL := cmp.Or(l.getLoadedConfig().CapabilitiesURL, capabilities.DefaultURL)
	if bis, ok := l.loadedBuiltins.Get(capsURL); ok {
		return bis
	}

	return rego.BuiltinsForCapabilities(ast.CapabilitiesForThisVersion())
}

func (l *LanguageServer) parseOpts(fileURI string, bis map[string]*ast.Builtin) updateParseOpts {
	return updateParseOpts{
		Cache:            l.cache,
		Store:            l.regoStore,
		FileURI:          fileURI,
		Builtins:         bis,
		RegoVersion:      l.regoVersionForURI(fileURI),
		WorkspaceRootURI: l.workspaceRootURI,
	}
}

func (l *LanguageServer) regalContext(uri string, req *rego.Requirements) rego.RegalContext {
	rctx := rego.RegalContext{
		Client: l.client,
		File: rego.File{
			Name:        l.toRelativePath(uri),
			RegoVersion: l.regoVersionForURI(uri).String(),
			Abs:         l.toPath(uri),
		},
		Environment: rego.Environment{
			PathSeparator:    string(os.PathSeparator),
			WebServerBaseURI: l.webServer.GetBaseURL(),
			WorkspaceRootURI: l.workspaceRootURI,
		},
	}

	if req == nil {
		return rctx
	}

	if req.File.Lines {
		if content, ok := l.cache.GetFileContents(uri); ok {
			rctx.File.Lines = strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		} else {
			l.log.Message("failed to get file contents for uri %q", uri)
		}
	}

	return rctx
}

func positionToOffset(text string, p types.Position) int {
	bytesRead := 0

	for i, line := range strings.Split(text, "\n") {
		if line == "" {
			bytesRead++
		} else {
			bytesRead += len(line) + 1
		}

		//nolint:gosec
		if i == int(p.Line)-1 {
			return bytesRead + int(p.Character)
		}
	}

	return -1
}

func sendToAll[T any](val T, ch ...chan T) {
	for _, c := range ch {
		c <- val
	}
}
