package linter

import (
	"bytes"
	"embed"
	"path/filepath"
	"slices"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/topdown"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/rules"
)

func TestLintWithDefaultBundle(t *testing.T) {
	t.Parallel()

	input := test.InputPolicy("p/p.rego", `package p

# TODO: fix this
camelCase if {
	input.one == 1
	input.two == 2
}
`)

	linter := NewLinter().WithEnableAll(true).WithInputModules(input)
	result := must.Return(linter.Lint(t.Context()))(t)

	testutil.AssertNumViolations(t, 2, result)

	assert.Equal(t, "todo-comment", result.Violations[0].Title, "unexpected violation")
	assert.Equal(t, 3, result.Violations[0].Location.Row, "unexpected line number")
	assert.Equal(t, 1, result.Violations[0].Location.Column, "unexpected column number")
	assert.Equal(t, "# TODO: fix this", *result.Violations[0].Location.Text, "unexpected location text")

	assert.Equal(t, "prefer-snake-case", result.Violations[1].Title, "unexpected violation")
	assert.Equal(t, 4, result.Violations[1].Location.Row, "unexpected line number")
	assert.Equal(t, 1, result.Violations[1].Location.Column, "unexpected column number")
	assert.Equal(t, "camelCase if {", *result.Violations[1].Location.Text, "unexpected location text")
}

func TestLintWithUserConfig(t *testing.T) {
	t.Parallel()

	input := test.InputPolicy("p/p.rego", "package p\n\nr := input.foo[_]\n")
	rules := map[string]config.Category{"bugs": {"rule-shadows-builtin": config.Rule{Level: "ignore"}}}

	result := must.Return(NewLinter().
		WithUserConfig(config.Config{Rules: rules}).
		WithInputModules(input).
		Lint(t.Context()))(t)

	testutil.AssertNumViolations(t, 1, result)
	assert.Equal(t, "top-level-iteration", result.Violations[0].Title, "unexpected first violation")
}

