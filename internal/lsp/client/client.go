package client

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/roast/encoding/read"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

//nolint:recvcheck // UnmarshalJSONFrom must use a pointer receiver
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

func (c *Client) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return read.Object(dec, func(dec *jsontext.Decoder, key string) (err error) {
		switch key {
		case "identifier":
			err = json.UnmarshalDecode(dec, &c.Identifier)
		case "init_options", "initializationOptions":
			err = json.UnmarshalDecode(dec, &c.InitOptions)
		case "capabilities":
			err = json.UnmarshalDecode(dec, &c.Capabilities, encoding.NoASTOptions)
		default:
			err = read.Unknown(dec, key)
		}

		return err
	})
}
