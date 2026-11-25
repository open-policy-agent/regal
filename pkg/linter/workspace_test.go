package linter_test

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/linter"
)

func TestWithWorkspaceLoadsConfigFromRegalDirectory(t *testing.T) {
	t.Parallel()

	root := testutil.TempDirectoryOf(t, map[string]string{
		".regal/config.yaml": "rules:\n  style:\n    opa-fmt:\n      level: ignore\n",
		"a.rego":             "package a\n\nx := 1 # not formatted",
	})

	l, _, _, err := linter.NewLinter().WithWorkspace(root, linter.WorkspaceOptions{})
	must.Equal(t, nil, err)

	report := must.Return(l.WithInputPaths([]string{filepath.Join(root, "a.rego")}).Lint(t.Context()))(t)

	for _, v := range report.Violations {
		if v.Title == "opa-fmt" {
			t.Errorf("expected opa-fmt to be ignored by the config in .regal, got: %v", v)
		}
	}
}

func TestWithWorkspaceReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	root := testutil.TempDirectoryOf(t, map[string]string{".regal/config.yaml": "rules: [not valid"})

	_, cfg, _, err := linter.NewLinter().WithWorkspace(root, linter.WorkspaceOptions{})

	testutil.ErrMustContain(err, "config.yaml")(t)
	assert.Equal(t, true, cfg == nil, "no config returned on error")
}

func TestWithWorkspaceSkipsCustomRulesWhenAsked(t *testing.T) {
	t.Parallel()

	// A rules directory with a module that does not parse stops the linter from preparing, so it shows whether it
	// was used.
	root := testutil.TempDirectoryOf(t, map[string]string{
		".regal/config.yaml":       "rules: {}\n",
		".regal/rules/broken.rego": "package custom.regal.rules.x.y\n\nnot rego {",
	})

	used, _, _, err := linter.NewLinter().WithWorkspace(root, linter.WorkspaceOptions{})
	must.Equal(t, nil, err)

	if _, err := used.Prepare(t.Context()); err == nil {
		t.Error("expected the custom rules in .regal/rules to be used, and to fail to prepare")
	}

	skipped, _, _, err := linter.NewLinter().WithWorkspace(root, linter.WorkspaceOptions{SkipCustomRules: true})
	must.Equal(t, nil, err)

	must.Return(skipped.Prepare(t.Context()))(t)
}
