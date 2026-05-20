package input

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/store"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

type (
	// Manager manages input files in the workspace, allowing for fast retrieval of the most specific input for a given
	// path, whether in Go (via [*Manager.FindForPath] and [*Manager.Get]) or in Rego (via `data.workspace.inputs`).
	Manager struct {
		rootPath string
		rootFS   fs.FS
		store    storage.Store
		inputs   map[string]file
		log      *log.Logger
		mut      sync.RWMutex
	}
	file struct {
		dir  string
		raw  []byte
		path storage.Path
	}
)

func NewManager(store storage.Store, log *log.Logger) *Manager {
	return &Manager{
		store:  store,
		inputs: make(map[string]file),
		log:    log,
		mut:    sync.RWMutex{},
	}
}

func (m *Manager) LoadFromRoot(ctx context.Context, rootPath string) error {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return fmt.Errorf("failed to open workspace root: %w", err)
	}

	return m.LoadFromFS(ctx, rootPath, root.FS())
}

func (m *Manager) LoadFromFS(ctx context.Context, rootPath string, rootFS fs.FS) error {
	m.mut.Lock()
	m.rootPath = rootPath
	m.rootFS = rootFS
	m.mut.Unlock()

	return files.DefaultWalker(".").
		WithFilters(filter.Not(filter.Suffixes("input.json", "input.yaml"))).
		WalkFS(rootFS, func(path string) error {
			return m.Update(ctx, path, nil)
		})
}

// FindForPath returns the most specific input path for the given path
// relative to the workspace root, or an empty string if no input file is found.
func (m *Manager) FindForPath(pathOrURI string) string {
	path := m.internalPath(pathOrURI)

	m.mut.RLock()
	defer m.mut.RUnlock()

	var closestDir, closestPath string
	for inputPath, file := range m.inputs {
		if strings.HasPrefix(path, file.dir) && len(file.dir) >= len(closestDir) {
			closestDir, closestPath = file.dir, inputPath
		}
	}

	return closestPath
}

// Get returns the input value for the given input path, as retrieved by FindForPath.
func (m *Manager) Get(ctx context.Context, inputPath string) ast.Value {
	inputPath = strings.Trim(filepath.ToSlash(inputPath), "/")

	m.mut.RLock()
	defer m.mut.RUnlock()

	if file, exists := m.inputs[inputPath]; exists {
		if inputAny, err := storage.ReadOne(ctx, m.store, file.path); err == nil {
			if inputValue, ok := inputAny.(ast.Value); ok {
				return inputValue
			}
		}
	}

	return ast.InternedEmptyObjectValue
}

// Update updates the input value for the given path or URI, caching it in the store for retrieval with Get.
// If content is nil, it will attempt to load it from disk.
func (m *Manager) Update(ctx context.Context, pathOrURI string, content []byte) (err error) {
	path := m.internalPath(pathOrURI)

	m.mut.Lock()
	defer m.mut.Unlock()

	if content == nil {
		if content, err = fs.ReadFile(m.rootFS, path); err != nil {
			return err
		}

		if len(content) == 0 {
			content = append(content, '{', '}') // file likely just created
		}
	}

	if curr, exists := m.inputs[path]; exists && bytes.Equal(curr.raw, content) {
		return nil
	}

	var val ast.Value

	suffix := path[strings.LastIndexByte(path, '.')+1:]
	switch suffix {
	case "json":
		val, err = encoding.OfValue().Decode(content)
	case "yaml":
		var res map[string]any
		if err = yaml.Unmarshal(content, &res); err == nil {
			val, err = transform.AnyToValue(res)
		}
	}

	if err == nil {
		storePath := m.storagePathFor(path)
		if err = store.Put(ctx, m.store, storePath, val); err == nil {
			dir := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(path, suffix), "input."), "/")

			m.inputs[path] = file{raw: content, path: storePath, dir: dir}
		}
	}

	return err
}

func (m *Manager) Delete(ctx context.Context, pathOrURI string) error {
	path := m.internalPath(pathOrURI)

	m.mut.Lock()
	defer m.mut.Unlock()

	err := store.Remove(ctx, m.store, m.storagePathFor(path))
	delete(m.inputs, path)

	return err
}

func (*Manager) HasInputSuffix(path string) bool {
	return util.HasAnySuffix(path, "input.json", "input.yaml")
}

// NOTE: must be called with m.mut held!
func (m *Manager) storagePathFor(path string) storage.Path {
	if existing, ok := m.inputs[path]; ok {
		return existing.path
	}

	return storage.Path{"workspace", "inputs", path}
}

func (m *Manager) internalPath(pathOrURI string) string {
	m.mut.RLock()
	rootPath := m.rootPath
	m.mut.RUnlock()

	return strings.TrimPrefix(strings.TrimPrefix(uri.ToPath(pathOrURI), rootPath), "/")
}
