//go:build e2e

package e2e

import (
	"fmt"
	"maps"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/mode"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
)

var _cwd = io.Getwd()

func TestCLIUsage(t *testing.T) {
	regal().expectStdout(contains("Available Commands:")).verify(t)
}

func TestLintEmptyDir(t *testing.T) {
	for _, tc := range []struct {
		format string
		check  verifier
	}{
		{
			format: "pretty",
			check:  equals("0 files linted. No violations found.\n"),
		},
		{
			format: "compact",
			check:  equals("\n"),
		},
		{
			format: "json",
			check: all(
				contains(`"violations": []`),
				contains(`"files_scanned": 0`),
				contains(`"files_failed": 0`),
				contains(`"rules_skipped": 0`),
				contains(`"num_violations": 0`),
			),
		},
	} {
		t.Run(tc.format, regal("lint", "--format", tc.format, t.TempDir()).expectStdout(tc.check).test)
	}
}

func TestLintFileFromStdin(t *testing.T) {
	var rep report.Report

	regal("lint", "-f", "json", "-").
		stdinFrom(strings.NewReader("package p\n\nallow = true")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	testutil.AssertOnlyViolations(t, rep, "opa-fmt", "use-assignment-operator")
}

func TestLintNonExistentDir(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "what", "ever")

	regal("lint", nonexistent).
		skip(onCondition(runtime.GOOS == "windows", "skipping on Windows as the error message is different")).
		expectExitCode(1).
		expectStderr(equals(
			"error(s) encountered while linting: errors encountered when reading files to lint: "+
				"failed to filter paths:\nstat %s: no such file or directory\n", nonexistent,
		)).
		verify(t)
}

func TestLintProposeToRunFix(t *testing.T) {
	regal("lint", "--config-file", cwd("e2e_conf.yaml"), cwd("testdata/v0/rule_named_if.rego")).
		skip(onCondition(!mode.Standalone, "test requires regal to be built with the 'regal_standalone' build ta")).
		expectExitCode(3).
		expectStdout(equals(
			"1 file linted. 2 violations found.\n\n" +
				"Hint: 2/2 violations can be automatically fixed (directory-package-mismatch, opa-fmt)\n" +
				"      Run regal fix --help for more details.\n\n",
		)).
		verify(t)
}

