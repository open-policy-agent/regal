package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type objectComprehensionCodec struct{}

func (*objectComprehensionCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*objectComprehensionCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	oc := *((*ast.ObjectComprehension)(ptr))

	write.ObjectStart(stream, nil)
	write.Val(stream, "key", oc.Key)
	write.Val(stream, "value", oc.Value)
	write.ValsArrayAttr(stream, "body", oc.Body)
	write.ObjectEnd(stream)
}
