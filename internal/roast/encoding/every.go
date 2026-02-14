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
	write.Val(stream, "key", every.Key)
	write.Val(stream, "value", every.Value)
	write.Val(stream, "domain", every.Domain)
	write.Val(stream, "body", every.Body)
	write.ObjectEnd(stream)
}
