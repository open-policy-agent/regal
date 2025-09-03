package lsp

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/completions/refs"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/hints"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/rules"
)

// diagnosticsRunOpts contains options for file and workspace linting.
type diagnosticsRunOpts struct {
	Cache            *cache.Cache
	RegalConfig      *config.Config
	WorkspaceRootURI string
	UpdateForRules   []string
	CustomRulesPath  string

	// File-specific
	FileURI string

	// Workspace-specific
	OverwriteAggregates bool
	AggregateReportOnly bool
}

// updateParseOpts contains options for updateParse function.
type updateParseOpts struct {
	Cache            *cache.Cache
	Store            storage.Store
	FileURI          string
	Builtins         map[string]*ast.Builtin
	RegoVersion      ast.RegoVersion
	WorkspaceRootURI string
}

// updateParse updates the module cache with the latest parse result for a given URI,
// if the module cannot be parsed, the parse errors are saved as diagnostics for the
// URI instead.
func updateParse(ctx context.Context, opts updateParseOpts) (bool, error) {
	content, ok := opts.Cache.GetFileContents(opts.FileURI)
	if !ok {
		return false, fmt.Errorf("failed to get file contents for uri %q", opts.FileURI)
	}

	lines := strings.Split(content, "\n")
	options := rparse.ParserOptions()
	options.RegoVersion = opts.RegoVersion

	presentedFileName := strings.TrimPrefix(opts.FileURI, opts.WorkspaceRootURI+"/")

	module, err := rparse.ModuleWithOpts(presentedFileName, content, options)
	if err == nil {
		// if the parse was ok, clear the parse errors
		opts.Cache.SetParseErrors(opts.FileURI, []types.Diagnostic{})
		opts.Cache.SetModule(opts.FileURI, module)
		opts.Cache.SetSuccessfulParseLineCount(opts.FileURI, len(lines))

		if err := PutFileMod(ctx, opts.Store, opts.FileURI, module); err != nil {
			return false, fmt.Errorf("failed to update rego store with parsed module: %w", err)
		}

		definedRefs := refs.DefinedInModule(module, opts.Builtins)

		opts.Cache.SetFileRefs(opts.FileURI, definedRefs)

		// TODO: consider how we use and generate these to avoid needing to have in the cache and the store
		var ruleRefs []string

		for _, ref := range definedRefs {
			if ref.Kind == types.Package {
				continue
			}

			ruleRefs = append(ruleRefs, ref.Label)
		}

		if err = PutFileRefs(ctx, opts.Store, opts.FileURI, ruleRefs); err != nil {
			return false, fmt.Errorf("failed to update rego store with defined refs: %w", err)
		}

		return true, nil
	}

	var astErrors []ast.Error

	// Check if err is of type ast.Errors
	var astErrs ast.Errors
	if errors.As(err, &astErrs) {
		for _, e := range astErrs {
			astErrors = append(astErrors, ast.Error{
				Code:     e.Code,
				Message:  e.Message,
				Location: e.Location,
			})
		}
	} else {
		// Check if err is a single ast.Error
		var singleAstErr *ast.Error
		if errors.As(err, &singleAstErr) {
			astErrors = append(astErrors, ast.Error{
				Code:     singleAstErr.Code,
				Message:  singleAstErr.Message,
				Location: singleAstErr.Location,
			})
		} else {
			// Unknown error type
			return false, fmt.Errorf("unknown error type: %T", err)
		}
	}

	diags := make([]types.Diagnostic, 0)

	for _, astError := range astErrors {
		line := max(astError.Location.Row-1, 0)

		lineLength := 1
		if line < len(lines) {
			lineLength = len(lines[line])
		}

		key := "regal/parse"
		link := "https://docs.styra.com/opa/category/rego-parse-error"

		hints, _ := hints.GetForError(err)
		if len(hints) > 0 {
			// there should only be one hint, so take the first
			key = hints[0]
			link = "https://docs.styra.com/opa/errors/" + hints[0]
		}

		diags = append(diags, types.Diagnostic{ /////////////////////// parse errors:
			Severity: util.Pointer(uint(1)),                         // - the only error Diagnostic the server sends
			Range:    types.RangeBetween(line, 0, line, lineLength), // - always highlights the whole line
			Message:  astError.Message,
			Source:   &key,
			Code:     strings.ReplaceAll(astError.Code, "_", "-"),
			CodeDescription: &types.CodeDescription{
				Href: link,
			},
		})
	}

	opts.Cache.SetParseErrors(opts.FileURI, diags)

	if len(diags) == 0 {
		return false, errors.New("failed to parse module, but no errors were set as diagnostics")
	}

	return false, nil
}

