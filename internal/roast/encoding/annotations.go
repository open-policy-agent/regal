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
	write.String(stream, "scope", a.Scope)

	if a.Title != "" {
		write.String(stream, "title", a.Title)
	}

	if a.Description != "" {
		write.String(stream, "description", a.Description)
	}

	if a.Entrypoint {
		write.Bool(stream, "entrypoint", a.Entrypoint)
	}

	if len(a.Organizations) > 0 {
		write.ValsArrayAttr(stream, "organizations", a.Organizations)
	}

	if len(a.RelatedResources) > 0 {
		write.ValsArrayAttr(stream, "related_resources", a.RelatedResources)
	}

	if len(a.Authors) > 0 {
		write.ValsArrayAttr(stream, "authors", a.Authors)
	}

	if len(a.Schemas) > 0 {
		write.ValsArrayAttr(stream, "schemas", a.Schemas)
	}

	if len(a.Custom) > 0 {
		write.Object(stream, "custom", a.Custom)
	}

	write.ObjectEnd(stream)
}