func TestLintWithUserConfigTable(t *testing.T) {
	t.Parallel()

	policy := `package p

boo := input.hoo[_]

 opa_fmt := "fail"

or := 1
`
	tests := map[string]struct {
		userConfig      *config.Config
		filename        string
		expViolations   []string
		expLevels       []string
		ignoreFilesFlag []string
		rootDir         string
	}{
		"baseline": {
			filename:      "p/p.rego",
			expViolations: []string{"top-level-iteration", "rule-shadows-builtin", "opa-fmt"},
		},
		"ignore rule": {
			userConfig: &config.Config{Rules: map[string]config.Category{
				"bugs":  {"rule-shadows-builtin": config.Rule{Level: "ignore"}},
				"style": {"opa-fmt": config.Rule{Level: "ignore"}},
			}},
			filename:      "p/p.rego",
			expViolations: []string{"top-level-iteration"},
		},
		"ignore all": {
			userConfig: &config.Config{Defaults: config.Defaults{Global: config.Default{Level: "ignore"}}},
			filename:   "p.rego",
		},
		"ignore all but bugs": {
			userConfig: &config.Config{
				Defaults: config.Defaults{
					Global:     config.Default{Level: "ignore"},
					Categories: map[string]config.Default{"bugs": {Level: "error"}},
				},
				Rules: map[string]config.Category{"bugs": {"rule-shadows-builtin": config.Rule{Level: "ignore"}}},
			},
			filename:      "p/p.rego",
			expViolations: []string{"top-level-iteration"},
		},
		"ignore style, no global default": {
			userConfig: &config.Config{
				Defaults: config.Defaults{Categories: map[string]config.Default{
					"bugs":  {Level: "error"},
					"style": {Level: "ignore"},
				}},
				Rules: map[string]config.Category{"bugs": {"rule-shadows-builtin": config.Rule{Level: "ignore"}}},
			},
			filename:      "p/p.rego",
			expViolations: []string{"top-level-iteration"},
		},
		"set level to warning": {
			userConfig: &config.Config{
				Defaults: config.Defaults{
					Global:     config.Default{Level: "warning"}, // will apply to all but style
					Categories: map[string]config.Default{"style": {Level: "error"}},
				},
				Rules: map[string]config.Category{},
			},
			filename:      "p/p.rego",
			expViolations: []string{"top-level-iteration", "rule-shadows-builtin", "opa-fmt"},
			expLevels:     []string{"warning", "warning", "error"},
		},
		"rule level ignore files": {
			userConfig: &config.Config{Rules: map[string]config.Category{
				"bugs": {"rule-shadows-builtin": config.Rule{
					Level:  "error",
					Ignore: &config.Ignore{Files: []string{"p/p.rego"}},
				}},
				"style": {"opa-fmt": config.Rule{
					Level:  "error",
					Ignore: &config.Ignore{Files: []string{"p/p.rego"}},
				}},
			}},
			filename:      "p/p.rego",
			expViolations: []string{"top-level-iteration"},
		},
		"user config global ignore files": {
			userConfig: &config.Config{Ignore: config.Ignore{Files: []string{"p.rego"}}},
			filename:   "p.rego",
		},
		"user config global ignore files with rootDir": {
			userConfig:    &config.Config{Ignore: config.Ignore{Files: []string{"foo/*"}}},
			filename:      "file:///wow/foo/p.rego",
			expViolations: []string{},
			rootDir:       "file:///wow",
		},
		"user config global ignore files with rootDir, not ignored": {
			userConfig: &config.Config{Ignore: config.Ignore{Files: []string{"bar/*"}}},
			filename:   "file:///wow/foo/p.rego",
			expViolations: []string{
				"top-level-iteration", "rule-shadows-builtin", "directory-package-mismatch", "opa-fmt",
			},
			rootDir: "file:///wow",
		},
		"CLI flag ignore files": {
			filename:        "p.rego",
			ignoreFilesFlag: []string{"p.rego"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			linter := NewLinter().
				WithPathPrefix(tc.rootDir).
				WithIgnore(tc.ignoreFilesFlag).
				WithInputModules(test.InputPolicy(tc.filename, policy))

			if tc.userConfig != nil {
				linter = linter.WithUserConfig(*tc.userConfig)
			}

			result := must.Return(linter.Lint(t.Context()))(t)

			testutil.AssertNumViolations(t, len(tc.expViolations), result)

			for idx, violation := range result.Violations {
				assert.Equal(t, tc.expViolations[idx], violation.Title, "unexpected violation at index %d", idx)
			}

			if len(tc.expLevels) > 0 {
				must.Equal(t, len(tc.expLevels), len(result.Violations), "number of levels")

				for idx, violation := range result.Violations {
					assert.Equal(t, tc.expLevels[idx], violation.Level, "unexpected level at index %d", idx)
				}
			}
		})
	}
}

func TestLintWithCustomRule(t *testing.T) {
	t.Parallel()

	result := must.Return(NewLinter().
		WithCustomRulesPaths(filepath.Join("testdata", "custom.rego")).
		WithInputModules(test.InputPolicy("p/p.rego", "package p\n\nimport rego.v1\n")).
		Lint(t.Context()))(t)

	testutil.AssertNumViolations(t, 1, result)
	assert.Equal(t, "acme-corp-package", result.Violations[0].Title, "unexpected first violation")
}

func TestLintWithErrorInEnable(t *testing.T) {
	t.Parallel()

	_, err := NewLinter().
		WithCustomRulesPaths(filepath.Join("testdata", "custom.rego")).
		WithEnabledRules("foo").
		WithInputModules(test.InputPolicy("p/p.rego", "package p")).
		Lint(t.Context())

	testutil.ErrMustContain(err, "unknown rules: [foo]")(t)
}

//go:embed testdata/*
var testLintWithCustomEmbeddedRulesFS embed.FS

func TestLintWithCustomEmbeddedRules(t *testing.T) {
	t.Parallel()

	result := must.Return(NewLinter().
		WithCustomRulesFromFS(testLintWithCustomEmbeddedRulesFS, "testdata").
		WithInputModules(test.InputPolicy("p/p.rego", "package p\n\nimport rego.v1\n")).
		Lint(t.Context()))(t)

	testutil.AssertNumViolations(t, 1, result)
	assert.Equal(t, "acme-corp-package", result.Violations[0].Title, "unexpected first violation")
}

