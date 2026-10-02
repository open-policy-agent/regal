package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func ArrayComprehensionMarshalToFn(enc *jsontext.Encoder, ac *ast.ArrayComprehension) error {
	enc.WriteToken(jsontext.BeginObject)
	write.FieldFn(enc, "term", ac.Term, TermMarshalToFn)
	write.FieldFn(enc, "body", ac.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func SetComprehensionMarshalToFn(enc *jsontext.Encoder, sc *ast.SetComprehension) error {
	enc.WriteToken(jsontext.BeginObject)
	write.FieldFn(enc, "term", sc.Term, TermMarshalToFn)
	write.FieldFn(enc, "body", sc.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func ObjectComprehensionMarshalToFn(enc *jsontext.Encoder, oc *ast.ObjectComprehension) error {
	enc.WriteToken(jsontext.BeginObject)
	write.FieldFn(enc, "key", oc.Key, TermMarshalToFn)
	write.FieldFn(enc, "value", oc.Value, TermMarshalToFn)
	write.FieldFn(enc, "body", oc.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}
