package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func NotMarshalToFn(enc *jsontext.Encoder, not *ast.Not) error {
	write.ObjectStart(enc, not.Location)
	write.StringField(enc, "type", "not")
	write.WhenTrue(enc, not.ExplicitBody, "explicit_body")

	enc.WriteToken(jsontext.String("body"))
	write.ArrayToFn(enc, not.Body, ExprMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}
