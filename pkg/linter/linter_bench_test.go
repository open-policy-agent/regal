package linter

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/bundle"
	"github.com/open-policy-agent/regal/internal/cache"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
)

// 752941230 ns/op	2513359004 B/op	52228468 allocs/op
// 731859438 ns/op	2481404780 B/op	51310839 allocs/op main
// 720459812 ns/op	2474991988 B/op	51155691 allocs/op
// 704425916 ns/op	2476541140 B/op	51151877 allocs/op

// BenchmarkRegalLintingItself-16    	       2	 717249188 ns/op	2477907556 B/op	51198136 allocs/op
func BenchmarkRegalLintingItself(b *testing.B) {
	conf := testutil.Must(config.FromPath(filepath.Join("..", "..", ".regal", "config.yaml")))(b)

	linter := NewLinter().
		WithInputPaths([]string{"../../bundle"}).
		WithBaseCache(cache.NewBaseCache()).
		WithUserConfig(conf)

	var rep report.Report

	for b.Loop() {
		rep = testutil.Must(linter.Lint(b.Context()))(b)
	}

	testutil.AssertNumViolations(b, 0, rep)
}

// 955670792 ns/op	3004937416 B/op	57408554 allocs/op
// ...
func BenchmarkRegalLintingItselfPrepareOnce(b *testing.B) {
	conf := testutil.Must(config.FromPath(filepath.Join("..", "..", ".regal", "config.yaml")))(b)

	linter := NewLinter().
		WithInputPaths([]string{"../../bundle"}).
		WithBaseCache(cache.NewBaseCache()).
		WithUserConfig(conf).
		MustPrepare(b.Context())

	var rep report.Report

	for b.Loop() {
		rep = testutil.Must(linter.Lint(b.Context()))(b)
	}

	testutil.AssertNumViolations(b, 0, rep)
}

// 62198250 ns/op	42255370 B/op	  985472 allocs/op
// ...
func BenchmarkOnlyPrepare(b *testing.B) {
	conf := testutil.Must(config.FromPath(filepath.Join("..", "..", ".regal", "config.yaml")))(b)
	linter := NewLinter().WithInputPaths([]string{"../../bundle"}).WithUserConfig(conf)

	for b.Loop() {
		linter.MustPrepare(b.Context())
	}
}

// 6	 168875708 ns/op	455586606 B/op	 8537889 allocs/op
// 8	 139592615 ns/op	326694376 B/op	 5996314 allocs/op
// ...
func BenchmarkRegalNoEnabledRules(b *testing.B) {
	linter := NewLinter().
		WithInputPaths([]string{"../../bundle"}).
		WithBaseCache(cache.NewBaseCache()).
		WithDisableAll(true)

	var rep report.Report

	for b.Loop() {
		rep = testutil.Must(linter.Lint(b.Context()))(b)
	}

	testutil.AssertNumViolations(b, 0, rep)
}

// 97546746 ns/op	401323023 B/op	 7427903 allocs/op
// ...
func BenchmarkRegalNoEnabledRulesPrepareOnce(b *testing.B) {
	linter := NewLinter().
		WithInputPaths([]string{"../../bundle"}).
		WithBaseCache(cache.NewBaseCache()).
		WithDisableAll(true).
		MustPrepare(b.Context())

	var rep report.Report

	for b.Loop() {
		rep = testutil.Must(linter.Lint(b.Context()))(b)
	}

	testutil.AssertNumViolations(b, 0, rep)
}

// Runs a separate benchmark for each rule in the bundle. Note that this will take *several* minutes to run,
// meaning you do NOT want to do this more than occasionally. You may however find it helpful to use this with
// a single, or handful of rules to get a better idea of how long they take to run, and relative to each other.
func BenchmarkEachRule(b *testing.B) {
	conf := testutil.Must(config.WithDefaultsFromBundle(bundle.LoadedBundle(), nil))(b)

	linter := NewLinter().
		WithInputPaths([]string{"../../bundle"}).
		WithBaseCache(cache.NewBaseCache()).
		WithDisableAll(true).
		MustPrepare(b.Context())

	for _, category := range conf.Rules {
		for ruleName := range category {
			// Uncomment / modify this to benchmark specific rule(s) only
			//
			// if ruleName != "metasyntactic-variable" {
			// 	continue
			// }
			b.Run(ruleName, func(b *testing.B) {
				var rep report.Report

				for b.Loop() {
					rep = testutil.Must(linter.WithEnabledRules(ruleName).Lint(b.Context()))(b)
				}

				testutil.AssertNumViolations(b, 0, rep)
			})
		}
	}
}
