package inlayhint

import (
	"fmt"
	"slices"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/types"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	lspTypes "github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/util"
)

var noInlayHints = make([]lspTypes.InlayHint, 0)

type builtinsMap = map[string]*ast.Builtin

func FromModule(module *ast.Module, builtins builtinsMap) []lspTypes.InlayHint {
	inlayHints := make([]lspTypes.InlayHint, 0)

	for _, call := range rego.AllBuiltinCalls(module, builtins) {
		for i, arg := range call.Builtin.Decl.NamedFuncArgs().Args {
			if len(call.Args) <= i {
				// avoid panic if provided a builtin function where the args
				// have yet to be provided, like if the user types `split()`
				continue
			}

			if named, ok := arg.(*types.NamedType); ok {
				inlayHints = append(inlayHints, lspTypes.InlayHint{
					Position:     rego.PositionFromLocation(call.Args[i].Location),
					Label:        named.Name + ":",
					Kind:         2,
					PaddingLeft:  false,
					PaddingRight: true,
					Tooltip:      *lspTypes.Markdown(createInlayTooltip(named)),
				})
			}
		}
	}

	return inlayHints
}

func Partial(parseErrors []lspTypes.Diagnostic, policy, uri string, builtins builtinsMap) []lspTypes.InlayHint {
	firstErrorLine := slices.MinFunc(parseErrors, func(a, b lspTypes.Diagnostic) int {
		return util.SafeUintToInt(a.Range.Start.Line - b.Range.Start.Line)
	}).Range.Start.Line

	// try parse the lines up to the first parse error
	if numLines := util.NumLines(policy); firstErrorLine > 0 && firstErrorLine < numLines {
		// (1-indexed so don't +1 here)
		if end := util.IndexByteNth(policy, '\n', firstErrorLine); end != -1 {
			if mod, err := parse.Module(uri, policy[:end]); err == nil {
				return FromModule(mod, builtins)
			}
		}
	}

	// if there are parse errors from line 0, we skip doing anything
	// if the last valid line is beyond the end of the file, we exit as something is up
	// if we still can't parse the bit we hoped was valid, we exit as this is 'too hard'
	return noInlayHints
}

func createInlayTooltip(named *types.NamedType) string {
	if named.Descr == "" {
		return fmt.Sprintf("Type: `%s`", named.Type.String())
	}

	return fmt.Sprintf("%s\n\nType: `%s`", named.Descr, named.Type.String())
}
