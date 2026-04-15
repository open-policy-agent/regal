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
	pathWorkspaceAggregates  = storage.Path{"workspace", "aggregates"}
)

func NewRegalStore() storage.Store {
	return inmem.NewFromObjectWithOpts(map[string]any{
		"workspace": map[string]any{
			"parsed": map[string]any{},
			// should map[string][]string{}, but since we don't round trip on write,
			// we'll need to conform to the most basic "JSON" format understood by the store
			"defined_refs": map[string]any{},
			"builtins":     map[string]any{},
			"aggregates":   map[string]any{},
		},
		// Pre-initialize paths for linter when using shared store (via WithBaseStore).
		// Allows linter to write to nested paths like {"internal", "prepared"} without NotFoundErr.
		"internal": map[string]any{},
		"eval":     map[string]any{},
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
	return Put(ctx, store, append(pathWorkspaceParsed, fileURI), mod)
}

func PutBuiltins(ctx context.Context, store storage.Store, builtins map[string]*ast.Builtin) error {
	return Put(ctx, store, pathWorkspaceBuiltins, builtins)
}

func PutConfig(ctx context.Context, store storage.Store, config *config.Config) error {
	return Put(ctx, store, pathWorkspaceConfig, config)
}

func Put[T any](ctx context.Context, store storage.Store, path storage.Path, value T) error {
	return storage.Txn(ctx, store, storage.WriteParams, func(txn storage.Transaction) error {
		// TODO: Since we use the AST backend, we should not have to round trip via JSON here.
		asMap, err := encoding.JSONRoundTripTo[map[string]any](value)
		if err != nil {
			return fmt.Errorf("failed to marshal value to JSON: %w", err)
		}

		return write(ctx, store, txn, path, asMap)
	})
}

// PutAST stores an AST value directly without JSON round-tripping.
// Implements copy-on-write by calling Copy() on AST types before storing.
func PutAST[T any](ctx context.Context, store storage.Store, path storage.Path, value T) error {
	return storage.Txn(ctx, store, storage.WriteParams, func(txn storage.Transaction) error {
		// Implement copy-on-write for AST types to avoid TOCTOU issues.
		// When aggregates are extracted from the store and later modified,
		// we need to ensure the stored copy is not affected.
		var valueToPut any

		switch v := any(value).(type) {
		case ast.Object:
			valueToPut = v.Copy()
		default:
			valueToPut = value
		}

		return write(ctx, store, txn, path, valueToPut)
	})
}

// GetAST reads an AST value directly from the store without JSON unmarshaling.
// If the path is not found, returns the zero value for T without error.
// Use this for runtime AST values like ast.Object that were stored with PutAST.
func GetAST[T any](ctx context.Context, store storage.Store, path storage.Path) (T, error) {
	var result T

	err := storage.Txn(ctx, store, storage.TransactionParams{}, func(txn storage.Transaction) error {
		value, err := store.Read(ctx, txn, path)
		if err != nil {
			var stErr *storage.Error
			if errors.As(err, &stErr) && stErr.Code == storage.NotFoundErr {
				return nil // Return zero value without error
			}

			return fmt.Errorf("failed to read from store: %w", err)
		}

		typed, ok := value.(T)
		if !ok {
			var zero T

			return fmt.Errorf("expected %T, got %T", zero, value)
		}

		result = typed

		return nil
	})

	return result, err
}

func write[T any](ctx context.Context, store storage.Store, txn storage.Transaction, path storage.Path, value T) error {
	var stErr *storage.Error

	// Ensure all intermediate path segments exist before writing the leaf.
	// Parent paths can be missing at startup or after a store reset.
	for i := 1; i < len(path); i++ {
		prefix := path[:i]
		if _, err := store.Read(ctx, txn, prefix); err != nil {
			if !errors.As(err, &stErr) || stErr.Code != storage.NotFoundErr {
				return fmt.Errorf("failed to read intermediate path %s in store: %w", prefix, err)
			}

			if err = store.Write(ctx, txn, storage.AddOp, prefix, map[string]any{}); err != nil {
				return fmt.Errorf("failed to create intermediate path %s in store: %w", prefix, err)
			}
		}
	}

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

// PutFileAggregates stores aggregate data for a specific file URI.
// The data parameter should be an ast.Object containing the aggregates for that file.
func PutFileAggregates(ctx context.Context, store storage.Store, fileURI string, data ast.Object) error {
	return PutAST(ctx, store, append(pathWorkspaceAggregates, fileURI), data)
}

// PutAllAggregates replaces all aggregate data in the store.
// The aggregates parameter should be an ast.Object with file URIs as keys.
func PutAllAggregates(ctx context.Context, store storage.Store, aggregates ast.Object) error {
	// In such cases, this makes it possible for per file updates in future.
	if aggregates == nil || aggregates.Len() == 0 {
		return PutAST(ctx, store, pathWorkspaceAggregates, map[string]any{})
	}

	return PutAST(ctx, store, pathWorkspaceAggregates, aggregates)
}

// RemoveFileAggregates removes aggregate data for a specific file URI.
func RemoveFileAggregates(ctx context.Context, store storage.Store, fileURI string) error {
	return storage.Txn(ctx, store, storage.WriteParams, func(txn storage.Transaction) error {
		return remove(ctx, store, txn, append(pathWorkspaceAggregates, fileURI))
	})
}
