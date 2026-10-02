package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func ImportMarshalToFn(enc *jsontext.Encoder, imp *ast.Import) (err error) {
	write.ObjectStart(enc, imp.Location)

	if imp.Path != nil {
		write.FieldFn(enc, "path", imp.Path, TermMarshalToFn)

		if imp.Alias != "" {
			enc.WriteToken(jsontext.String("alias"))
			_ = imp.Alias.MarshalJSONTo(enc)
		}
	}

	return enc.WriteToken(jsontext.EndObject)
}
