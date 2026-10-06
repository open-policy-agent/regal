package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func SetMarshalToFn(enc *jsontext.Encoder, s ast.Set) (err error) {
	return write.ArrayToFn(enc, s.Slice(), TermMarshalToFn)
}
