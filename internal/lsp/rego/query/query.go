package query

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/resolver"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"

	_ "github.com/open-policy-agent/regal/pkg/builtins"
)

const (
	RuleHeadLocations = "data.regal.ast.rule_head_locations"
	MainEval          = "data.regal.lsp.main.eval"
	TestLocations     = "data.regal.lsp.testlocations.result"
)

var (
	RegoEnablePrint      = rego.EnablePrintStatements(true)
	RegoGenJSONValue     = rego.GenerateJSON(genJSONValue)
	RegoEvalGenJSONValue = rego.EvalGenerateJSON(genJSONValue)
	RegoStderrPrintHook  = rego.PrintHook(StderrPrintHook)
	RegoCapabilities     = rego.Capabilities(io.Capabilities())
	RegoStoreReadAST     = rego.StoreReadAST(true)

	genJSONValue    = func(term *ast.Term, _ *rego.EvalContext) (any, error) { return term.Value, nil }
	StderrPrintHook = topdown.NewPrintHook(os.Stderr)
)

type (
	Cache struct {
		prepared *concurrent.Map[string, *Prepared]
	}

	Prepared struct {
		body     ast.Body
		prepared *rego.PreparedEvalQuery
		store    storage.Store
	}

	schemaResolver struct {
		value ast.Value
	}
	regoOptions = []func(*rego.Rego)
)

func SchemaResolvers() []func(*rego.Rego) {
	return schemaResolvers()
}

func NewCache() *Cache {
	return &Cache{prepared: concurrent.MapOf(make(map[string]*Prepared, 5))}
}

func (q *Prepared) EvalQuery() *rego.PreparedEvalQuery {
	return q.prepared
}

func (q *Prepared) String() string {
	return q.body.String()
}

func (c *Cache) Store(ctx context.Context, query string, store storage.Store) error {
	parsedQuery := parse.Query(query)

	pq, err := prepareQuery(ctx, parsedQuery, store)
	if err != nil {
		return fmt.Errorf("failed preparing query %q: %w", query, err)
	}

	c.prepared.Set(query, &Prepared{body: parsedQuery, prepared: pq, store: store})

	return nil
}

func (c *Cache) Get(query string) *Prepared {
	p, _ := c.prepared.Get(query)

	return p
}

func (c *Cache) GetOrSet(ctx context.Context, store storage.Store, query string) (*Prepared, error) {
	cq, ok := c.prepared.Get(query)
	if !ok {
		parsedQuery := parse.Query(query)

		pq, err := prepareQuery(ctx, parsedQuery, store)
		if err != nil {
			return nil, fmt.Errorf("failed preparing query %q: %w", query, err)
		}

		cq = &Prepared{body: parsedQuery, prepared: pq, store: store}

		c.prepared.Set(query, cq)

		return cq, nil
	}

	if rbundle.DevModeEnabled() {
		// In dev mode, we always prepare the query to ensure changes in the bundle are reflected
		// immediately. We can however reuse the query and the store (if set).
		pq, err := prepareQuery(ctx, cq.body, cq.store)
		if err != nil {
			return nil, fmt.Errorf("failed preparing query %q: %w", query, err)
		}

		cq.prepared = pq
	}

	return cq, nil
}

func prepareQuery(ctx context.Context, query ast.Body, store storage.Store) (*rego.PreparedEvalQuery, error) {
	args, txn := prepareQueryArgs(ctx, query, store, rbundle.Loaded())

	// Note that we currently don't provide metrics or profiling here, and
	// most likely we should — need to consider how to best make that conditional
	// and how to present it if enabled.
	pq, err := rego.New(args...).PrepareForEval(ctx)
	if err != nil {
		if store != nil {
			store.Abort(ctx, txn)
		}

		if rbundle.DevModeEnabled() {
			// Try falling back to the embedded bundle, or else we'll
			// easily have errors popping up as notifications, making it
			// really hard to fix the issue that broke the query (like a parse error)
			args, txn = prepareQueryArgs(ctx, query, store, rbundle.Embedded())
			if pq, err = rego.New(args...).PrepareForEval(ctx); err == nil {
				if store != nil && txn != nil {
					if err := store.Commit(ctx, txn); err != nil {
						return nil, err
					}
				}

				return &pq, nil
			}

			if store != nil {
				store.Abort(ctx, txn)
			}
		}

		return nil, err
	}

	if store != nil && txn != nil {
		if err := store.Commit(ctx, txn); err != nil {
			return nil, err
		}
	}

	return &pq, nil
}

func prepareQueryArgs(
	ctx context.Context,
	query ast.Body,
	store storage.Store,
	rb *bundle.Bundle,
) (regoOptions, storage.Transaction) {
	resolvers := SchemaResolvers()
	args := append(append(make([]func(*rego.Rego), 0, 8+len(resolvers)),
		RegoCapabilities, RegoGenJSONValue,
		RegoEnablePrint, RegoStderrPrintHook, // enabled for debugging, should probably be conditional
		rego.ParsedQuery(query), rego.ParsedBundle("regal", rb),
	), resolvers...)

	var txn storage.Transaction
	if store != nil {
		txn, _ = store.NewTransaction(ctx, storage.WriteParams)
		args = append(args, rego.Store(store), rego.Transaction(txn))
	} else {
		args = append(args, RegoStoreReadAST)
	}

	return args, txn
}

var schemaResolvers = sync.OnceValue(func() (resolvers []func(*rego.Rego)) {
	ss := compile.RegalSchemaSet()
	added := util.NewSet[string]()

	// Find all schema references in the bundle and add the schemas to the base cache.
	for _, module := range rbundle.Loaded().Modules {
		for _, annos := range module.Parsed.Annotations {
			for _, s := range annos.Schemas {
				if len(s.Schema) != 0 && !added.Contains(s.Schema.String()) {
					resolvers = append(resolvers, rego.Resolver(
						ast.DefaultRootRef.Extend(s.Schema),
						schemaResolver{value: ast.MustInterfaceToValue(ss.Get(s.Schema))},
					))
					added.Add(s.Schema.String())
				}
			}
		}
	}

	return resolvers
})

// Eval implements the resolver.Resolver interface to resolve schemas from annotations at runtime.
func (sr schemaResolver) Eval(context.Context, resolver.Input) (resolver.Result, error) {
	return resolver.Result{Value: sr.value}, nil
}
