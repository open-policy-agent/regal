package fileprovider

import (
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/rules"
)

type InMemoryFileProvider struct {
	files    map[string]string
	modified *util.Set[string]
	deleted  *util.Set[string]
}

func NewInMemoryFileProvider(files map[string]string) *InMemoryFileProvider {
	return &InMemoryFileProvider{files: files, modified: util.NewSet[string](), deleted: util.NewSet[string]()}
}

func NewInMemoryFileProviderFromFS(paths ...string) (*InMemoryFileProvider, error) {
	files := make(map[string]string, len(paths))

	for _, path := range paths {
		fc, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		files[path] = string(fc)
	}

	return &InMemoryFileProvider{files: files, modified: util.NewSet[string](), deleted: util.NewSet[string]()}, nil
}

func (p *InMemoryFileProvider) List() ([]string, error) {
	return outil.Keys(p.files), nil
}

func (p *InMemoryFileProvider) Get(file string) (string, error) {
	content, ok := p.files[file]
	if !ok {
		return "", fmt.Errorf("file %s not found", file)
	}

	return content, nil
}

func (p *InMemoryFileProvider) Put(file, content string) error {
	p.files[file] = content
	p.modified.Add(file)

	return nil
}

func (p *InMemoryFileProvider) Rename(from, to string) error {
	content, ok := p.files[from]
	if !ok {
		return fmt.Errorf("file %s not found", from)
	}

	if _, ok = p.files[to]; ok {
		return RenameConflictError{From: from, To: to}
	}

	p.Put(to, content) //nolint:errcheck // always returns nil

	return p.Delete(from)
}

func (p *InMemoryFileProvider) Delete(file string) error {
	p.deleted.Add(file)
	p.modified.Remove(file)
	delete(p.files, file)

	return nil
}

func (p *InMemoryFileProvider) ModifiedFiles() []string {
	return p.modified.Items()
}

func (p *InMemoryFileProvider) DeletedFiles() []string {
	return p.deleted.Items()
}

func (p *InMemoryFileProvider) ToInput(versionsMap map[string]ast.RegoVersion) (rules.Input, error) {
	return util.Wrap(rules.InputFromMap(p.files, versionsMap))("failed to create input")
}
