package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type setComprehensionCodec struct{}

func (*setComprehensionCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*setComprehensionCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	sc := *((*ast.SetComprehension)(ptr))

	stream.WriteObjectStart()
	stream.WriteObjectField("term")
	write.Term(stream, sc.Term)
	stream.WriteMore()
	stream.WriteObjectField("body")
	write.ValsArray(stream, sc.Body)
	stream.WriteObjectEnd()
}
