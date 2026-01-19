package cache

import (
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

// Cache is used to store: current file contents (which includes unsaved changes), the latest parsed modules, and
// diagnostics for each file (including diagnostics gathered from linting files alongside other files).
type Cache struct {
	// fileContents is a map of file URI to raw file contents received from the client
	fileContents *concurrent.Map[string, string]

	// ignoredFileContents is a similar map of file URI to raw file contents
	// but it's not queried for project level operations like goto definition,
	// linting etc.
	// ignoredFileContents is also cleared on the delete operation.
	ignoredFileContents *concurrent.Map[string, string]

	// modules is a map of file URI to parsed AST modules from the latest file contents value
	modules *concurrent.Map[string, *ast.Module]

	// aggregateData stores the aggregate data from evaluations for each file.
	// This is used to cache the results of expensive evaluations and can be used
	// to update aggregate diagnostics incrementally.
	aggregateData *concurrent.Object

	// diagnosticsFile is a map of file URI to diagnostics for that file
	diagnosticsFile *concurrent.Map[string, []types.Diagnostic]

	// diagnosticsParseErrors is a map of file URI to parse errors for that file
	diagnosticsParseErrors *concurrent.Map[string, []types.Diagnostic]

	// builtinPositionsFile is a map of file URI to builtin positions for that file
	builtinPositionsFile *concurrent.Map[string, map[uint][]types.BuiltinPosition]

	// keywordLocationsFile is a map of file URI to Rego keyword locations for that file
	// to be used for hover hints.
	keywordLocationsFile *concurrent.Map[string, map[uint][]types.KeywordLocation]

	// when a file is successfully parsed, the number of lines in the file is stored
	// here. This is used to gracefully fail when exiting unparsable files.
	successfulParseLineCounts *concurrent.Map[string, int]
}

func NewCache() *Cache {
	return &Cache{
		fileContents:              concurrent.MapOf(make(map[string]string)),
		ignoredFileContents:       concurrent.MapOf(make(map[string]string)),
		modules:                   concurrent.MapOf(make(map[string]*ast.Module)),
		diagnosticsFile:           concurrent.MapOf(make(map[string][]types.Diagnostic)),
		diagnosticsParseErrors:    concurrent.MapOf(make(map[string][]types.Diagnostic)),
		builtinPositionsFile:      concurrent.MapOf(make(map[string]map[uint][]types.BuiltinPosition)),
		keywordLocationsFile:      concurrent.MapOf(make(map[string]map[uint][]types.KeywordLocation)),
		successfulParseLineCounts: concurrent.MapOf(make(map[string]int)),
		aggregateData:             concurrent.NewObject(),
	}
}

func (c *Cache) GetAllFiles() map[string]string {
	return c.fileContents.Clone()
}

func (c *Cache) HasFileContents(fileURI string) bool {
	_, ok := c.fileContents.Get(fileURI)

	return ok
}

func (c *Cache) GetFileContents(fileURI string) (string, bool) {
	return c.fileContents.Get(fileURI)
}

func (c *Cache) SetFileContents(fileURI, content string) {
	c.fileContents.Set(fileURI, content)
}

func (c *Cache) GetIgnoredFileContents(fileURI string) (string, bool) {
	return c.ignoredFileContents.Get(fileURI)
}

func (c *Cache) SetIgnoredFileContents(fileURI, content string) {
	c.ignoredFileContents.Set(fileURI, content)
}

func (c *Cache) GetAllIgnoredFiles() map[string]string {
	return c.ignoredFileContents.Clone()
}

func (c *Cache) ClearIgnoredFileContents(fileURI string) {
	c.ignoredFileContents.Delete(fileURI)
}

func (c *Cache) GetAllModules() map[string]*ast.Module {
	return c.modules.Clone()
}

func (c *Cache) GetModule(fileURI string) (*ast.Module, bool) {
	return c.modules.Get(fileURI)
}

func (c *Cache) SetModule(fileURI string, module *ast.Module) {
	c.modules.Set(fileURI, module)
}

func (c *Cache) GetContentAndModule(fileURI string) (string, *ast.Module, bool) {
	content, ok := c.GetFileContents(fileURI)
	if !ok {
		return "", nil, false
	}

	module, ok := c.GetModule(fileURI)
	if !ok {
		return "", nil, false
	}

	return content, module, true
}

func (c *Cache) Rename(oldKey, newKey string) {
	c.fileContents.RenameKey(oldKey, newKey)
	c.ignoredFileContents.RenameKey(oldKey, newKey)
	c.modules.RenameKey(oldKey, newKey)
	c.aggregateData.RenameKey(oldKey, newKey)
	c.diagnosticsFile.RenameKey(oldKey, newKey)
	c.diagnosticsParseErrors.RenameKey(oldKey, newKey)
	c.builtinPositionsFile.RenameKey(oldKey, newKey)
	c.keywordLocationsFile.RenameKey(oldKey, newKey)
	c.successfulParseLineCounts.RenameKey(oldKey, newKey)
}

func (c *Cache) SetAggregates(aggregates ast.Object) {
	c.aggregateData.Reset(aggregates)
}

// SetFileAggregates sets aggregate data for the provided URI.
func (c *Cache) SetFileAggregates(fileURI string, data ast.Object) {
	if data != nil {
		c.aggregateData.Set(fileURI, ast.NewTerm(data))
	}
}

// GetFileAggregates is used to get aggregate data for a given list of files, or
// all files if no file URIs are provided. Note that the returned object includes
// also the provided file URIs as keys.. i.e. not just the values.
func (c *Cache) GetFileAggregates(fileURIs ...string) ast.Object {
	if len(fileURIs) == 0 {
		return c.aggregateData.UnsafeObject()
	}

	return c.aggregateData.Keep(fileURIs...)
}

func (c *Cache) GetFileDiagnostics(uri string) ([]types.Diagnostic, bool) {
	return c.diagnosticsFile.Get(uri)
}

func (c *Cache) SetFileDiagnostics(fileURI string, diags []types.Diagnostic) {
	c.diagnosticsFile.Set(fileURI, diags)
}

// SetFileDiagnosticsForRules will perform a partial update of the diagnostics
// for a file given a list of evaluated rules.
func (c *Cache) SetFileDiagnosticsForRules(fileURI string, rules []string, diags []types.Diagnostic) {
	c.diagnosticsFile.UpdateValue(fileURI, func(current []types.Diagnostic) []types.Diagnostic {
		ruleKeys := util.NewSet(rules...)
		preservedDiagnostics := make([]types.Diagnostic, 0, len(current))

		for i := range current {
			if !ruleKeys.Contains(current[i].Code) {
				preservedDiagnostics = append(preservedDiagnostics, current[i])
			}
		}

		return append(preservedDiagnostics, diags...)
	})
}

func (c *Cache) ClearFileDiagnostics() {
	c.diagnosticsFile.Clear()
}

func (c *Cache) GetParseErrors(uri string) ([]types.Diagnostic, bool) {
	return c.diagnosticsParseErrors.Get(uri)
}

func (c *Cache) SetParseErrors(fileURI string, diags []types.Diagnostic) {
	c.diagnosticsParseErrors.Set(fileURI, diags)
}

func (c *Cache) GetBuiltinPositions(fileURI string) (map[uint][]types.BuiltinPosition, bool) {
	return c.builtinPositionsFile.Get(fileURI)
}

func (c *Cache) SetBuiltinPositions(fileURI string, positions map[uint][]types.BuiltinPosition) {
	c.builtinPositionsFile.Set(fileURI, positions)
}

func (c *Cache) GetAllBuiltInPositions() map[string]map[uint][]types.BuiltinPosition {
	return c.builtinPositionsFile.Clone()
}

func (c *Cache) SetKeywordLocations(fileURI string, keywords map[uint][]types.KeywordLocation) {
	c.keywordLocationsFile.Set(fileURI, keywords)
}

func (c *Cache) GetKeywordLocations(fileURI string) (map[uint][]types.KeywordLocation, bool) {
	return c.keywordLocationsFile.Get(fileURI)
}

func (c *Cache) GetSuccessfulParseLineCount(fileURI string) (int, bool) {
	return c.successfulParseLineCounts.Get(fileURI)
}

func (c *Cache) SetSuccessfulParseLineCount(fileURI string, count int) {
	c.successfulParseLineCounts.Set(fileURI, count)
}

// Delete removes all cached data for a given URI. Ignored file contents are
// also removed if found for a matching URI.
func (c *Cache) Delete(fileURI string) {
	c.fileContents.Delete(fileURI)
	c.ignoredFileContents.Delete(fileURI)
	c.modules.Delete(fileURI)
	c.aggregateData.Delete(fileURI)
	c.diagnosticsFile.Delete(fileURI)
	c.diagnosticsParseErrors.Delete(fileURI)
	c.builtinPositionsFile.Delete(fileURI)
	c.keywordLocationsFile.Delete(fileURI)
	c.successfulParseLineCounts.Delete(fileURI)
}

func (c *Cache) UpdateForURIFromDisk(fileURI, path string) (bool, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, "", fmt.Errorf("failed to read file: %w", err)
	}

	currentContent := outil.ByteSliceToString(content)

	cachedContent, ok := c.GetFileContents(fileURI)
	if ok && cachedContent == currentContent {
		return false, cachedContent, nil
	}

	c.SetFileContents(fileURI, currentContent)

	return true, currentContent, nil
}
