package types

import (
	"encoding/json"
	"errors"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/roast/transforms"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
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
type RefKind uint8

const (
	Package RefKind = iota + 1
	Rule
	ConstantRule
	Function
)

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
	Identifier   clients.Identifier    `json:"identifier"`
	InitOptions  InitializationOptions `json:"init_options"`
	Capabilities ast.Value             `json:"capabilities,omitempty"`
}

func (c *Client) UnmarshalJSON(data []byte) (err error) {
	var m map[string]any
	if err := encoding.SafeNumberConfig.Unmarshal(data, &m); err != nil {
		return err
	}

	idNum, ok := m["identifier"].(json.Number)
	if !ok {
		return errors.New("invalid identifier type")
	}

	idInt, _ := idNum.Int64()

	c.Identifier = clients.Identifier(util.SafeIntToUint(int(idInt))) //nolint: gosec

	if initOptions, ok := m["initializationOptions"]; ok {
		if err := encoding.JSONRoundTrip(initOptions, &c.InitOptions); err != nil {
			return err
		}
	}

	c.Capabilities, err = transforms.AnyToValue(m["capabilities"])

	return err
}

// ServerContext is a type which is used to contain things from the server's
// state that is needed in RegalContext.
type ServerContext struct {
	FeatureFlags ServerFeatureFlags `json:"feature_flags"`
	Version      string             `json:"version"`
}
