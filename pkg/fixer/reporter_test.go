package fixer

import (
	"bytes"
	"testing"

	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
)

func TestPrettyReporterOutput(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	reporter := NewPrettyReporter(&buffer)
	report := NewReport()

	report.AddFileFix("/workspace/bundle1/policy1.rego", fixes.FixResult{
		Title: "rego-v1",
		Root:  "/workspace/bundle1",
	})
	report.AddFileFix("/workspace/bundle2/policy1.rego", fixes.FixResult{
		Title: "rego-v1",
		Root:  "/workspace/bundle2",
	})
	report.AddFileFix("/workspace/bundle1/policy1.rego", fixes.FixResult{
		Title: "directory-package-mismatch",
		Root:  "/workspace/bundle1",
	})
	report.AddFileFix("/workspace/bundle2/policy1.rego", fixes.FixResult{
		Title: "directory-package-mismatch",
		Root:  "/workspace/bundle2",
	})
	report.AddFileFix("/workspace/bundle1/policy3.rego", fixes.FixResult{
		Title: "no-whitespace-comment",
		Root:  "/workspace/bundle1",
	})
	report.AddFileFix("/workspace/bundle2/policy3.rego", fixes.FixResult{
		Title: "use-assignment-operator",
		Root:  "/workspace/bundle2",
	})

	report.MergeFixes(
		"/workspace/bundle1/main/policy1.rego",
		"/workspace/bundle1/policy1.rego",
	)

	report.RegisterOldPathForFile(
		"/workspace/bundle1/main/policy1.rego",
		"/workspace/bundle1/policy1.rego",
	)

	report.MergeFixes(
		"/workspace/bundle2/lib/policy2.rego",
		"/workspace/bundle2/policy1.rego",
	)

	report.RegisterOldPathForFile(
		"/workspace/bundle2/lib/policy2.rego",
		"/workspace/bundle2/policy1.rego",
	)

	if err := reporter.Report(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `6 fixes applied:
In project root: /workspace/bundle1
policy1.rego -> main/policy1.rego:
- rego-v1
- directory-package-mismatch
policy3.rego:
- no-whitespace-comment

In project root: /workspace/bundle2
policy1.rego -> lib/policy2.rego:
- rego-v1
- directory-package-mismatch
policy3.rego:
- use-assignment-operator
`

	if got := buffer.String(); got != expected {
		t.Fatalf("unexpected output:\nexpected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestPrettyReporterOutputWithConflicts(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	reporter := NewPrettyReporter(&buffer)
	report := NewReport()
	root := "/workspace/bundle1"

	// not conflicting rename
	report.RegisterOldPathForFile(
		"/workspace/bundle1/foo/policy1.rego",
		"/workspace/bundle1/baz/policy1.rego",
	)
	// conflicting renames
	report.RegisterOldPathForFile(
		"/workspace/bundle1/foo/policy1.rego",
		"/workspace/bundle1/baz/policy2.rego",
	)
	report.RegisterOldPathForFile(
		"/workspace/bundle1/foo/policy1.rego",
		"/workspace/bundle1/baz.rego",
	)
	report.RegisterConflictManyToOne(
		root,
		"/workspace/bundle1/foo/policy1.rego",
		"/workspace/bundle1/baz/policy2.rego",
	)
	report.RegisterConflictManyToOne(
		root,
		"/workspace/bundle1/foo/policy1.rego",
		"/workspace/bundle1/baz.rego",
	)

	// source file conflict
	report.RegisterOldPathForFile(
		// imagine that foo.rego existed already
		"/workspace/bundle1/foo.rego",
		"/workspace/bundle1/baz.rego",
	)
	report.RegisterConflictSourceFile(
		root,
		"/workspace/bundle1/foo.rego",
		"/workspace/bundle1/baz.rego",
	)

	if err := reporter.Report(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `Source file conflicts:
In project root: /workspace/bundle1
Cannot overwrite existing file: foo.rego
- baz.rego

Many to one conflicts:
In project root: /workspace/bundle1
Cannot move multiple files to: foo/policy1.rego
- baz.rego
- baz/policy1.rego
- baz/policy2.rego
`

	if got := buffer.String(); got != expected {
		t.Fatalf("unexpected output:\nexpected:\n%s\ngot:\n%s", expected, got)
	}
}
