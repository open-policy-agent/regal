package lsp

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/roast/transforms/module"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

var (
	pathWorkspaceParsed      = storage.Path{"workspace", "parsed"}
	pathWorkspaceDefinedRefs = storage.Path{"workspace", "defined_refs"}
	pathWorkspaceBuiltins    = storage.Path{"workspace", "builtins"}
	pathWorkspaceConfig      = storage.Path{"workspace", "config"}
	pathClient               = storage.Path{"client"}
	pathServer               = storage.Path{"server"}
)

func NewRegalStore() storage.Store {
	return inmem.NewFromObjectWithOpts(map[string]any{
		"workspace": map[string]any{
			"config": map[string]any{},
			"parsed": map[string]any{},
			// should map[string][]string{}, but since we don't round trip on write,
			// we'll need to conform to the most basic "JSON" format understood by the store
			"defined_refs": map[string]any{},
			"builtins":     map[string]any{},
		},
		"client": map[string]any{},
		"server": map[string]any{},
	}, inmem.OptRoundTripOnWrite(false), inmem.OptReturnASTValuesOnRead(true))
}

func RemoveFileMod(ctx context.Context, store storage.Store, fileURI string) error {
	return storage.Txn(ctx, store, storage.WriteParams, func(txn storage.Transaction) error {
		return remove(ctx, store, txn, append(pathWorkspaceParsed, fileURI))
	})
}

func PutFileRefs(ctx context.Context, store storage.Store, fileURI string, refs []string) error {
	return storage.Txn(ctx, store, storage.WriteParams, func(txn storage.Transaction) error {
		return write(ctx, store, txn, append(pathWorkspaceDefinedRefs, fileURI), refs)
	})
}

func PutFileMod(ctx context.Context, store storage.Store, fileURI string, mod *ast.Module) error {
	value, err := module.ToValue(mod)
	if err != nil {
		return fmt.Errorf("failed to convert module to value: %w", err)
	}

	return Put(ctx, store, append(pathWorkspaceParsed, fileURI), value)
}

func PutBuiltins(ctx context.Context, store storage.Store, builtins map[string]*ast.Builtin) error {
	return Put(ctx, store, pathWorkspaceBuiltins, builtins)
}

func PutConfig(ctx context.Context, store storage.Store, config *config.Config) error {
	return Put(ctx, store, pathWorkspaceConfig, rast.StructToValue(config))
}

func PutClient(ctx context.Context, store storage.Store, client types.Client) error {
	return Put(ctx, store, pathClient, rast.StructToValue(client))
}

func PutServer(ctx context.Context, store storage.Store, server types.ServerContext) error {
	return Put(ctx, store, pathServer, rast.StructToValue(server))
}

func Put[T any](ctx context.Context, store storage.Store, path storage.Path, value T) error {
	return storage.Txn(ctx, store, storage.WriteParams, func(txn storage.Transaction) error {
		return write(ctx, store, txn, path, value)
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
