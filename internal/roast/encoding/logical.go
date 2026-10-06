package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func LogicalAndMarshalToFn(enc *jsontext.Encoder, and *ast.LogicalAnd) error {
	write.ObjectStart(enc, and.Location)
	write.StringField(enc, "type", "and")
	write.WhenTrue(enc, and.ExplicitLhs, "explicit_lhs")
	write.WhenTrue(enc, and.ExplicitRhs, "explicit_rhs")
	write.ArrayFieldFn(enc, "lhs", and.Lhs, ExprMarshalToFn)
	write.ArrayFieldFn(enc, "rhs", and.Rhs, ExprMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func LogicalOrMarshalToFn(enc *jsontext.Encoder, or *ast.LogicalOr) error {
	write.ObjectStart(enc, or.Location)
	write.StringField(enc, "type", "or")
	write.WhenTrue(enc, or.ExplicitLhs, "explicit_lhs")
	write.WhenTrue(enc, or.ExplicitRhs, "explicit_rhs")
	write.ArrayFieldFn(enc, "lhs", or.Lhs, ExprMarshalToFn)
	write.ArrayFieldFn(enc, "rhs", or.Rhs, ExprMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}
