package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type importCodec struct{}

func (*importCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*importCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	imp := *((*ast.Import)(ptr))

	util.ObjectStart(stream, imp.Location)

	if imp.Path != nil {
		util.WriteVal(stream, strPath, imp.Path)

		if imp.Alias != "" {
			util.WriteVal(stream, strAlias, imp.Alias)
		}
	}

	util.ObjectEnd(stream)
}
