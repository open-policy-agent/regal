package rego

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp/handler"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/semantictokens"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	ruri "github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	emptyResponse = map[string]any{
		"textDocument/codeAction":        nil,
		"textDocument/documentLink":      nil,
		"textDocument/documentHighlight": nil,
		"textDocument/documentSymbol":    make([]types.DocumentSymbol, 0),
		"textDocument/codeLens":          make([]types.CodeLens, 0),
		"textDocument/hover":             make([]types.Hover, 0),
		"textDocument/signatureHelp":     nil,
	}
	errIgnored   = errors.New("ignored URI")
	valueDecoder = encoding.OfValue()

	inputValuePool = &sync.Pool{New: func() any {
		return ast.NewObjectWithCapacity(3)
	}}

	bufPool = &sync.Pool{New: func() any {
		return new(bytes.Buffer)
	}}
)

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
		ContextProvider              func(uri string, reqs *Requirements) *RegalContext
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
		handler  regoContextHandler
		resolver regoContextHandler
		requires *Requirements
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
		"textDocument/codeAction": {handler: passthrough},
		"textDocument/codeLens": {
			handler: textDocument[types.CodeLensParams, []types.CodeLens],
			requires: &Requirements{
				File: FileRequirements{
					Lines:                    true,
					SuccessfulParseLineCount: true,
					ParseErrors:              true,
				},
			},
		},
		"textDocument/completion": {
			handler: textDocument[types.CompletionParams, *types.CompletionList],
			requires: &Requirements{
				File:         FileRequirements{Lines: true},
				InputDotJSON: true,
			},
		},
		"textDocument/documentLink": {handler: passthrough},
		"textDocument/documentHighlight": {
			handler:  passthrough,
			requires: &Requirements{File: FileRequirements{Lines: true}},
		},
		"textDocument/foldingRange": {handler: passthrough},
		"textDocument/hover": {
			handler:  textDocument[types.HoverParams, *types.Hover],
			requires: &Requirements{File: FileRequirements{Lines: true}},
		},
		"textDocument/inlayHint": {
			handler:  textDocument[types.InlayHintParams, *[]types.InlayHint],
			requires: &Requirements{File: FileRequirements{Lines: true, ParseErrors: true}},
		},
		"textDocument/linkedEditingRange": {
			handler:  textDocument[types.LinkedEditingRangeParams, types.LinkedEditingRanges],
			requires: &Requirements{File: FileRequirements{Lines: true}},
		},
		"textDocument/selectionRange": {handler: passthrough},
		"textDocument/semanticTokens/full": {
			handler:  semanticTokensHandler,
			requires: &Requirements{File: FileRequirements{Lines: true}},
		},
		"textDocument/signatureHelp": {
			handler:  textDocument[types.SignatureHelpParams, *types.SignatureHelp],
			requires: &Requirements{File: FileRequirements{Lines: true}},
		},
		"completionItem/resolve": {
			resolver: resolve[types.CompletionItem],
		},
		"inlayHint/resolve": {
			resolver: resolve[types.InlayHint],
		},
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
		result, err := initialize(ctx, pq, req)
		if err != nil {
			return nil, err
		}

		if handler, ok := m.resultHandlers["initialize"]; ok {
			return handler(ctx, result)
		} else {
			// this could be removed, but since this is currently a hard dependency
			// for the server, better be safe and error out here in case it's missing
			return nil, errors.New("no result handler registered for initialize")
		}
	}

	if route, ok := m.routes[req.Method]; ok {
		if strings.HasSuffix(req.Method, "/resolve") && route.resolver != nil {
			return resolverFor(route)(ctx, pq, m.providers, req)
		}

		return handlerFor(route)(ctx, pq, m.providers, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: "method not supported: " + req.Method}
}

// handlerFor wraps a regoHandler which first verifies that the text document URI isn't
// ignored, and ensures that any custom requirements the handler may have are met.
func handlerFor(route Route) regoHandler {
	return func(ctx context.Context, query *query.Prepared, prvs Providers, req *jsonrpc2.Request) (any, error) {
		// This is mandatory requirement for all routes managed here.
		uri, err := decodeAndCheckURI(req, prvs.IgnoredProvider)
		if err != nil {
			if errors.Is(err, errIgnored) {
				return emptyResponse[req.Method], nil
			}

			return nil, fmt.Errorf("error handling route %s: %w", req.Method, err)
		}

		rctx, err := regalContextForRequirements(prvs, uri, route.requires)
		if err != nil {
			return nil, fmt.Errorf("error handling route %s: %w", req.Method, err)
		}

		if rctx == nil {
			return emptyResponse[req.Method], nil // e.g. file has always been unparsable
		}

		rctx.Query = query

		return route.handler(ctx, rctx, req)
	}
}

