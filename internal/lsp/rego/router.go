package rego

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/io"
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
			buf:    new(bytes.Buffer),
		}
	}}
	fileLines = Requirements{File: FileRequirements{Lines: true}}
)

type inputCacheItem struct {
	input  ast.Value
	params *ast.Term
	regctx *ast.Term
	buf    *bytes.Buffer
}

func init() {
	ast.InternStringTerm(
		"textDocument/codeAction", "textDocument/codeLens", "textDocument/completion", "textDocument/documentLink",
		"textDocument/documentHighlight", "textDocument/foldingRange", "textDocument/hover", "textDocument/inlayHint",
		"textDocument/linkedEditingRange", "textDocument/selectionRange", "textDocument/semanticTokens/full",
		"textDocument/signatureHelp", "completionItem/resolve", "inlayHint/resolve",

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
	}

	RegoRouter struct {
		routes         map[string]Route
		resultHandlers map[string]ResultHandler
		providers      Providers
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
			Client    types.Client `json:"client"`
			Workspace struct {
				URI string `json:"uri"`
			} `json:"workspace"`
			Warnings []string `json:"warnings"`
		} `json:"regal"`
	}
)

func NewRegoRouter(ctx context.Context, store storage.Store, qc *query.Cache, prvs Providers) *RegoRouter {
	if _, err := qc.GetOrSet(ctx, store, query.MainEval); err != nil {
		panic(err) // can't recover here
	}

	routes := map[string]Route{
		"textDocument/codeAction": {},
		"textDocument/codeLens": {requires: Requirements{File: FileRequirements{
			Lines:                    true,
			SuccessfulParseLineCount: true,
			ParseErrors:              true,
		}}},
		"textDocument/completion":          {requires: Requirements{File: FileRequirements{Lines: true}, InputDotJSON: true}},
		"textDocument/documentLink":        {requires: fileLines},
		"textDocument/documentHighlight":   {requires: fileLines},
		"textDocument/foldingRange":        {requires: fileLines},
		"textDocument/hover":               {requires: fileLines},
		"textDocument/inlayHint":           {requires: Requirements{File: FileRequirements{Lines: true, ParseErrors: true}}},
		"textDocument/linkedEditingRange":  {requires: fileLines},
		"textDocument/selectionRange":      {},
		"textDocument/semanticTokens/full": {requires: fileLines},
		"textDocument/signatureHelp":       {requires: fileLines},
		"completionItem/resolve":           {resolver: passthrough},
		"inlayHint/resolve":                {resolver: passthrough},
	}

	return &RegoRouter{routes: routes, providers: prvs, qc: qc}
}

func (m *RegoRouter) RegisterResultHandler(method string, handler ResultHandler) {
	if m.resultHandlers == nil {
		m.resultHandlers = make(map[string]ResultHandler)
	}

	if _, ok := m.resultHandlers[method]; ok {
		panic("result handler already registered for method: " + method)
	}

	m.resultHandlers[method] = handler
}

func (m *RegoRouter) Handle(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
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
		if strings.HasSuffix(req.Method, "/resolve") && route.resolver != nil {
			rctx := m.providers.ContextProvider("", Requirements{}) // No requirements for resolvers yet
			rctx.Query = pq

			return passthrough(ctx, rctx, req)
		}

		result, err := textDocumentPassthroughHandlerFor(route)(ctx, pq, m.providers, req)
		if err != nil {
			return nil, err
		}

		if handler, ok := m.resultHandlers[req.Method]; ok {
			return handler(ctx, result)
		}

		return result, nil
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: "method not supported: " + req.Method}
}

// textDocumentPassthroughHandlerFor wraps a regoHandler which first verifies that the text document URI isn't
// ignored, and ensures that any custom requirements the handler may have are met.
func textDocumentPassthroughHandlerFor(route Route) regoHandler {
	return func(ctx context.Context, query *query.Prepared, prvs Providers, req *jsonrpc2.Request) (any, error) {
		maybeURI := jsoniter.Get(*req.Params, "textDocument", "uri")
		if maybeURI.LastError() != nil {
			return nil, fmt.Errorf("expected textDocument.uri parameter at this point: %w", maybeURI.LastError())
		}

		docURI := maybeURI.ToString()
		if prvs.IgnoredProvider != nil && prvs.IgnoredProvider(docURI) {
			// This is not an error, but perhaps we should wire in some debug logging later
			return nil, nil
		}

		rctx, err := regalContextForRequirements(prvs, docURI, route.requires)
		if err != nil {
			return nil, fmt.Errorf("error handling route %s: %w", req.Method, err)
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

	if reqs.InputDotJSON {
		path := ruri.ToPath(uri)
		root := ruri.ToPath(rctx.Environment.WorkspaceRootURI)

		// TODO: Avoid the intermediate map[string]any step and unmarshal directly into ast.Value.
		inputDotJSONPath, inputDotJSONContent := io.FindInput(path, root)
		if inputDotJSONPath != "" && inputDotJSONContent != nil {
			inputDotJSONValue, err := transform.ToOPAInputValue(inputDotJSONContent)
			if err != nil {
				return nil, fmt.Errorf("failed to convert input.json to value: %w", err)
			}

			rctx.Environment.InputDotJSONPath = &inputDotJSONPath
			rctx.Environment.InputDotJSON = inputDotJSONValue
		}
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

		cached.buf.Reset()

		if err := encoding.OfValue().Encode(cached.buf, rsp); err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		raw := json.RawMessage(cached.buf.Bytes())

		return &raw, nil
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
