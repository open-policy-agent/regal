package lsp

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

var (
	pathWorkspaceParsed      = storage.Path{"workspace", "parsed"}
	pathWorkspaceDefinedRefs = storage.Path{"workspace", "defined_refs"}
	pathWorkspaceBuiltins    = storage.Path{"workspace", "builtins"}
	pathWorkspaceConfig      = storage.Path{"workspace", "config"}
)

func NewRegalStore() storage.Store {
	return inmem.NewFromObjectWithOpts(map[string]any{
		"workspace": map[string]any{
			"parsed": map[string]any{},
			// should map[string][]string{}, but since we don't round trip on write,
			// we'll need to conform to the most basic "JSON" format understood by the store
			"defined_refs": map[string]any{},
			"builtins":     map[string]any{},
		},
	}, inmem.OptRoundTripOnWrite(false), inmem.OptReturnASTValuesOnRead(true))
}

func RemoveFileMod(ctx context.Context, store storage.Store, fileURI string) error {
	return transact(ctx, store, func(txn storage.Transaction) error {
		return remove(ctx, store, txn, append(pathWorkspaceParsed, fileURI))
	})
}

func PutFileRefs(ctx context.Context, store storage.Store, fileURI string, refs []string) error {
	return transact(ctx, store, func(txn storage.Transaction) error {
		return write(ctx, store, txn, append(pathWorkspaceDefinedRefs, fileURI), refs)
	})
}

func PutFileMod(ctx context.Context, store storage.Store, fileURI string, mod *ast.Module) error {
	return Put(ctx, store, append(pathWorkspaceParsed, fileURI), mod)
}

func PutBuiltins(ctx context.Context, store storage.Store, builtins map[string]*ast.Builtin) error {
	return Put(ctx, store, pathWorkspaceBuiltins, builtins)
}

func PutConfig(ctx context.Context, store storage.Store, config *config.Config) error {
	return Put(ctx, store, pathWorkspaceConfig, config)
}

func Put[T any](ctx context.Context, store storage.Store, path storage.Path, value T) error {
	return transact(ctx, store, func(txn storage.Transaction) error {
		asMap, err := encoding.JSONRoundTripTo[map[string]any](value)
		if err != nil {
			return fmt.Errorf("failed to marshal value to JSON: %w", err)
		}

		return write(ctx, store, txn, path, asMap)
	})
}

func write[T any](ctx context.Context, store storage.Store, txn storage.Transaction, path storage.Path, value T) error {
	var stErr *storage.Error

	err := store.Write(ctx, txn, storage.ReplaceOp, path, value)
	if errors.As(err, &stErr) && stErr.Code == storage.NotFoundErr {
		if err = store.Write(ctx, txn, storage.AddOp, path, value); err != nil {
			return fmt.Errorf("failed to add value at path %s in store: %w", path, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to replace value at path %s in store: %w", path, err)
	}

	return nil
}

func remove(ctx context.Context, store storage.Store, txn storage.Transaction, path storage.Path) error {
	var stErr *storage.Error

	err := store.Write(ctx, txn, storage.RemoveOp, path, nil)
	if errors.As(err, &stErr) && stErr.Code == storage.NotFoundErr {
		return nil // No-op if the path does not exist
	} else if err != nil {
		return fmt.Errorf("failed to remove value at path %s in store: %w", path, err)
	}

	return nil
}

func transact(ctx context.Context, store storage.Store, op func(txn storage.Transaction) error) error {
	txn, err := store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	success := false

	defer func() {
		if !success {
			store.Abort(ctx, txn)
		}
	}()

	if err := op(txn); err != nil {
		return err
	}

	if err := store.Commit(ctx, txn); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	success = true

	return nil
}