func updateFileDiagnostics(ctx context.Context, opts diagnosticsRunOpts) error {
	if opts.OverwriteAggregates {
		return errors.New("OverwriteAggregates should not be set for updateFileDiagnostics")
	}

	if opts.AggregateReportOnly {
		return errors.New("AggregateReportOnly should not be set for updateFileDiagnostics")
	}

	module, ok := opts.Cache.GetModule(opts.FileURI)
	if !ok {
		// then there must have been a parse error
		return nil
	}

	contents, ok := opts.Cache.GetFileContents(opts.FileURI)
	if !ok {
		return fmt.Errorf("failed to get file contents for uri %q", opts.FileURI)
	}

	input := rules.NewInput(map[string]string{opts.FileURI: contents}, map[string]*ast.Module{opts.FileURI: module})

	regalInstance := linter.NewLinter().
		// needed to get the aggregateData for this file
		WithCollectQuery(true).
		// needed to get the aggregateData out so we can update the cache
		WithExportAggregates(true).
		WithInputModules(&input).
		WithPathPrefix(opts.WorkspaceRootURI)

	if opts.RegalConfig != nil {
		regalInstance = regalInstance.WithUserConfig(*opts.RegalConfig)
	}

	if opts.CustomRulesPath != "" {
		regalInstance = regalInstance.WithCustomRules([]string{opts.CustomRulesPath})
	}

	rpt, err := regalInstance.Lint(ctx)
	if err != nil {
		return fmt.Errorf("failed to lint: %w", err)
	}

	fileDiags := convertReportToDiagnostics(&rpt, opts.WorkspaceRootURI)

	for uri := range opts.Cache.GetAllFiles() {
		// if a file has parse errors, continue to show these until they're addressed
		parseErrs, ok := opts.Cache.GetParseErrors(uri)
		if ok && len(parseErrs) > 0 {
			continue
		}

		// For updateFileDiagnostics, we only update the file in question.
		if uri == opts.FileURI {
			fd, ok := fileDiags[uri]
			if !ok {
				fd = []types.Diagnostic{}
			}

			opts.Cache.SetFileDiagnosticsForRules(uri, opts.UpdateForRules, fd)
		}
	}

	opts.Cache.SetFileAggregates(opts.FileURI, rpt.Aggregates)

	return nil
}

func updateWorkspaceDiagnostics(ctx context.Context, opts diagnosticsRunOpts) error {
	if opts.FileURI != "" {
		return errors.New("FileURI should not be set for updateAllDiagnostics")
	}

	var err error

	modules := opts.Cache.GetAllModules()
	files := opts.Cache.GetAllFiles()

	regalInstance := linter.NewLinter().
		WithPathPrefix(opts.WorkspaceRootURI).
		// aggregates need only be exported if they're to be used to overwrite.
		WithExportAggregates(opts.OverwriteAggregates)

	if opts.RegalConfig != nil {
		regalInstance = regalInstance.WithUserConfig(*opts.RegalConfig)
	}

	if opts.CustomRulesPath != "" {
		regalInstance = regalInstance.WithCustomRules([]string{opts.CustomRulesPath})
	}

	if opts.AggregateReportOnly {
		regalInstance = regalInstance.WithAggregates(opts.Cache.GetFileAggregates())
	} else {
		input := rules.NewInput(files, modules)
		regalInstance = regalInstance.WithInputModules(&input)
	}

	rpt, err := regalInstance.Lint(ctx)
	if err != nil {
		return fmt.Errorf("failed to lint: %w", err)
	}

	fileDiags := convertReportToDiagnostics(&rpt, opts.WorkspaceRootURI)

	for uri := range files {
		parseErrs, ok := opts.Cache.GetParseErrors(uri)
		if ok && len(parseErrs) > 0 {
			continue
		}

		fd, ok := fileDiags[uri]
		if !ok {
			fd = []types.Diagnostic{}
		}

		// when only an aggregate report was run, then we must make sure to
		// only update diagnostics from these rules. So the report is
		// authoratative, but for those rules only.
		if opts.AggregateReportOnly {
			opts.Cache.SetFileDiagnosticsForRules(uri, opts.UpdateForRules, fd)
		} else {
			opts.Cache.SetFileDiagnostics(uri, fd)
		}
	}

	if opts.OverwriteAggregates {
		// clear all aggregates, and use these ones
		opts.Cache.SetAggregates(rpt.Aggregates)
	}

	return nil
}

func convertReportToDiagnostics(rpt *report.Report, workspaceRootURI string) map[string][]types.Diagnostic {
	fileDiags := make(map[string][]types.Diagnostic, len(rpt.Violations))

	// rangeValCopy necessary, as value copied in loop anyway
	//nolint:gocritic
	for _, item := range rpt.Violations {
		// here errors are presented as warnings, and warnings as info
		// to differentiate from parse errors
		severity := uint(2)
		if item.Level == "warning" {
			severity = 3
		}

		file := cmp.Or(item.Location.File, workspaceRootURI)
		source := "regal/" + item.Category

		fileDiags[file] = append(fileDiags[file], types.Diagnostic{
			Severity: &severity,
			Range:    getRangeForViolation(item),
			Message:  item.Description,
			Source:   &source,
			Code:     item.Title,
			CodeDescription: &types.CodeDescription{
				Href: fmt.Sprintf(
					"https://docs.styra.com/regal/rules/%s/%s",
					item.Category,
					item.Title,
				),
			},
		})
	}

	return fileDiags
}

func getRangeForViolation(item report.Violation) types.Range {
	startLine, startChar := max(item.Location.Row-1, 0), max(item.Location.Column-1, 0)

	if item.Location.End != nil {
		return types.RangeBetween(
			startLine, startChar,
			max(item.Location.End.Row-1, 0), max(item.Location.End.Column-1, 0),
		)
	}

	itemLen := 0
	if item.Location.Text != nil {
		itemLen = len(*item.Location.Text)
	}

	return types.RangeBetween(startLine, startChar, startLine, startChar+itemLen)
}
