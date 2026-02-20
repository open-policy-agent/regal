package linter

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
)

// 736486708 ns/op	2348230496 B/op	51198148 allocs/op // OPA v1.12.2
// 563373188 ns/op	1962281168 B/op	46929220 allocs/op // AST aggregates refactor
// 461607500 ns/op	1543793525 B/op	36938279 allocs/op // Performance refactor + follow-up
func BenchmarkRegalLintingItself(b *testing.B) {
	for b.Loop() {
		linter := NewLinter().
			WithInputPaths([]string{"../../bundle"}).
			WithUserConfig(must.Return(config.FromPath(filepath.Join("..", "..", ".regal", "config.yaml")))(b))

		testutil.AssertNumViolations(b, 0, must.Return(linter.Lint(b.Context()))(b))
	}
}

// 694275500 ns/op	2568604236 B/op	52506343 allocs/op // OPA v1.10.0
// 656495042 ns/op	2309640068 B/op	50264746 allocs/op // OPA v1.12.2
// 497374153 ns/op	1923188613 B/op	45981278 allocs/op // AST aggregates refactor
// 486491514 ns/op	1906786789 B/op	45640947 allocs/op // OPA v1.13.1 + fixes
// 420959403 ns/op	1534395760 B/op	36917131 allocs/op // Performance refactor
// 403029542 ns/op	1494994832 B/op	35884349 allocs/op // Performance refactor follow-up
// 403083139 ns/op	1520423272 B/op	36511766 allocs/op // 3 new rules added
//
// 349616986 ns/op	1260233392 B/op	31870693 allocs/op
func BenchmarkRegalLintingItselfPrepareOnce(b *testing.B) {
	benchmarkLint(b, bundleLinter(b, true).MustPrepare(b.Context()))
}

// 65815866 ns/op   43852693 B/op    1025467 allocs/op // OPA v1.10.0
// 64977849 ns/op   38570571 B/op     932404 allocs/op // OPA v1.12.2
// 61936272 ns/op	38191932 B/op	  921084 allocs/op // OPA v1.13.1
func BenchmarkOnlyPrepare(b *testing.B) {
	linter := bundleLinter(b, true)
	for b.Loop() {
		linter.MustPrepare(b.Context())
	}
}

// 127396828 ns/op	300739526 B/op	 5938689 allocs/op // OPA v1.10.0
// 123784616 ns/op	284724624 B/op	 5918990 allocs/op // OPA v1.12.2
// _95888368 ns/op	156125725 B/op	 3331213 allocs/op // With Rego prepare eval phase (!!!)
func BenchmarkRegalNoEnabledRules(b *testing.B) {
	benchmarkLint(b, bundleLinter(b, false).WithDisableAll(true))
}

// 53643340 ns/op	256599746 B/op	 4910862 allocs/op // OPA v1.10.0
// 53197442 ns/op	245969548 B/op	 4984253 allocs/op // OPA v1.12.2
// 32465720 ns/op	118513114 B/op	 2409371 allocs/op // With Rego prepare eval phase
func BenchmarkRegalNoEnabledRulesPrepareOnce(b *testing.B) {
	benchmarkLint(b, bundleLinter(b, false).WithDisableAll(true).MustPrepare(b.Context()))
}

// Runs a separate benchmark for each rule in the bundle. Note that this will take *several* minutes to run,
// meaning you do NOT want to do this more than occasionally. You may however find it helpful to use this with
// a single, or handful of rules to get a better idea of how long they take to run, and relative to each other.
func BenchmarkEachRule(b *testing.B) {
	config := must.Return(config.WithDefaultsFromBundle(bundle.Loaded(), nil))(b)
	linter := bundleLinter(b, false).WithDisableAll(true).MustPrepare(b.Context())

	for _, category := range config.Rules {
		for ruleName := range category {
			// Uncomment / modify this to benchmark specific rule(s) only
			//
			// if ruleName != "metasyntactic-variable" {
			// 	continue
			// }
			b.Run(ruleName, func(b *testing.B) {
				benchmarkLint(b, linter.WithEnabledRules(ruleName))
			})
		}
	}
}

func bundleLinter(b *testing.B, withConfig bool) Linter {
	b.Helper()

	linter := NewLinter().WithInputPaths([]string{"../../bundle"})

	if withConfig {
		config := must.Return(config.FromPath(filepath.Join("..", "..", ".regal", "config.yaml")))(b)
		linter = linter.WithUserConfig(config)
	}

	return linter
}

func benchmarkLint(b *testing.B, linter Linter) {
	b.Helper()

	var rep report.Report
	for b.Loop() {
		rep = must.Return(linter.Lint(b.Context()))(b)
	}

	testutil.AssertNumViolations(b, 0, rep)
}