func resolverFor(route Route) regoHandler {
	return func(ctx context.Context, query *query.Prepared, prvs Providers, req *jsonrpc2.Request) (any, error) {
		rctx := prvs.ContextProvider("", nil) // No requirements for resolvers yet
		rctx.Query = query

		return route.resolver(ctx, rctx, req)
	}
}

func regalContextForRequirements(prvs Providers, uri string, reqs *Requirements) (*RegalContext, error) {
	// Set up a basic RegalContext, which while not used by all routes, is provided for all.
	rctx := prvs.ContextProvider(uri, reqs)
	if reqs == nil {
		return rctx, nil
	}

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

// textDocument is a handler that requires TextDocumentParams (i.e. a document URI)
// embedded in parameter of type P, returning a result of type R.
func textDocument[P, R any](ctx context.Context, rctx *RegalContext, req *jsonrpc2.Request) (any, error) {
	params, err := decodeParams[P](req)
	if err != nil {
		return nil, err
	}

	result, err := QueryEval[P, R](ctx, rctx.Query, NewInput(req.Method, rctx, params))
	if err != nil {
		return nil, err
	}

	// For now we just unwrap the LSP response here, but may use other fields in the future.
	// In particular, we'll likely want to allow Rego handlers to return detailed error messages.
	return result.Response, nil
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

	params, err := valueDecoder.Decode(*req.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to decode params: %w\n%s", err, string(*req.Params))
	}

	paramsTerm := ast.TermPtrPool.Get()
	regctxTerm := ast.TermPtrPool.Get()
	inputValue := inputValuePool.Get().(ast.Object) //nolint:forcetypeassert

	defer func() {
		inputValuePool.Put(inputValue)
		ast.TermPtrPool.Put(paramsTerm)
		ast.TermPtrPool.Put(regctxTerm)
	}()

	paramsTerm.Value = params
	regctxTerm.Value = rast.StructToValue(rctx)

	rast.Insert(inputValue, "method", ast.InternedTerm(req.Method))
	rast.Insert(inputValue, "params", paramsTerm)
	rast.Insert(inputValue, "regal", regctxTerm)

	res, err := CachedQueryEvalUndecoded(ctx, rctx.Query, inputValue)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate prepared query: %w", err)
	}

	if obj, ok := res.(ast.Object); ok {
		rsp := obj.Get(ast.InternedTerm("response")).Value
		buf := bufPool.Get().(*bytes.Buffer) //nolint:forcetypeassert

		buf.Reset()
		defer bufPool.Put(buf)

		if err := encoding.OfValue().Encode(buf, rsp); err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		raw := json.RawMessage(buf.Bytes())

		return &raw, nil
	}

	return nil, fmt.Errorf("unexpected query result format: %v", res)
}

func semanticTokensHandler(ctx context.Context, rctx *RegalContext, req *jsonrpc2.Request) (any, error) {
	res, err := textDocument[types.SemanticTokensParams, semantictokens.SemanticTokensResult](ctx, rctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate prepared query: %w", err)
	}

	return semantictokens.Full(res.(semantictokens.SemanticTokensResult)) //nolint:forcetypeassert
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

// resolve handlers return the same type they receive as parameter, but enriched with data it resolves.
func resolve[P any](ctx context.Context, rctx *RegalContext, req *jsonrpc2.Request) (any, error) {
	params, err := decodeParams[P](req)
	if err != nil {
		return nil, err
	}

	result, err := QueryEval[P, P](ctx, rctx.Query, NewInput(req.Method, rctx, params))
	if err != nil {
		return nil, err
	}

	return result.Response, nil
}

func decodeAndCheckURI(req *jsonrpc2.Request, ignored func(string) bool) (string, error) {
	tdp, err := decodeParams[types.TextDocumentParams](req)
	if err != nil {
		return "", err
	}

	if ignored != nil && ignored(tdp.TextDocument.URI) {
		return "", errIgnored
	}

	return tdp.TextDocument.URI, nil
}

func decodeParams[P any](req *jsonrpc2.Request) (P, error) {
	var params P

	err := handler.Decode(req, &params)

	return params, err
}
