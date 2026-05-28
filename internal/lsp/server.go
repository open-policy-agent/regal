//nolint:nilnil
package lsp

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/ast/oracle"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/open-policy-agent/opa/v1/storage"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/capabilities"
	"github.com/open-policy-agent/regal/internal/compile"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/lsp/bundles"
	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/client"
	lsconfig "github.com/open-policy-agent/regal/internal/lsp/config"
	"github.com/open-policy-agent/regal/internal/lsp/documentsymbol"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/input"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/semantictokens"
	"github.com/open-policy-agent/regal/internal/lsp/store"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/lsp/window"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	"github.com/open-policy-agent/regal/internal/update"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/internal/web"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/fixer"
	"github.com/open-policy-agent/regal/pkg/fixer/fileprovider"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
	"github.com/open-policy-agent/regal/pkg/rules"
	"github.com/open-policy-agent/regal/pkg/version"
)

const (
	noInputFoundMsg = "No input.json/yaml file was found. " +
		"This file is used to provide input data for rule evaluation. " +
		"Would you like to create one?"

	inputCreateSuccessMsg      = "input.json created successfully! Running Evaluate will now pull from this file."
	crlfWarnMsg                = "CRLF line ending detected. Please change editor setting to use LF for line endings."
	methodTdPublishDiagnostics = "textDocument/publishDiagnostics"

	ruleNameOPAFmt    = "opa-fmt"
	ruleNameUseRegoV1 = "use-rego-v1"

	// rpcTimeout allows requests to complete independently from the server's ctx,
	// supporting graceful shutdown rather than immediate cancellation.
	rpcTimeout = 3 * time.Second
)

var (
	noDocumentSymbols                       any = make([]types.DocumentSymbol, 0)
	noTextEdits                             any = make([]types.TextEdit, 0)
	noWorkspaceFullDocumentDiagnosticReport any = make([]types.WorkspaceFullDocumentDiagnosticReport, 0)
	emptyStruct                             any = struct{}{}

	noDiagnostics = make([]types.Diagnostic, 0)

	validPathComponentPattern = regexp.MustCompile(`^\w+[\w\-]*\w+$`)

	fixFmt                    = &fixes.Fmt{OPAFmtOpts: format.Opts{}}
	fixUseRegoV1              = &fixes.Fmt{OPAFmtOpts: format.Opts{RegoVersion: ast.RegoV0CompatV1}}
	fixUseAssignmentOperator  = &fixes.UseAssignmentOperator{}
	fixNoWhitespaceComment    = &fixes.NoWhitespaceComment{}
	fixNonRawRegexPattern     = &fixes.NonRawRegexPattern{}
	fixPreferEqualsComparison = &fixes.PreferEqualsComparison{}
	fixConstantCondition      = &fixes.ConstantCondition{}
	fixRedundantExistence     = &fixes.RedundantExistenceCheck{}
)

// lintJob is sent to the lintJobs channel to trigger a linter run.
type lintJob struct {
	Reason string
}

type fileJob struct {
	Reason string
	URI    string
}

// DefaultServerFeatureFlags returns the default feature flags with all
// custom features enabled.
func DefaultServerFeatureFlags() *types.ServerFeatureFlags {
	return &types.ServerFeatureFlags{
		ExplorerProvider:         true,
		InlineEvaluationProvider: true,
		DebugProvider:            true,
		OPATestProvider:          true,
		TestCreationProvider:     true,
	}
}

type LanguageServerOptions struct {
	// Logger is the logger to use for the language server.
	Logger *log.Logger

	// WorkspaceDiagnosticsPoll, if set > 0 will cause a full workspace lint
	// to run on this interval. This is intended to be used where eventing
	// is not working, as expected. E.g. with a client that does not send
	// changes or when running in extremely slow environments like GHA with
	// the go race detector on. TODO, work out why this is required.
	WorkspaceDiagnosticsPoll time.Duration

	// FeatureFlags defines which custom features are enabled.
	// If not provided, DefaultServerFeatureFlags() will be used.
	FeatureFlags *types.ServerFeatureFlags
}

type LanguageServer struct {
	log          *log.Logger
	featureFlags types.ServerFeatureFlags

	regoStore storage.Store
	conn      *jsonrpc2.Conn
	window    *window.Window

	configWatcher               *lsconfig.Watcher
	loadedConfig                *config.Config
	loadedConfigLock            sync.RWMutex
	loadedConfigAllRegoVersions *concurrent.Map[string, ast.RegoVersion]
	loadedBuiltins              *concurrent.Map[string, map[string]*ast.Builtin]

	cache       *cache.Cache
	bundleCache *bundles.Cache
	queryCache  *query.Cache

	regoRouter *rego.Router

	// initializationGate blocks workers until the initialized notification is received
	initializationGate     chan struct{}
	initializationGateOnce sync.Once

	lintJobs         chan lintJob
	templateFileJobs chan fileJob
	testLocationJobs chan fileJob
	prepareQueryJobs chan struct{}
	commandRequest   chan types.ExecuteCommandParams

	input *input.Manager

	// templatingFiles tracks files currently being templated to ensure
	// other updates are not processed while the file is being updated.
	templatingFiles *concurrent.Map[string, bool]

	webServer *web.Server

	workspace                workspace.Workspace
	workspaceDiagnosticsPoll time.Duration

	// workersWg tracks all running worker goroutines to enable clean shutdown
	workersWg sync.WaitGroup

	// Flag used to suppress input.json prompt if user chooses to ignore it
	supressInputPrompt bool
}

