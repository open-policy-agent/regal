package parse

import (
	"regexp"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
)

var (
	simpleRefPattern    = regexp.MustCompile(`^[a-zA-Z.]$`)
	attemptVersionOrder = [2]ast.RegoVersion{ast.RegoV1, ast.RegoV0}
)

// Options provides parser options with annotation processing. JSONOptions are not included,
// as it is assumed that the caller will marshal the AST to JSON with the roast encoder rather than
// encoding/json (and consequently, the OPA marshaller implementations).
func Options() ast.ParserOptions {
	return ast.ParserOptions{
		ProcessAnnotation: true,
		// If not provided, OPA's parser will call ast.CapabilitiesForCurrentVersion()
		// on each Parse() call, which is a waste of resources as it builds the whole
		// structure from scratch each time. That should probably be fixed in OPA, but
		// we do this here for now.
		Capabilities: rio.Capabilities(),
	}
}

// ModuleWithOpts parses a module with the given options. If the Rego version is unknown, the function
// may attempt to run several parser versions to determine the correct version. Setting the Rego version
// in the parser options will skip this step, and is recommended whenever possible.
func ModuleWithOpts(path, policy string, opts ast.ParserOptions) (module *ast.Module, err error) {
	if opts.RegoVersion == ast.RegoUndefined && strings.HasSuffix(path, "_v0.rego") {
		opts.RegoVersion = ast.RegoV0
	}

	parse := ast.ParseModuleWithOpts
	if opts.RegoVersion == ast.RegoUndefined {
		parse = ModuleUnknownVersionWithOpts // We are parsing for an unknown Rego version
	}

	return parse(path, policy, opts)
}

// ModuleUnknownVersionWithOpts attempts to parse a module with an unknown Rego version. The function will
// attempt to parse the module with different parser versions, and determine the version of Rego based on
// which parser was successful. Note that this is not 100% accurate, and the conditions for determining the
// version may change over time. If the version is known beforehand, use ModuleWithOpts instead, and provide
// the target Rego version in the parser options.
func ModuleUnknownVersionWithOpts(filename, policy string, opts ast.ParserOptions) (mod *ast.Module, err error) {
	// Iterate over RegoV1 and RegoV0 in that order
	// If `import rego.v1`` is present in module, RegoV0CompatV1 is used
	for i := range attemptVersionOrder {
		opts.RegoVersion = attemptVersionOrder[i]

		if mod, err = ast.ParseModuleWithOpts(filename, policy, opts); err == nil {
			if hasRegoV1Import(mod.Imports) {
				mod.SetRegoVersion(ast.RegoV0CompatV1)
			} else {
				mod.SetRegoVersion(opts.RegoVersion)
			}

			return mod, nil
		}
	}

	// TODO: We probably need to return the errors from each parse attempt ?
	// as otherwise there could be very skewed error messages..

	return nil, err
}

func hasRegoV1Import(imports []*ast.Import) bool {
	return slices.ContainsFunc(imports, func(imp *ast.Import) bool {
		return ast.RegoV1CompatibleRef.Equal(imp.Path.Value)
	})
}

// MustParseModule works like ast.MustParseModule but with the Regal parser options applied.
func MustParseModule(policy string) *ast.Module {
	return ast.MustParseModuleWithOpts(policy, Options())
}

// Module works like ast.ParseModule but with the Regal parser options applied.
// Note that this function will parse using the RegoV1 parser version. If the version of
// the policy is unknown, use ModuleUnknownVersionWithOpts instead.
func Module(filename, policy string) (*ast.Module, error) {
	return util.Wrap(ast.ParseModuleWithOpts(filename, policy, Options()))("failed to parse module")
}

// Query parses a query string into an ast.Body, using a custom and very efficient
// method for simple dot-delimited paths ("refs") when possible and otherwise falls
// back on the standard parser. This function panics on invalid queries.
func Query(query string) ast.Body {
	if simpleRefPattern.MatchString(query) { // Try cheap parsing if possible
		return RefStringToBody(query)
	}

	return ast.MustParseBody(query)
}

// RefStringToBody converts a simple dot-delimited string path to an ast.Body.
// This is a lightweight alternative to ast.ParseBody that avoids the overhead of parsing,
// and benefits from using interned terms when possible. It is also nowhere near as competent,
// and can only handle simple string paths without vars, numbers, etc. Suitable for use with
// e.g. rego.ParsedQuery and other places where a simple ref is needed. Do *NOT* use the returned
// ast.Body anywhere it might be mutated (like having location data added), as that modifies the
// globally interned terms.
//
// Implementations tested:
// -----------------------
// 333.6 ns/op	     472 B/op	      19 allocs/op - SplitSeq
// 330.7 ns/op	     496 B/op	      16 allocs/op - Split
// 269.1 ns/op	     400 B/op	      15 allocs/op - IndexOf for loop (current).
func RefStringToBody(path string) ast.Body {
	return ast.NewBody(ast.NewExpr(ast.NewTerm(RefStringToRef(path))))
}

// RefStringToRef converts a simple dot-delimited string path to an ast.Ref in the most
// efficient way possible, using interned terms where possible. See RefStringToBody for
// more details on the limitations of this function.
func RefStringToRef(path string) ast.Ref {
	before, after, found := strings.Cut(path, ".")
	terms := append(make([]*ast.Term, 0, strings.Count(path, ".")+1), refHeadTerm(before))

	for found {
		before, after, found = strings.Cut(after, ".")
		terms = append(terms, ast.InternedTerm(before))
	}

	return ast.Ref(terms)
}

func refHeadTerm(name string) *ast.Term {
	switch name {
	case "data":
		return ast.DefaultRootDocument
	case "input":
		return ast.InputRootDocument
	default:
		return ast.VarTerm(name)
	}
}
