package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type packageCodec struct{}

func (*packageCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*packageCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	pkg := *((*ast.Package)(ptr))

	util.ObjectStart(stream, pkg.Location)

	if pkg.Path != nil {
		// Make a copy to avoid data race
		// https://github.com/open-policy-agent/regal/issues/1167
		pathCopy := pkg.Path.Copy()

		// Omit location of "data" part of path, at it isn't present in code
		pathCopy[0].Location = nil

		util.WriteVal(stream, strPath, pathCopy)
	}

	if stream.Attachment != nil {
		util.WriteVal(stream, strAnnotations, stream.Attachment)
	}

	util.ObjectEnd(stream)
}