type fileLoadFailure struct {
	URI   string
	Error error
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
	rstore := store.NewRegalStore()
	featureFlags := util.Or(opts.FeatureFlags, DefaultServerFeatureFlags)

	_ = store.PutServer(ctx, rstore, types.ServerContext{FeatureFlags: *featureFlags, Version: version.Version})

	ls := &LanguageServer{
		cache:              c,
		queryCache:         qc,
		loadedConfig:       cfg,
		regoStore:          rstore,
		log:                opts.Logger,
		featureFlags:       *featureFlags,
		input:              input.NewManager(rstore, opts.Logger),
		initializationGate: make(chan struct{}),
		lintJobs:           make(chan lintJob, 10),
		commandRequest:     make(chan types.ExecuteCommandParams, 10),
		templateFileJobs:   make(chan fileJob, 10),
		// at start up, we need to be able to fire many of these in quick succession for large repos
		// without blocking.
		testLocationJobs:            make(chan fileJob, 1000),
		prepareQueryJobs:            make(chan struct{}, 1),
		templatingFiles:             concurrent.MapOf(make(map[string]bool)),
		webServer:                   web.NewServer(opts.Logger),
		loadedBuiltins:              concurrent.MapOf(make(map[string]map[string]*ast.Builtin)),
		workspaceDiagnosticsPoll:    opts.WorkspaceDiagnosticsPoll,
		loadedConfigAllRegoVersions: concurrent.MapOf(make(map[string]ast.RegoVersion)),
	}

	ls.regoRouter = rego.NewRouter(ctx, rstore, qc, rego.Providers{
		ContextProvider:              ls.regalContext,
		IgnoredProvider:              ls.ignoreURI,
		ContentProvider:              ls.cache.GetFileContents,
		ParseErrorsProvider:          ls.cache.GetParseErrors,
		SuccessfulParseCountProvider: ls.cache.GetSuccessfulParseLineCount,
		InputPathProvider:            ls.input.FindForPath,
	}, opts.Logger)

	ls.regoRouter.RegisterResultHandler("initialize", ls.initializeResultHandler)
	ls.regoRouter.RegisterResultHandler("initialized", ls.initializedResultHandler)
	ls.regoRouter.RegisterResultHandler("textDocument/semanticTokens/full", semantictokens.ResultHandler)

	merged, _ := config.WithDefaultsFromBundle(bundle.Embedded(), cfg)

	// Even though user configuration (if provided) will overwrite some of the default configuration,
	// loading the default conf in the "constructor" ensures we can assume there's *some* configuration
	// set everywhere in the language server code.
	ls.loadConfig(ctx, merged)

	return ls
}

func (l *LanguageServer) Workspace() workspace.Workspace {
	l.loadedConfigLock.RLock()
	defer l.loadedConfigLock.RUnlock()

	return l.workspace
}

func (l *LanguageServer) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	l.log.Debug("received request: %s", req.Method)

	// null params are allowed, but only for certain methods
	if req.Params == nil && req.Method != "shutdown" && req.Method != "exit" {
		return nil, handler.ErrInvalidParams
	}

	switch req.Method {
	case "textDocument/definition":
		return handler.WithParams(req, l.handleTextDocumentDefinition)
	case "textDocument/diagnostic":
		return l.handleTextDocumentDiagnostic()
	case "textDocument/didOpen":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentDidOpen)
	case "textDocument/didClose":
		return handler.WithParams(req, l.handleTextDocumentDidClose)
	case "textDocument/didSave":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentDidSave)
	case "textDocument/documentSymbol":
		return handler.WithParams(req, l.handleTextDocumentDocumentSymbol)
	case "textDocument/didChange":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentDidChange)
	case "textDocument/formatting":
		return handler.WithContextAndParams(ctx, req, l.handleTextDocumentFormatting)
	case "workspace/didChangeWatchedFiles":
		return handler.WithContextAndParams(ctx, req, l.handleWorkspaceDidChangeWatchedFiles)
	case "workspace/diagnostic":
		return l.handleWorkspaceDiagnostic()
	case "workspace/didRenameFiles":
		return handler.WithContextAndParams(ctx, req, l.handleWorkspaceDidRenameFiles)
	case "workspace/didDeleteFiles":
		return handler.WithContextAndParams(ctx, req, l.handleWorkspaceDidDeleteFiles)
	case "workspace/didCreateFiles":
		return handler.WithContextAndParams(ctx, req, l.handleWorkspaceDidCreateFiles)
	case "workspace/executeCommand":
		return handler.WithParams(req, l.handleWorkspaceExecuteCommand)
	case "workspace/symbol":
		return l.handleWorkspaceSymbol()
	case "regal/runTests":
		return handler.WithContextAndParams(ctx, req, l.handleRunTests)
	case "shutdown":
		// no-op as we wait for the exit signal before closing channel
		return emptyStruct, nil
	case "exit":
		// close the channel, cancel the context for all workers, and exit
		if err := l.conn.Close(); err != nil {
			return nil, fmt.Errorf("failed to close connection: %w", err)
		}

		return emptyStruct, nil
	case "$/setTrace":
		return handler.WithParams(req, func(params types.TraceParams) (any, error) {
			if level, err := log.TraceValueToLevel(params.Value); err != nil {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: err.Error()}
			} else if level != log.LevelOff {
				// VS Code sets this to "off" a few seconds after initialization,
				// for no apparent reason. Perhaps we shouldn't use this level to
				// determine logging, but what else is it for?
				l.log.SetLevel(level)
			}

			return emptyStruct, nil
		})
	case "$/cancelRequest":
		// NOTE: no-op, implement if we want to support longer running, client-triggered operations
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#dollarRequests
		return emptyStruct, nil
	}

	// returns jsonrpc2.Error with code jsonrpc2.CodeMethodNotFound if provided unknown method.
	return l.regoRouter.Handle(ctx, l.conn, req)
}

func (l *LanguageServer) SetConn(conn *jsonrpc2.Conn) {
	l.conn = conn
	l.window = window.New(conn, l.log)
}

