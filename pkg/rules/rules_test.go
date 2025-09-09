package rules

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/parse"
)

func TestInputFromTextWithOptions(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Module      string
		RegoVersion ast.RegoVersion
	}{
		"regov1": {
			Module: `package test
p if { true }`,
			RegoVersion: ast.RegoV1,
		},
		"regov0": {
			Module: `package test
p { true }`,
			RegoVersion: ast.RegoV0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := parse.ParserOptions()
			opts.RegoVersion = tc.RegoVersion

			_, err := InputFromTextWithOptions("p.rego", tc.Module, opts)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestRegoVersionFromVersionsMap(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		VersionsMap     map[string]ast.RegoVersion
		Filename        string
		ExpectedVersion ast.RegoVersion
	}{
		"file has no root in version map": {
			VersionsMap: map[string]ast.RegoVersion{
				"foo/bar": ast.RegoV1,
				"bar":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/baz/qux.rego",
			ExpectedVersion: ast.RegoUndefined,
		},
		"use project value": {
			VersionsMap: map[string]ast.RegoVersion{
				"":        ast.RegoV1, // "" means the project default, rather than a defined root
				"bar":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/baz/qux.rego",
			ExpectedVersion: ast.RegoV1,
		},
		"file has version from current dir": {
			VersionsMap: map[string]ast.RegoVersion{
				"foo":     ast.RegoV1,
				"bar":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/foo/bar.rego",
			ExpectedVersion: ast.RegoV1,
		},
		"file has version from current dir (no leading slash)": {
			VersionsMap: map[string]ast.RegoVersion{
				"foo":     ast.RegoV1,
				"bar":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/foo/bar.rego",
			ExpectedVersion: ast.RegoV1,
		},
		"file has version from parent dir": {
			VersionsMap: map[string]ast.RegoVersion{
				"foo":     ast.RegoV1,
				"bar":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/foo/bar/baz.rego",
			ExpectedVersion: ast.RegoV1,
		},
		"file has version from grandparent dir": {
			VersionsMap: map[string]ast.RegoVersion{
				"foo":     ast.RegoV1,
				"bar":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/foo/bar/baz/qux.rego",
			ExpectedVersion: ast.RegoV1,
		},
		"project roots are subdirs and overlap": {
			VersionsMap: map[string]ast.RegoVersion{
				"foo/bar": ast.RegoV1,
				"foo":     ast.RegoV0,
				"unknown": ast.RegoUndefined,
			},
			Filename:        "/foo/bar/baz/qux.rego",
			ExpectedVersion: ast.RegoV1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actualVersion := RegoVersionFromMap(tc.VersionsMap, filepath.FromSlash(tc.Filename), ast.RegoUndefined)
			if actualVersion != tc.ExpectedVersion {
				t.Errorf("Expected %v, got %v", tc.ExpectedVersion, actualVersion)
			}
		})
	}
}

func TestInputFromMap(t *testing.T) {
	t.Parallel()

	versionsMap := map[string]ast.RegoVersion{
		"/foo/bar": ast.RegoV1,
		"/foo":     ast.RegoV0,
	}

	files := map[string]string{
		"/foo/bar/main.rego": `package main
# v1 syntax is allowed

allow if input.admin
`,
		"/foo/main.rego": `package main
# v0 syntax is allowed

allow[msg] { msg := "hello" }
`,
	}

	input, err := InputFromMap(files, versionsMap)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(input.Modules) != 2 {
		t.Fatalf("Expected 2 modules, got %d", len(input.Modules))
	}
}
