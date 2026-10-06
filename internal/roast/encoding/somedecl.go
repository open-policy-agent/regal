package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func SomeDeclMarshalToFn(enc *jsontext.Encoder, some *ast.SomeDecl) error {
	write.ObjectStart(enc, some.Location)
	write.ArrayFieldFn(enc, "symbols", some.Symbols, TermMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}
