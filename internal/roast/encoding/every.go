package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func EveryMarshalToFn(enc *jsontext.Encoder, every *ast.Every) error {
	write.ObjectStart(enc, every.Location)

	if every.Key != nil {
		write.FieldFn(enc, "key", every.Key, TermMarshalToFn)
	}

	write.FieldFn(enc, "value", every.Value, TermMarshalToFn)
	write.FieldFn(enc, "domain", every.Domain, TermMarshalToFn)
	write.FieldFn(enc, "body", every.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}