func TestLintWithCustomRuleAndCustomConfig(t *testing.T) {
	t.Parallel()

	linter := NewLinter().
		WithUserConfig(config.Config{Rules: map[string]config.Category{
			"naming": {"acme-corp-package": config.Rule{Level: "ignore"}},
		}}).
		WithCustomRulesPaths(filepath.Join("testdata", "custom.rego")).
		WithInputModules(test.InputPolicy("p/p.rego", "package p\n\nimport rego.v1\n"))

	testutil.AssertNumViolations(t, 0, must.Return(linter.Lint(t.Context()))(t))
}

func TestLintMergedConfigInheritsLevelFromProvided(t *testing.T) {
	t.Parallel()

	// Note that the user configuration does not provide a level
	linter := NewLinter().
		WithUserConfig(config.Config{Rules: map[string]config.Category{
			"style": {"file-length": config.Rule{Extra: config.ExtraAttributes{"max-file-length": 1}}},
		}}).
		WithInputModules(test.InputPolicy("p.rego", "package p\n\nx := 1\n"))

	// Since no level was provided, "error" should be inherited from the provided configuration for the rule
	mergedRules := must.Return(linter.GetConfig())(t).Rules
	assert.Equal(t, "error", mergedRules["style"]["file-length"].Level, "unexpected level for file-length rule")

	// Ensure the extra attributes are still there.
	fileLength := mergedRules["style"]["file-length"].Extra["max-file-length"]
	assert.Equal(t, 1, fileLength, "unexpected max-file-length extra attribute")
}

func TestLintMergedConfigUsesProvidedDefaults(t *testing.T) {
	t.Parallel()

	userConfig := config.Config{
		Defaults: config.Defaults{
			Global: config.Default{Level: "ignore"},
			Categories: map[string]config.Default{
				"style": {Level: "error"},
				"bugs":  {Level: "warning"},
			},
		},
		Rules: map[string]config.Category{"style": {"opa-fmt": config.Rule{Level: "warning"}}},
	}

	mergedConfig := must.Return(NewLinter().
		WithUserConfig(userConfig).
		WithInputModules(test.InputPolicy("p.rego", `package p`)).
		GetConfig())(t)

	// specifically configured rule should not be affected by the default
	assert.Equal(t, "warning", mergedConfig.Rules["style"]["opa-fmt"].Level)

	// other rule in style should have the default level for the category
	assert.Equal(t, "error", mergedConfig.Rules["style"]["pointless-reassignment"].Level)

	// rule in bugs should have the default level for the category
	assert.Equal(t, "warning", mergedConfig.Rules["bugs"]["constant-condition"].Level)

	// rule in unconfigured category should have the global default level
	assert.Equal(t, "ignore", mergedConfig.Rules["imports"]["avoid-importing-input"].Level)
}

func TestLintWithPrintHook(t *testing.T) {
	t.Parallel()

	var bb bytes.Buffer

	must.Return(NewLinter().
		WithCustomRulesPaths(filepath.Join("testdata", "printer.rego")).
		WithPrintHook(topdown.NewPrintHook(&bb)).
		WithInputModules(test.InputPolicy("p.rego", "package p")).
		Lint(t.Context()))(t)

	assert.Equal(t, "p.rego\n", bb.String(), "unexpected print hook output")
}

func TestLintWithAggregateRule(t *testing.T) {
	t.Parallel()

	policies := make(map[string]string, 2)
	policies["foo.rego"] = `package foo
		import data.bar

		default allow := false
	`
	policies["bar.rego"] = `package bar
		import data.foo.allow
	`

	result := must.Return(NewLinter().
		WithDisableAll(true).
		WithPrintHook(topdown.NewPrintHook(t.Output())).
		WithEnabledRules("prefer-package-imports").
		WithInputModules(new(rules.NewInput(policies, util.MapValues(policies, parse.MustParseModule)))).
		Lint(t.Context()))(t)

	testutil.AssertNumViolations(t, 1, result)

	violation := result.Violations[0]

	assert.Equal(t, "prefer-package-imports", violation.Title, "unexpected violation")
	assert.Equal(t, 2, violation.Location.Row, "unexpected line number")
	assert.Equal(t, 3, violation.Location.Column, "unexpected column number")
	assert.Equal(t, "import data.foo.allow", *violation.Location.Text, "unexpected location text")
}