// Shutdown waits for all worker goroutines to complete. The context can be
// used to set a timeout or cancel the wait if workers take too long to exit.
// The context passed to workers should be cancelled before calling this method.
func (l *LanguageServer) Shutdown(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		l.workersWg.Wait()

		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *LanguageServer) StartDiagnosticsWorker(ctx context.Context) {
	l.workersWg.Go(func() {
		var wg sync.WaitGroup

		if l.workspaceDiagnosticsPoll > 0 {
			wg.Go(func() {
				ticker := time.NewTicker(l.workspaceDiagnosticsPoll)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						l.lintJobs <- lintJob{Reason: "poll ticker"}
					}
				}
			})
		}

		// coalescing channel: non-blocking send ensures multiple triggers
		// coalesce into a single lint run, avoiding redundant expensive work.
		work := make(chan struct{}, 1)

		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-l.lintJobs:
					l.log.Debug("linting: %s", job.Reason)

					select {
					case work <- struct{}{}:
					default:
					}
				}
			}
		})

		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-work:
					l.log.Debug("linting workspace")

					err := updateWorkspaceDiagnostics(ctx, diagnosticsRunOpts{
						Cache:            l.cache,
						RegalConfig:      l.getLoadedConfig(),
						WorkspaceRootURI: l.Workspace().URI(),
						CustomRulesPath:  l.getCustomRulesPath(),
					})
					if err != nil {
						l.log.Message("failed to lint workspace: %s", err)

						continue
					}

					for fileURI := range l.cache.GetAllFiles() {
						l.sendFileDiagnostics(ctx, fileURI)
					}

					l.log.Debug("linting workspace done")
				}
			}
		})

		<-ctx.Done()
		wg.Wait()
	})
}

// StartQueryCacheWorker starts a worker that waits for query strings on the
// queryCacheJobs channel, and re-prepares and stores them in the query cache,
// upon receiving them. This is currently used only when the REGAL_BUNDLE_PATH
// development mode is set, to ensure we recompile on live bundle updates.
func (l *LanguageServer) StartQueryCacheWorker(ctx context.Context) {
	l.workersWg.Go(func() {
		if !bundle.DevModeEnabled() {
			l.log.Debug("LSP development mode not enabled — not starting query cache worker")

			return
		}

		bundle.Dev.Subscribe(l.prepareQueryJobs)

		for {
			select {
			case <-ctx.Done():
				return
			case <-l.prepareQueryJobs:
				if err := l.queryCache.Store(ctx, query.MainEval, l.regoStore); err != nil {
					l.log.Message("failed to prepare query %s: %s", query.MainEval, err)
				} else {
					l.log.Message("re-prepared query %s", query.MainEval)
				}
			}
		}
	})
}

func (l *LanguageServer) StartConfigWorker(ctx context.Context) {
	if err := l.configWatcher.Start(ctx); err != nil {
		l.log.Message("failed to start config watcher: %s", err)

		return
	}

	l.workersWg.Go(func() {
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

				mergedConfig, err := config.WithDefaultsFromBundle(bundle.Loaded(), &userConfig)
				if err != nil {
					l.log.Message("failed to load config: %s", err)

					continue
				}

				l.loadConfig(ctx, mergedConfig)

				l.workersWg.Go(func() {
					if l.getLoadedConfig().Features.Remote.CheckVersion &&
						os.Getenv(update.CheckVersionDisableEnvVar) == "" {
						update.CheckAndWarn(ctx, update.Options{
							CurrentVersion: version.Version,
							CurrentTime:    time.Now().UTC(),
							StateDir:       config.GlobalConfigDir(true),
						}, os.Stderr)
					}
				})

				l.lintJobs <- lintJob{Reason: "config file changed"}
			case <-l.configWatcher.Drop:
				l.loadedConfigLock.Lock()

				defaultConfig, _ := config.WithDefaultsFromBundle(bundle.Loaded(), nil)
				l.loadedConfig = &defaultConfig
				l.loadedConfigLock.Unlock()

				l.lintJobs <- lintJob{Reason: "config file dropped"}
			}
		}
	})
}

// StartWorkspaceStateWorker will poll for changes to the workspaces state that
// are not sent from the client. For example, when a file a is removed from the
// workspace after changing branch.
func (l *LanguageServer) StartWorkspaceStateWorker(ctx context.Context) {
	l.workersWg.Go(func() {
		timer := time.NewTicker(2 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				// first clear files that are missing from the workspaceDir
				for fileURI := range l.cache.GetAllFiles() {
					if _, err := os.Stat(uri.ToPath(fileURI)); os.IsNotExist(err) {
						// clear the cache first, then send the diagnostics based on the cleared cache
						l.cache.Delete(fileURI)
						l.sendFileDiagnostics(ctx, fileURI)
					}
				}

				// for this next operation, the workspace root must be set as it's
				// used to scan for new files.
				if l.Workspace().URI() == "" {
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
					parseSuccess, err := updateParse(ctx, l.parseOpts(cnURI, l.builtinsForCurrentCapabilities()))
					if err != nil {
						l.log.Message("failed to update module for %s: %s", cnURI, err)
					} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
						l.testLocationJobs <- fileJob{URI: cnURI}
					}

					l.lintJobs <- lintJob{Reason: "internal/workspaceStateWorker/changedOrNewFile"}
				}
			}
		}
	})
}

// StartWebServer starts the web server that serves explorer.
func (l *LanguageServer) StartWebServer(ctx context.Context) {
	l.webServer.Start(ctx)
}

// StartTemplateWorker runs the process of the server that templates newly
// created Rego files.
func (l *LanguageServer) StartTemplateWorker(ctx context.Context) {
	l.workersWg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-l.templateFileJobs:
				if err := l.processTemplateJob(ctx, job); err != nil {
					l.log.Message(err.Error())
				}
			}
		}
	})
}

// getLoadedConfig returns the currently loaded config, which may either be the default config embedded in Regal, or the
// default config merged with a user's config file. This function should never return nil, as even when a user config is
// not provided or fails to load, the default config is used as a fallback.
func (l *LanguageServer) getLoadedConfig() *config.Config {
	l.loadedConfigLock.RLock()
	defer l.loadedConfigLock.RUnlock()

	return l.loadedConfig
}

func (l *LanguageServer) getCustomRulesPath() string {
	if workspace := l.Workspace(); workspace.URI() != "" {
		if customRulesPath := workspace.Path(".regal", "rules"); rio.IsDir(customRulesPath) {
			return customRulesPath
		}
	}

	return ""
}

