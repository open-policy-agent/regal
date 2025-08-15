package fileprovider

import (
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/rules"
)

type InMemoryFileProvider struct {
	files         map[string]string
	modifiedFiles *util.Set[string]
	deletedFiles  *util.Set[string]
}

func NewInMemoryFileProvider(files map[string]string) *InMemoryFileProvider {
	return &InMemoryFileProvider{
		files:         files,
		modifiedFiles: util.NewSet[string](),
		deletedFiles:  util.NewSet[string](),
	}
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

	return &InMemoryFileProvider{
		files:         files,
		modifiedFiles: util.NewSet[string](),
		deletedFiles:  util.NewSet[string](),
	}, nil
}

func (p *InMemoryFileProvider) List() ([]string, error) {
	files := make([]string, 0)
	for file := range p.files {
		files = append(files, file)
	}

	return files, nil
}

func (p *InMemoryFileProvider) Get(file string) (string, error) {
	content, ok := p.files[file]
	if !ok {
		return "", fmt.Errorf("file %s not found", file)
	}

	return content, nil
}

func (p *InMemoryFileProvider) Put(file string, content string) error {
	p.files[file] = content

	p.modifiedFiles.Add(file)

	return nil
}

func (p *InMemoryFileProvider) Rename(from, to string) error {
	content, ok := p.files[from]
	if !ok {
		return fmt.Errorf("file %s not found", from)
	}

	_, ok = p.files[to]
	if ok {
		return RenameConflictError{
			From: from,
			To:   to,
		}
	}

	if err := p.Put(to, content); err != nil {
		return fmt.Errorf("failed to put file %s: %w", to, err)
	}

	if err := p.Delete(from); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", from, err)
	}

	return nil
}

func (p *InMemoryFileProvider) Delete(file string) error {
	p.deletedFiles.Add(file)
	p.modifiedFiles.Remove(file)
	delete(p.files, file)

	return nil
}

func (p *InMemoryFileProvider) ModifiedFiles() []string {
	return p.modifiedFiles.Items()
}

func (p *InMemoryFileProvider) DeletedFiles() []string {
	return p.deletedFiles.Items()
}

func (p *InMemoryFileProvider) ToInput(versionsMap map[string]ast.RegoVersion) (rules.Input, error) {
	input, err := rules.InputFromMap(p.files, versionsMap)
	if err != nil {
		return rules.Input{}, fmt.Errorf("failed to create input: %w", err)
	}

	return input, nil
}
