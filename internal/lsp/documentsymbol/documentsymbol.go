package documentsymbol

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"

	rast "github.com/open-policy-agent/regal/internal/ast"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/types/symbols"
	"github.com/open-policy-agent/regal/internal/util"
)

func All(contents string, module *ast.Module, builtins map[string]*ast.Builtin) []types.DocumentSymbol {
	// TODO: Rewrite in Rego.
	// Only pkgSymbols would likely suffice, but we're keeping docSymbols around in case
	// we ever want to add more top-level symbols than the package.
	docSymbols := make([]types.DocumentSymbol, 0)
	pkgSymbols := make([]types.DocumentSymbol, 0)

	lines := strings.Split(contents, "\n")
	pkgRange := types.RangeBetween(0, 0, len(lines)-1, len(lines[len(lines)-1]))
	pkg := documentSymbol(module.Package.Path.String(), symbols.Package, pkgRange)

	// Create groups of rules and functions sharing the same name
	ruleGroups := make(map[string][]*ast.Rule, len(module.Rules))

	for _, rule := range module.Rules {
		name := rule.Head.Ref().String()
		ruleGroups[name] = append(ruleGroups[name], rule)
	}

	for _, rules := range ruleGroups {
		if len(rules) == 1 {
			rule := rules[0]

			kind := symbols.Variable
			if rast.IsConstant(rule) {
				kind = symbols.Constant
			} else if rule.Head.Args != nil {
				kind = symbols.Function
			}

			ruleSymbol := documentSymbol(rule.Head.Ref().String(), kind, locationToRange(rule.Location))

			if detail := rast.GetRuleDetail(rule, builtins); detail != "" {
				ruleSymbol.Detail = &detail
			}

			pkgSymbols = append(pkgSymbols, ruleSymbol)
		} else {
			groupFirstRange := locationToRange(rules[0].Location)
			groupLastRange := locationToRange(rules[len(rules)-1].Location)
			groupRange := types.Range{Start: groupFirstRange.Start, End: groupLastRange.End}

			kind := symbols.Variable
			if rules[0].Head.Args != nil {
				kind = symbols.Function
			}

			groupSymbol := documentSymbol(rules[0].Head.Ref().String(), kind, groupRange)

			if detail := rast.GetRuleDetail(rules[0], builtins); detail != "" {
				groupSymbol.Detail = &detail
			}

			children := make([]types.DocumentSymbol, 0, len(rules))

			for i, rule := range rules {
				childRule := documentSymbol(fmt.Sprintf("#%d", i+1), kind, locationToRange(rule.Location))

				if childDetail := rast.GetRuleDetail(rule, builtins); childDetail != "" {
					childRule.Detail = &childDetail
				}

				children = append(children, childRule)
			}

			groupSymbol.Children = &children

			pkgSymbols = append(pkgSymbols, groupSymbol)
		}
	}

	if len(pkgSymbols) > 0 {
		pkg.Children = &pkgSymbols
	}

	return append(docSymbols, pkg)
}

func locationToRange(location *ast.Location) types.Range {
	startLine := util.SafeIntToUint(location.Row - 1)
	numLines := bytes.Count(location.Text, []byte{'\n'}) + 1

	endLine := startLine
	if numLines > 1 {
		endLine += util.SafeIntToUint(numLines - 1)
	}

	endLineContent := util.LineContents(location.Text, endLine-startLine)

	return types.RangeBetween(startLine, location.Col-1, endLine, len(endLineContent))
}

func toWorkspaceSymbol(docSym types.DocumentSymbol, docURL string) types.WorkspaceSymbol {
	return types.WorkspaceSymbol{
		Name:     docSym.Name,
		Kind:     docSym.Kind,
		Location: types.Location{URI: docURL, Range: docSym.Range},
	}
}

func ToWorkspaceSymbols(docSym []types.DocumentSymbol, docURL string, symbols *[]types.WorkspaceSymbol) {
	for _, sym := range docSym {
		// Only include the "main" symbol for incremental rules and functions
		// as numeric items isn't very useful in the workspace symbol list.
		if !strings.HasPrefix(sym.Name, "#") {
			*symbols = append(*symbols, toWorkspaceSymbol(sym, docURL))

			if sym.Children != nil {
				ToWorkspaceSymbols(*sym.Children, docURL, symbols)
			}
		}
	}
}

func documentSymbol(name string, kind symbols.SymbolKind, rang types.Range) types.DocumentSymbol {
	return types.DocumentSymbol{
		Name:           name,
		Kind:           kind,
		Range:          rang,
		SelectionRange: rang,
	}
}
