// Package capabilities provides convenient access to OPA capabilities
// definitions that are embedded within Regal.
package capabilities

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/capabilities/embedded"
	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
)

const (
	engineOPA  = "opa"
	engineEOPA = "eopa"

	DefaultURL = "regal:///capabilities/default"
)

var driveLetterPattern = regexp.MustCompile(`^\/[a-zA-Z]:`)

// Lookup attempts to retrieve capabilities from the requested RFC3986
// compliant URL.
//
// If the URL scheme is 'http', 'https', or 'file' then the specified document will
// be retrieved and parsed as JSON using ast.LoadCapabilitiesJSON().
//
// If the URL scheme is 'regal', then Lookup will retrieve the capabilities
// from Regal's embedded capabilities database. The path for the URL is treated
// according to the following rules:
//
// 'regal://capabilities/default' loads the capabilities from
// ast.CapabilitiesForThisVersion().
//
// 'regal://capabilities/{engine}' loads the latest capabilities for the
// specified engine, sorted according to semver. Versions that are not valid
// semver strings are sorted lexicographically, but are always sorted after
// valid semver strings.
//
// 'regal://capabilities/{engine}/{version}' loads the requested capabilities
// version for the specified engine.
func Lookup(ctx context.Context, rawURL string) (*ast.Capabilities, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %w", rawURL, err)
	}

	return LookupURL(ctx, parsedURL)
}

// LookupURL behaves identically to Lookup(), but allows using a pre-parsed
// URL to avoid a needless round-trip through a string.
func LookupURL(ctx context.Context, parsedURL *url.URL) (*ast.Capabilities, error) {
	switch parsedURL.Scheme {
	case "http":
		return lookupWebURL(ctx, parsedURL)
	case "https":
		return lookupWebURL(ctx, parsedURL)
	case "file":
		return lookupFileURL(parsedURL)
	case "regal":
		return lookupEmbeddedURL(parsedURL)
	default:
		return nil, fmt.Errorf("regal URL '%s' has unsupported scheme '%s'", parsedURL.String(), parsedURL.Scheme)
	}
}

func lookupEmbeddedURL(parsedURL *url.URL) (*ast.Capabilities, error) {
	// We need to consider the individual path elements of the URL. It
	// would arguably be more elegant to do this with regex and named
	// capture groups, but I trust the stdlib URL and path splitting
	// implementations more.
	elems := make([]string, 0)
	dir := path.Clean(parsedURL.Path)

	var file string

	for dir != "" {
		// leading and trailing / symbols confuse path.Split()
		dir, file = path.Split(strings.Trim(dir, "/"))
		elems = append(elems, file)
	}

	slices.Reverse(elems)

	if len(elems) < 1 {
		return nil, fmt.Errorf("regal URL '%s' has an empty path", parsedURL.String())
	}

	// The capabilities element should always be present so that if we want
	// to make other regal:// URLs later for other purposes, we don't cross
	// contaminate different subsystems.
	if elems[0] != "capabilities" {
		return nil, fmt.Errorf(
			"regal URL '%s' does not have 'capabilities' as it's first path element "+
				"- did you mean to try to load capabilities from this URL?",
			parsedURL.String(),
		)
	}

	if len(elems) > 3 {
		return nil, fmt.Errorf(
			"regal URL '%s' is malformed (too many path elements), "+
				"expected regal://capabilities/{engine}[/{version}]",
			parsedURL.String(),
		)
	}

	if elems[1] == "default" {
		return ast.CapabilitiesForThisVersion(), nil
	}

	engine := elems[1]

	var version string

	if len(elems) == 3 {
		version = elems[2]
	} else {
		// look up latest version if the caller did not explicitly
		// supply one. This relies on the behavior of List() to
		// sort the versions correctly.
		//
		// Right now, this does mean we are enumerating all of the
		// versions for all engines. Since there are only 2, that's not
		// an issue today. But in future we may need to expand List()
		// so that it filters to only a specific engine or something to
		// that effect.
		versionsList, err := List()
		if err != nil {
			return nil, fmt.Errorf(
				"while processing regal URL '%s', failed to determine the latest version for engine '%s': %w",
				parsedURL.String(),
				engine,
				err,
			)
		}

		versionsForEngine, ok := versionsList[engine]
		if !ok {
			return nil, fmt.Errorf(
				"while processing regal URL '%s', failed to determine "+
					"the latest version for engine '%s': engine not found in embedded database",
				parsedURL.String(),
				engine,
			)
		}

		if len(versionsForEngine) < 1 {
			return nil, fmt.Errorf(
				"while processing regal URL '%s', failed to determine the "+
					"latest version for engine '%s': engine found in embedded "+
					"database but has no versions associated with it",
				parsedURL.String(),
				engine,
			)
		}

		version = versionsForEngine[0]
	}

	switch engine {
	case engineOPA:
		return util.Wrap(ast.LoadCapabilitiesVersion(version))("failed to load capabilities")
	case engineEOPA:
		return util.Wrap(embedded.LoadCapabilitiesVersion(engineEOPA, version))("failed to load capabilities")
	default:
		return nil, fmt.Errorf("engine '%s' not present in embedded capabilities database", engine)
	}
}

