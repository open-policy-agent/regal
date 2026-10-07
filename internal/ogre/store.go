package ogre

import (
	"context"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/topdown"

	"github.com/open-policy-agent/regal/internal/cache"
)

type Store struct {
	store     storage.Store
	baseCache topdown.BaseCache
}

func NewStore() *Store {
	return NewStoreFromObject(ast.NewObject())
}

func NewStoreFromObject(data ast.Object) *Store {
	baseCache := cache.NewBaseCache()
	baseCache.Put(ast.InternedEmptyRefValue.(ast.Ref), data) //nolint:forcetypeassert

	return &Store{store: inmem.NewFromASTObject(data), baseCache: baseCache}
}

func (s *Store) Storage() storage.Store {
	return s.store
}

func (s *Store) BaseCache() topdown.BaseCache {
	return s.baseCache
}

func (s *Store) ReadTransaction(ctx context.Context) storage.Transaction {
	txn, _ := s.store.NewTransaction(ctx)

	return txn
}
