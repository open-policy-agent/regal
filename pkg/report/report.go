package report

import (
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

var emptyObject = ast.NewObject()

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
	Aggregates       ast.Object              `json:"aggregates,omitempty"`
	Metrics          map[string]any          `json:"metrics,omitempty"`
	AggregateProfile map[string]ProfileEntry `json:"-"`
	IgnoreDirectives ast.Object              `json:"-"`
	Violations       []Violation             `json:"violations"`
	Notices          []Notice                `json:"notices,omitempty"`
	Profile          []ProfileEntry          `json:"profile,omitempty"`
	Summary          Summary                 `json:"summary"`
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

func FromQueryResult(result ast.Value, aggregate bool) (r Report, err error) {
	obj, ok := result.(ast.Object)
	if !ok {
		return r, fmt.Errorf("expected result to be an object, got %T", result)
	}

	if aggregate {
		if aggObj, ok := rast.GetValue[ast.Object](obj, "aggregate"); ok {
			obj = aggObj
		}
	}

	r = Report{}

	if val, ok := rast.GetValue[ast.Set](obj, "violations"); ok {
		r.Violations = make([]Violation, 0, val.Len())
		for _, v := range val.Slice() {
			if vObj, ok := v.Value.(ast.Object); ok {
				r.Violations = append(r.Violations, violationFromObject(vObj))
			}
		}
	}

	if notices, ok := rast.GetValue[ast.Set](obj, "notices"); ok {
		for notice := range rast.ValuesOfType[ast.Object](notices.Slice()) {
			r.Notices = append(r.Notices, NoticeFromObject(notice))
		}
	}

	// Both aggregates and ignore_directives are internal transport fields passed
	// from the linter to the aggregate report phase. As such, they are best kept
	// as ast.Objects without conversion.
	r.Aggregates = emptyObject
	if val, ok := rast.GetValue[ast.Object](obj, "aggregates"); ok {
		r.Aggregates = val
	}

	if val, ok := rast.GetValue[ast.Object](obj, "ignore_directives"); ok {
		r.IgnoreDirectives = val
	}

	return r, err
}

func (r *Report) AddProfileEntries(prof map[string]ProfileEntry) {
	if r.AggregateProfile == nil {
		r.AggregateProfile = make(map[string]ProfileEntry, len(prof))
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

	if numResults <= 0 || numResults > len(r.Profile) {
		return
	}

	r.Profile = util.SortedFunc(r.Profile, ProfileEntry.ByTotalTimeNs)[:numResults]
}

func (p ProfileEntry) ByTotalTimeNs(other ProfileEntry) int {
	return int(p.TotalTimeNs - other.TotalTimeNs)
}

// ViolationsFileCount returns the number of files containing violations.
func (r *Report) ViolationsFileCount() map[string]int {
	fc := make(map[string]int, len(r.Violations)/2)
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

	buf := append(make([]byte, 0, len(l.File)+10), l.File...)

	return util.ByteSliceToString(util.AppendInt(append(util.AppendInt(append(buf, ':'), l.Row), ':'), l.Column))
}

func (n Notice) ByTitle(other Notice) int {
	return strings.Compare(n.Title, other.Title)
}

func violationFromObject(obj ast.Object) Violation {
	return Violation{
		Title:            rast.GetString(obj, "title"),
		Description:      rast.GetString(obj, "description"),
		Category:         rast.GetString(obj, "category"),
		Level:            rast.GetString(obj, "level"),
		RelatedResources: relatedResourcesValue(obj, "related_resources"),
		Location:         locationValue(obj, "location"),
	}
}

func NoticeFromObject(obj ast.Object) Notice {
	return Notice{
		Title:       rast.GetString(obj, "title"),
		Description: rast.GetString(obj, "description"),
		Category:    rast.GetString(obj, "category"),
		Level:       rast.GetString(obj, "level"),
		Severity:    rast.GetString(obj, "severity"),
	}
}

func LocationFromObject(obj ast.Object) Location {
	l := Location{
		File:   rast.GetString(obj, "file"),
		Row:    rast.GetInt(obj, "row"),
		Column: rast.GetInt(obj, "col"),
	}

	if endObj, ok := rast.GetValue[ast.Object](obj, "end"); ok {
		l.End = &Position{
			Row:    rast.GetInt(endObj, "row"),
			Column: rast.GetInt(endObj, "col"),
		}
	}

	if text := rast.GetString(obj, "text"); text != "" {
		l.Text = &text
	}

	return l
}

func relatedResourcesValue(obj ast.Object, key string) (resources []RelatedResource) {
	if arr, ok := rast.GetValue[*ast.Array](obj, key); ok {
		resources = make([]RelatedResource, 0, arr.Len())
		for i := range arr.Len() {
			term := arr.Elem(i)
			if resObj, ok := term.Value.(ast.Object); ok {
				resources = append(resources, RelatedResource{
					Description: rast.GetString(resObj, "description"),
					Reference:   rast.GetString(resObj, "ref"),
				})
			}
		}
	}

	return resources
}

func locationValue(obj ast.Object, key string) Location {
	if val, ok := rast.GetValue[ast.Object](obj, key); ok {
		return LocationFromObject(val)
	}

	return Location{}
}
