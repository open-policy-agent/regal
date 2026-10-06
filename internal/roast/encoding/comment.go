package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"
)

func CommentMarshalToFn(enc *jsontext.Encoder, c *ast.Comment) error {
	// Marshal only location string — text is retrieved dynamically via regal.file.lines
	return LocationMarshalToFn(enc, c.Location)
}
