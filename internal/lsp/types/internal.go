package types

import (
	"encoding/json"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
)

// Ref is a generic construct for an object found in a Rego module.
// Ref is designed to be used in completions and provides information
// relevant to the object with that operation in mind.
type Ref struct {
	// Label is a identifier for the object. e.g. data.package.rule.
	Label string `json:"label"`
	// Detail is a small amount of additional information about the object.
	Detail string `json:"detail"`
	// Description is a longer description of the object and uses Markdown formatting.
	Description string  `json:"description"`
	Kind        RefKind `json:"kind"`
}

// RefKind represents the kind of object that a Ref represents.
// This is intended to toggle functionality and which UI symbols to use.
type RefKind int

const (
	Package RefKind = iota + 1
	Rule
	ConstantRule
	Function
)

type BuiltinPosition struct {
	Builtin *ast.Builtin
	Line    uint
	Start   uint
	End     uint
}

type KeywordLocation struct {
	Name  string
	Line  uint
	Start uint
	End   uint
}

type CommandArgs struct {
	// Target is the URI of the document for which the command applies to
	Target string `json:"target"`

	// Optional arguments, command dependent
	// Diagnostic is the diagnostic that is to be fixed in the target
	Diagnostic *Diagnostic `json:"diagnostic,omitempty"`
	// Query is the query to evaluate
	Query string `json:"path,omitempty"`
	// Row is the row within the file where the command was run from
	Row int `json:"row,omitempty"`
}

type Client struct {
	Identifier   clients.Identifier     `json:"identifier"`
	InitOptions  *InitializationOptions `json:"init_options,omitempty"`
	Capabilities *json.RawMessage       `json:"capabilities,omitempty"`
}

func NewGenericClient() Client {
	return Client{Identifier: clients.IdentifierGeneric}
}
