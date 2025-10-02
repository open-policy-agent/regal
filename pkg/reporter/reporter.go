package reporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/jstemmer/go-junit-report/v2/junit"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/owenrumney/go-sarif/v2/sarif"

	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/mode"
	"github.com/open-policy-agent/regal/internal/novelty"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/fixer"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

// Reporter releases linter reports in a format decided by the implementation.
type Reporter interface {
	// Publish releases a report to any appropriate target
	Publish(context.Context, report.Report) error
}

// PrettyReporter is a Reporter for representing reports as tables.
type PrettyReporter struct {
	out io.Writer
}

// CompactReporter reports violations in a compact table.
type CompactReporter struct {
	out io.Writer
}

// JSONReporter reports violations as JSON.
type JSONReporter struct {
	out io.Writer
}

// GitHubReporter reports violations in a format suitable for GitHub Actions.
type GitHubReporter struct {
	out io.Writer
}

// FestiveReporter reports violations in a format suitable for the holidays.
type FestiveReporter struct {
	out io.Writer
}

// SarifReporter reports violations in the SARIF (https://sarifweb.azurewebsites.net/) format.
type SarifReporter struct {
	out io.Writer
}

// JUnitReporter reports violations in the JUnit XML format
// (https://github.com/junit-team/junit5/blob/main/platform-tests/src/test/resources/jenkins-junit.xsd).
type JUnitReporter struct {
	out io.Writer
}

// NewPrettyReporter creates a new PrettyReporter.
func NewPrettyReporter(out io.Writer) PrettyReporter {
	return PrettyReporter{out: out}
}

// NewCompactReporter creates a new CompactReporter.
func NewCompactReporter(out io.Writer) CompactReporter {
	return CompactReporter{out: out}
}

// NewJSONReporter creates a new JSONReporter.
func NewJSONReporter(out io.Writer) JSONReporter {
	return JSONReporter{out: out}
}

// NewGitHubReporter creates a new GitHubReporter.
func NewGitHubReporter(out io.Writer) GitHubReporter {
	return GitHubReporter{out: out}
}

// NewFestiveReporter creates a new FestiveReporter.
func NewFestiveReporter(out io.Writer) FestiveReporter {
	return FestiveReporter{out: out}
}

// NewSarifReporter creates a new SarifReporter.
func NewSarifReporter(out io.Writer) SarifReporter {
	return SarifReporter{out: out}
}

// NewJUnitReporter creates a new JUnitReporter.
func NewJUnitReporter(out io.Writer) JUnitReporter {
	return JUnitReporter{out: out}
}

// Publish prints a pretty report to the configured output.
func (tr PrettyReporter) Publish(_ context.Context, r report.Report) error {
	table := buildPrettyViolationsTable(r.Violations)

	numsWarning, numsError := 0, 0

	for i := range r.Violations {
		switch r.Violations[i].Level {
		case "warning":
			numsWarning++
		case "error":
			numsError++
		}
	}

	footer := fmt.Sprintf("%d %s linted.", r.Summary.FilesScanned, pluralize("file", r.Summary.FilesScanned))

	if r.Summary.NumViolations == 0 {
		footer += " No violations found."
	} else {
		footer += fmt.Sprintf(" %d %s ", r.Summary.NumViolations, pluralize("violation", r.Summary.NumViolations))

		if numsWarning > 0 {
			footer += fmt.Sprintf("(%d %s, %d %s) found",
				numsError, pluralize("error", numsError), numsWarning, pluralize("warning", numsWarning),
			)
		} else {
			footer += "found"
		}

		if r.Summary.FilesScanned > 1 && r.Summary.FilesFailed > 0 {
			footer += fmt.Sprintf(" in %d %s.", r.Summary.FilesFailed, pluralize("file", r.Summary.FilesFailed))
		} else {
			footer += "."
		}
	}

	if r.Summary.RulesSkipped > 0 {
		footer += fmt.Sprintf(" %d %s skipped:\n", r.Summary.RulesSkipped, pluralize("rule", r.Summary.RulesSkipped))
		for _, notice := range r.Notices {
			if notice.Severity != "none" {
				footer += fmt.Sprintf("- %s: %s\n", notice.Title, notice.Description)
			}
		}
	}

	_, err := fmt.Fprint(tr.out, table+footer+"\n")
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	if !mode.Standalone { // don't bother advertising `regal fix` when not in standalone mode
		return nil
	}

	f := fixer.NewFixer().RegisterFixes(fixes.NewDefaultFixes()...)

	fixableViolations := util.NewSet[string]()

	for i := range r.Violations {
		if fix, ok := f.GetFixForName(r.Violations[i].Title); ok {
			fixableViolations.Add(fix.Name())
		}
	}

	if fixableViolations.Size() > 0 {
		violationKeys := fixableViolations.Items()
		slices.Sort(violationKeys)

		_, err = fmt.Fprintf(
			tr.out,
			`
Hint: %d/%d violations can be automatically fixed (%s)
      Run regal fix --help for more details.
`,
			fixableViolations.Size(),
			r.Summary.NumViolations,
			strings.Join(violationKeys, ", "),
		)
	}

	return err
}

