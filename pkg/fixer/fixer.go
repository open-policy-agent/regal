package fixer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/fixer/fileprovider"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/report"
)

type OnConflictOperation string

const (
	OnConflictError  OnConflictOperation = "error"
	OnConflictRename OnConflictOperation = "rename"
)

// Fixer must be instantiated via NewFixer.
type Fixer struct {
	registeredFixes     map[string]any
	onConflictOperation OnConflictOperation
	registeredRoots     []string
	versionsMap         map[string]ast.RegoVersion
}

// NewFixer instantiates a Fixer.
func NewFixer() *Fixer {
	return &Fixer{
		registeredFixes:     make(map[string]any),
		registeredRoots:     make([]string, 0),
		onConflictOperation: OnConflictError,
	}
}

// SetOnConflictOperation sets the fixer's behavior when a conflict occurs.
func (f *Fixer) SetOnConflictOperation(operation OnConflictOperation) *Fixer {
	f.onConflictOperation = operation

	return f
}

// SetRegoVersionsMap sets the mapping of path prefixes to versions for the
// fixer to use when creating input for fixer runs.
func (f *Fixer) SetRegoVersionsMap(versionsMap map[string]ast.RegoVersion) *Fixer {
	f.versionsMap = versionsMap

	return f
}

// RegisterFixes sets the fixes that will be fixed if there are related linter
// violations that can be fixed by fixes.
func (f *Fixer) RegisterFixes(fixes ...fixes.Fix) *Fixer {
	for _, fix := range fixes {
		f.registeredFixes[fix.Name()] = fix
	}

	return f
}

// RegisterRoots sets the roots of the files that will be fixed.
// Certain fixes may require the nearest root of the file to be known,
// as fix operations could involve things like moving files, which
// will be moved relative to their nearest root.
func (f *Fixer) RegisterRoots(roots ...string) *Fixer {
	f.registeredRoots = append(f.registeredRoots, roots...)

	return f
}

func (f *Fixer) GetFixForName(name string) (fixes.Fix, bool) {
	fix, ok := f.registeredFixes[name]
	if !ok {
		return nil, false
	}

	fixInstance, ok := fix.(fixes.Fix)
	if !ok {
		return nil, false
	}

	return fixInstance, true
}

func (f *Fixer) Fix(ctx context.Context, l *linter.Linter, fp fileprovider.FileProvider) (*Report, error) {
	fixReport := NewReport()

	// If there are no registered fixes that require a linter, return the report
	if len(f.registeredFixes) == 0 {
		return fixReport, nil
	}

	// Apply fixes that require linter violation triggers
	if err := f.applyLinterFixes(ctx, l, fp, fixReport); err != nil {
		return nil, err
	}

	return fixReport, nil
}

func (f *Fixer) FixViolations(
	violations []report.Violation,
	fp fileprovider.FileProvider,
	config *config.Config,
) (*Report, error) {
	fixReport := NewReport()

	startingFiles, err := fp.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// rangeValCopy may be expensive, but this is not critical enough
	// to motivate cluttering the code
	//nolint:gocritic
	for _, violation := range violations {
		fixInstance, ok := f.GetFixForName(violation.Title)
		if !ok {
			return nil, fmt.Errorf("no fix for violation %s", violation.Title)
		}

		file := violation.Location.File

		fc, err := fp.Get(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get file %s: %w", file, err)
		}

		abs, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", file, err)
		}

		fixResults, err := fixInstance.Fix(&fixes.FixCandidate{Filename: file, Contents: fc}, &fixes.RuntimeOptions{
			BaseDir:   util.FindClosestMatchingRoot(abs, f.registeredRoots),
			Config:    config,
			Locations: []report.Location{violation.Location},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fix %s: %w", file, err)
		}

		if len(fixResults) == 0 {
			continue
		}

		if fixResults[0].Rename != nil {
			if err = f.handleRename(fp, fixReport, startingFiles, fixResults[0]); err != nil {
				return nil, fmt.Errorf("failed to handle rename: %w", err)
			}
		}

		// Write the fixed content to the file
		if err := fp.Put(file, fixResults[0].Contents); err != nil {
			return nil, fmt.Errorf("failed to write fixed content to file %s: %w", file, err)
		}

		fixReport.AddFileFix(file, fixResults[0])
	}

	return fixReport, nil
}