func (l *LanguageServer) loadConfig(ctx context.Context, conf config.Config) {
	l.loadedConfigLock.Lock()
	l.loadedConfig = &conf
	l.loadedConfigLock.Unlock()

	if err := store.PutConfig(ctx, l.regoStore, &conf); err != nil {
		l.log.Message("failed to update config in storage: %v", err)
	}

	workspace := l.Workspace()

	// Rego versions may have changed, so reload them.
	if workspace.Path() != "" {
		allRegoVersions, err := config.AllRegoVersions(workspace.Path(), &conf)
		if err != nil {
			l.log.Debug("failed to reload rego versions: %s", err)
		} else {
			l.loadedConfigAllRegoVersions.Clear()

			for k, v := range allRegoVersions {
				l.loadedConfigAllRegoVersions.Set(k, v)
			}
		}
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

	if err := store.PutBuiltins(ctx, l.regoStore, bis); err != nil {
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

		if err := store.RemoveFileMod(ctx, l.regoStore, k); err != nil {
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
		if ok := l.cache.HasFileContents(k); !ok {
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

// processTemplateJob handles the templating of a newly created Rego file.
func (l *LanguageServer) processTemplateJob(ctx context.Context, job fileJob) error {
	l.log.Debug("template worker received job: %s (reason: %s)", job.URI, job.Reason)

	// mark file as being templated to prevent race conditions
	l.templatingFiles.Set(job.URI, true)
	defer l.templatingFiles.Delete(job.URI)

	// disable the templating feature for files in the workspace root.
	if filepath.Dir(uri.ToPath(job.URI)) == l.Workspace().Path() {
		return nil
	}

	// determine the new contents for the file, if permitted
	newContents, err := l.templateContentsForFile(job.URI)
	if err != nil {
		return fmt.Errorf("failed to template new file: %w", err)
	}

	// set the contents of the new file in the cache immediately as
	// these must be update to date in order for fixRenameParams
	// to work
	l.cache.SetFileContents(job.URI, newContents)

	// determine if a rename is needed based on the new file package.
	// edits will be empty if no file rename is needed.
	additionalRenameEdits, err := l.fixRenameChanges(job.URI)
	if err != nil {
		return fmt.Errorf("failed to get rename params: %w", err)
	}

	// combine content changes with any additional rename changes
	params := workspace.NewApplyEditParams("Template new Rego file").
		WithTimeout(rpcTimeout).
		WithChanges(types.NewTextDocumentEdit(job.URI, ComputeEdits("", newContents))).
		WithChanges(additionalRenameEdits...)

	if err = l.Workspace().ApplyEdit(ctx, params); err != nil {
		return fmt.Errorf("failed to apply workspace edit for templating new file: %w", err)
	}

	// finally, trigger a diagnostics run for the new contents
	l.lintJobs <- lintJob{Reason: "internal/templateNewFile"}

	return nil
}

func (l *LanguageServer) templateContentsForFile(fileURI string) (string, error) {
	path := uri.ToPath(fileURI)

	// this function should not be called with files in the root, but if it is,
	// then it is an error to prevent unwanted behavior.
	if filepath.Dir(path) == l.Workspace().Path() {
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
		roots = []string{l.Workspace().Path()}
	} else {
		roots = append(roots, l.Workspace().Path())
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
			pkg += fmt.Sprintf("[%q]", part)
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
	docURI := params.TextDocument.URI
	if l.ignoreURI(docURI) {
		return nil, nil
	}

	contents, ok := l.cache.GetFileContents(docURI)
	if !ok {
		return nil, fmt.Errorf("textDocument/definition: failed to get file contents for uri %s", docURI)
	}

	// modules are loaded from the cache and keyed by their URI.
	modules, err := l.getFilteredModules()
	if err != nil {
		return nil, fmt.Errorf("failed to filter ignored paths: %w", err)
	}

	definition, err := oracle.New().
		WithCompiler(compile.NewCompilerWithRegalBuiltins()).
		FindDefinition(oracle.DefinitionQuery{
			// The value of Filename is used if the defn in the current buffer.
			Filename: l.Workspace().RelativePath(docURI),
			Pos:      params.Position.ToOffset(contents),
			Modules:  modules,
			Buffer:   outil.StringToByteSlice(contents),
		})
	if err != nil {
		if !util.IsAnyError(err, oracle.ErrNoDefinitionFound, oracle.ErrNoMatchFound) {
			l.log.Message("failed to find definition: %s", err)
		}

		return nil, nil // the user could have clicked anywhere. return "null" as per the spec
	}

	// res.File will be relative to the workspace root. The response here needs
	// a URI for the client to be able to navigate correctly.
	res := definition.Result
	resURI := l.Workspace().URI(res.File)
	resRng := types.RangeBetween(res.Row-1, res.Col-1, res.Row-1, res.Col-1)

	return types.Location{URI: resURI, Range: resRng}, nil
}

func (l *LanguageServer) handleTextDocumentDidOpen(
	ctx context.Context,
	params types.DidOpenTextDocumentParams,
) (any, error) {
	workspace := l.Workspace()
	// we have started the server, but not yet received a suitable root to use
	if workspace.URI() == "" {
		// get the URI of the file's immediate parent
		rootURI := workspace.URI(filepath.Dir(uri.ToPath(params.TextDocument.URI)))
		if err := l.loadWorkspace(ctx, rootURI, workspace.Client()); err != nil {
			l.log.Message("failed to update server root URI: %w", err)
		}
	}

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

		parseSuccess, err := updateParse(ctx, l.parseOpts(params.TextDocument.URI, l.builtinsForCurrentCapabilities()))
		if err != nil {
			l.log.Message("failed to update module for %s: %s", params.TextDocument.URI, err)
		} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
			l.testLocationJobs <- fileJob{URI: params.TextDocument.URI}
		}

		l.lintJobs <- lintJob{Reason: "textDocument/didOpen"}
	}

	return emptyStruct, nil
}

func (l *LanguageServer) handleTextDocumentDidClose(params types.DidCloseTextDocumentParams) (any, error) {
	// if the file being closed is ignored, we clear it from the ignored state in the cache.
	if l.ignoreURI(params.TextDocument.URI) {
		l.cache.Delete(params.TextDocument.URI)
	}

	return emptyStruct, nil
}

func (l *LanguageServer) handleTextDocumentDidChange(
	ctx context.Context,
	params types.DidChangeTextDocumentParams,
) (any, error) {
	if len(params.ContentChanges) == 0 {
		return emptyStruct, nil
	}

	var contents string

	for _, change := range params.ContentChanges {
		if change.Range == nil {
			// If no range is specified, the whole document is replaced.
			// This is currently the only change type we support.
			contents = change.Text
		}
	}

	if ignored := l.setMaybeIgnoredContents(params.TextDocument.URI, contents); !ignored {
		opts := l.parseOpts(params.TextDocument.URI, l.builtinsForCurrentCapabilities())

		parseSuccess, err := updateParse(ctx, opts)
		if err != nil {
			l.log.Message("failed to update module for %s: %s", params.TextDocument.URI, err)
		} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
			l.testLocationJobs <- fileJob{URI: params.TextDocument.URI}
		}

		l.lintJobs <- lintJob{Reason: "textDocument/didChange"}
	}

	return emptyStruct, nil
}

func (l *LanguageServer) maybeIgnoredContents(uri string) (string, bool) {
	if l.ignoreURI(uri) {
		return l.cache.GetIgnoredFileContents(uri)
	}

	return l.cache.GetFileContents(uri)
}

func (l *LanguageServer) setMaybeIgnoredContents(uri, contents string) bool {
	ignored := l.ignoreURI(uri)
	if ignored {
		l.cache.SetIgnoredFileContents(uri, contents)
	} else {
		l.cache.SetFileContents(uri, contents)
	}

	return ignored
}

func (l *LanguageServer) handleTextDocumentDidSave(
	ctx context.Context,
	params types.DidSaveTextDocumentParams,
) (any, error) {
	// If dev mode is enabled, reload the bundle on save. Otherwise, this is a no-op.
	bundle.Dev.Reload()

	if params.Text == nil || !strings.Contains(*params.Text, "\r\n") {
		return emptyStruct, nil
	}

	enabled, err := linter.NewLinter().WithUserConfig(*l.getLoadedConfig()).DetermineEnabledRules(ctx)
	if err != nil {
		l.log.Message("failed to determine enabled rules: %s", err)

		return emptyStruct, nil
	}

	if slices.ContainsFunc(enabled, util.EqualsAny(ruleNameOPAFmt, ruleNameUseRegoV1)) {
		l.window.ShowMessage(ctx, types.WarningMessage, crlfWarnMsg)
	}

	return emptyStruct, nil
}

func (l *LanguageServer) handleTextDocumentDocumentSymbol(params types.DocumentSymbolParams) (any, error) {
	if l.ignoreURI(params.TextDocument.URI) {
		return noDocumentSymbols, nil
	}

	contents, module, ok := l.cache.GetContentAndModule(params.TextDocument.URI)
	if !ok {
		l.log.Message("textDocument/documentSymbol: failed to get file contents for uri %q", params.TextDocument.URI)

		return noDocumentSymbols, nil
	}

	return documentsymbol.All(contents, module, l.builtinsForCurrentCapabilities()), nil
}

func (l *LanguageServer) handleTextDocumentFormatting(
	ctx context.Context,
	params types.DocumentFormattingParams,
) (any, error) {
	// Fetch the contents used for formatting from the appropriate cache location.
	oldContent, _ := l.maybeIgnoredContents(params.TextDocument.URI)
	if oldContent == "" {
		// if the file is empty, then the formatters will fail, so we template instead
		if filepath.Dir(uri.ToPath(params.TextDocument.URI)) == l.Workspace().Path() {
			// disable the templating feature for files in the workspace root.
			return noTextEdits, nil
		}

		newContent, err := l.templateContentsForFile(params.TextDocument.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to template contents as a templating fallback: %w", err)
		}

		l.cache.ClearFileDiagnostics()
		l.cache.SetFileContents(params.TextDocument.URI, newContent)

		parseSuccess, err := updateParse(ctx, l.parseOpts(params.TextDocument.URI, l.builtinsForCurrentCapabilities()))
		if err != nil {
			l.log.Message("failed to update module for %s: %s", params.TextDocument.URI, err)
		} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
			l.testLocationJobs <- fileJob{URI: params.TextDocument.URI}
		}

		l.lintJobs <- lintJob{Reason: "internal/templateFormattingFallback"}

		return ComputeEdits(oldContent, newContent), nil
	}

	// opa-fmt is the default formatter if not set in the client options
	formatter := cmp.Or(l.Workspace().Client().InitOptions.Formatter, "opa-fmt")

	var newContent string

	switch formatter {
	case "opa-fmt", "opa-fmt-rego-v1":
		opts := format.Opts{RegoVersion: l.regoVersionForURI(params.TextDocument.URI)}
		if formatter == "opa-fmt-rego-v1" {
			opts.RegoVersion = ast.RegoV0CompatV1
		}

		f := &fixes.Fmt{OPAFmtOpts: opts}

		fixResults, err := f.Fix(
			&fixes.FixCandidate{Filename: filepath.Base(uri.ToPath(params.TextDocument.URI)), Contents: oldContent},
			&fixes.RuntimeOptions{BaseDir: l.Workspace().Path()},
		)
		if err != nil {
			l.log.Message("failed to format file: %s", err)

			return nil, nil // return "null" as per the spec
		}

		if len(fixResults) == 0 {
			return noTextEdits, nil
		}

		newContent = fixResults[0].Contents
	case "regal-fix":
		// set up an in-memory file provider to pass to the fixer for this one file
		memfp := fileprovider.NewInMemoryFileProvider(map[string]string{params.TextDocument.URI: oldContent})

		input, err := memfp.ToInput(l.loadedConfigAllRegoVersions.Clone())
		if err != nil {
			return nil, fmt.Errorf("failed to create fixer input: %w", err)
		}

		roots, err := config.GetPotentialRoots(l.Workspace().Path(), uri.ToPath(params.TextDocument.URI))
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
			return noTextEdits, nil
		}

		if newContent, err = memfp.Get(params.TextDocument.URI); err != nil {
			return nil, fmt.Errorf("failed to get formatted contents: %w", err)
		}
	default:
		return nil, fmt.Errorf("unrecognized formatter %q", formatter)
	}

	return ComputeEdits(oldContent, newContent), nil
}

func (l *LanguageServer) handleWorkspaceDidCreateFiles(
	ctx context.Context,
	params types.CreateFilesParams,
) (any, error) {
	if l.ignoreURI(params.Files[0].URI) {
		return emptyStruct, nil
	}

	for _, createOp := range params.Files {
		if _, _, err := l.cache.UpdateForURIFromDisk(createOp.URI, uri.ToPath(createOp.URI)); err != nil {
			return nil, fmt.Errorf("failed to update cache for uri %q: %w", createOp.URI, err)
		}

		parseSuccess, err := updateParse(ctx, l.parseOpts(createOp.URI, l.builtinsForCurrentCapabilities()))
		if err != nil {
			l.log.Message("failed to update module for %s: %s", createOp.URI, err)
		} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
			l.testLocationJobs <- fileJob{URI: createOp.URI}
		}

		l.lintJobs <- lintJob{Reason: "textDocument/didCreate"}

		l.templateFileJobs <- fileJob{URI: createOp.URI}
	}

	return emptyStruct, nil
}

