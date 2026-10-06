package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func AnnotationsMarshalToFn(enc *jsontext.Encoder, a *ast.Annotations) error {
	write.ObjectStart(enc, a.Location)
	write.StringField(enc, "scope", a.Scope)

	if a.Title != "" {
		write.StringField(enc, "title", a.Title)
	}

	if a.Description != "" {
		write.StringField(enc, "description", a.Description)
	}

	write.WhenTrue(enc, a.Entrypoint, "entrypoint")

	if len(a.Organizations) > 0 {
		write.ArrayField(enc, "organizations", a.Organizations)
	}

	if len(a.RelatedResources) > 0 {
		write.ArrayField(enc, "related_resources", a.RelatedResources)
	}

	if len(a.Authors) > 0 {
		write.ArrayField(enc, "authors", a.Authors)
	}

	if len(a.Schemas) > 0 {
		write.ArrayField(enc, "schemas", a.Schemas)
	}

	if len(a.Custom) > 0 {
		write.Tokens(enc, jsontext.String("custom"), jsontext.BeginObject)

		for k, v := range a.Custom {
			_ = write.Field(enc, k, v)
		}

		enc.WriteToken(jsontext.EndObject)
	}

	return enc.WriteToken(jsontext.EndObject)
}