func lookupFileURL(parsedURL *url.URL) (*ast.Capabilities, error) {
	// the provided URL's path could be either a windows path or a unix one
	// we must account for both cases by stripping the leading / if found
	path := parsedURL.Path
	if driveLetterPattern.MatchString(path) {
		path = path[1:]
	}

	return io.WithOpen(path, func(fd *os.File) (*ast.Capabilities, error) {
		return util.Wrap(ast.LoadCapabilitiesJSON(fd))("failed to load capabilities")
	})
}

func lookupWebURL(ctx context.Context, parsedURL *url.URL) (*ast.Capabilities, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for URL '%s': %w", parsedURL.String(), err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL '%s': %w", parsedURL.String(), err)
	}
	defer resp.Body.Close()

	return util.Wrap(ast.LoadCapabilitiesJSON(resp.Body))("failed to load capabilities")
}

// semverSort uses the semver library to perform the string comparisons during
// sorting, but cleanly handles invalid semver strings without panicing or
// throwing an error.
func semverSort(stringVersions []string) {
	// For performance, we'll memoize conversion of strings to semver
	// versions.
	vCache := make(map[string]*semver.Version)

	sort.Slice(stringVersions, func(i, j int) bool {
		iStr := stringVersions[i]
		jStr := stringVersions[j]

		iValid := true
		jValid := true

		iVers, ok := vCache[iStr]
		if !ok {
			var err error

			iVers, err = semver.NewVersion(iStr)
			if err != nil {
				iValid = false
			} else {
				vCache[iStr] = iVers
			}
		}

		jVers, ok := vCache[jStr]
		if !ok {
			var err error

			jVers, err = semver.NewVersion(jStr)
			if err != nil {
				jValid = false
			} else {
				vCache[jStr] = jVers
			}
		}

		if iValid && jValid {
			return iVers.LessThan(*jVers)
		}

		// When i is valid semver and j is not, i always sorts first.
		if iValid && !jValid {
			return false
		}

		if !iValid && jValid {
			return true
		}

		// If neither string is valid semver, fall back to normal
		// string comparison.
		return iStr < jStr
	})

	// This sort sorts ascending, but we want descending. I can't figure
	// out how to get sort.Reverse() and sort.Slice() to play nice, and
	// these lists will generally be small anyway.
	slices.Reverse(stringVersions)
}

// List returns a map with keys being Rego engine types, and values being lists
// of capabilities versions present in the embedded capabilities database for
// that version. Versions are sorted descending according to semver (e.g. index
// 0 is the newest version), with version strings that are not valid semver
// versions sorting after all valid versions strings but otherwise being
// compared lexicographically.
func List() (map[string][]string, error) {
	opaCaps, err := ast.LoadCapabilitiesVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to load capabilities due to error: %w", err)
	}

	eopaCaps, err := embedded.LoadCapabilitiesVersions(engineEOPA)
	if err != nil {
		return nil, fmt.Errorf("failed to load capabilities due to error: %w", err)
	}

	semverSort(opaCaps)
	semverSort(eopaCaps)

	return map[string][]string{engineOPA: opaCaps, engineEOPA: eopaCaps}, nil
}
