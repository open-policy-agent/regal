package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type arrayComprehensionCodec struct{}

func (*arrayComprehensionCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*arrayComprehensionCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	ac := *((*ast.ArrayComprehension)(ptr))

	stream.WriteObjectStart()

	stream.WriteObjectField("term")
	write.Term(stream, ac.Term)
	stream.WriteMore()
	stream.WriteObjectField("body")
	write.ValsArray(stream, ac.Body)

	stream.WriteObjectEnd()
}