func TestEnabledRules(t *testing.T) {
	t.Parallel()

	enabledRules, _, err := NewLinter().
		WithDisableAll(true).
		WithEnabledRules("opa-fmt", "no-whitespace-comment").
		DetermineEnabledRules(t.Context())

	testutil.NoErr(err)(t)
	assert.Equal(t, 2, len(enabledRules), "enabled rules")
	assert.Equal(t, "no-whitespace-comment", enabledRules[0], "first enabled rule")
	assert.Equal(t, "opa-fmt", enabledRules[1], "second enabled rule")
}

func TestEnabledRulesWithConfig(t *testing.T) {
	t.Parallel()

	config := testutil.MustUnmarshalYAML[config.Config](t, []byte(`
rules:
  style:
    opa-fmt:
      level: ignore # go rule
  imports:
    unresolved-import: # agg rule
      level: ignore
  idiomatic:
    directory-package-mismatch: # non agg rule
      level: ignore
`))
	enabledRules, enabledAggRules, err := NewLinter().WithUserConfig(config).DetermineEnabledRules(t.Context())

	must.Equal(t, nil, err, "unexpected error")
	must.NotEqual(t, 0, len(enabledRules), "enabled aggregate rules")
	assert.False(t, slices.Contains(enabledRules, "directory-package-mismatch"))
	assert.False(t, slices.Contains(enabledRules, "opa-fmt"))
	assert.False(t, slices.Contains(enabledAggRules, "unresolved-import"))
}

func TestEnabledAggregateRules(t *testing.T) {
	t.Parallel()

	_, enabledRules, err := NewLinter().
		WithDisableAll(true).
		WithEnabledRules("opa-fmt", "unresolved-import", "use-assignment-operator").
		DetermineEnabledRules(t.Context())

	must.Equal(t, nil, err, "unexpected error")
	assert.SlicesEqual(t, []string{"unresolved-import"}, enabledRules, "unexpected enabled aggregate rules")
}

func TestLintWithCollectQuery(t *testing.T) {
	t.Parallel()

	result := must.Return(NewLinter().
		WithDisableAll(true).
		WithEnabledRules("unresolved-import").
		WithCollectQuery(true).     // needed since we have a single file input
		WithExportAggregates(true). // needed to be able to test the aggregates are set
		WithInputModules(test.InputPolicy("p.rego", "package p\n\nimport data.foo.bar.unresolved\n")).
		Lint(t.Context()))(t)

	must.Equal(t, 1, result.Aggregates.Len(), "aggregates count")

	_, err := result.Aggregates.Find(util.Map([]string{"p.rego", "imports/unresolved-import"}, ast.InternedTerm))

	assert.Equal(t, nil, err, "expected aggregates to contain 'p.rego/imports/unresolved-import'")
}

func TestLintWithCollectQueryAndAggregates(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"foo.rego": "package foo\n\nimport data.unresolved",
		"bar.rego": "package foo\n\nimport data.unresolved",
		"baz.rego": "package foo\n\nimport data.unresolved",
	}

	var ok bool

	allAggregates := ast.NewObject()

	for file, content := range files {
		linter := NewLinter().
			WithDisableAll(true).
			WithEnabledRules("unresolved-import").
			WithCollectQuery(true). // runs collect for a single file input
			WithExportAggregates(true).
			WithInputModules(test.InputPolicy(file, content))

		if allAggregates, ok = allAggregates.Merge(must.Return(linter.Lint(t.Context()))(t).Aggregates); !ok {
			t.Fatalf("failed to merge aggregates for file %s", file)
		}
	}

	result := must.Return(NewLinter().
		WithDisableAll(true).
		WithEnabledRules("unresolved-import").
		WithAggregates(allAggregates).
		Lint(t.Context()))(t)

	testutil.AssertNumViolations(t, 3, result)

	foundFiles := make([]string, 0, 3)

	for _, v := range result.Violations {
		assert.Equal(t, "unresolved-import", v.Title, "title")
		foundFiles = append(foundFiles, v.Location.File)
	}

	assert.SlicesEqual(t, []string{"bar.rego", "baz.rego", "foo.rego"}, util.Sorted(foundFiles), "files")
}
