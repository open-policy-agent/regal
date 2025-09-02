package rego

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestAllRuleHeadLocations(t *testing.T) {
	t.Parallel()

	contents := `package p

	default allow := false

	allow if 1
	allow if 2

	foo.bar[x] if x := 1
	foo.bar[x] if x := 2`

	module := parse.MustParseModule(contents)
	ruleHeads := testutil.Must(AllRuleHeadLocations(t.Context(), "p.rego", contents, module))(t)

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

	contents := `package p

	import data.foo

	my_set contains "x" if true
	`

	module := parse.MustParseModule(contents)
	keywords := testutil.Must(AllKeywords(t.Context(), "p.rego", contents, module))(t)

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

// 2	 532180958 ns/op	348596532 B/op	 7898069 allocs/op
func BenchmarkStoreAllCachedQueries(b *testing.B) {
	for b.Loop() {
		if err := StoreAllCachedQueries(b.Context(), nil); err != nil {
			b.Fatalf("failed to store all cached queries: %v", err)
		}
	}
}
