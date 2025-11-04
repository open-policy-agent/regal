package report

import (
	"fmt"
	"slices"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

// RelatedResource provides documentation on a violation.
type RelatedResource struct {
	Description string `json:"description"`
	Reference   string `json:"ref"`
}

type Position struct {
	Row    int `json:"row"`
	Column int `json:"col"`
}

// Location provides information on the location of a violation.
// End attribute added in v0.24.0 and ideally we'd have a Start attribute the same way.
// But as opposed to adding an optional End attribute, changing the structure of the existing
// struct would break all existing API clients.
type Location struct {
	End    *Position `json:"end,omitempty"`
	Text   *string   `json:"text,omitempty"`
	File   string    `json:"file"`
	Column int       `json:"col"`
	Row    int       `json:"row"`
	Offset int       `json:"offset,omitempty"`
}

// Violation describes any violation found by Regal.
type Violation struct {
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Category         string            `json:"category"`
	Level            string            `json:"level"`
	RelatedResources []RelatedResource `json:"related_resources,omitempty"`
	Location         Location          `json:"location"`
	IsAggregate      bool              `json:"-"`
}

// Notice describes any notice found by Regal.
type Notice struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Level       string `json:"level"`
	Severity    string `json:"severity"`
}

type Summary struct {
	FilesScanned  int `json:"files_scanned"`
	FilesFailed   int `json:"files_failed"`
	RulesSkipped  int `json:"rules_skipped"`
	NumViolations int `json:"num_violations"`
}

// Report aggregate of Violation as returned by a linter run.
type Report struct {
	// We don't have aggregates when publishing the final report (see JSONReporter), so omitempty is needed here
	// to avoid surfacing a null/empty field.
	Aggregates       map[string][]Aggregate         `json:"aggregates,omitempty"`
	Metrics          map[string]any                 `json:"metrics,omitempty"`
	AggregateProfile map[string]ProfileEntry        `json:"-"`
	IgnoreDirectives map[string]map[string][]string `json:"ignore_directives,omitempty"`
	Violations       []Violation                    `json:"violations"`
	Notices          []Notice                       `json:"notices,omitempty"`
	Profile          []ProfileEntry                 `json:"profile,omitempty"`
	Summary          Summary                        `json:"summary"`
}

// ProfileEntry is a single entry of profiling information, keyed by location.
// This data may have been aggregated across multiple runs.
type ProfileEntry struct {
	Location    string `json:"location"`
	TotalTimeNs int64  `json:"total_time_ns"`
	NumEval     int    `json:"num_eval"`
	NumRedo     int    `json:"num_redo"`
	NumGenExpr  int    `json:"num_gen_expr"`
}

// An Aggregate is data collected by some rule while processing a file AST, to be used later by other rules needing a
// global context (i.e. broader than per-file)
// Rule authors are expected to collect the minimum needed data, to avoid performance problems
// while working with large Rego code repositories.
type Aggregate map[string]any

func FromResultSet(resultSet rego.ResultSet, aggregate bool) (r Report, err error) {
	if len(resultSet) != 1 {
		return r, fmt.Errorf("expected 1 item in resultset, got %d", len(resultSet))
	}

	var binding any

	if lb, ok := resultSet[0].Bindings["lint"].(map[string]any); ok {
		binding = lb
		if aggregateBinding, ok := lb["aggregate"]; ok && aggregate {
			binding = aggregateBinding
		}

		if err = encoding.JSONRoundTrip(binding, &r); err != nil {
			err = fmt.Errorf("JSON rountrip failed for bindings: %v %w", binding, err)
		}
	}

	return r, err
}

func (a Aggregate) SourceFile() string {
	source, ok := a["aggregate_source"].(map[string]any)
	if !ok {
		return ""
	}

	file, ok := source["file"].(string)
	if !ok {
		return ""
	}

	return file
}

// IndexKey is the category/title of the rule that generated the aggregate.
// This key is generated in Rego during linting, this function replicates the
// functionality in Go for use in the cache when indexing aggregates.
func (a Aggregate) IndexKey() string {
	rule, ok := a["rule"].(map[string]any)
	if !ok {
		return ""
	}

	cat, ok := rule["category"].(string)
	if !ok {
		return ""
	}

	title, ok := rule["title"].(string)
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s/%s", cat, title)
}

func (r *Report) AddProfileEntries(prof map[string]ProfileEntry) {
	if r.AggregateProfile == nil {
		r.AggregateProfile = map[string]ProfileEntry{}
	}

	for loc, entry := range prof {
		if _, ok := r.AggregateProfile[loc]; !ok {
			r.AggregateProfile[loc] = entry
		} else {
			profCopy := r.AggregateProfile[loc]
			profCopy.NumEval += entry.NumEval
			profCopy.NumRedo += entry.NumRedo
			profCopy.NumGenExpr += entry.NumGenExpr
			profCopy.TotalTimeNs += entry.TotalTimeNs
			r.AggregateProfile[loc] = profCopy
		}
	}
}

func (r *Report) AggregateProfileToSortedProfile(numResults int) {
	r.Profile = make([]ProfileEntry, 0, len(r.AggregateProfile))
	for loc := range r.AggregateProfile {
		r.Profile = append(r.Profile, r.AggregateProfile[loc])
	}

	slices.SortFunc(r.Profile, func(a, b ProfileEntry) int {
		return int(b.TotalTimeNs - a.TotalTimeNs)
	})

	if numResults <= 0 || numResults > len(r.Profile) {
		return
	}

	r.Profile = r.Profile[:numResults]
}

// ViolationsFileCount returns the number of files containing violations.
func (r *Report) ViolationsFileCount() map[string]int {
	fc := map[string]int{}
	for i := range r.Violations {
		fc[r.Violations[i].Location.File]++
	}

	return fc
}

// String shorthand form for a Location.
func (l Location) String() string {
	if l.Row == 0 && l.Column == 0 {
		return l.File
	}

	return fmt.Sprintf("%s:%d:%d", l.File, l.Row, l.Column)
}
