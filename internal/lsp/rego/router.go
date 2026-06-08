package rego

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/lsp/client"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	ruri "github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	valueDecoder    = encoding.OfValue()
	inputValuesPool = &sync.Pool{New: func() any {
		return &inputCacheItem{
			input:  ast.NewObjectWithCapacity(3),
			params: ast.NewTerm(ast.NullValue),
			regctx: ast.NewTerm(ast.NullValue),
		}
	}}
	fileLines = Requirements{File: FileRequirements{Lines: true}}
)

func init() {
	ast.InternStringTerm(
		"textDocument/codeAction", "textDocument/codeLens", "textDocument/completion", "textDocument/documentLink",
		"textDocument/documentHighlight", "textDocument/foldingRange", "textDocument/hover", "textDocument/inlayHint",
		"textDocument/linkedEditingRange", "textDocument/selectionRange", "textDocument/semanticTokens/full",
		"textDocument/signatureHelp", "textDocument/references", "textDocument/prepareRename", "textDocument/rename",
		"completionItem/resolve", "inlayHint/resolve",

		"method", "params", "identifier",

		"feature_flags",
		"debug_provider",
		"explorer_provider",
		"inline_evaluation_provider",
		"opa_test_provider",
		"server",
		"content",
		"successful_parse_count",
		"parse_errors",
		"workspace_root_path",
		"foldingRange",
		"foldingrange",
		"bundle",
		"lineFoldingOnly",
		"init_options",
	)
}

type (
	Providers struct {
		ContextProvider              func(uri string, reqs Requirements) *RegalContext
		ContentProvider              func(uri string) (string, bool)
		IgnoredProvider              func(uri string) bool
		ParseErrorsProvider          func(uri string) ([]types.Diagnostic, bool)
		SuccessfulParseCountProvider func(uri string) (uint, bool)
		InputPathProvider            func(path string) string
	}

	Router struct {
		routes         map[string]Route
		resultHandlers map[string]ResultHandler
		providers      Providers
		log            *log.Logger
		qc             *query.Cache
	}

	Route struct {
		resolver regoContextHandler
		requires Requirements
	}

	ResultHandler      = func(context.Context, any) (any, error)
	regoHandler        = func(context.Context, *query.Prepared, Providers, *jsonrpc2.Request) (any, error)
	regoContextHandler = func(context.Context, *RegalContext, *jsonrpc2.Request) (any, error)

	InitializeResponse struct {
		Response struct {
			ServerInfo   types.ServerInfo `json:"serverInfo"`
			Capabilities any              `json:"capabilities"`
		} `json:"response"`
		Regal struct {
			Client    client.Client `json:"client"`
			Workspace struct {
				URI string `json:"uri"`
			} `json:"workspace"`
			Warnings []string `json:"warnings"`
		} `json:"regal"`
	}

	inputCacheItem struct {
		input  ast.Value
		params *ast.Term
		regctx *ast.Term
	}
)

func NewRouter(ctx context.Context, s storage.Store, qc *query.Cache, prvs Providers, log *log.Logger) *Router {
	if _, err := qc.GetOrSet(ctx, s, query.MainEval); err != nil {
		panic(err) // can't recover here
	}

	routes := map[string]Route{
		"textDocument/codeAction": {},
		"textDocument/codeLens": {requires: Requirements{File: FileRequirements{
			Lines:                    true,
			SuccessfulParseLineCount: true,
			ParseErrors:              true,
		}}},
		"textDocument/completion":          {requires: Requirements{File: FileRequirements{Lines: true}, InputPath: true}},
		"textDocument/documentLink":        {requires: fileLines},
		"textDocument/documentHighlight":   {requires: fileLines},
		"textDocument/foldingRange":        {requires: fileLines},
		"textDocument/hover":               {requires: fileLines},
		"textDocument/inlayHint":           {requires: Requirements{File: FileRequirements{Lines: true, ParseErrors: true}}},
		"textDocument/references":          {requires: fileLines},
		"textDocument/prepareRename":       {requires: fileLines},
		"textDocument/rename":              {requires: fileLines},
		"textDocument/linkedEditingRange":  {requires: fileLines},
		"textDocument/selectionRange":      {},
		"textDocument/semanticTokens/full": {requires: fileLines},
		"textDocument/signatureHelp":       {requires: fileLines},
		"completionItem/resolve":           {resolver: passthrough},
		"inlayHint/resolve":                {resolver: passthrough},

		"initialized": {}, // special case
	}

	return &Router{routes: routes, providers: prvs, qc: qc, log: log}
}