// Publish prints a festive report to the configured output.
func (tr FestiveReporter) Publish(ctx context.Context, r report.Report) error {
	if os.Getenv("CI") == "" && len(r.Violations) == 0 {
		if err := novelty.HappyHolidays(); err != nil {
			return fmt.Errorf("novelty message display failed: %w", err)
		}
	}

	return NewPrettyReporter(tr.out).Publish(ctx, r)
}

func buildPrettyViolationsTable(violations []report.Violation) string {
	sb := &strings.Builder{}
	table := tablewriter.NewTable(sb, tablewriter.WithConfig(tablewriter.Config{
		Row: tw.CellConfig{
			Padding: tw.CellPadding{PerColumn: []tw.Padding{{Right: " "}, tw.PaddingDefault}},
		},
	}), tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
		Borders: tw.BorderNone,
		Settings: tw.Settings{
			Separators: tw.Separators{BetweenRows: tw.Off, BetweenColumns: tw.Off},
		},
	})))

	numViolations := len(violations)

	// Note: it's tempting to use table.SetColumnColor here, but for whatever reason, that requires using
	// table.SetHeader as well, and we don't want a header for this format.

	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// rangeValCopy, but Not performance sensitive
	//nolint:gocritic
	for i, violation := range violations {
		description := red(violation.Description)
		if violation.Level == "warning" {
			description = yellow(violation.Description)
		}

		table.Append([]string{yellow("Rule:"), violation.Title})

		// if there is no support for color, then we show the level in addition
		// so that the level of the violation is still clear
		if color.NoColor {
			table.Append([]string{"Level:", violation.Level})
		}

		table.Append([]string{yellow("Description:"), description})
		table.Append([]string{yellow("Category:"), violation.Category})
		// End location ignored here as it's not too interesting in this format and line:column
		// allows click-to-open.
		table.Append([]string{yellow("Location:"), cyan(violation.Location.String())})

		if violation.Location.Text != nil {
			if len(*violation.Location.Text) > 117 {
				table.Append([]string{yellow("Text:"), (*violation.Location.Text)[:117] + "..."})
			} else {
				table.Append([]string{yellow("Text:"), strings.TrimSpace(*violation.Location.Text)})
			}
		}

		table.Append([]string{yellow("Documentation:"), cyan(getDocumentationURL(violation))})

		if i+1 < numViolations {
			table.Append([]string{""})
		}
	}

	end := ""
	if numViolations > 0 {
		end = "\n"
	}

	table.Render()

	return sb.String() + end
}