// applyLinterFixes handles the application of fixes that require linter violation triggers.
func (f *Fixer) applyLinterFixes(
	ctx context.Context,
	l *linter.Linter,
	fp fileprovider.FileProvider,
	fixReport *Report,
) error {
	enabledRules, err := l.DetermineEnabledRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine enabled rules: %w", err)
	}

	var fixableEnabledRules []string

	for _, rule := range enabledRules {
		if _, ok := f.GetFixForName(rule); ok {
			fixableEnabledRules = append(fixableEnabledRules, rule)
		}
	}

	startingFiles, err := fp.List()
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	var versionsMap map[string]ast.RegoVersion
	if f.versionsMap != nil {
		versionsMap = f.versionsMap
	}

	for {
		fixMadeInIteration := false

		in, err := fp.ToInput(versionsMap)
		if err != nil {
			return fmt.Errorf("failed to create linter input: %w", err)
		}

		rep, err := l.WithDisableAll(true).WithEnabledRules(fixableEnabledRules...).WithInputModules(&in).Lint(ctx)
		if err != nil {
			return fmt.Errorf("failed to lint before fixing: %w", err)
		}

		if len(rep.Violations) == 0 {
			break
		}

		//nolint:gocritic
		for _, violation := range rep.Violations {
			fixInstance, ok := f.GetFixForName(violation.Title)
			if !ok {
				return fmt.Errorf("no fix for violation %s", violation.Title)
			}

			config, err := l.GetConfig()
			if err != nil {
				return fmt.Errorf("failed to get config: %w", err)
			}

			file := violation.Location.File

			abs, err := filepath.Abs(file)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", file, err)
			}

			fc, err := fp.Get(file)
			if err != nil {
				return fmt.Errorf("failed to get file %s: %w", file, err)
			}

			fixCandidate := fixes.FixCandidate{Filename: file, Contents: fc}

			fixResults, err := fixInstance.Fix(&fixCandidate, &fixes.RuntimeOptions{
				BaseDir:   util.FindClosestMatchingRoot(abs, f.registeredRoots),
				Config:    config,
				Locations: []report.Location{violation.Location},
			})
			if err != nil {
				return fmt.Errorf("failed to fix %s: %w", file, err)
			}

			if len(fixResults) == 0 {
				continue
			}

			fixResult := fixResults[0]

			if fixResult.Rename != nil {
				if err = f.handleRename(fp, fixReport, startingFiles, fixResult); err != nil {
					return err
				}

				fixMadeInIteration = true

				break // Restart the loop after handling a rename
			}

			// Write the fixed content to the file
			if err := fp.Put(file, fixResult.Contents); err != nil {
				return fmt.Errorf("failed to write fixed content to file %s: %w", file, err)
			}

			fixReport.AddFileFix(file, fixResult)

			fixMadeInIteration = true
		}

		if !fixMadeInIteration {
			break
		}
	}

	return nil
}

// handleRename processes the rename operation and resolves conflicts if necessary.
func (f *Fixer) handleRename(
	fp fileprovider.FileProvider,
	fixReport *Report,
	startingFiles []string,
	fixResult fixes.FixResult,
) error {
	to := fixResult.Rename.ToPath
	from := fixResult.Rename.FromPath

	for {
		err := fp.Rename(from, to)
		if err == nil {
			// if there is no error, and no conflict, we have nothing to do
			break
		}

		if !errors.As(err, &fileprovider.RenameConflictError{}) {
			return fmt.Errorf("failed to rename file: %w", err)
		}

		switch f.onConflictOperation {
		case OnConflictError:
			// OnConflictError is the default, these operations are taken to
			// ensure the correct state in the report for outputting the
			// verbose conflict report.
			// clean the old file to prevent repeated fixes
			if err := fp.Delete(from); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", from, err)
			}

			if slices.Contains(startingFiles, to) {
				fixReport.RegisterConflictSourceFile(fixResult.Root, to, from)
			} else {
				fixReport.RegisterConflictManyToOne(fixResult.Root, to, from)
			}

			fixReport.AddFileFix(to, fixResult)
			fixReport.MergeFixes(to, from)
			fixReport.RegisterOldPathForFile(to, from)

			return nil
		case OnConflictRename:
			// OnConflictRename will select a new filename until there is no
			// conflict.
			to = renameCandidate(to)

			continue
		default:
			return fmt.Errorf("unsupported conflict operation: %v", f.onConflictOperation)
		}
	}

	// update the fix result with the new path for consistency
	if to != fixResult.Rename.ToPath {
		fixResult.Rename.ToPath = to
	}

	fixReport.AddFileFix(to, fixResult)
	fixReport.MergeFixes(to, from)
	fixReport.RegisterOldPathForFile(to, from)

	return nil
}
