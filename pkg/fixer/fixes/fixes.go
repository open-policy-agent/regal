package fixes

import (
	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/report"
)

// NewDefaultFixes returns a list of default fixes that are applied by the fix command.
// When a new fix is added, it should be added to this list.
func NewDefaultFixes() []Fix {
	return []Fix{
		&Fmt{},
		&Fmt{
			// this effectively maps the fix for violations from the
			// use-rego-v1 rule to just format the file.
			NameOverride: "use-rego-v1",
		},
		&UseAssignmentOperator{},
		&NoWhitespaceComment{},
		&DirectoryPackageMismatch{},
		&NonRawRegexPattern{},
		&PreferEqualsComparison{},
	}
}

// NewDefaultFormatterFixes returns a list of default fixes that are applied by the formatter.
// Notably, this does not include fixers that move files around.
func NewDefaultFormatterFixes() []Fix {
	return []Fix{
		&Fmt{},
		&UseAssignmentOperator{},
		&NoWhitespaceComment{},
		&NonRawRegexPattern{},
		&PreferEqualsComparison{},
	}
}

// Fix is the interface that must be implemented by all fixes.
type Fix interface {
	// Name returns the unique name for the fix, this should correlate with the
	// violation title & rule name that the fix is meant to address.
	Name() string
	Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error)
}

// RuntimeOptions are the options that are passed to the Fix method when the Fix is executed.
// Location based fixes will have the locations populated by the caller.
type RuntimeOptions struct {
	Config *config.Config
	// BaseDir is the base directory for the files being fixed. This is often the same as the
	// workspace root directory, but not necessarily.
	BaseDir   string
	Locations []report.Location
	Client    clients.Identifier
}

// FixCandidate is the input to a Fix method and represents a file in need of fixing.
type FixCandidate struct {
	Filename    string
	Contents    string
	RegoVersion ast.RegoVersion
}

// Rename represents a file that has been moved (renamed).
type Rename struct {
	FromPath string
	ToPath   string
}

// FixResult is returned from the Fix method and contains the new contents or fix recommendations.
// In future this might support diff based updates.
type FixResult struct {
	// Rename is used to indicate that a rename operation should be performed by the **caller**.
	// An example of this would be the DirectoryPackageMismatch fix, which in the context of
	// `regal fix` renames files as part of the fix, while when invoked as a LSP Code Action will
	// defer the actual rename back to the client.
	Rename *Rename
	// Title is the name of the fix applied.
	Title string
	// Root is the project root of the file fixed. This is persisted for presentation purposes,
	// as it makes it easier to understand the context of the fix.
	Root string
	// Contents is the new contents of the file. May be nil or identical to the original contents,
	// as not all fixes involve content changes. It is the responsibility of the caller to handle
	// this.
	Contents string
}
