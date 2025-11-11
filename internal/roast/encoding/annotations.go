package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type annotationsCodec struct{}

func (*annotationsCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*annotationsCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	a := *((*ast.Annotations)(ptr))

	write.ObjectStart(stream, a.Location)
	write.String(stream, strScope, a.Scope)

	if a.Title != "" {
		write.String(stream, strTitle, a.Title)
	}

	if a.Description != "" {
		write.String(stream, strDescription, a.Description)
	}

	if a.Entrypoint {
		write.Bool(stream, strEntrypoint, a.Entrypoint)
	}

	if len(a.Organizations) > 0 {
		write.ValsArrayAttr(stream, strOrganizations, a.Organizations)
	}

	if len(a.RelatedResources) > 0 {
		write.ValsArrayAttr(stream, strRelatedResources, a.RelatedResources)
	}

	if len(a.Authors) > 0 {
		write.ValsArrayAttr(stream, strAuthors, a.Authors)
	}

	if len(a.Schemas) > 0 {
		write.ValsArrayAttr(stream, strSchemas, a.Schemas)
	}

	if len(a.Custom) > 0 {
		write.Object(stream, strCustom, a.Custom)
	}

	write.ObjectEnd(stream)
}
