package completions

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/completions/providers"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

type Manager struct {
	c         *cache.Cache
	opts      *ManagerOptions
	providers []Provider
}

type ManagerOptions struct{}

type Provider interface {
	Run(context.Context, *cache.Cache, types.CompletionParams, *providers.Options) ([]types.CompletionItem, error)
	Name() string
}

func NewManager(c *cache.Cache, opts *ManagerOptions) *Manager {
	return &Manager{c: c, opts: opts}
}

func NewDefaultManager(ctx context.Context, c *cache.Cache, store storage.Store) *Manager {
	m := NewManager(c, &ManagerOptions{})

	m.RegisterProvider(&providers.BuiltIns{})
	m.RegisterProvider(&providers.PackageRefs{})
	m.RegisterProvider(providers.NewPolicy(ctx, store))

	return m
}

func (m *Manager) Run(
	ctx context.Context,
	params types.CompletionParams,
	opts *providers.Options,
) ([]types.CompletionItem, error) {
	if m.isInsideOfComment(params) {
		// Exit early if caret position is inside a comment. We currently don't have any provider
		// where doing completions inside of a comment makes much sense. Behavior is also editor-specific:
		// - Zed: always on, with no way to disable
		// - VSCode: disabled but can be enabled with "editor.quickSuggestions.comments" setting
		return []types.CompletionItem{}, nil
	}

	var completionsList []types.CompletionItem

	for _, provider := range m.providers {
		providerCompletions, err := provider.Run(ctx, m.c, params, opts)
		if err != nil {
			return nil, fmt.Errorf("error running completion provider: %w", err)
		}

		for _, completion := range providerCompletions {
			completion.Regal = nil
			completionsList = append(completionsList, completion)
		}
	}

	return completionsList, nil
}

func (m *Manager) RegisterProvider(provider Provider) {
	m.providers = append(m.providers, provider)
}

func (m *Manager) isInsideOfComment(params types.CompletionParams) bool {
	if module, ok := m.c.GetModule(params.TextDocument.URI); ok {
		for _, comment := range module.Comments {
			cp := rego.PositionFromLocation(comment.Location)

			if cp.Line == params.Position.Line {
				if cp.Character <= params.Position.Character {
					return true
				}
			}
		}
	}

	return false
}
