package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type someDeclCodec struct{}

func (*someDeclCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*someDeclCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	some := *((*ast.SomeDecl)(ptr))

	write.ObjectStart(stream, some.Location)
	write.ValsArrayAttr(stream, "symbols", some.Symbols)
	write.ObjectEnd(stream)
}
