package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type annotationsCodec struct{}

func (*annotationsCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*annotationsCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	a := *((*ast.Annotations)(ptr))

	util.ObjectStart(stream, a.Location)
	util.WriteString(stream, strScope, a.Scope)

	if a.Title != "" {
		util.WriteString(stream, strTitle, a.Title)
	}

	if a.Description != "" {
		util.WriteString(stream, strDescription, a.Description)
	}

	if a.Entrypoint {
		util.WriteBool(stream, strEntrypoint, a.Entrypoint)
	}

	if len(a.Organizations) > 0 {
		util.WriteValsArrayAttr(stream, strOrganizations, a.Organizations)
	}

	if len(a.RelatedResources) > 0 {
		util.WriteValsArrayAttr(stream, strRelatedResources, a.RelatedResources)
	}

	if len(a.Authors) > 0 {
		util.WriteValsArrayAttr(stream, strAuthors, a.Authors)
	}

	if len(a.Schemas) > 0 {
		util.WriteValsArrayAttr(stream, strSchemas, a.Schemas)
	}

	if len(a.Custom) > 0 {
		util.WriteObject(stream, strCustom, a.Custom)
	}

	util.ObjectEnd(stream)
}
