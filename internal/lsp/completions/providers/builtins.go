package providers

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/hover"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/types/completion"
)

var (
	patternWhiteSpace = regexp.MustCompile(`\s+`)
	patternRuleBody   = regexp.MustCompile(`^\s+`)
)

type BuiltIns struct{}

func (*BuiltIns) Name() string {
	return "builtins"
}

func (*BuiltIns) Run(
	_ context.Context,
	c *cache.Cache,
	params types.CompletionParams,
	opts *Options,
) ([]types.CompletionItem, error) {
	if opts == nil {
		return nil, errors.New("builtins provider requires options")
	}

	lines, line := completionLineHelper(c, params.TextDocument.URI, params.Position.Line)
	if len(lines) < 1 || line == "" || !inRuleBody(line) || strings.HasPrefix(strings.TrimSpace(line), "default ") {
		return []types.CompletionItem{}, nil
	}

	words := patternWhiteSpace.Split(strings.TrimSpace(line), -1)
	lastWord := words[len(words)-1]
	items := make([]types.CompletionItem, 0, len(opts.Builtins))
	p := params.Position

	for _, builtIn := range opts.Builtins {
		if builtIn.Infix != "" || builtIn.IsDeprecated() || !strings.HasPrefix(builtIn.Name, lastWord) {
			continue
		}

		items = append(items, types.CompletionItem{
			Label:         builtIn.Name,
			Kind:          completion.Function,
			Detail:        "built-in function",
			Documentation: types.Markdown(hover.CreateHoverContent(builtIn)),
			TextEdit: &types.TextEdit{
				Range:   types.RangeBetween(p.Line, p.Character-uint(len(lastWord)), p.Line, p.Character),
				NewText: builtIn.Name,
			},
		})
	}

	return items, nil
}

// inRuleBody is a best-effort helper to determine if the current line is in a rule body.
func inRuleBody(currentLine string) bool {
	switch {
	case strings.Contains(currentLine, " if "):
		return true
	case strings.Contains(currentLine, " contains "):
		return true
	case strings.Contains(currentLine, " else "):
		return true
	case strings.Contains(currentLine, "= "):
		return true
	case patternRuleBody.MatchString(currentLine):
		return true
	}

	return false
}
