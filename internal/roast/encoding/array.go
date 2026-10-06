package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"
)

func ArrayMarshalToFn(enc *jsontext.Encoder, arr *ast.Array) error {
	enc.WriteToken(jsontext.BeginArray)

	for i := range arr.Len() {
		if err := TermMarshalToFn(enc, arr.Elem(i)); err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndArray)
}
