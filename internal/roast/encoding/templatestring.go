package encoding

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func TemplateStringMarshalToFn(enc *jsontext.Encoder, ts *ast.TemplateString) (err error) {
	enc.WriteToken(jsontext.BeginObject)
	write.WhenTrue(enc, ts.MultiLine, "multi_line")
	write.Tokens(enc, jsontext.String("parts"), jsontext.BeginArray)

	for _, part := range ts.Parts {
		if expr, ok := part.(*ast.Expr); ok {
			err = writeExpr(enc, expr, true)
		} else {
			err = json.MarshalEncode(enc, part)
		}

		if err != nil {
			return err
		}
	}

	return write.Tokens(enc, jsontext.EndArray, jsontext.EndObject)
}
