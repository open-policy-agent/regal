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

// 745276520 ns/op	2612562272 B/op	53531222 allocs/op // OPA v1.10.0
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

// 694275500 ns/op	2568604236 B/op	52506343 allocs/op // OPA v1.10.0
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

// 65815866 ns/op	43852693 B/op	 1025467 allocs/op // OPA v1.10.0
func BenchmarkOnlyPrepare(b *testing.B) {
	conf := testutil.Must(config.FromPath(filepath.Join("..", "..", ".regal", "config.yaml")))(b)
	linter := NewLinter().WithInputPaths([]string{"../../bundle"}).WithUserConfig(conf)

	for b.Loop() {
		linter.MustPrepare(b.Context())
	}
}

// 127396828 ns/op	300739526 B/op	 5938689 allocs/op // OPA v1.10.0
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

// 53643340 ns/op	256599746 B/op	 4910862 allocs/op // OPA v1.10.0
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
	conf := testutil.Must(config.WithDefaultsFromBundle(bundle.Loaded(), nil))(b)

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
