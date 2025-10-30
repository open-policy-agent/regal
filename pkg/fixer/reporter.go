package fixer

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
)

// Reporter is responsible for outputting a fix report in a specific format.
type Reporter interface {
	Report(*Report) error
	SetDryRun(bool)
}

// ReporterForFormat returns a suitable Reporter for outputting a fix report in the given format.
func ReporterForFormat(format string, outputWriter io.Writer) (Reporter, error) {
	switch format {
	case "pretty":
		return NewPrettyReporter(outputWriter), nil
	default:
		return nil, fmt.Errorf("unsupported format %s", format)
	}
}

// PrettyReporter outputs a fix report in a human-readable format.
type PrettyReporter struct {
	outputWriter io.Writer
	dryRun       bool
}

func NewPrettyReporter(outputWriter io.Writer) *PrettyReporter {
	return &PrettyReporter{outputWriter: outputWriter}
}

func (r *PrettyReporter) SetDryRun(dryRun bool) {
	r.dryRun = dryRun
}

func (r *PrettyReporter) ReportConflicts(fixReport *Report) error {
	roots := util.KeysSorted(fixReport.conflictsSourceFile)

	if len(roots) > 0 {
		fmt.Fprintln(r.outputWriter, "Source file conflicts:")

		for i, rootKey := range roots {
			if i > 0 {
				fmt.Fprintln(r.outputWriter)
			}

			cs, ok := fixReport.conflictsSourceFile[rootKey]
			if !ok {
				continue
			}

			conflictingFiles := util.KeysSorted(cs)

			fmt.Fprintln(r.outputWriter, "In project root:", rootKey)

			for _, file := range conflictingFiles {
				conflicts := fixReport.conflictsSourceFile[rootKey][file]
				slices.Sort(conflicts)

				rootKeyPrefix := rootKey + string(filepath.Separator)

				fmt.Fprintln(r.outputWriter, "Cannot overwrite existing file:", strings.TrimPrefix(file, rootKeyPrefix))

				for _, oldPath := range conflicts {
					fmt.Fprintln(r.outputWriter, "-", strings.TrimPrefix(oldPath, rootKeyPrefix))
				}
			}
		}
	}

	roots = util.KeysSorted(fixReport.conflictsManyToOne)
	if len(roots) > 0 {
		if len(fixReport.conflictsSourceFile) > 0 {
			fmt.Fprintln(r.outputWriter)
		}

		fmt.Fprintln(r.outputWriter, "Many to one conflicts:")

		for i, rootKey := range roots {
			if i > 0 {
				fmt.Fprintln(r.outputWriter)
			}

			cs, ok := fixReport.conflictsManyToOne[rootKey]
			if !ok {
				continue
			}

			conflictingFiles := util.KeysSorted(cs)

			fmt.Fprintln(r.outputWriter, "In project root:", rootKey)

			rootKeyPrefix := rootKey + string(filepath.Separator)

			for _, file := range conflictingFiles {
				fmt.Fprintln(r.outputWriter, "Cannot move multiple files to:", strings.TrimPrefix(file, rootKeyPrefix))

				// get the old paths from the movedFiles since that includes all the files moved, not just the conflicting ones
				oldPaths := fixReport.movedFiles[file]
				slices.Sort(oldPaths)

				for _, oldPath := range oldPaths {
					fmt.Fprintln(r.outputWriter, "-", strings.TrimPrefix(oldPath, rootKeyPrefix))
				}
			}
		}
	}

	return nil
}

func (r *PrettyReporter) Report(fixReport *Report) error {
	action := "applied"
	if r.dryRun {
		action = "to apply"
	}

	if fixReport.HasConflicts() {
		return r.ReportConflicts(fixReport)
	}

	switch x := fixReport.TotalFixes(); x {
	case 0:
		fmt.Fprintf(r.outputWriter, "No fixes %s.\n", action)

		return nil
	case 1:
		fmt.Fprintf(r.outputWriter, "1 fix %s:\n", action)
	default:
		fmt.Fprintf(r.outputWriter, "%d fixes %s:\n", x, action)
	}

	byRoot := make(map[string]map[string][]fixes.FixResult)

	for file, fxs := range fixReport.fileFixes {
		for _, fix := range fxs {
			if _, ok := byRoot[fix.Root]; !ok {
				byRoot[fix.Root] = make(map[string][]fixes.FixResult)
			}

			byRoot[fix.Root][file] = append(byRoot[fix.Root][file], fix)
		}
	}

	i := 0

	for _, root := range util.KeysSorted(byRoot) {
		if i > 0 {
			fmt.Fprintln(r.outputWriter)
		}

		fixesByFile := byRoot[root]
		files := util.KeysSorted(fixesByFile)

		fmt.Fprintf(r.outputWriter, "In project root: %s\n", root)

		for _, file := range files {
			fxs := fixesByFile[file]

			rel := relOrDefault(root, file, file)

			oldPath, ok := fixReport.OldPathForFile(file)
			if ok {
				fmt.Fprintf(r.outputWriter, "%s -> %s:\n", relOrDefault(root, oldPath, oldPath), rel)
			} else {
				fmt.Fprintf(r.outputWriter, "%s:\n", rel)
			}

			for _, fix := range fxs {
				fmt.Fprintf(r.outputWriter, "- %s\n", fix.Title)
			}

			if len(files) > 3 {
				fmt.Fprintln(r.outputWriter, "")
			}
		}

		i++
	}

	return nil
}

func relOrDefault(root, path, defaultValue string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return defaultValue
	}

	return rel
}