func TestLintV1Violations(t *testing.T) {
	var rep report.Report

	regal("lint", "--format", "json", "--config-file", cwd("e2e_conf.yaml"), cwd("testdata/violations")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	ruleNames := util.NewSet[string]()
	excludedRules := util.NewSet(
		"implicit-future-keywords",
		"use-if",
		"use-contains",
		"internal-entrypoint",
		"file-length",
		"rule-named-if",
		"use-rego-v1",
		"deprecated-builtin",
		"import-shadows-import",
	)

	cfg := readProvidedConfig(t)
	for _, category := range cfg.Rules {
		for ruleName, rule := range category {
			if !excludedRules.Contains(ruleName) && rule.Level != "ignore" {
				ruleNames.Add(ruleName)
			}
		}
	}

	violationNames := testutil.ViolationTitles(rep)
	for ruleName := range ruleNames.Diff(violationNames).Values() {
		t.Errorf("expected violation for rule %q", ruleName)
	}
}

func TestLintV0NoRegoV1ImportViolations(t *testing.T) {
	var rep report.Report

	regal("lint", "--format", "json", "--config-file", cwd("testdata/configs/v0.yaml"), cwd("testdata/v0/")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	testutil.AssertContainsViolations(t, rep, "implicit-future-keywords", "use-if", "use-contains")
}

func TestLintV0WithRegoV1ImportViolations(t *testing.T) {
	var rep report.Report

	regal("lint",
		"--format", "json",
		"--config-file", cwd("testdata/configs/v0-with-import-rego-v1.yaml"), cwd("testdata/v0/")).
		skip(onCondition(runtime.GOOS == "windows", "skipping on Windows")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	testutil.AssertContainsViolations(t, rep, "use-if", "use-contains", "use-rego-v1", "rule-named-if")
}

func TestLintFailsNonExistentConfigFile(t *testing.T) {
	expected := "no such file or directory"
	if runtime.GOOS == "windows" {
		expected = "The system cannot find the file specified"
	}

	regal("lint", "--config-file", cwd("testdata/configs/non_existent_test_file.yaml"), cwd("testdata/violations")).
		expectExitCode(1).
		expectStderr(contains(expected)).
		verify(t)
}

func TestLintRuleIgnoreFiles(t *testing.T) {
	var rep report.Report

	regal("lint", "--format", "json", "--config-file", cwd("testdata/configs/ignore_files_prefer_snake_case.yaml"),
		cwd("testdata/violations")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	testutil.AssertNotContainsViolations(t, rep, "prefer-snake-case")
}

func TestLintWithDebugOption(t *testing.T) {
	regal("lint", "--debug", "--config-file", cwd("testdata/configs/ignore_files_prefer_snake_case.yaml"),
		cwd("testdata/violations")).
		expectExitCode(3).
		expectStdout(notEmpty()).
		expectStderr(contains("rules:")).
		verify(t)
}

func TestLintRuleNamingConventionFromCustomCategory(t *testing.T) {
	var rep report.Report

	regal("lint", "--format", "json", "--config-file", cwd("testdata/configs/custom_naming_convention.yaml"),
		cwd("testdata/custom_naming_convention")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	testutil.AssertNumViolations(t, 2, rep)

	expectedViolations := []string{
		`Naming convention violation: package name "custom_naming_convention" does not match pattern '^acmecorp\.[a-z_\.]+$'`,
		`Naming convention violation: rule name "naming_convention_fail" does not match pattern '^_[a-z_]+$|^allow$'`,
	}

	for _, violation := range rep.Violations {
		if !slices.Contains(expectedViolations, violation.Description) {
			t.Errorf("unexpected violation: %s", violation.Description)
		}
	}
}

func TestAggregatesAreCollectedAndUsed(t *testing.T) {
	t.Run("two policies — no violations expected", func(t *testing.T) {
		regal("lint", "--format", "json", "--config-file", cwd("e2e_conf.yaml"), "--disable=unresolved-reference",
			cwd("testdata/aggregates/custom/regal/rules/testcase/aggregates/custom_rules_using_aggregates.rego"),
			cwd("testdata/aggregates/two_policies")).
			expectStdout(notEmpty()). // JSON output is never empty
			verify(t)
	})

	t.Run("single policy — no aggregate violations expected", func(t *testing.T) {
		regal("lint", "--format", "json", "--rules",
			cwd("testdata/aggregates/custom/regal/rules/testcase/aggregates/custom_rules_using_aggregates.rego"),
			cwd("testdata/aggregates/two_policies/policy_1.rego")).
			expectStdout(notEmpty()). // JSON output is never empty
			verify(t)
	})

	t.Run("three policies - violation expected", func(t *testing.T) {
		var rep report.Report

		regal("lint", "--format", "json", "--config-file", cwd("e2e_conf.yaml"), "--rules",
			cwd("testdata/aggregates/custom/regal/rules/testcase/aggregates/custom_rules_using_aggregates.rego"),
			cwd("testdata/aggregates/three_policies")).
			expectExitCode(3).
			expectStdout(unmarshalsTo(&rep)).
			verify(t)

		testutil.AssertNumViolations(t, 1, rep)
	})

	t.Run("custom policy where nothing aggregate is a violation", func(t *testing.T) {
		var rep report.Report

		regal("lint", "--format", "json", "--config-file", cwd("e2e_conf.yaml"), "--rules",
			cwd("testdata/aggregates/custom/regal/rules/testcase/empty_aggregate/"),
			cwd("testdata/aggregates/two_policies")).
			expectExitCode(3).
			expectStdout(unmarshalsTo(&rep)).
			verify(t)

		testutil.AssertNumViolations(t, 1, rep)
	})
}

func TestLintAggregateIgnoreDirective(t *testing.T) {
	var rep report.Report

	regal("lint", "--format", "json", "--config-file", cwd("e2e_conf.yaml"), cwd("testdata/aggregates/ignore_directive")).
		expectExitCode(3).
		expectStdout(unmarshalsTo(&rep)).
		verify(t)

	testutil.AssertNumViolations(t, 2, rep)
	testutil.AssertOnlyViolations(t, rep, "no-defined-entrypoint", "unresolved-import")

	// ensure that it's the file without the ignore directive that has the violation
	if !strings.HasSuffix(rep.Violations[1].Location.File, "second.rego") {
		t.Errorf("expected violation in second.rego, got %q", rep.Violations[1].Location.File)
	}
}

func TestTestRegalBundledBundle(t *testing.T) {
	var res []tester.Result

	regal("test", "--format", "json", cwd("../bundle")).expectStdout(unmarshalsTo(&res)).verify(t)
}

func TestTestRegalBundledRules(t *testing.T) {
	var res []tester.Result

	regal("test", "--format", "json", cwd("testdata/custom_rules")).expectStdout(unmarshalsTo(&res)).verify(t)
}

func TestTestRegalTestWithExtendedASTTypeChecking(t *testing.T) {
	regal("test", cwd("testdata/ast_type_failure")).
		expectExitCode(1).
		expectStderr(
			hasPrefix("1 error occurred: "),
			hasSuffix(
				"rego_type_error: undefined ref: input.foo\n\tinput.foo\n\t      ^\n\t      "+
					"have: \"foo\"\n\t      want (one of): [\"comments\" \"imports\" \"package\" \"regal\" \"rules\"]\n",
			),
		).
		verify(t)
}

// Both of the template creating tests are skipped on Windows for the time being,
// as the "regal test" command fails with a "no such file or directory" error, even
// though the files are seemingly created. Will need to look into this, but these
// tests are not critical.

func TestCreateNewCustomRuleFromTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := join(tmpDir, ".regal/rules/custom/regal/rules/naming/foo-bar-baz")

	r := regal("new", "rule", "--category", "naming", "--name", "foo-bar-baz", "--output", tmpDir).
		skip(onCondition(runtime.GOOS == "windows", "temporarily skipping this test on Windows")).
		expectStdout(hasPrefix(`Created custom rule "foo-bar-baz" in %s`, outDir)).
		expectFiles(
			exists(outDir, "foo_bar_baz.rego"),
			exists(outDir, "foo_bar_baz_test.rego"),
		).
		verify(t)

	r.regal("test", tmpDir).expectStdout(hasPrefix("PASS: 1/1")).verify(t)
}

func TestCreateNewBuiltinRuleFromTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	r := regal("new", "rule", "--type", "builtin", "--category", "naming", "--name", "foo-bar-baz", "--output", tmpDir).
		skip(onCondition(runtime.GOOS == "windows", "temporarily skipping this test on Windows")).
		inDirectory(filepath.Dir(cwd(""))).
		expectStdout(
			contains(`Created builtin rule "foo-bar-baz"`),
			contains(`Created doc template for builtin rule "foo-bar-baz"`),
			contains(`Wrote configuration update`),
		).
		expectFiles(
			exists(tmpDir, "bundle/regal/rules/naming/foo-bar-baz/foo_bar_baz.rego"),
			exists(tmpDir, "bundle/regal/rules/naming/foo-bar-baz/foo_bar_baz.rego"),
			exists(tmpDir, "bundle/regal/rules/naming/foo-bar-baz/foo_bar_baz_test.rego"),
			exists(tmpDir, "bundle/regal/config/provided/data.yaml"),
			exists(tmpDir, "docs/rules/naming/foo-bar-baz.md"),
		).
		verify(t)

	r.regal("test", tmpDir).expectStdout(hasPrefix("PASS: 1/1")).verify(t)
}

func TestMergeRuleConfigWithoutLevel(t *testing.T) {
	// No violations from the built-in configuration in the policy provided, but
	// the user --config-file changes the max-file-length to 1, so this should fail
	regal("lint", "--config-file", cwd("testdata/configs/rule_without_level.yaml"),
		cwd("testdata/custom_naming_convention")).
		expectExitCode(3).
		expectStdout(notEmpty()).
		verify(t)
}

func TestConfigDefaultingWithDisableDirective(t *testing.T) {
	regal("lint", "--disable-category=testing", "--config-file", cwd("testdata/configs/defaulting.yaml"),
		cwd("testdata/defaulting")).
		expectExitCode(3).
		expectStdout(
			notContains("print-or-trace-call"), // ignored by flag ignore directive
			notContains("opa-fmt"),             // ignored by config
			contains("top-level-iteration"),    // this error should not be ignored
		).
		verify(t)
}

func TestConfigDefaultingWithEnableDirective(t *testing.T) {
	regal("lint", "--enable-all", "--config-file", cwd("testdata/configs/defaulting.yaml"), cwd("testdata/defaulting")).
		expectExitCode(3).
		expectStdout(
			contains("print-or-trace-call"), // re-enabled by flag enable directive
			contains("opa-fmt"),             // re-enabled by flag enable directive
			contains("top-level-iteration"), // this error should not be ignored
		).
		verify(t)
}

// Test that the custom-has-key rule is skipped due to the custom capabilities provided where we
// use OPA v0.46.0 as a target (the `object.keys` built-in function was introduced in v0.47.0)
func TestLintWithCustomCapabilitiesAndUnmetRequirement(t *testing.T) {
	regal("lint", "--config-file", cwd("testdata/configs/opa_v46_capabilities.yaml"),
		cwd("testdata/capabilities/custom_has_key.rego")).
		expectStdout(equals(
			"1 file linted. No violations found. 3 rules skipped:\n" +
				"- custom-has-key-construct: Missing capability for built-in function `object.keys`\n" +
				"- use-rego-v1: Missing capability for `import rego.v1`\n" +
				"- use-strings-count: Missing capability for built-in function `strings.count`\n\n",
		)).
		verify(t)
}

func TestLintWithCustomCapabilitiesAndUnmetRequirementMultipleFiles(t *testing.T) {
	regal("lint", "--config-file", cwd("testdata/configs/opa_v46_capabilities.yaml"), cwd("testdata/capabilities/")).
		expectStdout(equals(
			"2 files linted. No violations found. 3 rules skipped:\n" +
				"- custom-has-key-construct: Missing capability for built-in function `object.keys`\n" +
				"- use-rego-v1: Missing capability for `import rego.v1`\n" +
				"- use-strings-count: Missing capability for built-in function `strings.count`\n\n",
		)).
		verify(t)
}

func TestLintPprof(t *testing.T) {
	// this overrides the ignore directives for e2e loaded from the config file
	regal("lint", "--ignore-files=none", "--pprof", "clock", cwd("testdata/violations")).
		cleanup(testutil.RemoveIgnoreErr("clock.pprof")).
		expectExitCode(3).
		expectStdout(notEmpty()).
		expectFiles(exists("clock.pprof")).
		verify(t)
}

func TestFix(t *testing.T) {
	initialState := map[string]string{
		".regal/config.yaml": `
project:
  rego-version: 1
  roots:
    - path: v0
      rego-version: 0
    - path: v1
      rego-version: 1
`,
		"foo/main.rego": `package wow

#comment

allow if {
	true
}
`,
		"foo/main_test.rego": `package wow_test

test_allow if {
	true
}
`,
		"foo/foo.rego": `package foo

# present and correct

allow if {
	input.admin
}
`,
		"bar/main.rego": `package wow["foo-bar"].baz
`,
		"bar/main_test.rego": `package wow["foo-bar"].baz_test
test_allow if {
	true
}
`,
		"v0/main.rego": `package v0

#comment
allow { true }
`,
		"v1/main.rego": `package v1

#comment
allow if { true }
`,
		"unrelated.txt": `foobar`,
	}

	td := testutil.TempDirectoryOf(t, initialState)
	exp := fmt.Sprintf(`12 fixes applied:
In project root: %[1]s
bar/main.rego -> wow/foo-bar/baz/main.rego:
- directory-package-mismatch

bar/main_test.rego -> wow/foo-bar/baz/main_test.rego:
- directory-package-mismatch
- opa-fmt

foo/main.rego -> wow/main.rego:
- directory-package-mismatch
- no-whitespace-comment
- opa-fmt

foo/main_test.rego -> wow/main_test.rego:
- directory-package-mismatch
- opa-fmt


In project root: %[2]s
main.rego:
- use-rego-v1
- no-whitespace-comment

In project root: %[3]s
main.rego:
- no-whitespace-comment
- opa-fmt
`, td, filepath.Join(td, "v0"), filepath.Join(td, "v1"))

	expectedState := map[string]string{
		"foo/foo.rego": `package foo

# present and correct

allow if {
	input.admin
}
`,
		filepath.FromSlash("wow/foo-bar/baz/main.rego"): `package wow["foo-bar"].baz
`,
		filepath.FromSlash("wow/foo-bar/baz/main_test.rego"): `package wow["foo-bar"].baz_test

test_allow := true
`,
		"wow/main.rego": `package wow

# comment

allow := true
`,
		"wow/main_test.rego": `package wow_test

test_allow := true
`,
		"v0/main.rego": `package v0

import rego.v1

# comment
allow := true
`,
		"v1/main.rego": `package v1

# comment
allow := true
`,
		"unrelated.txt": `foobar`,
	}

	// --force is required to make the changes when there is no git repo
	regal("fix", "--force", join(td, "foo"), join(td, "bar"), join(td, "v0"), join(td, "v1")).
		expectStdout(equals(exp)).
		expectFiles(
			notExists(td, "bar"),
			contentMatchesMap(td, expectedState),
		).
		verify(t)
}

func TestFixWithConflicts(t *testing.T) {
	initialState := map[string]string{
		filepath.FromSlash(".regal/config.yaml"):   "",              // needed to find the root in the right place
		filepath.FromSlash("foo/foo.rego"):         "package foo\n", // this file is in the correct location
		filepath.FromSlash("quz/foo.rego"):         "package foo\n", // should be at foo/foo.rego, but that file already exists
		filepath.FromSlash("foo/bar.rego"):         "package bar\n", // should be at bar/bar.rego, but they cannot all be moved there
		filepath.FromSlash("baz/bar.rego"):         "package bar\n", // should be at bar/bar.rego, but they cannot all be moved there
		filepath.FromSlash("bax/foo/wow/bar.rego"): "package bar\n", // should be at bar/bar.rego, but they cannot all be moved there
	}

	td := testutil.TempDirectoryOf(t, initialState)

	// --force is required to make the changes when there is no git repo
	regal("fix", "--force", td).
		expectExitCode(1).
		expectStdout(equals(
			"Source file conflicts:\n"+
				"In project root: %[1]s\n"+
				"Cannot overwrite existing file: foo/foo.rego\n"+
				"- quz/foo.rego\n\n"+
				"Many to one conflicts:\n"+
				"In project root: %[1]s\n"+
				"Cannot move multiple files to: bar/bar.rego\n"+
				"- bax/foo/wow/bar.rego\n"+
				"- baz/bar.rego\n"+
				"- foo/bar.rego\n", td,
		)).
		expectStderr(notEmpty()). // ignore empty config file warning
		expectFiles(contentMatchesMap(td, initialState)).
		verify(t)
}

func TestFixWithConflictRenaming(t *testing.T) {
	initialState := map[string]string{
		".regal/config.yaml": "",                   // needed to find the root in the right place
		"foo/foo.rego":       "package foo\n",      // this file is in the correct location
		"foo/foo_test.rego":  "package foo_test\n", // this file is in the correct location
		"quz/foo.rego":       "package foo\n",      // should be at foo/foo.rego, but that file already exists
		"quz/foo_test.rego":  "package foo_test\n", // should be at foo/foo_test.rego, but that file already exists
		"foo/bar.rego":       "package bar\n",      // this file should be at bar/bar.rego and is not a conflict
	}

	td := testutil.TempDirectoryOf(t, initialState)

	// --force is required to make the changes when there is no git repo
	// --conflict=rename will rename inbound files when there is a conflict
	regal("fix", "--force", "--on-conflict=rename", td).
		expectStdout(equals("3 fixes applied:\n"+
			"In project root: %[1]s\n"+
			"foo/bar.rego -> bar/bar.rego:\n"+
			"- directory-package-mismatch\n"+
			"quz/foo.rego -> foo/foo_1.rego:\n"+
			"- directory-package-mismatch\n"+
			"quz/foo_test.rego -> foo/foo_1_test.rego:\n"+
			"- directory-package-mismatch\n", td,
		)).
		expectStderr(notEmpty()). // ignore empty config file warning
		expectFiles(
			notExists(td, "quz/foo.rego"),
			notExists(td, "quz/foo_test.rego"),
			hasContent(join(td, ".regal/config.yaml"), initialState[".regal/config.yaml"]),
			hasContent(join(td, "foo/foo.rego"), initialState["foo/foo.rego"]),
			hasContent(join(td, "bar/bar.rego"), initialState["foo/bar.rego"]),
			hasContent(join(td, "foo/foo_1.rego"), "package foo\n"),           // renamed to permit its new location
			hasContent(join(td, "foo/foo_1_test.rego"), "package foo_test\n"), // renamed to permit its new location
		).
		verify(t)
}

func TestFixSingleFileNested(t *testing.T) {
	initialState := map[string]string{
		".regal/config.yaml": `
project:
  rego-version: 1
`,
		"foo/.regal.yaml": `
project:
  rego-version: 1
`,
		"foo/foo.rego": "package wow\n",
	}
	td := testutil.TempDirectoryOf(t, initialState)

	expectedState := maps.Clone(initialState)
	expectedState["foo/wow/foo.rego"] = initialState["foo/foo.rego"]
	delete(expectedState, "foo/foo.rego")

	// --force is required to make the changes when there is no git repo
	regal("fix", "--force", join(td, "foo/foo.rego")).
		expectStdout(equals(
			"1 fix applied:\n"+
				"In project root: %[1]s\n"+
				"foo.rego -> wow/foo.rego:\n"+
				"- directory-package-mismatch\n", join(td, "foo"),
		)).
		expectFiles(contentMatchesMap(td, expectedState)).
		verify(t)
}

// verify fix for https://github.com/open-policy-agent/regal/issues/1082
func TestLintAnnotationCustomAttributeMultipleItems(t *testing.T) {
	regal("lint", "--config-file", cwd("e2e_conf.yaml"), "--disable=directory-package-mismatch",
		cwd("testdata/bugs/issue_1082.rego")).
		expectStdout(equals("1 file linted. No violations found.\n")).
		verify(t)
}

func join(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

func cwd(rel string) string {
	return join(_cwd, rel)
}

func readProvidedConfig(t *testing.T) config.Config {
	t.Helper()
	return testutil.Must(config.FromPath(cwd("../bundle/regal/config/provided/data.yaml")))(t)
}

// use for skipping tests on simple conditions, such as OS or build mode.
func onCondition(cond bool, msg string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()
		if cond {
			t.Skip(msg)
		}
	}
}
