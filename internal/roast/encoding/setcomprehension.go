package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
)

type setComprehensionCodec struct{}

func (*setComprehensionCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*setComprehensionCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	sc := *((*ast.SetComprehension)(ptr))

	stream.WriteObjectStart()
	stream.WriteObjectField(strTerm)
	stream.WriteVal(sc.Term)
	stream.WriteMore()
	stream.WriteObjectField(strBody)
	stream.WriteVal(sc.Body)
	stream.WriteObjectEnd()
}
