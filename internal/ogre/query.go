package ogre

import (
	"context"
	"maps"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"
	"github.com/open-policy-agent/opa/v1/topdown/print"

	rbundle "github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/io"
)

type Query struct {
	query           ast.Body
	modules         map[string]*ast.Module
	compiler        *ast.Compiler
	store           *Store
	metrics         metrics.Metrics
	printHook       print.Hook
	instrumentation *topdown.Instrumentation
	txn             storage.Transaction
}

func New(query ast.Body) *Query {
	q := &Query{query: query}

	// The modules provided by bundle.Loaded() are typically those embedded in the Regal binary,
	// but could also be sourced from the file system when development mode is enabled.
	q.modules = modulesFromBundle(rbundle.Loaded().Modules)

	return q
}

func (q *Query) Evaluator() *Evaluator {
	return &Evaluator{prepared: q, txn: q.txn}
}

func (q *Query) WithStore(store *Store) *Query {
	q.store = store

	return q
}

func (q *Query) WithModules(modules map[string]*ast.Module) *Query {
	maps.Copy(q.modules, modules)

	return q
}

func (q *Query) WithMetrics(m metrics.Metrics) *Query {
	q.metrics = m

	// In case instrumentation was enabled before metrics were set,
	// recreate the instrumentation with the new metrics.
	if q.instrumentation != nil {
		q.instrumentation = topdown.NewInstrumentation(q.metrics)
	}

	return q
}

func (q *Query) Metrics() metrics.Metrics {
	return q.metrics
}

func (q *Query) Compiler() *ast.Compiler {
	return q.compiler
}

func (q *Query) Store() *Store {
	return q.store
}

func (q *Query) WithInstrumentation(enabled bool) *Query {
	if !enabled {
		q.instrumentation = nil

		return q
	}

	if q.metrics == nil {
		q.metrics = metrics.New()
	}

	q.instrumentation = topdown.NewInstrumentation(q.metrics)

	return q
}

func (q *Query) WithPrintHook(hook print.Hook) *Query {
	q.printHook = hook

	return q
}

func (q *Query) StartReadTransaction(ctx context.Context) *Query {
	q.txn = q.store.ReadTransaction(ctx)

	return q
}

func (q *Query) EndReadTransaction(ctx context.Context) *Query {
	if q.txn != nil {
		q.store.Storage().Abort(ctx, q.txn)
		q.txn = nil
	}

	return q
}

func (q *Query) Prepare(_ context.Context) (*Query, error) {
	if q.metrics == nil {
		q.metrics = metrics.NoOp()
	}

	// Compile all modules
	q.compiler = q.newCompiler()

	if q.compiler.Compile(q.modules); q.compiler.Failed() {
		if rbundle.DevModeEnabled() {
			// In dev mode, try compiling again with the embedded bundle modules only
			q.modules = modulesFromBundle(rbundle.Embedded().Modules)
			q.compiler = q.newCompiler()

			if q.compiler.Compile(q.modules); !q.compiler.Failed() {
				return q, nil
			}
		}

		return nil, q.compiler.Errors
	}

	if q.store == nil {
		q.store = NewStore()
	}

	return q, nil
}

func (q *Query) newCompiler() *ast.Compiler {
	return ast.NewCompiler().
		WithCapabilities(io.Capabilities()).
		WithKeepModules(true).
		WithMetrics(q.metrics).
		WithEnablePrintStatements(q.printHook != nil).
		WithDefaultRegoVersion(ast.RegoV1)
}

func modulesFromBundle(files []bundle.ModuleFile) map[string]*ast.Module {
	m := make(map[string]*ast.Module, len(files))
	for i := range files {
		m[files[i].Path] = files[i].Parsed
	}

	return m
}