// Publish prints a compact report to the configured output.
func (tr CompactReporter) Publish(_ context.Context, r report.Report) error {
	if len(r.Violations) == 0 {
		_, err := fmt.Fprintln(tr.out)

		return err
	}

	sb := &strings.Builder{}
	table := tablewriter.NewTable(sb, tablewriter.WithConfig(tablewriter.Config{
		Row: tw.CellConfig{
			Formatting:   tw.CellFormatting{AutoWrap: tw.WrapNormal},
			Alignment:    tw.CellAlignment{Global: tw.AlignLeft},
			ColMaxWidths: tw.CellWidth{Global: 80},
		},
	}))

	table.Header([]string{"Location", "Description"})

	for i := range r.Violations {
		table.Append([]string{r.Violations[i].Location.String(), r.Violations[i].Description})
	}

	summary := fmt.Sprintf("%d %s linted , %d %s found.",
		r.Summary.FilesScanned, pluralize("file", r.Summary.FilesScanned),
		r.Summary.NumViolations, pluralize("violation", r.Summary.NumViolations))

	table.Render()

	_, err := fmt.Fprintln(tr.out, strings.TrimSuffix(sb.String(), ""), summary)

	return err
}

// Publish prints a JSON report to the configured output.
func (tr JSONReporter) Publish(_ context.Context, r report.Report) error {
	r.Violations = util.NullToEmpty(r.Violations)

	bs, err := encoding.JSON().MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshalling of report failed: %w", err)
	}

	_, err = fmt.Fprintln(tr.out, outil.ByteSliceToString(bs))

	return err
}

// Publish first prints the pretty formatted report to console for easy access in the logs. It then goes on
// to print the GitHub Actions annotations for each violation. Finally, it prints a summary of the report suitable
// for the GitHub Actions UI.
func (tr GitHubReporter) Publish(ctx context.Context, r report.Report) error {
	if err := NewPrettyReporter(tr.out).Publish(ctx, r); err != nil {
		return err
	}

	r.Violations = util.NullToEmpty(r.Violations)

	for _, violation := range r.Violations { //nolint:gocritic
		if _, err := fmt.Fprintf(tr.out,
			"::%s file=%s,line=%d,col=%d::%s\n",
			violation.Level,
			violation.Location.File,
			violation.Location.Row,
			violation.Location.Column,
			fmt.Sprintf("%s. To learn more, see: %s", violation.Description, getDocumentationURL(violation)),
		); err != nil {
			return err
		}
	}

	// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#adding-a-job-summary
	if summaryFileLoc, ok := os.LookupEnv("GITHUB_STEP_SUMMARY"); ok && summaryFileLoc != "" {
		summaryFile, err := os.OpenFile(summaryFileLoc, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}

		defer func() {
			_ = summaryFile.Close()
		}()

		fmt.Fprintf(summaryFile, "### Regal Lint Report\n\n")
		fmt.Fprintf(summaryFile, "%d %s linted.", r.Summary.FilesScanned, pluralize("file", r.Summary.FilesScanned))

		if r.Summary.NumViolations == 0 {
			fmt.Fprintf(summaryFile, " No violations found")
		} else {
			fmt.Fprintf(summaryFile, " %d %s found", r.Summary.NumViolations, pluralize("violation", r.Summary.NumViolations))

			if r.Summary.FilesScanned > 1 && r.Summary.FilesFailed > 0 {
				fmt.Fprintf(summaryFile, " in %d %s.", r.Summary.FilesFailed, pluralize("file", r.Summary.FilesFailed))
				fmt.Fprintf(summaryFile, " See Files tab in PR for locations and details.\n\n")

				fmt.Fprintf(summaryFile, "#### Violations\n\n")

				for description, url := range getUniqueViolationURLs(r.Violations) {
					fmt.Fprintf(summaryFile, "* [%s](%s)\n", description, url)
				}
			}
		}
	}

	return nil
}

