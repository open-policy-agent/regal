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

var errNoResultHandler = errors.New("no result handler provided")

type Evaluator struct {
	txn           storage.Transaction
	input         ast.Value
	resultHandler ResultHandler
	profiler      *profiler.Profiler
	prepared      *Query
	id            int64
}

func (e *Evaluator) WithID(id int64) *Evaluator {
	e.id = id

	return e
}

func (e *Evaluator) ID() int64 {
	return e.id
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

func (e *Evaluator) WithResultHandler(r ResultHandler) *Evaluator {
	e.resultHandler = r

	return e
}

func (e *Evaluator) Eval(ctx context.Context) (err error) {
	if e.resultHandler == nil {
		return errNoResultHandler
	}

	if e.prepared == nil {
		return errors.New("query must be prepared before eval")
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
		WithQueryTracer(e.profiler).
		WithInput(inputTerm)

	err = q.Iter(ctx, e.handleResult)

	inputTerm.Value = nil
	ast.TermPtrPool.Put(inputTerm)

	if !txnProvided {
		store.Abort(ctx, txn)
	}

	return err
}

func (e *Evaluator) handleResult(qr topdown.QueryResult) error {
	if e.resultHandler == nil {
		return errNoResultHandler
	}

	// Since we expect a single output binding, extract the bound variable from the query
	// and create an output handler that maps to the provided function. This is merely a
	// convenience to avoid topdown.QueryResult boilerplate code at every call site.
	if terms, ok := e.prepared.query[0].Terms.([]*ast.Term); ok {
		if bound, ok := terms[1].Value.(ast.Var); ok {
			if output, ok := qr[bound]; ok {
				return e.resultHandler.Handle(Result{Evaluator: e, Value: output.Value})
			}

			return fmt.Errorf("expected variable %q in result", bound)
		}
	}

	return fmt.Errorf("unsupported query format: %v", e.prepared.query)
}