func (l *LanguageServer) handleWorkspaceDidDeleteFiles(ctx context.Context, dfp types.DeleteFilesParams) (any, error) {
	for _, deleteOp := range dfp.Files {
		if !l.ignoreURI(deleteOp.URI) {
			l.cache.Delete(deleteOp.URI)
			l.sendFileDiagnostics(ctx, deleteOp.URI)
		}
	}

	return emptyStruct, nil
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
			_, content, err = l.cache.UpdateForURIFromDisk(renameOp.NewURI, uri.ToPath(renameOp.NewURI))
			if err != nil {
				return nil, fmt.Errorf("failed to update cache for uri %q: %w", renameOp.NewURI, err)
			}
		}

		// clear the cache and send diagnostics for the old URI to clear the client
		l.cache.Delete(renameOp.OldURI)
		l.sendFileDiagnostics(ctx, renameOp.OldURI)

		if l.ignoreURI(renameOp.NewURI) {
			continue
		}

		l.cache.SetFileContents(renameOp.NewURI, content)

		parseSuccess, err := updateParse(ctx, l.parseOpts(renameOp.NewURI, l.builtinsForCurrentCapabilities()))
		if err != nil {
			l.log.Message("failed to update module for %s: %s", renameOp.NewURI, err)
		} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
			l.testLocationJobs <- fileJob{URI: renameOp.NewURI}
		}

		l.lintJobs <- lintJob{Reason: "textDocument/didRename"}

		l.templateFileJobs <- fileJob{URI: renameOp.NewURI}
	}

	return emptyStruct, nil
}

