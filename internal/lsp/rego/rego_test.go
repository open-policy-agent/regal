package rego

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/open-policy-agent/regal/internal/lsp/rego/query"
	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestAllRuleHeadLocations(t *testing.T) {
	t.Parallel()

	qc := query.NewCache()
	contents := `package p

	default allow := false

	allow if 1
	allow if 2

	foo.bar[x] if x := 1
	foo.bar[x] if x := 2`

	pq := testutil.Must(qc.GetOrSet(t.Context(), inmem.New(), query.RuleHeadLocations))(t)
	module := parse.MustParseModule(contents)
	ruleHeads := testutil.Must(AllRuleHeadLocations(t.Context(), pq, "p.rego", contents, module))(t)

	if len(ruleHeads) != 2 {
		t.Fatalf("expected 2 code lenses, got %d", len(ruleHeads))
	}

	if len(ruleHeads["data.p.allow"]) != 3 {
		t.Fatalf("expected 3 allow rule heads, got %d", len(ruleHeads["data.p.allow"]))
	}

	if len(ruleHeads["data.p.foo.bar"]) != 2 {
		t.Fatalf("expected 2 foo.bar rule heads, got %d", len(ruleHeads["data.p.foo.bar"]))
	}
}

func TestAllKeywords(t *testing.T) {
	t.Parallel()

	qc := query.NewCache()

	contents := `package p

	import data.foo

	my_set contains "x" if true
	`

	pq := testutil.Must(qc.GetOrSet(t.Context(), inmem.New(), query.Keywords))(t)
	keywords := testutil.Must(AllKeywords(t.Context(), pq, "p.rego", contents, parse.MustParseModule(contents)))(t)

	// this is "lines with keywords", not number of keywords
	if len(keywords) != 3 {
		t.Fatalf("expected 1 keyword, got %d", len(keywords))
	}

	if len(keywords["1"]) != 1 {
		t.Fatalf("expected 1 keywords on line 1, got %d", len(keywords["1"]))
	}

	if len(keywords["3"]) != 1 {
		t.Fatalf("expected 1 keywords on line 3, got %d", len(keywords["1"]))
	}

	if len(keywords["5"]) != 2 {
		t.Fatalf("expected 2 keywords on line 5, got %d", len(keywords["1"]))
	}
}
