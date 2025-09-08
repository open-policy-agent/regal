package hover

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/types"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/examples"
	"github.com/open-policy-agent/regal/internal/lsp/rego"
	types2 "github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

var builtinCache = concurrent.MapOf(make(map[*ast.Builtin]string))

func writeFunctionSnippet(sb *strings.Builder, builtin *ast.Builtin) {
	sb.WriteString("```rego\n")

	resType := builtin.Decl.NamedResult()
	if n, ok := resType.(*types.NamedType); ok {
		sb.WriteString(n.Name)
	} else {
		sb.WriteString("output")
	}

	sb.WriteString(" := ")

	sb.WriteString(builtin.Name)
	sb.WriteString("(")

	for i, arg := range builtin.Decl.NamedFuncArgs().Args {
		if i > 0 {
			sb.WriteString(", ")
		}

		if n, ok := arg.(*types.NamedType); ok {
			sb.WriteString(n.Name)
		} else {
			sb.WriteString(arg.String())
		}
	}

	sb.WriteString(")\n```")
}

func CreateHoverContent(builtin *ast.Builtin) string {
	if content, ok := builtinCache.Get(builtin); ok {
		return content
	}

	link := fmt.Sprintf(
		"https://www.openpolicyagent.org/docs/policy-reference/#builtin-%s-%s",
		rego.BuiltinCategory(builtin),
		strings.ReplaceAll(builtin.Name, ".", ""),
	)

	// EOPA supports custom links via categories from 1.29.1
	for _, category := range builtin.Categories {
		if strings.HasPrefix(category, "url=") {
			link = category[4:]

			break
		}
	}

	title := fmt.Sprintf("[%s](%s)", builtin.Name, link)
	sb := &strings.Builder{}

	sb.WriteString("### ")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	if exampleLink, ok := examples.GetBuiltInLink(builtin.Name); ok {
		fmt.Fprintf(sb, "[View Usage Examples](%s)\n", exampleLink)
	}

	writeFunctionSnippet(sb, builtin)

	sb.WriteString("\n\n")
	sb.WriteString(builtin.Description)
	sb.WriteString("\n")

	if len(builtin.Decl.FuncArgs().Args) == 0 {
		return sb.String()
	}

	sb.WriteString("\n\n#### Arguments\n\n")

	for _, arg := range builtin.Decl.NamedFuncArgs().Args {
		sb.WriteString("- ")

		if n, ok := arg.(*types.NamedType); ok {
			sb.WriteString("`")
			sb.WriteString(n.Name)
			sb.WriteString("` ")
			sb.WriteString(n.Type.String())

			if n.Descr != "" {
				sb.WriteString(" — ")
				sb.WriteString(n.Descr)
			}
		} else {
			sb.WriteString(arg.String())
		}

		sb.WriteString("\n")
	}

	sb.WriteString("\n\nReturns ")

	ret := builtin.Decl.NamedResult()
	if n, ok := ret.(*types.NamedType); ok {
		sb.WriteString("`")
		sb.WriteString(n.Name)
		sb.WriteString("` of type `")
		sb.WriteString(n.Type.String())
		sb.WriteString("`: ")
		sb.WriteString(n.Descr)
	} else if ret != nil {
		sb.WriteString(ret.String())
	}

	sb.WriteString("\n")

	result := sb.String()

	builtinCache.Set(builtin, result)

	return result
}

func UpdateBuiltinPositions(cache *cache.Cache, uri string, builtins map[string]*ast.Builtin) error {
	module, ok := cache.GetModule(uri)
	if !ok {
		return fmt.Errorf("failed to update builtin positions: no parsed module for uri %q", uri)
	}

	builtinsOnLine := map[uint][]types2.BuiltinPosition{}

	//nolint:gosec
	for _, call := range rego.AllBuiltinCalls(module, builtins) {
		line := uint(call.Location.Row)

		builtinsOnLine[line] = append(builtinsOnLine[line], types2.BuiltinPosition{
			Builtin: call.Builtin,
			Line:    line,
			Start:   uint(call.Location.Col),
			End:     uint(call.Location.Col + len(call.Builtin.Name)),
		})
	}

	cache.SetBuiltinPositions(uri, builtinsOnLine)

	return nil
}

func UpdateKeywordLocations(ctx context.Context, cache *cache.Cache, uri string) error {
	fileContents, module, ok := cache.GetContentAndModule(uri)
	if !ok {
		return fmt.Errorf("failed to determine keyword locations: missing file contents for uri %q", uri)
	}

	keywords, err := rego.AllKeywords(ctx, filepath.Base(uri), fileContents, module)
	if err != nil {
		return fmt.Errorf("failed to determine keyword locations: %w", err)
	}

	keywordLocations := make(map[uint][]types2.KeywordLocation)

	for line, uses := range keywords {
		lineNumber32, err := strconv.ParseUint(line, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse line number: %w", err)
		}

		lineNumber := uint(lineNumber32)

		for _, use := range uses {
			keywordLocations[lineNumber] = append(keywordLocations[lineNumber], types2.KeywordLocation{
				Name:  use.Name,
				Line:  lineNumber,
				Start: use.Location.Col,
				End:   use.Location.Col + uint(len(use.Name)),
			})
		}
	}

	cache.SetKeywordLocations(uri, keywordLocations)

	return nil
}
