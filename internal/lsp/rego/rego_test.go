package rego

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestAllRuleHeadLocations(t *testing.T) {
	t.Parallel()

	contents := `package p

	default allow := false

	allow if 1
	allow if 2

	foo.bar[x] if x := 1
	foo.bar[x] if x := 2`

	pq := must.Return(query.NewCache().GetOrSet(t.Context(), inmem.New(), query.RuleHeadLocations))(t)
	module := parse.MustParseModule(contents)
	ruleHeads := must.Return(AllRuleHeadLocations(t.Context(), pq, "p.rego", contents, module))(t)

	must.Equal(t, 2, len(ruleHeads), "rules with heads")
	must.Equal(t, 3, len(ruleHeads["data.p.allow"]), "allow rule heads")
	must.Equal(t, 2, len(ruleHeads["data.p.foo.bar"]), "foo.bar rule heads")
}

func TestAllKeywords(t *testing.T) {
	t.Parallel()

	contents := `package p

	import data.foo

	my_set contains "x" if true`

	pq := must.Return(query.NewCache().GetOrSet(t.Context(), inmem.New(), query.Keywords))(t)
	keywords := must.Return(AllKeywords(t.Context(), pq, "p.rego", contents, parse.MustParseModule(contents)))(t)

	// this is "lines with keywords", not number of keywords
	must.Equal(t, 3, len(keywords), "lines with keywords")
	must.Equal(t, 1, len(keywords["1"]), "line 1 keywords")
	must.Equal(t, 1, len(keywords["3"]), "line 3 keywords")
	must.Equal(t, 2, len(keywords["5"]), "line 5 keywords")
}
