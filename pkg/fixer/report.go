package fixer

import (
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
)

// Report contains updated file contents and summary information about the fixes that were applied
// during a fix operation.
type Report struct {
	fileFixes           map[string][]fixes.FixResult
	movedFiles          map[string][]string
	conflictsManyToOne  map[string]map[string][]string
	conflictsSourceFile map[string]map[string][]string
	totalFixes          uint
}

func NewReport() *Report {
	return &Report{
		fileFixes:           make(map[string][]fixes.FixResult),
		movedFiles:          make(map[string][]string),
		conflictsManyToOne:  make(map[string]map[string][]string),
		conflictsSourceFile: make(map[string]map[string][]string),
	}
}

func (r *Report) AddFileFix(file string, fix fixes.FixResult) {
	r.fileFixes[file] = append(r.fileFixes[file], fix)
	r.totalFixes++
}

func (r *Report) FixesForFile(file string) []fixes.FixResult {
	return r.fileFixes[file]
}

func (r *Report) MergeFixes(path1, path2 string) {
	r.fileFixes[path1] = append(r.FixesForFile(path1), r.FixesForFile(path2)...)
	delete(r.fileFixes, path2)
}

func (r *Report) RegisterOldPathForFile(newPath, oldPath string) {
	r.movedFiles[newPath] = append(r.movedFiles[newPath], oldPath)
}

func (r *Report) OldPathForFile(newPath string) (string, bool) {
	oldPaths, ok := r.movedFiles[newPath]

	if !ok || len(oldPaths) == 0 {
		return "", false
	}

	return oldPaths[0], true
}

func (r *Report) FixedFiles() []string {
	return util.KeysSorted(r.fileFixes)
}

func (r *Report) TotalFixes() uint {
	// totalFixes is incremented for each unique violation that is fixed
	return r.totalFixes
}

func (r *Report) RegisterConflictManyToOne(root, newPath, oldPath string) {
	if _, ok := r.conflictsManyToOne[root]; !ok {
		r.conflictsManyToOne[root] = make(map[string][]string)
	}

	if _, ok := r.conflictsManyToOne[root][newPath]; !ok {
		r.conflictsManyToOne[root][newPath] = make([]string, 0)
	}

	r.conflictsManyToOne[root][newPath] = append(r.conflictsManyToOne[root][newPath], oldPath)
}

func (r *Report) RegisterConflictSourceFile(root, newPath, oldPath string) {
	if _, ok := r.conflictsSourceFile[root]; !ok {
		r.conflictsSourceFile[root] = make(map[string][]string)
	}

	if _, ok := r.conflictsSourceFile[root][newPath]; !ok {
		r.conflictsSourceFile[root][newPath] = make([]string, 0)
	}

	r.conflictsSourceFile[root][newPath] = append(r.conflictsSourceFile[root][newPath], oldPath)
}

func (r *Report) HasConflicts() bool {
	return len(r.conflictsManyToOne) > 0 || len(r.conflictsSourceFile) > 0
}
