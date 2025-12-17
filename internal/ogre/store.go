package ogre

import (
	"context"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/topdown"

	"github.com/open-policy-agent/regal/internal/cache"
	"github.com/open-policy-agent/regal/pkg/roast/intern"
)

type Store struct {
	store     storage.Store
	baseCache topdown.BaseCache
}

func NewStore() *Store {
	return &Store{
		store:     inmem.NewWithOpts(inmem.OptReturnASTValuesOnRead(true)),
		baseCache: cache.NewBaseCache(),
	}
}

func NewStoreFromObject(ctx context.Context, data ast.Object) *Store {
	s := NewStore()

	if err := storage.WriteOne(ctx, s.store, storage.AddOp, storage.RootPath, data); err != nil {
		panic(err)
	}

	s.baseCache.Put(intern.EmptyRef, data)

	return s
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
