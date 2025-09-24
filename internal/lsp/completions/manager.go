package completions

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/completions/providers"
	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/util"
)

type Manager struct {
	c      *cache.Cache
	policy *providers.Policy
}

func NewDefaultManager(ctx context.Context, c *cache.Cache, store storage.Store, qc *query.Cache) *Manager {
	return &Manager{c: c, policy: providers.NewPolicy(ctx, store, qc)}
}

func (m *Manager) Run(
	ctx context.Context,
	params types.CompletionParams,
	opts *providers.Options,
) ([]types.CompletionItem, error) {
	completions, err := m.policy.Run(ctx, m.c, params, opts)
	if err != nil {
		return nil, fmt.Errorf("error running completion provider: %w", err)
	}

	return util.Map(completions, removeMetadata), nil
}

func removeMetadata(item types.CompletionItem) types.CompletionItem {
	item.Regal = nil

	return item
}
