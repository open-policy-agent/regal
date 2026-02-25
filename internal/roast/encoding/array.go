package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
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

		write.Term(stream, arr.Elem(i))
	}

	stream.WriteArrayEnd()
}
