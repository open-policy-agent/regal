package fixer

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/fixer/fileprovider"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/linter"
	"github.com/open-policy-agent/regal/pkg/report"
)

func TestFixer(t *testing.T) {
	t.Parallel()

	policies := map[string]string{
		filepath.FromSlash("/root/main/main.rego"): `package test

allow if {
true #no space
}
deny = true
`,
	}

	memfp := fileprovider.NewInMemoryFileProvider(policies)

	input, err := memfp.ToInput(map[string]ast.RegoVersion{
		filepath.FromSlash("/root/main"): ast.RegoV1,
	})
	if err != nil {
		t.Fatalf("failed to create input: %v", err)
	}

	l := linter.NewLinter().WithEnableAll(true).WithInputModules(&input)

	f := NewFixer().RegisterFixes(fixes.NewDefaultFixes()...).RegisterRoots(filepath.FromSlash("/root")).
		SetRegoVersionsMap(map[string]ast.RegoVersion{filepath.FromSlash("/root/main"): ast.RegoV1})

	fixReport, err := f.Fix(t.Context(), &l, memfp)
	if err != nil {
		t.Fatalf("failed to fix: %v", err)
	}

	expectedFileFixedViolations := map[string][]string{
		// use-assigment-operator is correct in formatting so does not appear.
		filepath.FromSlash("/root/test/main.rego"): {"directory-package-mismatch", "no-whitespace-comment", "opa-fmt"},
	}
	expectedFileContents := map[string]string{
		filepath.FromSlash("/root/test/main.rego"): `package test

allow := true

# no space

deny := true
`,
	}

	if got, exp := fixReport.TotalFixes(), uint(3); got != exp {
		t.Fatalf("expected a total of %d fixes, got %d", exp, got)
	}

	fpFiles, err := memfp.List()
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	for _, file := range fpFiles {
		// check that the content is correct
		expectedContent, ok := expectedFileContents[file]
		_, moved := fixReport.movedFiles[file]

		if !ok && !moved {
			t.Fatalf("unexpected file found in resulting file provider %s", file)
		}

		content, err := memfp.Get(file)
		if err != nil {
			t.Fatalf("failed to get file %s: %v", file, err)
		}

		if content != expectedContent {
			t.Fatalf(
				"unexpected content for %s:\ngot:\n%s---\nexpected:\n%s---",
				file,
				content,
				expectedContent,
			)
		}

		// check that the fixed violations are correct
		fxs := fixReport.FixesForFile(file)

		var fixes []string
		for _, fx := range fxs {
			fixes = append(fixes, fx.Title)
		}

		expectedFixes, ok := expectedFileFixedViolations[file]
		if !ok {
			t.Fatalf("unexpected file was fixed %s", file)
		}

		slices.Sort(expectedFixes)
		slices.Sort(fixes)

		if !slices.Equal(expectedFixes, fixes) {
			t.Fatalf("unexpected fixes for %s:\ngot: %v\nexpected: %v", file, fixes, expectedFixes)
		}
	}
}

func TestFixViolations(t *testing.T) {
	t.Parallel()

	// targeted specific fix for a given violation
	violations := []report.Violation{{
		Title:    "directory-package-mismatch",
		Location: report.Location{File: filepath.FromSlash("root/main.rego")},
	}}

	memfp := fileprovider.NewInMemoryFileProvider(map[string]string{
		filepath.FromSlash("root/main.rego"):         "package foo.bar\n\nallow := true\n",
		filepath.FromSlash("root/foo/bar/main.rego"): "package foo.bar\n\nallow := true\n", // file in correct place
	})

	f := NewFixer().SetOnConflictOperation(OnConflictRename).RegisterFixes(fixes.NewDefaultFixes()...)

	fixReport, err := f.FixViolations(violations, memfp, &config.Config{})
	if err != nil {
		t.Fatalf("failed to fix: %v", err)
	}

	expectedFileFixedViolations := map[string][]string{
		filepath.FromSlash("root/main.rego"):           {"directory-package-mismatch"},
		filepath.FromSlash("root/foo/bar/main_1.rego"): {"directory-package-mismatch"},
		filepath.FromSlash("root/foo/bar/main.rego"):   {}, // no fixes
	}
	expectedFileContents := map[string]string{
		filepath.FromSlash("root/main.rego"):           "package foo.bar\n\nallow := true\n", // old file yet to be deleted
		filepath.FromSlash("root/foo/bar/main_1.rego"): "package foo.bar\n\nallow := true\n",
		filepath.FromSlash("root/foo/bar/main.rego"):   "package foo.bar\n\nallow := true\n", // file in correct place
	}

	if got, exp := fixReport.TotalFixes(), uint(2); got != exp {
		t.Fatalf("expected %d fixes, got %d", exp, got)
	}

	fpFiles, err := memfp.List()
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	for _, file := range fpFiles {
		// check that the content is correct
		expectedContent, ok := expectedFileContents[file]
		if !ok {
			t.Fatalf("unexpected file %s", file)
		}

		content, err := memfp.Get(file)
		if err != nil {
			t.Fatalf("failed to get file %s: %v", file, err)
		}

		if content != expectedContent {
			t.Fatalf("unexpected content for %s:\ngot:\n%s---\nexpected:\n%s---", file, content, expectedContent)
		}

		// check that the fixed violations are correct
		var fixes []string
		for _, fx := range fixReport.FixesForFile(file) {
			fixes = append(fixes, fx.Title)
		}

		expectedFixes, ok := expectedFileFixedViolations[file]
		if !ok {
			t.Fatalf("unexpected file was fixed %s", file)
		}

		slices.Sort(expectedFixes)
		slices.Sort(fixes)

		if !slices.Equal(expectedFixes, fixes) {
			t.Fatalf("unexpected fixes for %s:\ngot: %v\nexpected: %v", file, fixes, expectedFixes)
		}
	}
}
