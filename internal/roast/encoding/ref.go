package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func RefMarshalToFn(enc *jsontext.Encoder, r ast.Ref) (err error) {
	return write.ArrayToFn(enc, r, TermMarshalToFn)
}

func CallMarshalToFn(enc *jsontext.Encoder, call ast.Call) (err error) {
	return write.ArrayToFn(enc, call, TermMarshalToFn)
}