func (l *LanguageServer) handleWorkspaceDiagnostic() (any, error) {
	// we can't provide workspace diagnostics without a workspace root being set (e.g. single file mode)
	rootURI := l.Workspace().URI()
	if rootURI == "" {
		return noWorkspaceFullDocumentDiagnosticReport, nil
	}

	wkspceDiags, ok := l.cache.GetFileDiagnostics(rootURI)
	if !ok {
		wkspceDiags = noDiagnostics
	}

	return types.WorkspaceDiagnosticReport{Items: []types.WorkspaceFullDocumentDiagnosticReport{{
		URI:   rootURI,
		Kind:  "full",
		Items: wkspceDiags,
	}}}, nil
}

func (l *LanguageServer) initializeResultHandler(ctx context.Context, result any) (any, error) {
	if bundle.DevModeEnabled() {
		l.log.Message("Development mode enabled. Will attempt to build bundle from:", os.Getenv("REGAL_BUNDLE_PATH"))
		bundle.Dev.SetPath(os.Getenv("REGAL_BUNDLE_PATH"))
	}

	if os.Getenv("REGAL_DEBUG") != "" {
		l.log.SetLevel(log.LevelDebug)
		l.log.Message("Debug mode enabled")
	}

	response, ok := result.(rego.InitializeResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected result type for initialize: %T", result)
	}

	if err := l.loadWorkspace(ctx, response.Regal.Workspace.URI, response.Regal.Client); err != nil {
		l.log.Message("failed to load workspace: %w", err)
	}

	for _, warning := range response.Regal.Warnings {
		l.log.Message(warning)
	}

	return response.Response, nil
}

func (l *LanguageServer) loadWorkspace(ctx context.Context, rootURI string, client client.Client) error {
	if err := store.PutClient(ctx, l.regoStore, client); err != nil {
		return fmt.Errorf("failed to store client in rego store: %w", err)
	}

	// rootURI not expected to have a trailing slash, remove if present for consistency
	normalizedRootURI := strings.TrimSuffix(rootURI, string(os.PathSeparator))

	configRoots, err := lsconfig.FindConfigRoots(uri.ToPath(normalizedRootURI))
	if err != nil {
		return fmt.Errorf("failed to find config roots: %w", err)
	}

	var workspaceRootURI string

	switch {
	case len(configRoots) > 1:
		l.log.Message("warning: multiple configuration root directories found in workspace:"+
			"\n%s\nusing %q as workspace root directory",
			strings.Join(configRoots, "\n"), configRoots[0],
		)

		workspaceRootURI = client.URIFromPath(configRoots[0])
	case len(configRoots) == 1:
		l.log.Message("using %q as workspace root directory", configRoots[0])

		workspaceRootURI = client.URIFromPath(configRoots[0])
	default:
		workspaceRootURI = rootURI

		l.log.Message(
			"using workspace root directory: %q, custom config not found — may be inherited from parent directory",
			rootURI,
		)
	}

	workspace := workspace.New(workspaceRootURI).WithClient(client.WithConnection(l.conn))

	var configFilePath string
	if configFile, err := config.Find(workspace.Path()); err == nil {
		configFilePath = configFile.Name()
	} else if globalConfigDir := config.GlobalConfigDir(false); globalConfigDir != "" {
		// the file might not exist and we only want to log we're using the global file if it does.
		if globalConfigFile := filepath.Join(globalConfigDir, "config.yaml"); rio.IsFile(globalConfigFile) {
			configFilePath = globalConfigFile
		}
	}

	l.loadedConfigLock.Lock()
	defer l.loadedConfigLock.Unlock()

	l.bundleCache = bundles.NewCache(workspace.Path(), l.log)
	l.workspace = workspace

	if configFilePath != "" {
		l.log.Message("using config file: %s", configFilePath)
		l.configWatcher.Watch(configFilePath)
	} else {
		l.log.Message("no config file found for workspace")
	}

	l.input.LoadFromWorkspace(ctx, workspace)

	return nil
}

