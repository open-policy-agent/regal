package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
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
				Text:   new("package illegal"),
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
				Text:   new("default allow = true"),
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
	must.Equal(t, nil, NewPrettyReporter(&buf).Publish(t.Context(), rep))
	assert.Equal(t, must.ReadFile(t, "testdata/pretty/reporter.txt"), buf.String(), "pretty output")
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
		Violations: []report.Violation{{
			Title:       "long-violation",
			Description: "violation with a long description",
			Category:    "long",
			Location: report.Location{
				File:   "b.rego",
				Row:    22,
				Column: 18,
				Text:   new(strings.Repeat("long,", 1000)),
			},
			RelatedResources: []report.RelatedResource{{
				Description: "documentation",
				Reference:   "https://example.com/to-long",
			}},
			Level: "warning",
		}},
	}

	var buf bytes.Buffer
	must.Equal(t, nil, NewPrettyReporter(&buf).Publish(t.Context(), longRep))
	assert.Equal(t, must.ReadFile(t, "testdata/pretty/reporter-long-text.txt"), buf.String(), "pretty output")
}

func TestPrettyReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewPrettyReporter(&buf).Publish(t.Context(), report.Report{}))
	assert.Equal(t, "0 files linted. No violations found.\n", buf.String(), "pretty output")
}

func TestCompactReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewCompactReporter(&buf).Publish(t.Context(), rep))

	expect := `┌──────────────┬──────────────────────────────┐
│   LOCATION   │         DESCRIPTION          │
├──────────────┼──────────────────────────────┤
│ a.rego:1:1   │ Rego must not break the law! │
│ b.rego:22:18 │ Questionable decision found  │
└──────────────┴──────────────────────────────┘
 3 files linted , 2 violations found.
`
	assert.Equal(t, expect, buf.String(), "compact output")
}

func TestCompactReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewCompactReporter(&buf).Publish(t.Context(), report.Report{}))
	assert.Equal(t, "\n", buf.String(), "compact output")
}

func TestJSONReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewJSONReporter(&buf).Publish(t.Context(), rep))
	assert.Equal(t, must.ReadFile(t, "testdata/json/reporter.json"), buf.String(), "json output")
}

func TestJSONReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewJSONReporter(&buf).Publish(t.Context(), report.Report{}))
	assert.Equal(t, must.ReadFile(t, "testdata/json/reporter-no-violations.json"), buf.String(), "json output")
}

func TestGitHubReporterPublish(t *testing.T) {
	// Can't use t.Parallel() here because t.Setenv() forbids that
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	var buf bytes.Buffer
	must.Equal(t, nil, NewGitHubReporter(&buf).Publish(t.Context(), rep))

	if expectTable := "Rule:           breaking-the-law"; !strings.Contains(buf.String(), expectTable) {
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
	must.Equal(t, nil, NewGitHubReporter(&buf).Publish(t.Context(), report.Report{}))
	assert.Equal(t, "0 files linted. No violations found.\n", buf.String(), "GitHub reporter output")
}

func TestSarifReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewSarifReporter(&buf).Publish(t.Context(), rep))
	assert.Equal(t, must.ReadFile(t, "testdata/sarif/reporter.json"), buf.String(), "sarif output")
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
			Reference:   "https://www.openpolicyagent.org/projects/regal/rules/style/opa-fmt",
		}},
		Level: "error",
	}}}

	var buf bytes.Buffer
	must.Equal(t, nil, NewSarifReporter(&buf).Publish(t.Context(), rep))

	if diff := cmp.Diff(must.ReadFile(t, "testdata/sarif/reporter-no-region.json"), buf.String()); diff != "" {
		t.Errorf("unexpected output (-want, +got):\n%s", diff)
	}
}

func TestSarifReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewSarifReporter(&buf).Publish(t.Context(), report.Report{}))

	if diff := cmp.Diff(must.ReadFile(t, "testdata/sarif/reporter-no-violation.json"), buf.String()); diff != "" {
		t.Errorf("unexpected output (-want, +got):\n%s", diff)
	}
}

func TestJUnitReporterPublish(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewJUnitReporter(&buf).Publish(t.Context(), rep))

	if diff := cmp.Diff(must.ReadFile(t, "testdata/junit/reporter.xml"), buf.String()); diff != "" {
		t.Errorf("unexpected output (-want, +got):\n%s", diff)
	}
}

func TestJUnitReporterPublishNoViolations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewJUnitReporter(&buf).Publish(t.Context(), report.Report{}))
	assert.Equal(t, "<testsuites name=\"regal\"></testsuites>\n", buf.String(), "JUnit output")
}

func TestJUnitReporterPublishViolationWithoutText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	must.Equal(t, nil, NewJUnitReporter(&buf).Publish(t.Context(), report.Report{
		Violations: []report.Violation{{Title: "no-text"}},
	}))
}
