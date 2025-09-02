package inlayhint

import (
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/types"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	lspTypes "github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/parse"
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

func Partial(parseErrors []lspTypes.Diagnostic, contents, uri string, builtins builtinsMap) []lspTypes.InlayHint {
	firstErrorLine := uint(0)
	for _, parseError := range parseErrors {
		if parseError.Range.Start.Line > firstErrorLine {
			firstErrorLine = parseError.Range.Start.Line
		}
	}

	split := strings.Split(contents, "\n")

	if firstErrorLine == 0 || firstErrorLine > uint(len(split)) {
		// if there are parse errors from line 0, we skip doing anything
		// if the last valid line is beyond the end of the file, we exit as something is up
		return noInlayHints
	}

	// select the lines from the contents up to the first parse error
	lines := strings.Join(split[:firstErrorLine], "\n")

	// parse the part of the module that might work
	module, err := parse.Module(uri, lines)
	if err != nil {
		// if we still can't parse the bit we hoped was valid, we exit as this is 'too hard'
		return noInlayHints
	}

	return FromModule(module, builtins)
}

func createInlayTooltip(named *types.NamedType) string {
	if named.Descr == "" {
		return fmt.Sprintf("Type: `%s`", named.Type.String())
	}

	return fmt.Sprintf("%s\n\nType: `%s`", named.Descr, named.Type.String())
}
