package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/report"
)

var rep = report.Report{
	Summary: report.Summary{FilesScanned: 3, NumViolations: 2, FilesFailed: 2, RulesSkipped: 1},
	Violations: []report.Violation{
		{
			Title:       "breaking-the-law",
			Description: "Rego must not break the law!",
			Category:    "legal",
			Location: report.Location{
				File:   "a.rego",
				Row:    1,
				Column: 1,
				Text:   util.Pointer("package illegal"),
				End:    &report.Position{Row: 1, Column: 14},
			},
			RelatedResources: []report.RelatedResource{{
				Description: "documentation",
				Reference:   "https://example.com/illegal",
			}},
			Level: "error",
		},
		{
			Title:       "questionable-decision",
			Description: "Questionable decision found",
			Category:    "really?",
			Location: report.Location{
				File:   "b.rego",
				Row:    22,
				Column: 18,
				Text:   util.Pointer("default allow = true"),
			},
			RelatedResources: []report.RelatedResource{{
				Description: "documentation",
				Reference:   "https://example.com/questionable",
			}},
			Level: "warning",
		},
	},
	Notices: []report.Notice{
		{
			Title:       "rule-made-obsolete",
			Description: "Rule made obsolete by capability foo",
			Category:    "some-category",
			Severity:    "none",
			Level:       "notice",
		},
		{
			Title:       "rule-missing-capability",
			Description: "Rule missing capability bar",
			Category:    "some-category",
			Severity:    "warning",
			Level:       "notice",
		},
	},
}

func TestPrettyReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewPrettyReporter(&buf).Publish(t.Context(), rep))(t)

	if expect := testutil.MustReadFile(t, "testdata/pretty/reporter.txt"); buf.String() != expect {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

func TestPrettyReporterPublishLongText(t *testing.T) {
	t.Parallel()

	longRep := report.Report{
		Summary: report.Summary{
			FilesScanned:  3,
			NumViolations: 1,
			FilesFailed:   0,
			RulesSkipped:  0,
		},
		Violations: []report.Violation{
			{
				Title:       "long-violation",
				Description: "violation with a long description",
				Category:    "long",
				Location: report.Location{
					File:   "b.rego",
					Row:    22,
					Column: 18,
					Text:   util.Pointer(strings.Repeat("long,", 1000)),
				},
				RelatedResources: []report.RelatedResource{{
					Description: "documentation",
					Reference:   "https://example.com/to-long",
				}},
				Level: "warning",
			},
		},
	}

	var buf bytes.Buffer
	testutil.NoErr(NewPrettyReporter(&buf).Publish(t.Context(), longRep))(t)

	if expect := testutil.MustReadFile(t, "testdata/pretty/reporter-long-text.txt"); expect != buf.String() {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

func TestPrettyReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewPrettyReporter(&buf).Publish(t.Context(), report.Report{}))(t)

	if buf.String() != "0 files linted. No violations found.\n" {
		t.Errorf(`expected "0 files linted. No violations found.\n", got %q`, buf.String())
	}
}

func TestCompactReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewCompactReporter(&buf).Publish(t.Context(), rep))(t)

	expect := `+--------------+------------------------------+
|   Location   |         Description          |
+--------------+------------------------------+
| a.rego:1:1   | Rego must not break the law! |
| b.rego:22:18 | Questionable decision found  |
+--------------+------------------------------+
 3 files linted , 2 violations found.
`

	if buf.String() != expect {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

func TestCompactReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewCompactReporter(&buf).Publish(t.Context(), report.Report{}))(t)

	if buf.String() != "\n" {
		t.Errorf("expected %q, got %q", "", buf.String())
	}
}

func TestJSONReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewJSONReporter(&buf).Publish(t.Context(), rep))(t)

	if expect := testutil.MustReadFile(t, "testdata/json/reporter.json"); expect != buf.String() {
		t.Errorf("expected %q, got %q", expect, buf.String())
	}
}

func TestJSONReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewJSONReporter(&buf).Publish(t.Context(), report.Report{}))(t)

	if expect := testutil.MustReadFile(t, "testdata/json/reporter-no-violations.json"); expect != buf.String() {
		t.Errorf("expected %q, got %q", expect, buf.String())
	}
}

func TestGitHubReporterPublish(t *testing.T) {
	// Can't use t.Parallel() here because t.Setenv() forbids that
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	var buf bytes.Buffer
	testutil.NoErr(NewGitHubReporter(&buf).Publish(t.Context(), rep))(t)

	if expectTable := "Rule:         \tbreaking-the-law"; !strings.Contains(buf.String(), expectTable) {
		t.Errorf("expected table output %q, got %q", expectTable, buf.String())
	}

	//nolint:lll
	expectGithub := `::error file=a.rego,line=1,col=1::Rego must not break the law!. To learn more, see: https://example.com/illegal
::warning file=b.rego,line=22,col=18::Questionable decision found. To learn more, see: https://example.com/questionable
`
	if !strings.Contains(buf.String(), expectGithub) {
		t.Errorf("expected workflow command output %q, got %q", expectGithub, buf.String())
	}
}

func TestGitHubReporterPublishNoViolations(t *testing.T) {
	// Can't use t.Parallel() here because t.Setenv() forbids that
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	var buf bytes.Buffer
	if err := NewGitHubReporter(&buf).Publish(t.Context(), report.Report{}); err != nil {
		t.Fatal(err)
	}

	if buf.String() != "0 files linted. No violations found.\n" {
		t.Errorf("expected %q, got %q", "", buf.String())
	}
}

func TestSarifReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewSarifReporter(&buf).Publish(t.Context(), rep))(t)

	if expect := testutil.MustReadFile(t, "testdata/sarif/reporter.json"); buf.String() != expect {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

// https://github.com/open-policy-agent/regal/issues/514
func TestSarifReporterViolationWithoutRegion(t *testing.T) {
	t.Parallel()

	rep := report.Report{Violations: []report.Violation{{
		Title:       "opa-fmt",
		Description: "File should be formatted with `opa fmt`",
		Category:    "style",
		Location:    report.Location{File: "policy.rego"},
		RelatedResources: []report.RelatedResource{{
			Description: "documentation",
			Reference:   "https://docs.styra.com/regal/rules/style/opa-fmt",
		}},
		Level: "error",
	}}}

	var buf bytes.Buffer
	testutil.NoErr(NewSarifReporter(&buf).Publish(t.Context(), rep))(t)

	if expect := testutil.MustReadFile(t, "testdata/sarif/reporter-no-region.json"); buf.String() != expect {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

func TestSarifReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewSarifReporter(&buf).Publish(t.Context(), report.Report{}))(t)

	if expect := testutil.MustReadFile(t, "testdata/sarif/reporter-no-violation.json"); buf.String() != expect {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

func TestJUnitReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewJUnitReporter(&buf).Publish(t.Context(), rep))(t)

	if expect := testutil.MustReadFile(t, "testdata/junit/reporter.xml"); buf.String() != expect {
		t.Errorf("expected %s, got %s", expect, buf.String())
	}
}

func TestJUnitReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewJUnitReporter(&buf).Publish(t.Context(), report.Report{}))(t)

	if expect := "<testsuites name=\"regal\"></testsuites>\n"; buf.String() != expect {
		t.Errorf("expected \n%s, got \n%s", expect, buf.String())
	}
}

func TestJUnitReporterPublishViolationWithoutText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	testutil.NoErr(NewJUnitReporter(&buf).Publish(t.Context(), report.Report{
		Violations: []report.Violation{{Title: "no-text"}},
	}))(t)
}
