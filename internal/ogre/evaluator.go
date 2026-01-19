// Package ogre is the Regal equivalence of OPA's rego package, providing a simplified
// interface for preparing and evaluating Rego queries in the context of Regal and its
// embedded policies and data.
package ogre

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/profiler"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/topdown"

	_ "github.com/open-policy-agent/regal/pkg/builtins"
)

var errNoResultHandler = errors.New("result handler must be provided")

type Evaluator struct {
	prepared      *Query
	input         ast.Value
	outputHandler func(result topdown.QueryResult) error
	profiler      *profiler.Profiler
	txn           storage.Transaction
}

type qcWrapper struct {
	*profiler.Profiler
}

func (w qcWrapper) Enabled() bool {
	return w.Profiler != nil
}

func (e *Evaluator) WithInput(input ast.Value) *Evaluator {
	e.input = input

	return e
}

func (e *Evaluator) WithProfiler(p *profiler.Profiler) *Evaluator {
	e.profiler = p

	return e
}

func (e *Evaluator) WithTransaction(txn storage.Transaction) *Evaluator {
	e.txn = txn

	return e
}

func (e *Evaluator) Profiler() *profiler.Profiler {
	return e.profiler
}

func (e *Evaluator) WithResultHandler(f func(ast.Value) error) *Evaluator {
	// Since we expect a single output binding, extract the bound variable from the query
	// and create an output handler that maps to the provided function. This is merely a
	// convenience to avoid topdown.QueryResult boilerplate code at every call site.
	if terms, ok := e.prepared.query[0].Terms.([]*ast.Term); ok {
		if bound, ok := terms[1].Value.(ast.Var); ok {
			e.outputHandler = func(r topdown.QueryResult) error {
				if output, ok := r[bound]; ok {
					return f(output.Value)
				}

				return fmt.Errorf("expected variable %q in result", bound)
			}
		}
	}

	return e
}

func (e *Evaluator) Eval(ctx context.Context) (err error) {
	if e.outputHandler == nil {
		return errNoResultHandler
	}

	store := e.prepared.store.store
	txnProvided := e.txn != nil

	txn := e.txn
	if !txnProvided {
		if txn, err = store.NewTransaction(ctx); err != nil {
			return err
		}
	}

	inputTerm := ast.TermPtrPool.Get()
	inputTerm.Value = cmp.Or(e.input, ast.InternedEmptyObject.Value)

	q := topdown.NewQuery(e.prepared.query).
		WithCompiler(e.prepared.compiler).
		WithMetrics(e.prepared.metrics).
		WithInstrumentation(e.prepared.instrumentation).
		WithPrintHook(e.prepared.printHook).
		WithStore(store).
		WithTransaction(txn).
		WithBaseCache(e.prepared.store.BaseCache()).
		WithQueryTracer(qcWrapper{Profiler: e.profiler}).
		WithInput(inputTerm)

	err = q.Iter(ctx, e.outputHandler)

	inputTerm.Value = nil
	ast.TermPtrPool.Put(inputTerm)

	if !txnProvided {
		store.Abort(ctx, txn)
	}

	return err
}
