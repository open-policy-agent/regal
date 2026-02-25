package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type everyCodec struct{}

func (*everyCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*everyCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	every := *((*ast.Every)(ptr))

	write.ObjectStart(stream, every.Location)

	if every.Key != nil {
		stream.WriteObjectField("key")
		write.Term(stream, every.Key)
		stream.WriteMore()
	}

	stream.WriteObjectField("value")
	write.Term(stream, every.Value)
	stream.WriteMore()

	stream.WriteObjectField("domain")
	write.Term(stream, every.Domain)
	stream.WriteMore()

	stream.WriteObjectField("body")
	write.ValsArray(stream, every.Body)
	stream.WriteObjectEnd()
}
