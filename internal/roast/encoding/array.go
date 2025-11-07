package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
)

type arrayCodec struct{}

func (*arrayCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*arrayCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	arr := *((*ast.Array)(ptr))

	stream.WriteArrayStart()

	for i := range arr.Len() {
		if i > 0 {
			stream.WriteMore()
		}

		stream.WriteVal(arr.Elem(i))
	}

	stream.WriteArrayEnd()
}
