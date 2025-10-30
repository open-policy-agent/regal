package fixer

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
)

func TestPrettyReporterOutput(t *testing.T) {
	t.Parallel()

	report := NewReport()
	root1, root2 := filepath.FromSlash("/workspace/bundle1"), filepath.FromSlash("/workspace/bundle2")
	inRoot1, inRoot2 := util.FilepathsJoiner(root1), util.FilepathsJoiner(root2)

	report.AddFileFix(inRoot1("policy1.rego"), fixes.FixResult{Title: "rego-v1", Root: root1})
	report.AddFileFix(inRoot2("policy1.rego"), fixes.FixResult{Title: "rego-v1", Root: root2})
	report.AddFileFix(inRoot1("policy1.rego"), fixes.FixResult{Title: "directory-package-mismatch", Root: root1})
	report.AddFileFix(inRoot2("policy1.rego"), fixes.FixResult{Title: "directory-package-mismatch", Root: root2})
	report.AddFileFix(inRoot1("policy3.rego"), fixes.FixResult{Title: "no-whitespace-comment", Root: root1})
	report.AddFileFix(inRoot2("policy3.rego"), fixes.FixResult{Title: "use-assignment-operator", Root: root2})

	report.MergeFixes(inRoot1("main", "policy1.rego"), inRoot1("policy1.rego"))
	report.RegisterOldPathForFile(inRoot1("main", "policy1.rego"), inRoot1("policy1.rego"))

	report.MergeFixes(inRoot2("lib", "policy2.rego"), inRoot2("policy1.rego"))
	report.RegisterOldPathForFile(inRoot2("lib", "policy2.rego"), inRoot2("policy1.rego"))

	var buffer bytes.Buffer
	if err := NewPrettyReporter(&buffer).Report(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := fmt.Sprintf(`6 fixes applied:
In project root: %s
policy1.rego -> %s:
- rego-v1
- directory-package-mismatch
policy3.rego:
- no-whitespace-comment

In project root: %s
policy1.rego -> %s:
- rego-v1
- directory-package-mismatch
policy3.rego:
- use-assignment-operator
`, root1, filepath.FromSlash("main/policy1.rego"), root2, filepath.FromSlash("lib/policy2.rego"))

	if got := buffer.String(); got != expected {
		t.Fatalf("unexpected output:\nexpected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestPrettyReporterOutputWithConflicts(t *testing.T) {
	t.Parallel()

	report := NewReport()
	root := filepath.FromSlash("/workspace/bundle1")
	inRoot := util.FilepathsJoiner(root)

	// not conflicting rename
	report.RegisterOldPathForFile(inRoot("foo", "policy1.rego"), inRoot("baz", "policy1.rego"))
	// conflicting renames
	report.RegisterOldPathForFile(inRoot("foo", "policy1.rego"), inRoot("baz", "policy2.rego"))
	report.RegisterOldPathForFile(inRoot("foo", "policy1.rego"), inRoot("baz.rego"))

	report.RegisterConflictManyToOne(root, inRoot("foo", "policy1.rego"), inRoot("baz", "policy2.rego"))
	report.RegisterConflictManyToOne(root, inRoot("foo", "policy1.rego"), inRoot("baz.rego"))

	// source file conflict, imagine that foo.rego existed already
	report.RegisterOldPathForFile(inRoot("foo.rego"), inRoot("baz.rego"))
	report.RegisterConflictSourceFile(root, inRoot("foo.rego"), inRoot("baz.rego"))

	var buffer bytes.Buffer
	if err := NewPrettyReporter(&buffer).Report(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := fmt.Sprintf(`Source file conflicts:
In project root: %s
Cannot overwrite existing file: foo.rego
- baz.rego

Many to one conflicts:
In project root: %s
Cannot move multiple files to: %s
- baz.rego
- %s
- %s
`, root, root, filepath.FromSlash("foo/policy1.rego"), filepath.FromSlash("baz/policy1.rego"),
		filepath.FromSlash("baz/policy2.rego"),
	)

	if got := buffer.String(); got != expected {
		t.Fatalf("unexpected output:\nexpected:\n%s\ngot:\n%s", expected, got)
	}
}
