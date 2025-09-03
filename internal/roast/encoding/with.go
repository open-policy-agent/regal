package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type withCodec struct{}

func (*withCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*withCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	with := *((*ast.With)(ptr))

	util.ObjectStart(stream, with.Location)
	util.WriteVal(stream, strTarget, with.Target)
	util.WriteVal(stream, strValue, with.Value)
	util.ObjectEnd(stream)
}