// Publish prints a SARIF report to the configured output.
func (tr SarifReporter) Publish(_ context.Context, r report.Report) error {
	rep, err := sarif.New(sarif.Version210)
	if err != nil {
		return err
	}

	run := sarif.NewRunWithInformationURI("Regal", "https://docs.styra.com/regal")

	for _, violation := range r.Violations { //nolint:gocritic
		pb := sarif.NewPropertyBag()
		pb.Add("category", violation.Category)

		run.AddRule(violation.Title).
			WithDescription(violation.Description).
			WithHelpURI(getDocumentationURL(violation)).
			WithProperties(pb.Properties)

		run.AddDistinctArtifact(violation.Location.File)

		run.CreateResultForRule(violation.Title).
			WithLevel(violation.Level).
			WithMessage(sarif.NewTextMessage(violation.Description)).
			AddLocation(getLocation(violation))
	}

	for _, notice := range r.Notices {
		if notice.Severity == "none" {
			// no need to report on notices like rules skipped due to
			// having been deprecated or made obsolete
			continue
		}

		pb := sarif.NewPropertyBag()
		pb.Add("category", notice.Category)

		run.AddRule(notice.Title).
			WithDescription(notice.Description).
			WithProperties(pb.Properties)

		run.CreateResultForRule(notice.Title).
			WithKind("informational").
			WithLevel("none").
			WithMessage(sarif.NewTextMessage(notice.Description))
	}

	rep.AddRun(run)

	return rep.PrettyWrite(tr.out)
}

func getLocation(violation report.Violation) *sarif.Location {
	physicalLocation := sarif.NewPhysicalLocation().
		WithArtifactLocation(sarif.NewSimpleArtifactLocation(violation.Location.File))

	region := sarif.NewRegion().WithStartLine(violation.Location.Row).WithStartColumn(violation.Location.Column)
	if violation.Location.End != nil {
		region = region.WithEndLine(violation.Location.End.Row).WithEndColumn(violation.Location.End.Column)
	}

	if violation.Location.Row > 0 && violation.Location.Column > 0 {
		physicalLocation = physicalLocation.WithRegion(region)
	}

	return sarif.NewLocationWithPhysicalLocation(physicalLocation)
}

func getDocumentationURL(violation report.Violation) string {
	for _, resource := range violation.RelatedResources {
		if resource.Description == "documentation" {
			return resource.Reference
		}
	}

	return ""
}

func getUniqueViolationURLs(violations []report.Violation) map[string]string {
	urls := make(map[string]string, len(violations))
	for i := range violations {
		urls[violations[i].Description] = getDocumentationURL(violations[i])
	}

	return urls
}

// Publish prints a JUnit XML report to the configured output.
func (tr JUnitReporter) Publish(_ context.Context, r report.Report) error {
	testSuites := junit.Testsuites{Name: "regal"}

	// group by file & sort by file
	files := make([]string, 0)
	violationsPerFile := map[string][]report.Violation{}

	for _, violation := range r.Violations { //nolint:gocritic
		files = append(files, violation.Location.File)
		violationsPerFile[violation.Location.File] = append(violationsPerFile[violation.Location.File], violation)
	}

	slices.Sort(files)

	for _, file := range files {
		testsuite := junit.Testsuite{Name: file}

		for _, violation := range violationsPerFile[file] { //nolint:gocritic
			text := ""
			if violation.Location.Text != nil {
				text = strings.TrimSpace(*violation.Location.Text)
			}

			testsuite.AddTestcase(junit.Testcase{
				Name:      fmt.Sprintf("%s/%s: %s", violation.Category, violation.Title, violation.Description),
				Classname: violation.Location.String(),
				Failure: &junit.Result{
					Message: fmt.Sprintf("%s. To learn more, see: %s", violation.Description, getDocumentationURL(violation)),
					Type:    violation.Level,
					Data: fmt.Sprintf("Rule: %s\nDescription: %s\nCategory: %s\nLocation: %s\nText: %s\nDocumentation: %s",
						violation.Title,
						violation.Description,
						violation.Category,
						violation.Location.String(),
						text,
						getDocumentationURL(violation)),
				},
			})
		}

		testSuites.AddSuite(testsuite)
	}

	return testSuites.WriteXML(tr.out)
}

func pluralize(singular string, count int) string {
	if count == 1 {
		return singular
	}

	return singular + "s"
}
