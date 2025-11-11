package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

// ParserOptions provides parser options with annotation processing. JSONOptions are not included,
// as it is assumed that the caller will marshal the AST to JSON with the roast encoder rather than
// encoding/json (and consequently, the OPA marshaller implementations).
func ParserOptions() ast.ParserOptions {
	return ast.ParserOptions{
		ProcessAnnotation: true,
		// If not provided, OPA's parser will call ast.CapabilitiesForCurrentVersion()
		// on each Parse() call, which is a waste of resources as it builds the whole
		// structure from scratch each time. That should probably be fixed in OPA, but
		// we do this here for now.
		Capabilities: rio.Capabilities(),
	}
}

var attemptVersionOrder = [2]ast.RegoVersion{ast.RegoV1, ast.RegoV0}

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
	return ast.MustParseModuleWithOpts(policy, ParserOptions())
}

// Module works like ast.ParseModule but with the Regal parser options applied.
// Note that this function will parse using the RegoV1 parser version. If the version of
// the policy is unknown, use ModuleUnknownVersionWithOpts instead.
func Module(filename, policy string) (*ast.Module, error) {
	return util.Wrap(ast.ParseModuleWithOpts(filename, policy, ParserOptions()))("failed to parse module")
}

// PrepareAST prepares the AST to be used as linter input.
//
// Deprecated: New code should use the `transform` package from roast, as this avoids an
// expensive intermediate step in module -> ast.Value conversions.
func PrepareAST(name string, content string, module *ast.Module) (preparedAST map[string]any, err error) {
	if err = encoding.JSONRoundTrip(module, &preparedAST); err != nil {
		return nil, fmt.Errorf("JSON rountrip failed for module: %w", err)
	}

	abs, _ := filepath.Abs(name)

	preparedAST["regal"] = map[string]any{
		"file": map[string]any{
			"name":         name,
			"lines":        strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n"),
			"abs":          abs,
			"rego_version": module.RegoVersion().String(),
		},
		"environment": map[string]any{
			"path_separator": string(os.PathSeparator),
		},
	}

	return preparedAST, nil
}
