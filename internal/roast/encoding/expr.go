package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func ExprMarshalToFn(enc *jsontext.Encoder, expr *ast.Expr) error {
	return writeExpr(enc, expr, false)
}

func writeExpr(enc *jsontext.Encoder, expr *ast.Expr, interpolated bool) (err error) {
	write.ObjectStart(enc, expr.Location)
	write.WhenTrue(enc, expr.Negated, "negated")
	write.WhenTrue(enc, expr.Generated, "generated")
	write.WhenTrue(enc, interpolated, "interpolated")

	if len(expr.With) > 0 {
		write.ArrayFieldFn(enc, "with", expr.With, WithMarshalToFn)
	}

	if expr.Terms != nil {
		enc.WriteToken(jsontext.String("terms"))

		switch t := expr.Terms.(type) {
		case *ast.Term:
			err = TermMarshalToFn(enc, t)
		case []*ast.Term:
			err = write.ArrayToFn(enc, t, TermMarshalToFn)
		case *ast.SomeDecl:
			err = SomeDeclMarshalToFn(enc, t)
		case *ast.Every:
			err = EveryMarshalToFn(enc, t)
		case *ast.Not:
			err = NotMarshalToFn(enc, t)
		case *ast.LogicalAnd:
			err = LogicalAndMarshalToFn(enc, t)
		case *ast.LogicalOr:
			err = LogicalOrMarshalToFn(enc, t)
		}

		if err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndObject)
}
