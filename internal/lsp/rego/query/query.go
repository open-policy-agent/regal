package query

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/resolver"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/compile"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/builtins"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

const (
	Keywords          = "data.regal.ast.keywords"
	RuleHeadLocations = "data.regal.ast.rule_head_locations"
	MainEval          = "data.regal.lsp.main.eval"
)

var simpleRefPattern = regexp.MustCompile(`^[a-zA-Z.]$`)

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
	parsedQuery := parseQuery(query)

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
		parsedQuery := parseQuery(query)

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
	args := make([]func(*rego.Rego), 0, 5+len(builtins.RegalBuiltinRegoFuncs))
	args = append(args, rego.ParsedQuery(query), rego.ParsedBundle("regal", rb))
	args = append(args, builtins.RegalBuiltinRegoFuncs...)

	// For debugging
	args = append(args, rego.EnablePrintStatements(true), rego.PrintHook(topdown.NewPrintHook(os.Stderr)))
	args = append(args, SchemaResolvers()...)

	var txn storage.Transaction
	if store != nil {
		txn, _ = store.NewTransaction(ctx, storage.WriteParams)
		args = append(args, rego.Store(store), rego.Transaction(txn))
	} else {
		args = append(args, rego.StoreReadAST(true))
	}

	return args, txn
}

func parseQuery(query string) ast.Body {
	if simpleRefPattern.MatchString(query) { // Try cheap parsing if possible
		return rast.RefStringToBody(query)
	}

	return ast.MustParseBody(query)
}

var schemaResolvers = sync.OnceValue(func() (resolvers []func(*rego.Rego)) {
	ss := compile.RegalSchemaSet()
	added := util.NewSet[string]()

	// Find all schema references in the bundle and add the schemas to the base cache.
	for _, module := range rbundle.Loaded().Modules {
		for _, annos := range module.Parsed.Annotations {
			for _, s := range annos.Schemas {
				if len(s.Schema) == 0 || added.Contains(s.Schema.String()) {
					continue
				}
				resolvers = append(resolvers, rego.Resolver(
					ast.DefaultRootRef.Extend(s.Schema),
					schemaResolver{value: ast.MustInterfaceToValue(ss.Get(s.Schema))},
				))
				added.Add(s.Schema.String())
			}
		}
	}

	return resolvers
})

// Eval implements the resolver.Resolver interface to resolve schemas from annotations at runtime.
func (sr schemaResolver) Eval(context.Context, resolver.Input) (resolver.Result, error) {
	return resolver.Result{Value: sr.value}, nil
}
