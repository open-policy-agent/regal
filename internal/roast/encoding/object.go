package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/funsafe"
	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func ObjectMarshalToFn(enc *jsontext.Encoder, obj ast.Object) error {
	enc.WriteToken(jsontext.BeginArray)

	elems := funsafe.ObjectElems(obj)
	for i := range elems {
		write.FieldFn(enc, "key", elems[i].Key(), TermMarshalToFn)
		write.FieldFn(enc, "value", elems[i].Value(), TermMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndArray)
}