type fileToLoad struct {
	uri  string
	path string
}

func (l *LanguageServer) loadWorkspaceContents(ctx context.Context, newOnly bool) ([]string, []fileLoadFailure, error) {
	workspace := l.Workspace()
	if workspace.Path() == "" {
		// this happens in single file cases
		l.log.Debug("skipping loading of workspace files as path is empty")

		return nil, nil, nil
	}

	// Walk the workspace and enqueue files that need loading from disk.
	walkErr := make(chan error, 1)
	fileCh := make(chan fileToLoad, 1000)

	go func() {
		defer close(fileCh)

		walkErr <- files.DefaultWalker(workspace.Path()).Walk(func(path string) error {
			fileURI := workspace.URI(path)
			if l.ignoreURI(fileURI) {
				return nil
			}

			if newOnly {
				if ok := l.cache.HasFileContents(fileURI); ok {
					return nil
				}
			}

			fileCh <- fileToLoad{uri: fileURI, path: path}

			return nil
		})
	}()

	var (
		mu               sync.Mutex
		changedOrNewURIs = make([]string, 0)
		failed           = make([]fileLoadFailure, 0)
		wg               sync.WaitGroup
	)

	for range 10 {
		wg.Go(func() {
			for f := range fileCh {
				changed, _, err := l.cache.UpdateForURIFromDisk(f.uri, f.path)
				if err != nil {
					mu.Lock()

					failed = append(failed,
						fileLoadFailure{URI: f.uri, Error: fmt.Errorf("failed to update cache for uri %q: %w", f.path, err)},
					)
					mu.Unlock()

					continue
				}

				if !changed {
					continue
				}

				if _, err := updateParse(ctx, l.parseOpts(f.uri, l.builtinsForCurrentCapabilities())); err != nil {
					l.log.Message("error parsing file %s", f.uri)
					mu.Lock()

					failed = append(failed,
						fileLoadFailure{URI: f.uri, Error: fmt.Errorf("failed to update parse: %w", err)},
					)
					mu.Unlock()
				}

				mu.Lock()

				changedOrNewURIs = append(changedOrNewURIs, f.uri)
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	if err := <-walkErr; err != nil {
		return nil, nil, fmt.Errorf("failed to walk workspace dir %q: %w", l.Workspace().Path(), err)
	}

	if l.bundleCache != nil {
		if _, err := l.bundleCache.Refresh(); err != nil {
			return nil, nil, fmt.Errorf("failed to refresh the bundle cache: %w", err)
		}
	}

	return changedOrNewURIs, failed, nil
}

func (l *LanguageServer) initializedResultHandler(ctx context.Context, result any) (any, error) {
	// If the client supports dynamic registration, register for any the Rego
	// handler returned. Currently this is workspace/didChangeWatchedFiles only.
	if raw, ok := result.(*json.RawMessage); ok && len(*raw) > 4 { // = len("null")
		if err := l.conn.Call(ctx, "client/registerCapability", &raw, nil); err != nil {
			l.log.Message("failed to register workspace/didChangeWatchedFiles capability: %s", err)
		}
	}

	// Load workspace contents and start jobs asynchronously
	// This allows us to respond to the client immediately while workspace
	// loading happens in the background
	go func() {
		// Use newOnly=true to ensure that files already in the cache from editor messages
		// (e.g., textDocument/didOpen) are not clobbered during workspace initialization
		newURIs, failed, err := l.loadWorkspaceContents(ctx, true)
		for _, f := range failed {
			l.log.Message("failed to load file %s: %s", f.URI, f.Error)
		}

		if err != nil {
			l.log.Message("failed to load workspace contents: %s", err)
		}

		// must start other workers here otherwise the test locations block
		l.initializationGateOnce.Do(func() { close(l.initializationGate) })

		builtins := l.builtinsForCurrentCapabilities()

		for _, cnURI := range newURIs {
			parseSuccess, err := updateParse(ctx, l.parseOpts(cnURI, builtins))
			if err != nil {
				l.log.Message("failed to update module for %s: %s", cnURI, err)
			} else if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
				l.testLocationJobs <- fileJob{URI: cnURI}
			}
		}

		l.lintJobs <- lintJob{Reason: "Workspace Initialization"}
	}()

	return emptyStruct, nil
}

func (*LanguageServer) handleTextDocumentDiagnostic() (any, error) {
	// this is a no-op. Because we accept the textDocument/didChange event, which contains the new content,
	// we don't need to do anything here as once the new content has been parsed, the diagnostics will be sent
	// on the channel regardless of this request.
	return nil, nil
}

func (l *LanguageServer) handleWorkspaceDidChangeWatchedFiles(
	ctx context.Context,
	params types.WorkspaceDidChangeWatchedFilesParams,
) (any, error) {
	changes := false

	for _, change := range slices.Compact(params.Changes) {
		switch {
		case change.URI == "":
		case l.ignoreURI(change.URI):
			if l.input.HasInputSuffix(change.URI) {
				switch change.Type {
				case 1, 2:
					if err := l.input.Update(ctx, change.URI, nil); err != nil {
						l.log.Message("failed to update input for %s: %s", change.URI, err)
					}
				case 3:
					if err := l.input.Delete(ctx, change.URI); err != nil {
						l.log.Message("failed to delete input entry for %s: %s", change.URI, err)
					}
				}
			} else if change.Type == 1 && config.HasConfigSuffix(change.URI) {
				// this handles the case of a new config file being created when one did not exist before
				if configFile, err := config.Find(l.Workspace().Path()); err == nil {
					l.configWatcher.Watch(configFile.Name())
					rio.CloseIgnore(configFile)
				}
			}
		case strings.HasSuffix(change.URI, ".rego"):
			parseSuccess, err := updateParse(ctx, l.parseOpts(change.URI, l.builtinsForCurrentCapabilities()))
			if err == nil {
				if l.Workspace().Client().InitOptions.EnableServerTesting && parseSuccess {
					l.testLocationJobs <- fileJob{URI: change.URI}
				}

				changes = true
			} else {
				l.log.Message("failed to update module for %s: %s", change.URI, err)
			}
		}
	}

	if changes {
		l.lintJobs <- lintJob{Reason: "workspace/didChangeWatchedFiles"}
	}

	return emptyStruct, nil
}

func (l *LanguageServer) sendFileDiagnostics(ctx context.Context, fileURI string) {
	if l.conn == nil {
		l.log.Debug("sendFileDiagnostics called with no connection: %s", fileURI)

		return
	}

	// first, set the diagnostics for the file to the current parse errors
	fileDiags, _ := l.cache.GetParseErrors(fileURI)
	if len(fileDiags) == 0 {
		// if there are no parse errors, then we can check for lint errors
		fileDiags, _ = l.cache.GetFileDiagnostics(fileURI)
	}

	// must be a non-nil slice, otherwise diagnostics may not be cleared by the client.
	if fileDiags == nil {
		fileDiags = noDiagnostics
	}

	err := l.conn.Notify(ctx, methodTdPublishDiagnostics, types.FileDiagnostics{URI: fileURI, Items: fileDiags})
	if err != nil {
		l.log.Message("failed to send file diagnostic %w", err)
	}
}

func (l *LanguageServer) getFilteredModules() (map[string]*ast.Module, error) {
	allModules := l.cache.GetAllModules()
	ignore := l.getLoadedConfig().Ignore.Files

	filtered, err := config.FilterIgnoredPaths(outil.Keys(allModules), ignore, false, l.Workspace().URI())
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
	paths, err := config.FilterIgnoredPaths([]string{uri.ToPath(fileURI)}, cfg.Ignore.Files, false, l.Workspace().Path())

	return err != nil || len(paths) == 0
}

func (l *LanguageServer) regoVersionForURI(fileURI string) ast.RegoVersion {
	if l.loadedConfigAllRegoVersions != nil {
		return rules.RegoVersionFromMap(
			l.loadedConfigAllRegoVersions.Clone(),
			strings.TrimPrefix(uri.ToPath(fileURI), l.Workspace().Path()),
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

	return rego.BuiltinsForDefaultCapabilities()
}

func (l *LanguageServer) parseOpts(fileURI string, bis map[string]*ast.Builtin) updateParseOpts {
	return updateParseOpts{
		Cache:       l.cache,
		Store:       l.regoStore,
		FileURI:     fileURI,
		Builtins:    bis,
		RegoVersion: l.regoVersionForURI(fileURI),
		Workspace:   l.Workspace(),
	}
}

func (l *LanguageServer) regalContext(fileURI string, _ rego.Requirements) *rego.RegalContext {
	return &rego.RegalContext{
		File: rego.File{
			Name:        l.Workspace().RelativePath(fileURI),
			RegoVersion: l.regoVersionForURI(fileURI).String(),
			URI:         fileURI,
		},
		Environment: rego.Environment{
			PathSeparator:     string(os.PathSeparator),
			WorkspaceRootURI:  l.Workspace().URI(),
			WorkspaceRootPath: l.Workspace().Path(),
		},
	}
}

func (l *LanguageServer) handleInputSkeletonPrompt(
	ctx context.Context,
	target, ruleName string,
	row int,
) (bool, error) {
	compiler := compile.NewCompilerWithRegalBuiltins()
	compiler.Compile(l.cache.GetAllModules())

	if compiler.Failed() {
		l.log.Message("failed to compile workspace modules for input skeleton: %v", compiler.Errors)
	}

	// Using the compiled modules to parse the rules. The dependencies package used in inputSkeletonFromRule
	// relies on compiled modules to resolve transitive dependencies.
	var compiledRule *ast.Rule

	if compiledModule, ok := compiler.Modules[target]; ok {
		for _, rule := range compiledModule.Rules {
			if rule.Head.Name.String() == ruleName && rule.Location.Row == row {
				compiledRule = rule

				break
			}
		}
	}

	if compiledRule == nil {
		return false, nil
	}

	skeleton := inputSkeletonFromRule(compiledRule, compiler)
	if len(skeleton) == 0 {
		return false, nil
	}

	msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	action := l.window.ShowMessageRequest(msgCtx, types.InfoMessage, noInputFoundMsg, "Yes", "No", "Ignore")

	switch action {
	case "Yes":
		data, err := json.MarshalIndent(skeleton, "", "  ")
		if err != nil {
			return false, fmt.Errorf("failed to marshal input skeleton: %w", err)
		}

		workspace := l.Workspace()

		inputFile := workspace.Path("input.json")
		if err = os.WriteFile(inputFile, append(data, '\n'), 0o600); err != nil {
			return false, fmt.Errorf("failed to create input.json: %w", err)
		}

		openAction := l.window.ShowMessageRequest(ctx, types.InfoMessage, inputCreateSuccessMsg, "Open")
		if openAction == "Open" {
			l.window.ShowDocument(ctx, workspace.URI(inputFile), false)
		}

		return true, nil
	case "Ignore":
		l.supressInputPrompt = true
	}

	return false, nil
}
