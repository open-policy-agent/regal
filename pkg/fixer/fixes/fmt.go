package fixes

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/format"

	"github.com/open-policy-agent/regal/internal/parse"
)

type Fmt struct {
	// OPAFmtOpts are the options to pass to OPA's format.SourceWithOpts
	// function.
	OPAFmtOpts format.Opts
	// NameOverride allows this fix config to also be registered under another name, see note
	// in Name().
	NameOverride string
}

func (f *Fmt) Name() string {
	// this allows this fix config to also be registered under another name so that different
	// configurations can be registered under other linter rule names.
	return cmp.Or(f.NameOverride, "opa-fmt")
}

func (f *Fmt) Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error) {
	if opts == nil {
		return nil, errors.New("runtime options are required")
	}

	if fc.Filename == "" {
		return nil, errors.New("filename is required when formatting")
	}

	popts := parse.ParserOptions()
	if fc.RegoVersion != ast.RegoUndefined {
		popts.RegoVersion = fc.RegoVersion
	}

	module, err := parse.ModuleWithOpts(fc.Filename, fc.Contents, popts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module: %w", err)
	}

	f.OPAFmtOpts.RegoVersion = module.RegoVersion()
	if f.OPAFmtOpts.RegoVersion == ast.RegoV0 {
		f.OPAFmtOpts.RegoVersion = ast.RegoV0CompatV1
	}

	formatted, err := format.AstWithOpts(module, f.OPAFmtOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to format: %w", err)
	}

	formattedStr := string(formatted)
	if fc.Contents == formattedStr {
		return nil, nil
	}

	return []FixResult{{Title: f.Name(), Root: opts.BaseDir, Contents: formattedStr}}, nil
}