func (m *Router) RegisterResultHandler(method string, handler ResultHandler) {
	if m.resultHandlers == nil {
		m.resultHandlers = make(map[string]ResultHandler)
	}

	if _, ok := m.resultHandlers[method]; ok {
		panic("result handler already registered for method: " + method)
	}

	m.resultHandlers[method] = handler
}

func (m *Router) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	pq := m.qc.Get(query.MainEval)
	if pq == nil {
		return nil, fmt.Errorf("no prepared query for %s", query.MainEval)
	}

	if req.Method == "initialize" {
		if handler, ok := m.resultHandlers["initialize"]; ok {
			result, err := initialize(ctx, pq, req)
			if err != nil {
				return nil, err
			}

			return handler(ctx, result)
		}
		// this could be removed, but since this is currently a hard dependency
		// for the server, better be safe and error out here in case it's missing
		return nil, errors.New("no result handler registered for initialize")
	}

	if route, ok := m.routes[req.Method]; ok {
		if strings.HasPrefix(req.Method, "textDocument/") {
			result, err = m.textDocumentPassthroughHandlerFor(route)(ctx, pq, m.providers, req)
		} else {
			rctx := m.providers.ContextProvider("", route.requires) // No requirements for resolvers yet
			rctx.Query = pq

			result, err = passthrough(ctx, rctx, req)
		}

		if err == nil {
			if resultHandler, ok := m.resultHandlers[req.Method]; ok {
				result, err = resultHandler(ctx, result)
			}
		}

		return result, err
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: "method not supported: " + req.Method}
}

// handleError logs the given message and returns nil, unless REGAL_DEBUG
// is set, in which case msg is returned as an error. This makes issues
// more visible during development, while not annoying actual users.
func (m *Router) handleError(msg string) error {
	if os.Getenv("REGAL_DEBUG") != "" {
		return errors.New(msg)
	}

	m.log.Message(msg)

	return nil
}

// textDocumentPassthroughHandlerFor wraps a regoHandler which first verifies that the text document URI isn't
// ignored, and ensures that any custom requirements the handler may have are met.
func (m *Router) textDocumentPassthroughHandlerFor(route Route) regoHandler {
	return func(ctx context.Context, query *query.Prepared, prvs Providers, req *jsonrpc2.Request) (any, error) {
		maybeURI := jsoniter.Get(*req.Params, "textDocument", "uri")
		if maybeURI.LastError() != nil {
			return nil, fmt.Errorf("expected textDocument.uri parameter: %w", maybeURI.LastError())
		}

		docURI := maybeURI.ToString()
		if !strings.HasSuffix(docURI, ".rego") || prvs.IgnoredProvider != nil && prvs.IgnoredProvider(docURI) {
			// This is not an error, but perhaps we should wire in some debug logging later
			return nil, nil
		}

		rctx, err := regalContextForRequirements(prvs, docURI, route.requires)
		if err != nil {
			return nil, m.handleError(fmt.Sprintf("error handling route %s: %v", req.Method, err))
		} else if rctx == nil {
			return nil, nil // e.g. file has always been unparsable
		}

		rctx.Query = query

		return passthrough(ctx, rctx, req)
	}
}

