package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func WithMarshalToFn(enc *jsontext.Encoder, with *ast.With) error {
	write.ObjectStart(enc, with.Location)
	write.FieldFn(enc, "target", with.Target, TermMarshalToFn)
	write.FieldFn(enc, "value", with.Value, TermMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}
