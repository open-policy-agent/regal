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
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/completions/refs"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	rparse "github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/hints"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/rules"
)

var (
	emptyDiagnostics = []types.Diagnostic{}

	errParseFailNoErrors = errors.New("failed to parse module, but no errors were set as diagnostics")

	diagErrorLevel = new(uint(1))
	diagWarnLevel  = new(uint(2))
	diagInfoLevel  = new(uint(3))
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
}

// updateParseOpts contains options for updateParse function.
type updateParseOpts struct {
	Cache            *cache.Cache
	Store            storage.Store
	FileURI          string
	Builtins         map[string]*ast.Builtin
	RegoVersion      ast.RegoVersion
	WorkspaceRootURI string
	ClientIdentifier clients.Identifier
}

// updateParse updates the module cache with the latest parse result for a given URI,
// if the module cannot be parsed, the parse errors are saved as diagnostics for the
// URI instead.
func updateParse(ctx context.Context, opts updateParseOpts) (bool, error) {
	content, ok := opts.Cache.GetFileContents(opts.FileURI)
	if !ok {
		return false, fmt.Errorf("failed to get file contents for uri %q", opts.FileURI)
	}

	options := rparse.ParserOptions()
	options.RegoVersion = opts.RegoVersion

	numLines := util.NumLines(content)
	presentedFileName := uri.ToRelativePath(opts.FileURI, opts.WorkspaceRootURI)

	module, err := rparse.ModuleWithOpts(presentedFileName, content, options)
	if err == nil {
		// if the parse was ok, clear the parse errors
		opts.Cache.SetParseErrors(opts.FileURI, emptyDiagnostics)
		opts.Cache.SetModule(opts.FileURI, module)
		opts.Cache.SetSuccessfulParseLineCount(opts.FileURI, numLines)

		if err := PutFileMod(ctx, opts.Store, opts.FileURI, module); err != nil {
			return false, fmt.Errorf("failed to update rego store with parsed module: %w", err)
		}

		definedRefs := refs.DefinedInModule(module, opts.Builtins)

		ruleRefs := make([]string, 0, len(definedRefs)-1)
		for _, ref := range definedRefs {
			if ref.Kind != types.Package {
				ruleRefs = append(ruleRefs, ref.Label)
			}
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
			astErrors = append(astErrors, ast.Error{Code: e.Code, Message: e.Message, Location: e.Location})
		}
	} else {
		// Check if err is a single ast.Error
		var e *ast.Error
		if errors.As(err, &e) {
			astErrors = append(astErrors, ast.Error{Code: e.Code, Message: e.Message, Location: e.Location})
		} else {
			return false, fmt.Errorf("unknown error type: %T", err)
		}
	}

	diags := make([]types.Diagnostic, 0, len(astErrors))

	for _, astError := range astErrors {
		line := uint(max(astError.Location.Row-1, 0))
		text, _ := util.Line(content, line+1)
		lineLength := cmp.Or(uint(len(text)), 1)
		key := "regal/parse"
		link := "https://www.openpolicyagent.org/docs/errors/" // overview page

		hints, _ := hints.GetForError(err)
		if len(hints) > 0 {
			// there should only be one hint, so take the first
			key = hints[0]
			link += hints[0]
		}

		diags = append(diags, types.Diagnostic{
			Severity:        diagErrorLevel,                                // - only error Diagnostic the server sends
			Range:           types.RangeBetween(line, 0, line, lineLength), // - always highlights the whole line
			Message:         astError.Message,
			Source:          &key,
			Code:            strings.ReplaceAll(astError.Code, "_", "-"),
			CodeDescription: &types.CodeDescription{Href: link},
		})
	}

	opts.Cache.SetParseErrors(opts.FileURI, diags)

	if len(diags) == 0 {
		return false, errParseFailNoErrors
	}

	return false, nil
}

func updateWorkspaceDiagnostics(ctx context.Context, opts diagnosticsRunOpts) (err error) {
	if opts.FileURI != "" {
		return errors.New("FileURI should not be set for updateAllDiagnostics")
	}

	modules := opts.Cache.GetAllModules()
	files := opts.Cache.GetAllFiles()

	regalInstance := linter.NewLinter().
		WithPathPrefix(opts.WorkspaceRootURI).
		WithCustomRulesPaths(opts.CustomRulesPath)

	if opts.RegalConfig != nil {
		regalInstance = regalInstance.WithUserConfig(*opts.RegalConfig)
	}

	input := rules.NewInput(files, modules)
	regalInstance = regalInstance.WithInputModules(&input)

	preparedInstance, err := regalInstance.Prepare(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare linter: %w", err)
	}

	rpt, err := preparedInstance.Lint(ctx)
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
			fd = emptyDiagnostics
		}

		opts.Cache.SetFileDiagnostics(uri, fd)
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
		severity := diagWarnLevel
		if item.Level == "warning" {
			severity = diagInfoLevel
		}

		file := cmp.Or(item.Location.File, workspaceRootURI)

		fileDiags[file] = append(fileDiags[file], types.Diagnostic{
			Severity: severity,
			Range:    getRangeForViolation(item),
			Message:  item.Description,
			Source:   new("regal/" + item.Category),
			Code:     item.Title,
			CodeDescription: &types.CodeDescription{
				Href: fmt.Sprintf("https://www.openpolicyagent.org/projects/regal/rules/%s/%s", item.Category, item.Title),
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
