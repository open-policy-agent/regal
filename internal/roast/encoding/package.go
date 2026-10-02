package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func PackageMarshalToFn(enc *jsontext.Encoder, p *ast.Package) (err error) {
	return writePackage(enc, p, nil)
}

var pkgDataTerm = jsontext.Value(`{"type":"var","value":"data"}`)

func writePackage(enc *jsontext.Encoder, p *ast.Package, annotations []*ast.Annotations) (err error) {
	write.ObjectStart(enc, p.Location)

	if path := p.Path; len(path) > 0 {
		write.Tokens(enc, jsontext.String("path"), jsontext.BeginArray)

		if value, ok := path[0].Value.(ast.Var); ok && value == "data" {
			enc.WriteValue(pkgDataTerm)

			path = path[1:]
		}

		for _, term := range path {
			if err := TermMarshalToFn(enc, term); err != nil {
				return err
			}
		}

		enc.WriteToken(jsontext.EndArray)
	}

	if len(annotations) > 0 {
		write.ArrayFieldFn(enc, "annotations", annotations, AnnotationsMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}
