package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type withCodec struct{}

func (*withCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*withCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	with := *((*ast.With)(ptr))

	write.ObjectStart(stream, with.Location)
	write.Val(stream, "target", with.Target)
	write.Val(stream, "value", with.Value)
	write.ObjectEnd(stream)
}
