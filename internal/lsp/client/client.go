//nolint:recvcheck
package client

import (
	"encoding/json"
	"errors"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/roast/transforms"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

type Client struct {
	Identifier   clients.Identifier          `json:"identifier"`
	InitOptions  types.InitializationOptions `json:"init_options"`
	Capabilities ast.Value                   `json:"capabilities,omitempty"`

	conn *jsonrpc2.Conn
}

func NewGeneric() Client {
	return Client{Identifier: clients.IdentifierGeneric}
}

func (c Client) URIFromPath(path string) string {
	return uri.FromPath(c.Identifier, path)
}

func (c Client) URIFromRelativePath(relPath, rootURI string) string {
	return uri.FromRelativePath(c.Identifier, relPath, rootURI)
}

func (c Client) Connection() *jsonrpc2.Conn {
	return c.conn
}

func (c Client) WithConnection(conn *jsonrpc2.Conn) Client {
	c.conn = conn

	return c
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
