package fileprovider

import (
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/rules"
)

type CacheFileProvider struct {
	Cache            *cache.Cache
	ClientIdentifier clients.Identifier

	modifiedFiles *util.Set[string]
	deletedFiles  *util.Set[string]

	toPath func(string) string
}

func NewCacheFileProvider(c *cache.Cache, ci clients.Identifier) *CacheFileProvider {
	return &CacheFileProvider{
		Cache:            c,
		ClientIdentifier: ci,
		modifiedFiles:    util.NewSet[string](),
		deletedFiles:     util.NewSet[string](),
		toPath:           uri.ToPath,
	}
}

func (c *CacheFileProvider) List() ([]string, error) {
	return util.MapKeys(c.Cache.GetAllFiles(), c.toPath), nil
}

func (c *CacheFileProvider) Get(file string) (string, error) {
	contents, ok := c.Cache.GetFileContents(uri.FromPath(c.ClientIdentifier, file))
	if !ok {
		return "", fmt.Errorf("failed to get file %s", file)
	}

	return contents, nil
}

func (c *CacheFileProvider) Put(file, content string) error {
	c.Cache.SetFileContents(file, content)

	return nil
}

func (c *CacheFileProvider) Delete(file string) error {
	c.Cache.Delete(uri.FromPath(c.ClientIdentifier, file))

	return nil
}

func (c *CacheFileProvider) Rename(from, to string) error {
	fromURI := uri.FromPath(c.ClientIdentifier, from)
	toURI := uri.FromPath(c.ClientIdentifier, to)

	content, ok := c.Cache.GetFileContents(fromURI)
	if !ok {
		return fmt.Errorf("file %s not found", from)
	}

	if ok := c.Cache.HasFileContents(toURI); ok {
		return RenameConflictError{From: from, To: to}
	}

	c.Cache.SetFileContents(toURI, content)
	c.modifiedFiles.Add(to)
	c.Cache.Delete(fromURI)
	c.modifiedFiles.Remove(from)
	c.deletedFiles.Add(from)

	return nil
}

func (c *CacheFileProvider) ToInput(versionsMap map[string]ast.RegoVersion) (rules.Input, error) {
	return util.Wrap(rules.InputFromMap(c.Cache.GetAllFiles(), versionsMap))("failed to create input")
}