func regalContextForRequirements(prvs Providers, uri string, reqs Requirements) (*RegalContext, error) {
	// Set up a basic RegalContext, which while not used by all routes, is provided for all.
	rctx := prvs.ContextProvider(uri, reqs)
	if reqs.File.Lines && rctx.File.Lines == nil {
		if prvs.ContentProvider == nil {
			return nil, errors.New("content provider required but not provided")
		}

		content, ok := prvs.ContentProvider(uri)
		if !ok {
			return nil, errors.New("content provider failed to provide content for URI: " + uri)
		}

		rctx.File.Lines = strings.Split(content, "\n")
	}

	if reqs.File.SuccessfulParseLineCount {
		if prvs.SuccessfulParseCountProvider == nil {
			return nil, errors.New("successful parse count provider required but not provided")
		}

		if splc, ok := prvs.SuccessfulParseCountProvider(uri); ok {
			rctx.File.SuccessfulParseCount = splc
		} else {
			// if the file has always been unparsable, we can return early
			return nil, nil //nolint:nilnil
		}
	}

	if reqs.File.ParseErrors {
		if prvs.ParseErrorsProvider == nil {
			return nil, errors.New("parse errors provider required but not provided")
		}

		if rctx.File.ParseErrors, _ = prvs.ParseErrorsProvider(uri); rctx.File.ParseErrors == nil {
			rctx.File.ParseErrors = make([]types.Diagnostic, 0)
		}
	}

	if reqs.InputPath {
		if prvs.InputPathProvider == nil {
			return nil, errors.New("input.json path provider required but not provided")
		}

		path := ruri.ToRelativePath(uri, rctx.Environment.WorkspaceRootURI)
		rctx.Environment.InputPath = prvs.InputPathProvider(path)
	}

	return rctx, nil
}

// passthrough is a handler that:
//  1. Parses provided input directly to an ast.Value without having 'params' roundtrip to an LSP Go type
//  2. Returns the result of evaluation as the Rego handler provides it, without an intermediate Go type in between
//
// This is much more efficient compared to the textDocument handler — which does both of those things — at the cost of
// potentially letting invalid input or output through. Runtime validation of types can however be done in Rego too,
// and that's where future handlers validation should take place when needed.
func passthrough(ctx context.Context, rctx *RegalContext, req *jsonrpc2.Request) (any, error) {
	if req.Params == nil {
		bs, _ := req.MarshalJSON()

		return nil, fmt.Errorf("expected request containing 'params', got %v", string(bs))
	}

	cached := inputValuesPool.Get().(*inputCacheItem) //nolint:forcetypeassert
	defer inputValuesPool.Put(cached)

	params, err := valueDecoder.Decode(*req.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to decode params: %w\n%s", err, string(*req.Params))
	}

	cached.params.Value = params
	cached.regctx.Value = rast.StructToValue(rctx)

	inputObj := cached.input.(ast.Object) //nolint:forcetypeassert
	rast.Insert(inputObj, "method", ast.InternedTerm(req.Method))
	rast.Insert(inputObj, "params", cached.params)
	rast.Insert(inputObj, "regal", cached.regctx)

	res, err := CachedQueryEvalUndecoded(ctx, rctx.Query, cached.input)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate prepared query: %w", err)
	}

	if obj, ok := res.(ast.Object); ok {
		rsp := obj.Get(ast.InternedTerm("response")).Value

		var buf bytes.Buffer

		if err := encoding.OfValue().Encode(&buf, rsp); err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		return new(json.RawMessage(buf.Bytes())), nil
	}

	return nil, fmt.Errorf("unexpected query result format: %v", res)
}

func initialize(ctx context.Context, pq *query.Prepared, req *jsonrpc2.Request) (any, error) {
	var result InitializeResponse

	paramsValue, err := transform.AnyToValue(req.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to decode initialize params: %w", err)
	}

	err = CachedQueryEval(ctx, pq, ast.NewObject(
		rast.Item("method", ast.InternedTerm(req.Method)),
		rast.Item("params", ast.NewTerm(paramsValue)),
	), &result)

	return result, err
}
